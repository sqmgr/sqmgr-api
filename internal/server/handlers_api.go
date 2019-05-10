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
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/internal/validator"
	"github.com/weters/sqmgr/pkg/smjwt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) apiGridLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)

		if !jcd.Claim.IsAdmin {
			s.ServeJSONError(w, http.StatusUnauthorized, "")
			return

		}

		grid, err := jcd.Grid.Logs(r.Context(), 0, 1000) // TODO: pagination???
		if err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		res := jsonResponse{
			Status: responseOK,
			Result: grid,
		}

		s.ServeJSON(w, http.StatusOK, res)
		return
	}
}

func (s *Server) apiGridSquaresHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)

		squares, err := jcd.Grid.Squares()
		if err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		s.ServeJSON(w, http.StatusOK, jsonResponse{
			Status: responseOK,
			Result: squares,
		})
	}
}

// GET|POST /api/grid/squares/:square
// may want to refactor since this handler is pretty complex
func (s *Server) apiGridSquaresSquareHandler() http.HandlerFunc {
	type postPayload struct {
		Claimant string                `json:"claimant"`
		State    model.GridSquareState `json:"state"`
		Note     string                `json:"note"`
		Unclaim  bool                  `json:"unclaim"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)
		squareID, _ := strconv.Atoi(mux.Vars(r)["square"])
		square, err := jcd.Grid.SquareBySquareID(squareID)
		if err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		if r.Method == http.MethodPost {
			dec := json.NewDecoder(r.Body)
			var payload postPayload
			if err := dec.Decode(&payload); err != nil {
				s.ServeJSONError(w, http.StatusInternalServerError, "", err)
				return
			}

			// int64 will decode into float64 if the type is interface{}. Need to convert back
			// to int64 if this is the case
			userID := jcd.Claim.EffectiveUserID
			if val, ok := userID.(float64); ok {
				userID = int64(val)
			}

			if len(payload.Claimant) > 0 {
				v := validator.New()
				claimant := v.Printable("name", payload.Claimant)
				claimant = v.ContainsWordChar("name", claimant)

				if !v.OK() {
					s.ServeJSONError(w, http.StatusBadRequest, v.String())
					return
				}

				square.Claimant = claimant
				square.State = model.GridSquareStateClaimed
				square.SetUserIdentifier(userID)

				logrus.WithField("claimant", payload.Claimant).Info("claiming square")
				if err := square.Save(r.Context(), false, model.GridSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       "user: initial claim",
				}); err != nil {
					s.ServeJSONError(w, http.StatusInternalServerError, "", err)
					return
				}
			} else if payload.Unclaim && square.UserIdentifier() == userID {
				square.State = model.GridSquareStateUnclaimed
				square.SetUserIdentifier(userID)

				if err := square.Save(r.Context(), false, model.GridSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       fmt.Sprintf("user: `%s` unclaimed", square.Claimant),
				}); err != nil {
					s.ServeJSONError(w, http.StatusInternalServerError, "", err)
					return
				}
			} else if jcd.Claim.IsAdmin {
				if payload.State.IsValid() {
					square.State = payload.State
				}

				if err := square.Save(r.Context(), true, model.GridSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       payload.Note,
				}); err != nil {
					s.ServeJSONError(w, http.StatusInternalServerError, "", err)
					return
				}
			} else {
				logrus.WithField("remoteAddr", r.RemoteAddr).Warn("non-admin tried to administer squares")
				s.ServeJSONError(w, http.StatusForbidden, "")
				return
			}
		}

		if jcd.Claim.IsAdmin {
			if err := square.LoadLogs(r.Context()); err != nil {
				s.ServeJSONError(w, http.StatusInternalServerError, "", err)
				return
			}
		}

		s.ServeJSON(w, http.StatusOK, jsonResponse{
			Status: responseOK,
			Result: square,
		})
	}
}

/* context handlers */

type jwtContextData struct {
	Claim *tokenJWTClaim
	Grid  *model.Grid
}

func (s *Server) apiGridJWTHandler(next http.Handler) http.HandlerFunc {
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
		token, err := s.jwt.Validate(tokenStr, &tokenJWTClaim{})
		if err != nil {
			if err != smjwt.ErrExpired {
				logrus.WithError(err).Error("unexpected error from smjwt.Validate()")
			}
			s.ServeJSONError(w, http.StatusUnauthorized, "")
			return
		}

		// let it panic
		claims := token.Claims.(*tokenJWTClaim)

		vars := mux.Vars(r)
		if vars["token"] != claims.Token {
			s.ServeJSONError(w, http.StatusUnauthorized, "")
			return
		}

		grid, err := s.model.GridByToken(r.Context(), claims.Token)
		if err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		newCtx := context.WithValue(r.Context(), ctxKeyJWT, &jwtContextData{
			Claim: claims,
			Grid:  grid,
		})

		next.ServeHTTP(w, r.WithContext(newCtx))
	}
}

type jsonResponse struct {
	Status string      `json:"status"`
	Error  string      `json:"error,omitempty"`
	Result interface{} `json:"result,omitempty"`
}

// ServeJSON will serve JSON to the user
func (s *Server) ServeJSON(w http.ResponseWriter, statusCode int, content interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	enc := json.NewEncoder(w)
	if err := enc.Encode(content); err != nil {
		logrus.WithError(err).Error("could not encode JSON")
		return
	}
}

// ServeJSONError will render a JSON error message
func (s *Server) ServeJSONError(w http.ResponseWriter, statusCode int, userMessage string, err ...error) {
	if userMessage == "" {
		userMessage = http.StatusText(statusCode)
	}

	res := jsonResponse{
		Status: "Error",
		Error:  userMessage,
	}

	if len(err) > 0 {
		logrus.WithError(err[0]).Error("could not serve request")
	}

	s.ServeJSON(w, statusCode, res)
}
