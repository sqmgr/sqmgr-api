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
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr-api/internal/model"
	"net/http"
	"strings"
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

			audSlice, ok := claims["aud"].([]interface{})
			if ok {
				audMatched := false
				for _, iAud := range audSlice {
					aud, _ := iAud.(string)
					if aud == audienceSqMGR {
						audMatched = true
						break
					}
				}

				if !audMatched {
					return token, errors.New("invalid audience")
				}
			} else if !claims.VerifyAudience(audienceSqMGR, true) {
				return token, errors.New("invalid audience")
			}

			if claims.VerifyIssuer(model.IssuerAuth0, true) {
				issuer = model.IssuerAuth0
				cert, err := s.keyLocker.GetPEMCert(token)
				if err != nil {
					return token, err
				}
				return jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			} else if claims.VerifyIssuer(model.IssuerSqMGR, true) {
				issuer = model.IssuerSqMGR
				return s.smjwt.PublicKey(), nil
			}

			return token, errors.New("invalid issuer")
		})

		if err != nil {
			logrus.WithError(err).WithField("token", parts[1]).Warn("could not validate token")
			s.writeErrorResponse(w, http.StatusUnauthorized, nil)
			return
		}

		sub, ok := token.Claims.(jwt.MapClaims)["sub"].(string)
		if !ok {
			logrus.WithField("token", parts[1]).Error("token did not have sub")
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
		ctx := context.WithValue(r.Context(), ctxUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
