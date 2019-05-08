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
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"strconv"
	"time"
)

type claims struct {
	jwt.StandardClaims
	UserID interface{}
}

func (s *Server) tokensSessionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := s.EffectiveUser(r)
		if err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		var id string
		switch val := user.UserID(r.Context()).(type) {
		case string:
			id = val
		case int64:
			id = strconv.FormatInt(val, 10)
		default:
			panic(fmt.Sprintf("unknown type: %T", val))
		}

		jwtStr, err := s.jwt.Sign(jwt.StandardClaims{
			Id:        id,
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		})
		if err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		s.ServeJSON(w, http.StatusOK, jsonResponse{
			Status: responseOK,
			Result: jwtStr,
		})
		return
	}
}
