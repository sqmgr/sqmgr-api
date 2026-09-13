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
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

const audienceSqMGR = "api.sqmgr.com"

// authOptions controls how a bearer token is validated and resolved to a user.
type authOptions struct {
	// audiences lists the "aud" values accepted, in order of preference. A
	// token matching mcpAudience is an OAuth access token issued by this API
	// for the MCP endpoint; anything else is a regular API token.
	audiences []string
	// mcpAudience, when non-empty, is the audience of MCP access tokens.
	mcpAudience string
	// resourceMetadataURL, when non-empty, is advertised in the
	// WWW-Authenticate header of 401 responses so OAuth-capable clients can
	// discover how to obtain a token.
	resourceMetadataURL string
}

// authHandler authenticates regular API requests with an Auth0 or SqMGR guest JWT.
func (s *Server) authHandler(next http.Handler) http.Handler {
	return s.authMiddleware(next, authOptions{audiences: []string{audienceSqMGR}})
}

// mcpAuthHandler authenticates requests to the MCP endpoint. It accepts the
// OAuth access tokens issued for that endpoint as well as regular API JWTs.
func (s *Server) mcpAuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.authMiddleware(next, authOptions{
			audiences:           []string{mcpResourceURL(), audienceSqMGR},
			mcpAudience:         mcpResourceURL(),
			resourceMetadataURL: mcpResourceMetadataURL(),
		}).ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler, opts authOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		user, ok := s.authenticate(w, r, opts)
		if !ok {
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// unauthorized writes a 401, advertising the resource metadata URL when the
// route has one, and counts the failure against the caller's IP.
func (s *Server) unauthorized(w http.ResponseWriter, ip string, opts authOptions, err error) {
	s.authRateLimiter.RecordFailure(ip)
	if opts.resourceMetadataURL != "" {
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+opts.resourceMetadataURL+`"`)
	}
	s.writeErrorResponse(w, http.StatusUnauthorized, err)
}

// authenticate validates the request's bearer token and resolves it to a
// user. On failure it writes the response and returns false.
func (s *Server) authenticate(w http.ResponseWriter, r *http.Request, opts authOptions) (*model.User, bool) {
	// Check auth rate limit before processing
	ip := getIP(r)
	if s.authRateLimiter.IsLimited(ip) {
		s.writeErrorResponse(w, http.StatusTooManyRequests, errors.New("too many failed authentication attempts"))
		return nil, false
	}

	authz := r.Header.Get("Authorization")
	parts := strings.SplitN(authz, " ", 2)
	if len(parts) != 2 || !strings.HasPrefix(strings.ToLower(parts[0]), "bearer") {
		s.unauthorized(w, ip, opts, nil)
		return nil, false
	}

	issuer := ""
	audience := ""
	token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
		claims := token.Claims.(jwt.MapClaims)

		// Check audience - handle both string and array formats
		audience = matchAudience(claims["aud"], opts.audiences)
		if audience == "" {
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
		s.unauthorized(w, ip, opts, nil)
		return nil, false
	}

	sub, ok := token.Claims.(jwt.MapClaims)["sub"].(string)
	if !ok {
		logrus.Error("token did not have sub")
		s.writeErrorResponse(w, http.StatusInternalServerError, nil)
		return nil, false
	}

	if issuer == "" {
		s.writeErrorResponse(w, http.StatusInternalServerError, errors.New("issuer could not be determined"))
		return nil, false
	}

	// MCP access tokens are minted by this API for a specific user ID and
	// never create accounts.
	if opts.mcpAudience != "" && audience == opts.mcpAudience {
		if issuer != model.IssuerSqMGR {
			s.unauthorized(w, ip, opts, nil)
			return nil, false
		}
		userID, err := strconv.ParseInt(sub, 10, 64)
		if err != nil {
			s.unauthorized(w, ip, opts, nil)
			return nil, false
		}
		user, err := s.model.GetUserByID(r.Context(), userID)
		if err != nil {
			if isNotFound(err) {
				s.unauthorized(w, ip, opts, nil)
				return nil, false
			}
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return nil, false
		}
		user.Token = token
		return user, true
	}

	user, err := s.model.GetUser(r.Context(), issuer, sub)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, err)
		return nil, false
	}

	user.Token = token

	// Check guest user expiration
	if user.Store == model.UserStoreSqMGR {
		expired, err := s.model.IsGuestUserExpired(r.Context(), user.Store, user.StoreID)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return nil, false
		}
		if expired {
			s.unauthorized(w, ip, opts, errors.New("guest account has expired"))
			return nil, false
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

	return user, true
}

// matchAudience returns the first accepted audience present in the token's
// "aud" claim, which may be a string or an array, or "" when none match.
func matchAudience(aud interface{}, accepted []string) string {
	var present []string
	switch v := aud.(type) {
	case string:
		present = []string{v}
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				present = append(present, str)
			}
		}
	}

	for _, want := range accepted {
		for _, have := range present {
			if have == want {
				return want
			}
		}
	}
	return ""
}
