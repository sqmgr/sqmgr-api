/*
Copyright (C) 2019 Tom Peters

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

const sseKeepaliveInterval = 30 * time.Second

func (s *Server) getPoolTokenEventsEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		poolToken := mux.Vars(r)["token"]

		// Authenticate via query parameter since EventSource doesn't support headers
		accessToken := r.FormValue("access_token")
		if accessToken == "" {
			s.writeErrorResponse(w, http.StatusUnauthorized, nil)
			return
		}

		// Validate the JWT (same logic as authHandler but inline)
		user, err := s.authenticateToken(r, accessToken)
		if err != nil {
			logrus.WithError(err).Debug("SSE auth failed")
			s.writeErrorResponse(w, http.StatusUnauthorized, nil)
			return
		}

		// Load pool and verify membership
		pool, err := s.model.PoolByToken(r.Context(), poolToken)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if !user.IsSiteAdmin {
			isMember, err := user.IsMemberOf(r.Context(), pool)
			if err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
			if !isMember {
				// Check if auto-join is possible
				if (pool.IsLocked() && pool.OpenAccessOnLock()) || !pool.PasswordRequired() {
					if err := user.JoinPool(r.Context(), pool); err != nil {
						s.writeErrorResponse(w, http.StatusInternalServerError, err)
						return
					}
				} else {
					s.writeErrorResponse(w, http.StatusForbidden, nil)
					return
				}
			}
		}

		// Verify the response writer supports flushing
		flusher, ok := w.(http.Flusher)
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, errors.New("streaming not supported"))
			return
		}

		// Clear the write deadline so the server's WriteTimeout doesn't
		// kill this long-lived connection after a few seconds.
		rc := http.NewResponseController(w)
		if err := rc.SetWriteDeadline(time.Time{}); err != nil {
			logrus.WithError(err).Error("could not clear write deadline for SSE")
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		// Subscribe to pool events
		ch := s.broker.Subscribe(poolToken)
		defer s.broker.Unsubscribe(poolToken, ch)

		keepalive := time.NewTicker(sseKeepaliveInterval)
		defer keepalive.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case event, ok := <-ch:
				if !ok {
					return
				}
				data, err := json.Marshal(event)
				if err != nil {
					logrus.WithError(err).Error("could not marshal SSE event")
					continue
				}
				if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data); err != nil {
					return
				}
				flusher.Flush()
			case <-keepalive.C:
				if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

// authenticateToken validates a JWT and returns the associated user
func (s *Server) authenticateToken(r *http.Request, rawToken string) (*model.User, error) {
	issuer := ""
	token, err := jwt.Parse(rawToken, func(token *jwt.Token) (interface{}, error) {
		claims := token.Claims.(jwt.MapClaims)

		// Check audience
		audMatched := false
		if audSlice, ok := claims["aud"].([]interface{}); ok {
			for _, iAud := range audSlice {
				if aud, _ := iAud.(string); aud == audienceSqMGR {
					audMatched = true
					break
				}
			}
		} else if aud, ok := claims["aud"].(string); ok && aud == audienceSqMGR {
			audMatched = true
		}

		if !audMatched {
			return nil, errors.New("invalid audience")
		}

		if iss, ok := claims["iss"].(string); ok {
			if iss == model.IssuerAuth0 {
				issuer = model.IssuerAuth0
				cert, err := s.keyLocker.GetPEMCert(token)
				if err != nil {
					return nil, err
				}
				return jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			} else if iss == model.IssuerSqMGR {
				issuer = model.IssuerSqMGR
				return s.smjwt.PublicKey(), nil
			}
		}

		return nil, errors.New("invalid issuer")
	})

	if err != nil {
		return nil, fmt.Errorf("validating token: %w", err)
	}

	sub, ok := token.Claims.(jwt.MapClaims)["sub"].(string)
	if !ok {
		return nil, errors.New("token missing sub claim")
	}

	if issuer == "" {
		return nil, errors.New("issuer could not be determined")
	}

	user, err := s.model.GetUser(r.Context(), issuer, sub)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	// Check guest user expiration
	if user.Store == model.UserStoreSqMGR {
		expired, err := s.model.IsGuestUserExpired(r.Context(), user.Store, user.StoreID)
		if err != nil {
			return nil, fmt.Errorf("checking guest expiration: %w", err)
		}
		if expired {
			return nil, errors.New("guest account has expired")
		}
	}

	return user, nil
}

// extractBearerToken extracts the token from an Authorization header
func extractBearerToken(header string) (string, bool) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	return parts[1], true
}
