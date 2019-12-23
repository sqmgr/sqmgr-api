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
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr-api/internal/validator"
	"github.com/weters/sqmgr-api/pkg/model"
	"net/http"
	"strconv"
	"time"
)

const minJoinPasswordLength = 6
const validationErrorMessage = "There were one or more errors with your request"
const sqmgrInviteAudience = "com.sqmgr.invite"

var inviteTokenTTL = time.Hour * 24 * 365 // 1 year

type inviteClaims struct {
	*jwt.StandardClaims
	CheckID int `json:"chid"`
}

func (s *Server) poolHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := mux.Vars(r)["token"]
		pool, err := s.model.PoolByToken(r.Context(), token)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		user := r.Context().Value(ctxUserKey).(*model.User)

		isMemberOf, err := user.IsMemberOf(r.Context(), pool)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if !isMemberOf {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxPoolKey, pool)))
	})
}

func (s *Server) postPoolTokenEndpoint() http.HandlerFunc {
	type payload struct {
		Action          string  `json:"action"`
		IDs             []int64 `json:"ids"`
		Name            string  `json:"name"`
		Password        string  `json:"password"`
		ResetMembership bool    `json:"resetMembership"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		user := r.Context().Value(ctxUserKey).(*model.User)

		if isAdmin, err := user.IsAdminOf(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		} else if !isAdmin {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		var resp payload
		if ok := s.parseJSONPayload(w, r, &resp); !ok {
			return
		}

		var err error
		switch resp.Action {
		case "lock":
			pool.SetLocks(time.Now())
			err = pool.Save(r.Context())
		case "unlock":
			pool.SetLocks(time.Time{})
			err = pool.Save(r.Context())
		case "reorderGrids":
			err = pool.SetGridsOrder(r.Context(), resp.IDs)
		case "archive":
			pool.SetArchived(true)
			err = pool.Save(r.Context())
		case "unarchive":
			pool.SetArchived(false)
			err = pool.Save(r.Context())
		case "changeJoinPassword":
			v := validator.New()
			password := v.Password("Join Password", resp.Password, minJoinPasswordLength)
			if !v.OK() {
				s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
					Status:           statusError,
					Error:            validationErrorMessage,
					ValidationErrors: v.Errors,
				})
				return
			}

			if err := pool.SetPassword(password); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			pool.IncrementCheckID()
			if err := pool.Save(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			if resp.ResetMembership {
				if err := pool.RemoveAllMembers(r.Context()); err != nil {
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
			}
		case "rename":
			v := validator.New()
			name := v.Printable("Name", resp.Name, false)
			name = v.MaxLength("Name", name, model.NameMaxLength)
			if !v.OK() {
				s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
					Status:           statusError,
					Error:            validationErrorMessage,
					ValidationErrors: v.Errors,
				})
				return
			}

			pool.SetName(name)
			err = pool.Save(r.Context())
		default:
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("unsupported action %s", resp.Action))
			return
		}

		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, poolResponse{
			PoolJSON: pool.JSON(),
			IsAdmin:  true,
		})
	}
}

func (s *Server) getPoolTokenLogEndpoint() http.HandlerFunc {
	const defaultPerPage = 100
	const maxPerPage = 100

	type response struct {
		Logs  []*model.PoolSquareLogJSON `json:"logs"`
		Total int64                      `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		user := r.Context().Value(ctxUserKey).(*model.User)

		if isAdmin, err := user.IsAdminOf(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		} else if !isAdmin {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit <= 0 {
			limit = defaultPerPage
		}

		if limit > maxPerPage {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("limit cannot exceed %d", maxPerPage))
		}

		logs, err := pool.Logs(r.Context(), offset, limit)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		count, err := pool.LogsCount(r.Context())
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		logsJSON := make([]*model.PoolSquareLogJSON, len(logs))
		for i, log := range logs {
			logsJSON[i] = log.JSON()
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			Logs:  logsJSON,
			Total: count,
		})
	}
}

func (s *Server) deletePoolTokenGridIDEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		id, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

		grid, err := pool.GridByID(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if err := grid.Delete(r.Context()); err != nil {
			if err == model.ErrLastGrid {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("you cannot delete the last grid"))
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusNoContent, nil)
	}
}

func (s *Server) getPoolConfiguration() http.HandlerFunc {
	type keyDescription struct {
		Key         model.GridType `json:"key"`
		Description string         `json:"description"`
	}

	gridTypes := model.GridTypes()
	gridTypesSlice := make([]keyDescription, len(gridTypes))
	for i, gt := range gridTypes {
		gridTypesSlice[i] = keyDescription{
			Key:         gt,
			Description: gt.Description(),
		}
	}

	resp := struct {
		ClaimantMaxLength     int                     `json:"claimantMaxLength"`
		NameMaxLength         int                     `json:"nameMaxLength"`
		NotesMaxLength        int                     `json:"notesMaxLength"`
		TeamNameMaxLength     int                     `json:"teamNameMaxLength"`
		PoolSquareStates      []model.PoolSquareState `json:"poolSquareStates"`
		GridTypes             []keyDescription        `json:"gridTypes"`
		MinJoinPasswordLength int                     `json:"minJoinPasswordLength"`
	}{
		ClaimantMaxLength:     model.ClaimantMaxLength,
		NameMaxLength:         model.NameMaxLength,
		NotesMaxLength:        model.NotesMaxLength,
		TeamNameMaxLength:     model.TeamNameMaxLength,
		PoolSquareStates:      model.PoolSquareStates,
		GridTypes:             gridTypesSlice,
		MinJoinPasswordLength: minJoinPasswordLength,
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		logrus.WithError(err).Fatal("could not encode pool configuration")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(jsonResp); err != nil {
			logrus.WithError(err).Error("could not write response")
		}
	}
}

func (s *Server) postPoolEndpoint() http.HandlerFunc {
	type payload struct {
		Name         string `json:"name"`
		GridType     string `json:"gridType"`
		JoinPassword string `json:"joinPassword"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(ctxUserKey).(*model.User)
		if !user.HasPermission(model.PermissionCreatePool) {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		var data payload
		if ok := s.parseJSONPayload(w, r, &data); !ok {
			return
		}

		v := validator.New()
		name := v.Printable("Squares Pool Name", data.Name)
		gridType := v.GridType("Grid Configuration", data.GridType)
		password := v.Password("Join Password", data.JoinPassword, minJoinPasswordLength)

		if err := user.Can(r.Context(), model.ActionCreatePool, user); err != nil {
			if _, ok := err.(model.ActionError); ok {
				s.writeErrorResponse(w, http.StatusBadRequest, err)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if !v.OK() {
			s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
				Status:           statusError,
				Error:            validationErrorMessage,
				ValidationErrors: v.Errors,
			})
			return
		}

		pool, err := s.model.NewPool(r.Context(), user.ID, name, gridType, password)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusCreated, poolResponse{
			PoolJSON: pool.JSON(),
			IsAdmin:  true,
		})
	}
}

func (s *Server) getPoolTokenEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(ctxUserKey).(*model.User)
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		isAdminOf, err := user.IsAdminOf(r.Context(), pool)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, poolResponse{
			PoolJSON: pool.JSON(),
			IsAdmin:  isAdminOf,
		})
	}
}

func (s *Server) getPoolTokenInviteTokenEndpoint() http.HandlerFunc {
	type response struct {
		JWT string `json:"jwt"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(ctxUserKey).(*model.User)
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		if isAdmin, err := user.IsAdminOf(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		} else if !isAdmin {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		claim := &inviteClaims{
			StandardClaims: &jwt.StandardClaims{
				Audience:  sqmgrInviteAudience,
				ExpiresAt: time.Now().Add(inviteTokenTTL).Unix(),
				Issuer:    model.IssuerSqMGR,
				NotBefore: 0,
				Subject:   pool.Token(),
			},
			CheckID: pool.CheckID(),
		}

		sign, err := s.smjwt.Sign(claim)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, response{JWT: sign})
	}
}

func (s *Server) getPoolTokenGridEndpoint() http.HandlerFunc {
	const defaultPerPage = model.MaxGridsPerPool
	const maxPerPage = model.MaxGridsPerPool

	type response struct {
		Grids      []*model.GridJSON `json:"grids"`
		Total      int64             `json:"total"`
		MaxAllowed int               `json:"maxAllowed"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)

		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit < 1 {
			limit = defaultPerPage
		} else if limit > maxPerPage {
			s.writeErrorResponse(w, http.StatusBadGateway, fmt.Errorf("limit cannot exceed %d", maxPerPage))
			return
		}

		grids, err := pool.Grids(r.Context(), offset, limit)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		count, err := pool.GridsCount(r.Context())
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		gridsJSON := make([]*model.GridJSON, len(grids))
		for i, grid := range grids {
			gridsJSON[i] = grid.JSON()
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			Grids:      gridsJSON,
			Total:      count,
			MaxAllowed: model.MaxGridsPerPool,
		})
	}
}

func (s *Server) getPoolTokenGridIDEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		id, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

		grid, err := pool.GridByID(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if err := grid.LoadSettings(r.Context()); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, grid.JSON())
	}
}

func (s *Server) getPoolTokenSquareEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)

		squares, err := pool.Squares()
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		squaresJSON := make(map[int]*model.PoolSquareJSON)
		for key, square := range squares {
			squaresJSON[key] = square.JSON()
		}

		s.writeJSONResponse(w, http.StatusOK, squaresJSON)
	}
}

func (s *Server) getPoolTokenSquareIDEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		user := r.Context().Value(ctxUserKey).(*model.User)

		squareID, _ := strconv.Atoi(mux.Vars(r)["id"])
		square, err := pool.SquareBySquareID(squareID)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if isAdmin, err := user.IsAdminOf(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		} else if isAdmin {
			if err := square.LoadLogs(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		}

		s.writeJSONResponse(w, http.StatusOK, square.JSON())
	}
}

func (s *Server) postPoolTokenSquareIDEndpoint() http.HandlerFunc {
	type postPayload struct {
		Claimant          string                `json:"claimant"`
		State             model.PoolSquareState `json:"state"`
		Note              string                `json:"note"`
		Unclaim           bool                  `json:"unclaim"`
		Rename            bool                  `json:"rename"`
		SecondarySquareID int                   `json:"secondarySquareId"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		user := r.Context().Value(ctxUserKey).(*model.User)
		squareID, _ := strconv.Atoi(mux.Vars(r)["id"])
		square, err := pool.SquareBySquareID(squareID)
		if err != nil {
			if err == sql.ErrNoRows {
				logrus.WithFields(logrus.Fields{
					"pool":   pool.ID(),
					"square": squareID,
				}).Error("could not find square")
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		lr := logrus.WithField("square-id", squareID)

		isAdmin, err := user.IsAdminOf(r.Context(), pool)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		// if the user isn't an admin and the grid is locked, do not let the user do anything
		if pool.IsLocked() && !isAdmin {
			s.writeErrorResponse(w, http.StatusForbidden, errors.New("the grid is locked"))
			return
		}

		dec := json.NewDecoder(r.Body)
		var payload postPayload
		if err := dec.Decode(&payload); err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		if pool.GridType() != model.GridTypeRoll100 && payload.SecondarySquareID > 0 {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("secondary squares are not used with this grid type"))
			return
		}

		var secondSquare *model.PoolSquare
		if payload.SecondarySquareID > 0 {
			secondSquare, err = pool.SquareBySquareID(payload.SecondarySquareID)
			if err != nil {
				if err == sql.ErrNoRows {
					logrus.WithFields(logrus.Fields{
						"pool":   pool.ID(),
						"square": squareID,
					}).Error("could not find square")
					s.writeErrorResponse(w, http.StatusNotFound, nil)
					return
				}

				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		}

		if payload.Rename {
			if !isAdmin {
				s.writeErrorResponse(w, http.StatusForbidden, errors.New("only an admin can rename a square"))
				return
			}

			v := validator.New()
			claimant := v.Printable("name", payload.Claimant)
			claimant = v.ContainsWordChar("name", claimant)

			if claimant == square.Claimant() {
				v.AddError("claimant", "must be a different name")
			}

			if !v.OK() {
				s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
					Status:           statusError,
					Error:            validationErrorMessage,
					ValidationErrors: v.Errors,
				})
				return
			}

			oldClaimant := square.Claimant()
			square.SetClaimant(claimant)
			lr.WithFields(logrus.Fields{
				"oldClaimant": oldClaimant,
				"claimant":    claimant,
			}).Info("renaming square")

			if err := square.Save(r.Context(), s.model.DB, true, model.PoolSquareLog{
				RemoteAddr: r.RemoteAddr,
				Note:       fmt.Sprintf("admin: changed claimant from %s", oldClaimant),
			}); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else if len(payload.Claimant) > 0 {
			// making a claim
			v := validator.New()
			claimant := v.Printable("name", payload.Claimant)
			claimant = v.ContainsWordChar("name", claimant)

			if !v.OK() {
				s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
					Status:           statusError,
					Error:            validationErrorMessage,
					ValidationErrors: v.Errors,
				})
				return
			}

			square.SetClaimant(claimant)
			square.State = model.PoolSquareStateClaimed
			square.SetUserID(user.ID)

			tx, err := s.model.DB.BeginTx(r.Context(), nil)
			if err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			lr.WithField("claimant", payload.Claimant).Info("claiming square")
			if err := square.Save(r.Context(), tx, false, model.PoolSquareLog{
				RemoteAddr: r.RemoteAddr,
				Note:       "user: initial claim",
			}); err != nil {
				_ = tx.Rollback()
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			if secondSquare != nil {
				secondSquare.SetClaimant(claimant)
				secondSquare.State = model.PoolSquareStateClaimed
				secondSquare.SetUserID(user.ID)

				if err := secondSquare.Save(r.Context(), tx, false, model.PoolSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       "user: initial claim (secondary)",
				}); err != nil {
					_ = tx.Rollback()
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}

				if err := secondSquare.SetParentSquare(r.Context(), tx, square); err != nil {
					_ = tx.Rollback()
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
			}

			if err := tx.Commit(); err != nil {
				_ = tx.Rollback()
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else if payload.Unclaim && square.UserID() == user.ID {
			tx, err := square.Model.DB.BeginTx(r.Context(), nil)
			if err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			squares := []*model.PoolSquare{square}
			if square.ParentID > 0 {
				pSq, err := pool.SquareBySquareID(square.ParentSquareID)
				if err != nil {
					_ = tx.Rollback()
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
				squares = append(squares, pSq)
			}

			childSquares, err := square.ChildSquares(r.Context(), tx)
			if err != nil {
				_ = tx.Rollback()
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
			squares = append(squares, childSquares...)

			for _, square := range squares {
				// trying to unclaim as user
				square.State = model.PoolSquareStateUnclaimed
				square.SetUserID(user.ID)

				if err := square.Save(r.Context(), tx, false, model.PoolSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       fmt.Sprintf("user: `%s` unclaimed", square.Claimant()),
				}); err != nil {
					_ = tx.Rollback()
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
			}

			if err := tx.Commit(); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else if isAdmin {
			// admin actions
			if payload.State.IsValid() {
				square.State = payload.State
			}

			if err := square.Save(r.Context(), s.model.DB, true, model.PoolSquareLog{
				RemoteAddr: r.RemoteAddr,
				Note:       payload.Note,
			}); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else {
			lr.WithField("remoteAddr", r.RemoteAddr).Warn("non-admin tried to administer squares")
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		if isAdmin {
			if err := square.LoadLogs(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		}

		s.writeJSONResponse(w, http.StatusOK, square.JSON())
	}
}

func (s *Server) postPoolTokenGridIDEndpoint() http.HandlerFunc {
	type payload struct {
		Action string `json:"action"`
		Data   *struct {
			EventDate      string `json:"eventDate"`
			Notes          string `json:"notes"`
			Rollover       bool   `json:"rollover"`
			HomeTeamName   string `json:"homeTeamName"`
			HomeTeamColor1 string `json:"homeTeamColor1"`
			HomeTeamColor2 string `json:"homeTeamColor2"`
			AwayTeamName   string `json:"awayTeamName"`
			AwayTeamColor1 string `json:"awayTeamColor1"`
			AwayTeamColor2 string `json:"awayTeamColor2"`

			HomeTeamNumbers []int `json:"homeTeamNumbers"`
			AwayTeamNumbers []int `json:"awayTeamNumbers"`
		} `json:"data,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		user := r.Context().Value(ctxUserKey).(*model.User)

		if isAdmin, err := user.IsAdminOf(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		} else if !isAdmin {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		var data payload
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&data); err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		gridID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			panic(err)
		}

		var grid *model.Grid
		if gridID > 0 {
			var err error
			grid, err = pool.GridByID(r.Context(), gridID)
			if err != nil {
				if err == sql.ErrNoRows {
					s.writeErrorResponse(w, http.StatusNotFound, nil)
					return
				}

				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			if err := grid.LoadSettings(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else if data.Action != "save" {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("cannot call action %s without an ID", data.Action))
			return
		}

		switch data.Action {
		case "drawManualNumbers":
			if err := grid.SetManualNumbers(data.Data.HomeTeamNumbers, data.Data.AwayTeamNumbers); err != nil {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("the numbers supplied are not valid"))
			}

			if err := grid.Save(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			s.writeJSONResponse(w, http.StatusOK, grid.JSON())
			return
		case "drawNumbers":
			if err := grid.SelectRandomNumbers(); err != nil {
				if err == model.ErrNumbersAlreadyDrawn {
					s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("the numbers have already been drawn"))
					return
				}

				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			if err := grid.Save(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			s.writeJSONResponse(w, http.StatusOK, grid.JSON())
			return
		case "save":
			if data.Data == nil {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("missing data in payload"))
				return
			}

			v := validator.New()
			eventDate := v.Datetime("Event Date", data.Data.EventDate, "00:00", "0", true)
			homeTeamName := v.Printable("Home Team Name", data.Data.HomeTeamName, true)
			homeTeamName = v.MaxLength("Home Team Name", homeTeamName, model.TeamNameMaxLength)
			homeTeamColor1 := v.Color("Home Team Colors", data.Data.HomeTeamColor1, true)
			homeTeamColor2 := v.Color("Home Team Colors", data.Data.HomeTeamColor2, true)
			awayTeamName := v.Printable("Away Team Name", data.Data.AwayTeamName, true)
			awayTeamName = v.MaxLength("Away Team Name", awayTeamName, model.TeamNameMaxLength)
			awayTeamColor1 := v.Color("Away Team Colors", data.Data.AwayTeamColor1, true)
			awayTeamColor2 := v.Color("Away Team Colors", data.Data.AwayTeamColor2, true)
			notes := v.PrintableWithNewline("Notes", data.Data.Notes, true)
			notes = v.MaxLength("Notes", notes, model.NotesMaxLength)

			if pool.GridType() != model.GridTypeRoll100 && data.Data.Rollover {
				v.AddError("rollover", "Rollover is not valid for this pool type")
			}

			if !v.OK() {
				s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
					Status:           statusError,
					Error:            "There were one or more validation errors",
					ValidationErrors: v.Errors,
				})
				return
			}

			if grid == nil {
				grid = pool.NewGrid()
			}

			grid.SetEventDate(eventDate)
			grid.SetHomeTeamName(homeTeamName)
			grid.SetAwayTeamName(awayTeamName)
			grid.SetRollover(data.Data.Rollover)
			settings := grid.Settings()
			settings.SetNotes(notes)
			settings.SetHomeTeamColor1(homeTeamColor1)
			settings.SetHomeTeamColor2(homeTeamColor2)
			settings.SetAwayTeamColor1(awayTeamColor1)
			settings.SetAwayTeamColor2(awayTeamColor2)

			if err := grid.Save(r.Context()); err != nil {
				if err == model.ErrGridLimit {
					s.writeErrorResponse(w, http.StatusBadRequest, err)
					return
				}

				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			s.writeJSONResponse(w, http.StatusAccepted, grid.JSON())
			return
		}

		s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("unsupported action %s", data.Action))
		return
	}
}

func (s *Server) postPoolTokenMemberEndpoint() http.HandlerFunc {
	type payload struct {
		Password string `json:"password"`
		JWT      string `json:"jwt"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(ctxUserKey).(*model.User)
		token := mux.Vars(r)["token"]
		pool, err := s.model.PoolByToken(r.Context(), token)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		var data payload
		if ok := s.parseJSONPayload(w, r, &data); !ok {
			return
		}

		if data.JWT != "" {
			j, err := s.smjwt.Validate(data.JWT, &inviteClaims{})
			if err != nil {
				s.writeErrorResponse(w, http.StatusBadRequest, err)
				return
			}

			claims := j.Claims.(*inviteClaims)

			if !claims.VerifyAudience(sqmgrInviteAudience, true) ||
				!claims.VerifyIssuer(model.IssuerSqMGR, true) ||
				!pool.CheckIDIsValid(claims.CheckID) {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("invalid join token"))
				return
			}
		} else if !pool.PasswordIsValid(data.Password) {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("password is invalid"))
			return
		}

		if err := user.JoinPool(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type poolResponse struct {
	*model.PoolJSON
	IsAdmin bool `json:"isAdmin"`
}
