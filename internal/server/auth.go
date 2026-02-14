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
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

const audienceSqMGR = "api.sqmgr.com"

func (s *Server) authHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		// Check auth rate limit before processing
		ip := getIP(r)
		if s.authRateLimiter.IsLimited(ip) {
			s.writeErrorResponse(w, http.StatusTooManyRequests, errors.New("too many failed authentication attempts"))
			return
		}

		authz := r.Header.Get("Authorization")
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) != 2 || !strings.HasPrefix(strings.ToLower(parts[0]), "bearer") {
			s.authRateLimiter.RecordFailure(ip)
			s.writeErrorResponse(w, http.StatusUnauthorized, nil)
			return
		}

		issuer := ""
		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			claims := token.Claims.(jwt.MapClaims)

			// Check audience - handle both string and array formats
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

			// Check issuer and return appropriate key
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
			logrus.WithError(err).Warn("could not validate token")
			s.authRateLimiter.RecordFailure(ip)
			s.writeErrorResponse(w, http.StatusUnauthorized, nil)
			return
		}

		sub, ok := token.Claims.(jwt.MapClaims)["sub"].(string)
		if !ok {
			logrus.Error("token did not have sub")
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
		}

		if issuer == "" {
			s.writeErrorResponse(w, http.StatusInternalServerError, errors.New("issuer could not be determined"))
			return
		}

		user, err := s.model.GetUser(r.Context(), issuer, sub)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		user.Token = token

		// Check guest user expiration
		if user.Store == model.UserStoreSqMGR {
			expired, err := s.model.IsGuestUserExpired(r.Context(), user.Store, user.StoreID)
			if err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
			if expired {
				s.authRateLimiter.RecordFailure(ip)
				s.writeErrorResponse(w, http.StatusUnauthorized, errors.New("guest account has expired"))
				return
			}
		}

		// Extract and store email for Auth0 users from namespaced claim
		if issuer == model.IssuerAuth0 && user.Email == nil {
			email, _ := token.Claims.(jwt.MapClaims)[model.ClaimNamespace+"/email"].(string)
			if email != "" {
				if err := user.SetEmail(r.Context(), email); err != nil {
					logrus.WithError(err).Warn("could not save user email")
				}
			}
		}

		ctx := context.WithValue(r.Context(), ctxUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
