/*
Copyright 2019 Tom Peters

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
		authz := r.Header.Get("Authorization")
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) != 2 || !strings.HasPrefix(strings.ToLower(parts[0]), "bearer") {
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

		// Extract and store email from JWT if present (Auth0 tokens include email)
		if issuer == model.IssuerAuth0 {
			if email, ok := token.Claims.(jwt.MapClaims)["email"].(string); ok && email != "" {
				// Only update if email is different or not stored yet
				if user.Email == nil || *user.Email != email {
					if err := user.SetEmail(r.Context(), email); err != nil {
						logrus.WithError(err).Warn("could not save user email")
					}
				}
			}
		}

		ctx := context.WithValue(r.Context(), ctxUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
