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
	"database/sql"
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
	"time"
)

func (s *Server) apiPoolLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)

		if !jcd.Claim.IsAdmin {
			s.ServeJSONError(w, http.StatusUnauthorized, "")
			return

		}

		grid, err := jcd.Pool.Logs(r.Context(), 0, 1000) // TODO: pagination???
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

func (s *Server) apiPoolHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)

		s.ServeJSON(w, http.StatusOK, jsonResponse{
			Status: responseOK,
			Result: jcd.Pool,
		})
	}
}

func (s *Server) apiPoolPostHandler() http.HandlerFunc {
	type Action string
	const ActionLock Action = "LOCK"
	const ActionUnlock Action = "UNLOCK"
	const ActionReorderGrids Action = "REORDER_GRIDS"

	type postPayload struct {
		Action Action  `json:"action"`
		IDs    []int64 `json:"ids"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)
		if !jcd.Claim.IsAdmin {
			s.ServeJSONError(w, http.StatusForbidden, "")
			return
		}

		dec := json.NewDecoder(r.Body)
		var p postPayload
		if err := dec.Decode(&p); err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		switch p.Action {
		case ActionLock:
			jcd.Pool.SetLocks(time.Now())
			if err := jcd.Pool.Save(r.Context()); err != nil {
				s.ServeJSONError(w, http.StatusInternalServerError, "", err)
				return
			}
		case ActionUnlock:
			jcd.Pool.SetLocks(time.Time{})
			if err := jcd.Pool.Save(r.Context()); err != nil {
				s.ServeJSONError(w, http.StatusInternalServerError, "", err)
				return
			}
		case ActionReorderGrids:
			if err := jcd.Pool.SetGridsOrder(r.Context(), p.IDs); err != nil {
				s.ServeJSONError(w, http.StatusInternalServerError, "", err)
				return
			}
		default:
			s.ServeJSONError(w, http.StatusBadRequest, fmt.Sprintf("unknown action: %s", p.Action))
			return
		}

		s.ServeJSON(w, http.StatusOK, jsonResponse{
			Status: responseOK,
			Result: jcd.Pool,
		})
		return
	}
}

func (s *Server) apiPoolGamesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)

		// TODO support pagination
		grids, err := jcd.Pool.Grids(r.Context(), 0, 100)
		if err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		s.ServeJSON(w, http.StatusOK, jsonResponse{
			Status: responseOK,
			Result: grids,
		})
	}
}

func (s *Server) apiPoolSquaresHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)

		squares, err := jcd.Pool.Squares()
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

// may want to refactor since this handler is pretty complex
func (s *Server) apiPoolSquaresSquareHandler() http.HandlerFunc {
	type postPayload struct {
		Claimant string                `json:"claimant"`
		State    model.PoolSquareState `json:"state"`
		Note     string                `json:"note"`
		Unclaim  bool                  `json:"unclaim"`
		Rename   bool                  `json:"rename"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)
		squareID, _ := strconv.Atoi(mux.Vars(r)["square"])
		square, err := jcd.Pool.SquareBySquareID(squareID)
		if err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		lr := logrus.WithField("square-id", squareID)

		if r.Method == http.MethodPost {
			// if the user isn't an admin and the grid is locked, do not let the user do anything
			if !jcd.Claim.IsAdmin && jcd.Pool.IsLocked() {
				s.ServeJSONError(w, http.StatusForbidden, "The grid is locked")
				return
			}

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

			if payload.Rename {
				if !jcd.Claim.IsAdmin {
					s.ServeJSONError(w, http.StatusForbidden, "")
					return
				}

				v := validator.New()
				claimant := v.Printable("name", payload.Claimant)
				claimant = v.ContainsWordChar("name", claimant)

				if claimant == square.Claimant {
					v.AddError("claimant", "must be a different name")
				}

				if !v.OK() {
					s.ServeJSONError(w, http.StatusBadRequest, v.String())
					return
				}

				oldClaimant := square.Claimant
				square.Claimant = claimant
				lr.WithFields(logrus.Fields{
					"oldClaimant": oldClaimant,
					"claimant":    claimant,
				}).Info("renaming sqaure")

				if err := square.Save(r.Context(), true, model.PoolSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       fmt.Sprintf("admin: changed claimant from %s", oldClaimant),
				}); err != nil {
					s.ServeJSONError(w, http.StatusInternalServerError, "", err)
					return
				}
			} else if len(payload.Claimant) > 0 {
				// making a claim
				v := validator.New()
				claimant := v.Printable("name", payload.Claimant)
				claimant = v.ContainsWordChar("name", claimant)

				if !v.OK() {
					s.ServeJSONError(w, http.StatusBadRequest, v.String())
					return
				}

				square.Claimant = claimant
				square.State = model.PoolSquareStateClaimed
				square.SetUserIdentifier(userID)

				lr.WithField("claimant", payload.Claimant).Info("claiming square")
				if err := square.Save(r.Context(), false, model.PoolSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       "user: initial claim",
				}); err != nil {
					s.ServeJSONError(w, http.StatusInternalServerError, "", err)
					return
				}
			} else if payload.Unclaim && square.UserIdentifier() == userID {
				// trying to unclaim as user
				square.State = model.PoolSquareStateUnclaimed
				square.SetUserIdentifier(userID)

				if err := square.Save(r.Context(), false, model.PoolSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       fmt.Sprintf("user: `%s` unclaimed", square.Claimant),
				}); err != nil {
					s.ServeJSONError(w, http.StatusInternalServerError, "", err)
					return
				}
			} else if jcd.Claim.IsAdmin {
				// admin actions
				if payload.State.IsValid() {
					square.State = payload.State
				}

				if err := square.Save(r.Context(), true, model.PoolSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       payload.Note,
				}); err != nil {
					s.ServeJSONError(w, http.StatusInternalServerError, "", err)
					return
				}
			} else {
				lr.WithField("remoteAddr", r.RemoteAddr).Warn("non-admin tried to administer squares")
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

func (s *Server) apiPoolGameHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)
		if !jcd.Claim.IsAdmin {
			s.ServeJSONError(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}

		gridID, _ := strconv.ParseInt(mux.Vars(r)["grid"], 10, 64)
		grid, err := jcd.Pool.GridByID(r.Context(), gridID)
		if err != nil {
			if err == sql.ErrNoRows {
				s.ServeJSONError(w, http.StatusNotFound, "")
				return
			}

			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		if err := grid.LoadSettings(r.Context()); err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		s.ServeJSON(w, http.StatusOK, jsonResponse{
			Status: responseOK,
			Result: grid,
		})
	}
}

func (s *Server) apiPoolGameDeleteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)
		if !jcd.Claim.IsAdmin {
			s.ServeJSONError(w, http.StatusForbidden, "")
			return
		}

		gridID, _ := strconv.ParseInt(mux.Vars(r)["grid"], 10, 64)
		grid, err := jcd.Pool.GridByID(r.Context(), gridID)
		if err != nil {
			if err == sql.ErrNoRows {
				s.ServeJSONError(w, http.StatusNotFound, "")
				return
			}

			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		if err := grid.Delete(r.Context()); err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		s.ServeJSON(w, http.StatusOK, jsonResponse{
			Status: responseOK,
		})
	}
}

func (s *Server) apiPoolGamePostHandler() http.HandlerFunc {
	type postPayload struct {
		Action string `json:"action"`
		Data   *struct {
			EventDate      string `json:"eventDate"`
			Notes          string `json:"notes"`
			HomeTeamName   string `json:"homeTeamName"`
			HomeTeamColor1 string `json:"homeTeamColor1"`
			HomeTeamColor2 string `json:"homeTeamColor2"`
			AwayTeamName   string `json:"awayTeamName"`
			AwayTeamColor1 string `json:"awayTeamColor1"`
			AwayTeamColor2 string `json:"awayTeamColor2"`
		} `json:"data,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		jcd := r.Context().Value(ctxKeyJWT).(*jwtContextData)
		if !jcd.Claim.IsAdmin {
			s.ServeJSONError(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}

		var payload postPayload
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&payload); err != nil {
			s.ServeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		gridID, _ := strconv.ParseInt(mux.Vars(r)["grid"], 10, 64)

		var grid *model.Grid
		if gridID > 0 {
			var err error
			grid, err = jcd.Pool.GridByID(r.Context(), gridID)
			if err != nil {
				if err == sql.ErrNoRows {
					s.ServeJSONError(w, http.StatusNotFound, "")
					return
				}

				s.ServeJSONError(w, http.StatusInternalServerError, "", err)
				return
			}

			if err := grid.LoadSettings(r.Context()); err != nil {
				s.ServeJSONError(w, http.StatusInternalServerError, "", err)
				return
			}
		} else if payload.Action != "save" {
			s.ServeJSONError(w, http.StatusBadRequest, fmt.Sprintf("cannot call action %s without an ID", payload.Action))
			return
		}

		switch payload.Action {
		case "drawNumbers":
			if err := grid.SelectRandomNumbers(); err != nil {
				if err == model.ErrNumbersAlreadyDrawn {
					s.ServeJSONError(w, http.StatusBadRequest, "The numbers have already been drawn")
					return
				}

				s.ServeJSONError(w, http.StatusInternalServerError, "", err)
				return
			}

			if err := grid.Save(r.Context()); err != nil {
				s.ServeJSONError(w, http.StatusInternalServerError, "", err)
				return
			}

			s.ServeJSON(w, http.StatusOK, jsonResponse{
				Status: responseOK,
				Result: grid,
			})
			return
		case "save":
			if payload.Data == nil {
				s.ServeJSONError(w, http.StatusBadRequest, "missing data in payload")
				return
			}

			v := validator.New()
			eventDate := v.Datetime("Event Date", payload.Data.EventDate, "00:00", "0", true)
			homeTeamName := v.Printable("Home Team Name", payload.Data.HomeTeamName, true)
			homeTeamName = v.MaxLength("Home Team Name", homeTeamName, model.TeamNameMaxLength)
			homeTeamColor1 := v.Color("Home Team Colors", payload.Data.HomeTeamColor1, true)
			homeTeamColor2 := v.Color("Home Team Colors", payload.Data.HomeTeamColor2, true)
			awayTeamName := v.Printable("Away Team Name", payload.Data.AwayTeamName, true)
			awayTeamName = v.MaxLength("Away Team Name", awayTeamName, model.TeamNameMaxLength)
			awayTeamColor1 := v.Color("Away Team Colors", payload.Data.AwayTeamColor1, true)
			awayTeamColor2 := v.Color("Away Team Colors", payload.Data.AwayTeamColor2, true)
			notes := v.PrintableWithNewline("Notes", payload.Data.Notes, true)
			notes = v.MaxLength("Notes", notes, model.NotesMaxLength)

			if !v.OK() {
				s.ServeJSON(w, http.StatusBadRequest, jsonResponse{
					Status: responseFail,
					Error:  "one or more errors",
					Result: v.Errors,
				})
				return
			}

			if grid == nil {
				grid = jcd.Pool.NewGrid()
			}

			grid.SetEventDate(eventDate)
			grid.SetHomeTeamName(homeTeamName)
			grid.SetAwayTeamName(awayTeamName)
			settings := grid.Settings()
			settings.SetNotes(notes)
			settings.SetHomeTeamColor1(homeTeamColor1)
			settings.SetHomeTeamColor2(homeTeamColor2)
			settings.SetAwayTeamColor1(awayTeamColor1)
			settings.SetAwayTeamColor2(awayTeamColor2)

			if err := grid.Save(r.Context()); err != nil {
				s.ServeJSONError(w, http.StatusInternalServerError, "", err)
				return
			}

			s.ServeJSON(w, http.StatusAccepted, jsonResponse{
				Status: responseOK,
				Result: grid,
			})
			return
		}

		s.ServeJSONError(w, http.StatusBadRequest, "unknown action")
		return
	}
}

/* context handlers */

type jwtContextData struct {
	Claim *tokenJWTClaim
	Pool  *model.Pool
}

func (s *Server) apiPoolJWTHandler(next http.Handler) http.HandlerFunc {
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

		grid, err := s.model.PoolByToken(r.Context(), claims.Token)
		if err != nil {
			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		newCtx := context.WithValue(r.Context(), ctxKeyJWT, &jwtContextData{
			Claim: claims,
			Pool:  grid,
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
