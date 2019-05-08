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
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr/pkg/smjwt"
	"net/http"
	"strings"
)

func (s *Server) apiGridSquaresHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authzHeader := r.Header.Get("Authorization")
		if authzHeader == "" {
			s.ServeJSONError(w, http.StatusUnauthorized, "Bearer token not provided")
			return
		}

		authzHeaderParts := strings.Split(authzHeader, " ")
		if len(authzHeaderParts) != 2 || strings.ToLower(authzHeaderParts[0]) != "bearer" {
			s.ServeJSONError(w, http.StatusUnauthorized, "Bearer token not provided")
			return
		}

		tokenStr := authzHeaderParts[1]
		token, err := s.jwt.Validate(tokenStr)
		if err != nil {
			if err != smjwt.ErrExpired {
				logrus.WithError(err).Error("unexpected error from smjwt.Validate()")
			}
			s.ServeJSONError(w, http.StatusUnauthorized, "")
		}

		claims, _ := token.Claims.(*jwt.StandardClaims)
		logrus.Info(claims.Id)
	}
}
