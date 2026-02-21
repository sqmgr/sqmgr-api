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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/internal/validator"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

const minJoinPasswordLength = 6
const validationErrorMessage = "There were one or more errors with your request"
const sqmgrInviteAudience = "com.sqmgr.invite"

var inviteTokenTTL = time.Hour * 24 * 365 // 1 year

type inviteClaims struct {
	jwt.RegisteredClaims
	CheckID int `json:"chid"`
}

func (s *Server) poolHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := mux.Vars(r)["token"]
		pool, err := s.model.PoolByToken(r.Context(), token)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		user, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		// Site admins can access any pool without joining
		if !user.IsAdmin {
			if (pool.IsLocked() && pool.OpenAccessOnLock()) || !pool.PasswordRequired() {
				// Auto-join user to pool since no password is required
				if err := user.JoinPool(r.Context(), pool); err != nil {
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
			} else {
				isMemberOf, err := user.IsMemberOf(r.Context(), pool)
				if err != nil {
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}

				if !isMemberOf {
					s.writeErrorResponse(w, http.StatusForbidden, nil)
					return
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxPoolKey, pool)))
	})
}

func (s *Server) poolGridHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		gridID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		grid, err := pool.GridByID(r.Context(), gridID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxGridKey, grid)))
	})
}

func (s *Server) poolAdminHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		user, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		if isAdmin, err := user.IsAdminOf(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		} else if !isAdmin {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) poolGridSquareAdminHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		user, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		squareID, err := strconv.Atoi(mux.Vars(r)["square_id"])
		if err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		if squareID < 1 || squareID > pool.NumberOfSquares() {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("invalid square ID"))
			return
		}

		if isAdmin, err := user.IsAdminOf(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		} else if !isAdmin {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxSquareIDKey, squareID)))
	})
}

func (s *Server) postPoolTokenEndpoint() http.HandlerFunc {
	type payload struct {
		Action           string  `json:"action"`
		IDs              []int64 `json:"ids"`
		Name             string  `json:"name"`
		Password         string  `json:"password"`
		ResetMembership  bool    `json:"resetMembership"`
		PasswordRequired bool    `json:"passwordRequired"`
		OpenAccessOnLock bool    `json:"openAccessOnLock"`
		NumberSetConfig  string  `json:"numberSetConfig"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
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
		case "passwordRequired":
			pool.SetPasswordRequired(resp.PasswordRequired)
			err = pool.Save(r.Context())
		case "accessOnLock":
			pool.SetOpenAccessOnLock(resp.OpenAccessOnLock)
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
		case "changeNumberSetConfig":
			if !model.IsValidNumberSetConfig(resp.NumberSetConfig) {
				s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
					Status: statusError,
					Error:  "Invalid number set configuration",
				})
				return
			}

			var canChange bool
			canChange, err = pool.CanChangeNumberSetConfig(r.Context())
			if err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			if !canChange {
				s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
					Status: statusError,
					Error:  "Cannot change number set configuration after numbers have been drawn for any game",
				})
				return
			}

			// Validate the config is valid for all linked events
			newConfig := model.NumberSetConfig(resp.NumberSetConfig)
			var grids []*model.Grid
			grids, err = pool.Grids(r.Context(), 0, model.MaxGridsPerPool)
			if err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			for _, grid := range grids {
				if err := grid.LoadBDLEvent(r.Context()); err != nil {
					continue
				}
				if grid.BDLEvent() != nil {
					if !model.IsValidNumberSetConfigForLeague(newConfig, grid.BDLEvent().League) {
						s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
							Status: statusError,
							Error:  fmt.Sprintf("Cannot use '%s' configuration: one or more grids are linked to %s games which don't support this configuration", resp.NumberSetConfig, grid.BDLEvent().League),
						})
						return
					}
				}
			}

			pool.SetNumberSetConfig(newConfig)
			err = pool.Save(r.Context())
		default:
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("unsupported action %s", resp.Action))
			return
		}

		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		canChange, canChangeErr := pool.CanChangeNumberSetConfig(r.Context())
		if canChangeErr != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, canChangeErr)
			return
		}

		s.broker.Publish(pool.Token(), PoolEvent{Type: EventPoolUpdated})

		s.writeJSONResponse(w, http.StatusOK, poolResponse{
			PoolJSON:                 pool.JSON(),
			IsAdmin:                  true,
			CanChangeNumberSetConfig: canChange,
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
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
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
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		id, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

		grid, err := pool.GridByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
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

		s.broker.Publish(pool.Token(), PoolEvent{Type: EventGridUpdated})
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
		ClaimantMaxLength     int                                             `json:"claimantMaxLength"`
		NameMaxLength         int                                             `json:"nameMaxLength"`
		NotesMaxLength        int                                             `json:"notesMaxLength"`
		TeamNameMaxLength     int                                             `json:"teamNameMaxLength"`
		PoolSquareStates      []model.PoolSquareState                         `json:"poolSquareStates"`
		GridTypes             []keyDescription                                `json:"gridTypes"`
		NumberSetConfigs      []model.NumberSetConfigInfo                     `json:"numberSetConfigs"`
		NumberSetTypeInfos    map[model.NumberSetType]model.NumberSetTypeInfo `json:"numberSetTypeInfos"`
		MinJoinPasswordLength int                                             `json:"minJoinPasswordLength"`
		GridAnnotationIcons   model.GridAnnotationIconMapping                 `json:"gridAnnotationIcons"`
	}{
		ClaimantMaxLength:     model.ClaimantMaxLength,
		NameMaxLength:         model.NameMaxLength,
		NotesMaxLength:        model.NotesMaxLength,
		TeamNameMaxLength:     model.TeamNameMaxLength,
		PoolSquareStates:      model.PoolSquareStates,
		GridTypes:             gridTypesSlice,
		NumberSetConfigs:      model.ValidNumberSetConfigs(),
		NumberSetTypeInfos:    model.NumberSetTypeInfos(),
		MinJoinPasswordLength: minJoinPasswordLength,
		GridAnnotationIcons:   model.AnnotationIcons,
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
		Name            string `json:"name"`
		GridType        string `json:"gridType"`
		NumberSetConfig string `json:"numberSetConfig"`
		JoinPassword    string `json:"joinPassword"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
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

		// Default to "single" if not provided (backwards compatibility)
		numberSetConfig := model.NumberSetConfig(data.NumberSetConfig)
		if numberSetConfig == "" {
			numberSetConfig = model.NumberSetConfigStandard
		}
		if !model.IsValidNumberSetConfig(string(numberSetConfig)) {
			v.AddError("numberSetConfig", "Invalid number set configuration")
		}

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

		pool, err := s.model.NewPool(r.Context(), user.ID, name, gridType, password, numberSetConfig)
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
		user, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		isAdminOf, err := user.IsAdminOf(r.Context(), pool)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		resp := poolResponse{
			PoolJSON: pool.JSON(),
			IsAdmin:  isAdminOf,
		}

		if isAdminOf {
			canChange, err := pool.CanChangeNumberSetConfig(r.Context())
			if err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
			resp.CanChangeNumberSetConfig = canChange
		}

		s.writeJSONResponse(w, http.StatusOK, resp)
	}
}

func (s *Server) getPoolTokenInviteTokenEndpoint() http.HandlerFunc {
	type response struct {
		Token string `json:"token"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		invite, err := pool.ActiveInvite(r.Context())
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if invite == nil {
			invite, err = s.model.NewPoolInvite(r.Context(), pool.ID(), pool.CheckID(), inviteTokenTTL)
			if err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		}

		s.writeJSONResponse(w, http.StatusOK, response{Token: invite.Token})
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
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit < 1 {
			limit = defaultPerPage
		} else if limit > maxPerPage {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("limit cannot exceed %d", maxPerPage))
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
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		id, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

		grid, err := pool.GridByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
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

		if err := grid.LoadAnnotations(r.Context()); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		// Load number sets for multi-set configurations
		// Need to load if pool config OR grid's payout config is non-standard
		effectiveConfig := pool.NumberSetConfig()
		if grid.PayoutConfig() != nil {
			effectiveConfig = *grid.PayoutConfig()
		}
		if effectiveConfig != model.NumberSetConfigStandard {
			if err := grid.LoadNumberSets(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		}

		// Load BDL event if linked
		if err := grid.LoadBDLEvent(r.Context()); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, grid.JSONWithWinningSquares(pool.NumberSetConfig(), pool.GridType()))
	}
}

func (s *Server) getPoolTokenSquareEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

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

func (s *Server) getPoolTokenSquaresPublicEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := mux.Vars(r)["token"]

		// Load pool from database
		pool, err := s.model.PoolByToken(r.Context(), token)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		// Auth required when password is required AND pool is not in open-access state
		// Open access: !PasswordRequired() OR (IsLocked() AND OpenAccessOnLock())
		authRequired := pool.PasswordRequired() && (!pool.IsLocked() || !pool.OpenAccessOnLock())
		if authRequired {
			_, password, ok := r.BasicAuth()
			if !ok || !pool.PasswordIsValid(password) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Pool Access"`)
				s.writeErrorResponse(w, http.StatusUnauthorized, errors.New("authentication required"))
				return
			}
		}

		// Retrieve squares
		squares, err := pool.Squares()
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		// Convert to JSON (no admin fields populated by default)
		squaresJSON := make(map[int]*model.PoolSquareJSON)
		for key, square := range squares {
			squaresJSON[key] = square.JSON()
		}

		s.writeJSONResponse(w, http.StatusOK, squaresJSON)
	}
}

func (s *Server) getPoolTokenSquareIDEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		user, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		squareID, _ := strconv.Atoi(mux.Vars(r)["id"])
		square, err := pool.SquareBySquareID(squareID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		squareJSON := square.JSON()

		// Handle optional gridId parameter for winning periods
		if gridIDStr := r.FormValue("gridId"); gridIDStr != "" {
			gridID, err := strconv.ParseInt(gridIDStr, 10, 64)
			if err == nil {
				grid, err := pool.GridByID(r.Context(), gridID)
				if err == nil {
					if err := grid.LoadBDLEvent(r.Context()); err == nil && grid.BDLEvent() != nil {
						// Load number sets if needed
						effectiveConfig := pool.NumberSetConfig()
						if grid.PayoutConfig() != nil {
							effectiveConfig = *grid.PayoutConfig()
						}
						if effectiveConfig != model.NumberSetConfigStandard {
							_ = grid.LoadNumberSets(r.Context())
						}

						winningSquares := grid.GetGridWinningSquares(grid.BDLEvent(), effectiveConfig, pool.GridType())

						// Use team abbreviations from live event
						var homeTeamName, awayTeamName string
						if grid.BDLEvent().HomeTeam() != nil {
							homeTeamName = grid.BDLEvent().HomeTeam().Abbreviation
						}
						if grid.BDLEvent().AwayTeam() != nil {
							awayTeamName = grid.BDLEvent().AwayTeam().Abbreviation
						}

						squareJSON.WinningPeriods = model.GetWinningPeriodsForSquare(squareID, winningSquares, grid.BDLEvent(), homeTeamName, awayTeamName)
					}
				}
			}
		}

		if isAdmin, err := user.IsAdminOf(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		} else if isAdmin {
			if err := square.LoadLogs(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
			squareJSON.Logs = square.Logs

			// Add user info for admins
			if square.UserID() > 0 {
				squareUser, err := s.model.GetUserByID(r.Context(), square.UserID())
				if err != nil {
					logrus.WithError(err).WithField("userId", square.UserID()).Warn("could not get square user")
				} else {
					userInfo := &model.SquareUserInfoJSON{}

					if squareUser.Store == model.UserStoreAuth0 {
						userInfo.UserType = "registered"

						// Get email: first try local, then fallback to Auth0 API
						if squareUser.Email != nil && *squareUser.Email != "" {
							userInfo.Email = *squareUser.Email
						} else if s.auth0Client.IsConfigured() {
							email, err := s.auth0Client.GetUserEmail(r.Context(), squareUser.StoreID)
							if err != nil {
								logrus.WithError(err).WithField("storeId", squareUser.StoreID).Warn("could not get email from Auth0")
							} else if email != "" {
								userInfo.Email = email
								// Cache the email for next time
								if err := squareUser.SetEmail(r.Context(), email); err != nil {
									logrus.WithError(err).Warn("could not cache user email")
								}
							}
						}
					} else {
						userInfo.UserType = "guest"
					}

					squareJSON.UserInfo = userInfo
				}
			}
		}

		s.writeJSONResponse(w, http.StatusOK, squareJSON)
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
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		user, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		squareID, _ := strconv.Atoi(mux.Vars(r)["id"])
		square, err := pool.SquareBySquareID(squareID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
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
				if errors.Is(err, sql.ErrNoRows) {
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

			if square.ParentID > 0 {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("cannot rename a secondary square directly; rename the primary square instead"))
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

			tx, err := s.model.DB.BeginTx(r.Context(), nil)
			if err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			if err := square.Save(r.Context(), tx, true, model.PoolSquareLog{
				RemoteAddr: r.RemoteAddr,
				Note:       fmt.Sprintf("admin: changed claimant from %s", oldClaimant),
			}); err != nil {
				_ = tx.Rollback()
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			// also rename any secondary squares linked to this primary
			childSquares, err := square.ChildSquares(r.Context(), tx)
			if err != nil {
				_ = tx.Rollback()
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			for _, child := range childSquares {
				child.SetClaimant(claimant)
				if err := child.Save(r.Context(), tx, true, model.PoolSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       fmt.Sprintf("admin: changed claimant from %s (via primary square %d)", oldClaimant, square.SquareID),
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

				if err == model.ErrSquareAlreadyClaimed {
					s.writeErrorResponse(w, http.StatusBadRequest, err)
				} else {
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
				}

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
				if square.State == model.PoolSquareStateUnclaimed && payload.State != model.PoolSquareStateUnclaimed {
					s.writeErrorResponse(w, http.StatusBadRequest, errors.New("cannot change state of an unclaimed square"))
					return
				}
				square.State = payload.State
			}

			tx, err := s.model.DB.BeginTx(r.Context(), nil)
			if err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			if err := square.Save(r.Context(), tx, true, model.PoolSquareLog{
				RemoteAddr: r.RemoteAddr,
				Note:       payload.Note,
			}); err != nil {
				_ = tx.Rollback()
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			// when unclaiming a primary square, also unclaim its secondary squares
			if square.State == model.PoolSquareStateUnclaimed {
				childSquares, err := square.ChildSquares(r.Context(), tx)
				if err != nil {
					_ = tx.Rollback()
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}

				for _, child := range childSquares {
					child.State = model.PoolSquareStateUnclaimed
					if err := child.Save(r.Context(), tx, true, model.PoolSquareLog{
						RemoteAddr: r.RemoteAddr,
						Note:       fmt.Sprintf("admin: unclaimed (secondary of square %d)", square.SquareID),
					}); err != nil {
						_ = tx.Rollback()
						s.writeErrorResponse(w, http.StatusInternalServerError, err)
						return
					}
				}
			}

			if err := tx.Commit(); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else {
			lr.WithField("remoteAddr", r.RemoteAddr).Warn("non-admin tried to administer squares")
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		s.broker.Publish(pool.Token(), PoolEvent{Type: EventSquareUpdated})

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
	type numberSetPayload struct {
		HomeTeamNumbers []int `json:"homeTeamNumbers"`
		AwayTeamNumbers []int `json:"awayTeamNumbers"`
	}

	type payload struct {
		Action string `json:"action"`
		Data   *struct {
			EventDate        string `json:"eventDate"`
			Notes            string `json:"notes"`
			Rollover         bool   `json:"rollover"`
			Label            string `json:"label"`
			HomeTeamName     string `json:"homeTeamName"`
			HomeTeamColor1   string `json:"homeTeamColor1"`
			HomeTeamColor2   string `json:"homeTeamColor2"`
			AwayTeamName     string `json:"awayTeamName"`
			AwayTeamColor1   string `json:"awayTeamColor1"`
			AwayTeamColor2   string `json:"awayTeamColor2"`
			BrandingImageURL string `json:"brandingImageUrl"`
			BrandingImageAlt string `json:"brandingImageAlt"`

			// BDL Event linking (optional)
			BDLEventID *int64 `json:"bdlEventId,omitempty"`

			// Payout configuration (optional, overrides pool's numberSetConfig for payout periods)
			PayoutConfig *string `json:"payoutConfig,omitempty"`

			// Legacy single set (for "single" config)
			HomeTeamNumbers []int `json:"homeTeamNumbers"`
			AwayTeamNumbers []int `json:"awayTeamNumbers"`

			// Multiple number sets (for multi-set configs)
			NumberSets map[model.NumberSetType]numberSetPayload `json:"numberSets"`

			// LockPool controls whether to lock the pool after drawing numbers
			// nil = default (lock if not already locked), true = lock, false = don't lock
			LockPool *bool `json:"lockPool,omitempty"`
		} `json:"data,omitempty"`
	}

	type drawResponse struct {
		*model.GridJSON
		PoolLocks time.Time `json:"poolLocks"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
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
			s.writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		var grid *model.Grid
		if gridID > 0 {
			var err error
			grid, err = pool.GridByID(r.Context(), gridID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
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

			if err := grid.LoadAnnotations(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else if data.Action != "save" {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("cannot call action %s without an ID", data.Action))
			return
		}

		switch data.Action {
		case "drawManualNumbers":
			config := pool.NumberSetConfig()

			// Handle multi-set configs
			if config != model.NumberSetConfigStandard && data.Data.NumberSets != nil {
				// Convert payload to model input
				numberSets := make(map[model.NumberSetType]model.NumberSetInput)
				for setType, ns := range data.Data.NumberSets {
					numberSets[setType] = model.NumberSetInput{
						HomeNumbers: ns.HomeTeamNumbers,
						AwayNumbers: ns.AwayTeamNumbers,
					}
				}

				if err := grid.DrawAllNumbersManual(r.Context(), config, numberSets); err != nil {
					s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("could not set manual numbers: %w", err))
					return
				}
			} else {
				// Legacy single set behavior
				if err := grid.SetManualNumbers(data.Data.HomeTeamNumbers, data.Data.AwayTeamNumbers); err != nil {
					s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("could not set manual numbers: %w", err))
					return
				}

				if err := grid.Save(r.Context()); err != nil {
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
			}

			// Load number sets for response
			if config != model.NumberSetConfigStandard {
				if err := grid.LoadNumberSets(r.Context()); err != nil {
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
			}

			// Lock the pool if requested (defaults to true when not already locked)
			shouldLock := !pool.IsLocked() // Default: lock if not already locked
			if data.Data != nil && data.Data.LockPool != nil {
				shouldLock = *data.Data.LockPool && !pool.IsLocked()
			}
			if shouldLock {
				pool.SetLocks(time.Now())
				if err := pool.Save(r.Context()); err != nil {
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
			}

			// Load the BDL event so it's included in the response
			if err := grid.LoadBDLEvent(r.Context()); err != nil {
				logrus.WithError(err).Warn("could not load BDL event after draw")
			}

			s.broker.Publish(pool.Token(), PoolEvent{Type: EventGridUpdated})

			s.writeJSONResponse(w, http.StatusOK, drawResponse{
				GridJSON:  grid.JSON(),
				PoolLocks: pool.Locks(),
			})
			return
		case "drawNumbers":
			config := pool.NumberSetConfig()

			// Handle multi-set configs
			if config != model.NumberSetConfigStandard {
				if err := grid.DrawAllNumbersRandom(r.Context(), config); err != nil {
					if err == model.ErrNumbersAlreadyDrawn {
						s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("the numbers have already been drawn"))
						return
					}
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
			} else {
				// Legacy single set behavior
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
			}

			// Load number sets for response
			if config != model.NumberSetConfigStandard {
				if err := grid.LoadNumberSets(r.Context()); err != nil {
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
			}

			// Lock the pool if requested (defaults to true when not already locked)
			shouldLock := !pool.IsLocked() // Default: lock if not already locked
			if data.Data != nil && data.Data.LockPool != nil {
				shouldLock = *data.Data.LockPool && !pool.IsLocked()
			}
			if shouldLock {
				pool.SetLocks(time.Now())
				if err := pool.Save(r.Context()); err != nil {
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
			}

			// Load the BDL event so it's included in the response
			if err := grid.LoadBDLEvent(r.Context()); err != nil {
				logrus.WithError(err).Warn("could not load BDL event after draw")
			}

			s.broker.Publish(pool.Token(), PoolEvent{Type: EventGridUpdated})

			s.writeJSONResponse(w, http.StatusOK, drawResponse{
				GridJSON:  grid.JSON(),
				PoolLocks: pool.Locks(),
			})
			return
		case "save":
			if data.Data == nil {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("missing data in payload"))
				return
			}

			v := validator.New()
			eventDate := v.Datetime("Event Date", data.Data.EventDate, "00:00", "0", true)
			label := v.Printable("Label", data.Data.Label, true)
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
			brandingImageURL := v.URL("Branding Image URL", data.Data.BrandingImageURL, true)
			brandingImageURL = v.MaxLength("Branding Image URL", brandingImageURL, model.BrandingImageURLMaxLength)
			brandingImageAlt := v.Printable("Branding Image Alt", data.Data.BrandingImageAlt, true)
			brandingImageAlt = v.MaxLength("Branding Image Alt", brandingImageAlt, model.BrandingImageAltMaxLength)

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

			// Prevent changing linked event if current event is final
			if grid.BDLEventID() != nil {
				if err := grid.LoadBDLEvent(r.Context()); err != nil {
					s.writeErrorResponse(w, http.StatusInternalServerError, err)
					return
				}
				if grid.BDLEvent() != nil && grid.BDLEvent().Status == model.BDLEventStatusFinal {
					// Check if trying to change to a different event (or unlink)
					currentID := *grid.BDLEventID()
					newID := data.Data.BDLEventID
					if newID == nil || *newID != currentID {
						s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
							Status: statusError,
							Error:  "Cannot change linked event after the game has ended",
						})
						return
					}
				}
			}

			grid.SetBDLEventID(data.Data.BDLEventID)

			// Auto-populate team names and colors when event is linked (only if empty)
			if data.Data.BDLEventID != nil {
				event, err := s.model.SportsEventByIDWithTeams(r.Context(), *data.Data.BDLEventID)
				if err == nil && event != nil {
					// Set full team names only if not provided
					if homeTeamName == "" && event.HomeTeam() != nil {
						homeTeamName = event.HomeTeam().FullName
					}
					if awayTeamName == "" && event.AwayTeam() != nil {
						awayTeamName = event.AwayTeam().FullName
					}

					// Set colors only if not provided and team has them
					if homeTeamColor1 == "" && event.HomeTeam() != nil && event.HomeTeam().Color != nil {
						homeTeamColor1 = "#" + *event.HomeTeam().Color
					}
					if homeTeamColor2 == "" && event.HomeTeam() != nil && event.HomeTeam().AlternateColor != nil {
						homeTeamColor2 = "#" + *event.HomeTeam().AlternateColor
					}
					if awayTeamColor1 == "" && event.AwayTeam() != nil && event.AwayTeam().Color != nil {
						awayTeamColor1 = "#" + *event.AwayTeam().Color
					}
					if awayTeamColor2 == "" && event.AwayTeam() != nil && event.AwayTeam().AlternateColor != nil {
						awayTeamColor2 = "#" + *event.AwayTeam().AlternateColor
					}
				}
			}

			grid.SetEventDate(eventDate)
			grid.SetLabel(label)
			grid.SetHomeTeamName(homeTeamName)
			grid.SetAwayTeamName(awayTeamName)
			grid.SetRollover(data.Data.Rollover)

			// Handle payout config - validate and set if provided
			if data.Data.PayoutConfig != nil {
				if *data.Data.PayoutConfig == "" {
					// Empty string means clear the payout config (use pool default)
					grid.SetPayoutConfig(nil)
				} else if !model.IsValidNumberSetConfig(*data.Data.PayoutConfig) {
					s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
						Status: statusError,
						Error:  "Invalid payout configuration",
					})
					return
				} else {
					config := model.NumberSetConfig(*data.Data.PayoutConfig)

					// Validate config is valid for the linked event's league
					if data.Data.BDLEventID != nil {
						event, err := s.model.SportsEventByID(r.Context(), *data.Data.BDLEventID)
						if err == nil && event != nil {
							if !model.IsValidNumberSetConfigForLeague(config, event.League) {
								s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
									Status: statusError,
									Error:  fmt.Sprintf("The payout configuration '%s' is not valid for %s games", *data.Data.PayoutConfig, event.League),
								})
								return
							}
						}
					}

					grid.SetPayoutConfig(&config)
				}
			}
			settings := grid.Settings()
			settings.SetNotes(notes)
			settings.SetHomeTeamColor1(homeTeamColor1)
			settings.SetHomeTeamColor2(homeTeamColor2)
			settings.SetAwayTeamColor1(awayTeamColor1)
			settings.SetAwayTeamColor2(awayTeamColor2)
			settings.SetBrandingImageURL(brandingImageURL)
			settings.SetBrandingImageAlt(brandingImageAlt)

			if err := grid.Save(r.Context()); err != nil {
				if err == model.ErrGridLimit {
					s.writeErrorResponse(w, http.StatusBadRequest, err)
					return
				}

				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			// Load the BDL event so it's included in the response
			if err := grid.LoadBDLEvent(r.Context()); err != nil {
				logrus.WithError(err).Warn("could not load BDL event after save")
			}

			s.broker.Publish(pool.Token(), PoolEvent{Type: EventGridUpdated})

			s.writeJSONResponse(w, http.StatusAccepted, grid.JSON())
			return
		}

		s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("unsupported action %s", data.Action))
	}
}

func (s *Server) postPoolTokenGridIDSquareSquareIDAnnotationEndpoint() http.HandlerFunc {
	type payload struct {
		Annotation string `json:"annotation"`
		Icon       int16  `json:"icon"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		grid, ok := gridFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		squareID, ok := squareIDFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		var payloadData payload
		if err := json.NewDecoder(r.Body).Decode(&payloadData); err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		v := validator.New()
		annotation := v.Printable("annotation", payloadData.Annotation, false)

		if !model.AnnotationIcons.IsValidIcon(payloadData.Icon) {
			v.AddError("icon", "%d is not a valid annotation icon", payloadData.Icon)
		}

		if !v.OK() {
			s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
				Status:           statusError,
				Error:            "There were one or more validation errors",
				ValidationErrors: v.Errors,
			})
			return
		}

		a, err := grid.AnnotationBySquareID(r.Context(), squareID)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		a.Annotation = annotation
		a.Icon = payloadData.Icon
		isNew := a.Created.IsZero()
		if err := a.Save(r.Context()); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		pool, _ := poolFromContext(r.Context())
		if pool != nil {
			s.broker.Publish(pool.Token(), PoolEvent{Type: EventGridUpdated})
		}

		status := http.StatusOK
		if isNew {
			status = http.StatusCreated
		}

		s.writeJSONResponse(w, status, a)
	}
}

func (s *Server) deletePoolTokenGridIDSquareSquareIDAnnotationEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		grid, ok := gridFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		squareID, ok := squareIDFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		if err := grid.DeleteAnnotationBySquareID(r.Context(), squareID); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		pool, _ := poolFromContext(r.Context())
		if pool != nil {
			s.broker.Publish(pool.Token(), PoolEvent{Type: EventGridUpdated})
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) postPoolTokenMemberEndpoint() http.HandlerFunc {
	type payload struct {
		Password string `json:"password"`
		JWT      string `json:"jwt"`
		Invite   string `json:"invite"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		token := mux.Vars(r)["token"]
		pool, err := s.model.PoolByToken(r.Context(), token)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
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

		if data.Invite != "" {
			invite, err := s.model.PoolInviteByToken(r.Context(), data.Invite)
			if err != nil {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("invalid invite token"))
				return
			}

			if invite.PoolID != pool.ID() ||
				!pool.CheckIDIsValid(invite.CheckID) ||
				time.Now().After(invite.ExpiresAt) {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("invalid invite token"))
				return
			}
		} else if data.JWT != "" {
			j, err := s.smjwt.Validate(data.JWT, &inviteClaims{})
			if err != nil {
				s.writeErrorResponse(w, http.StatusBadRequest, err)
				return
			}

			claims := j.Claims.(*inviteClaims)

			// Verify audience
			audValid := false
			for _, aud := range claims.Audience {
				if aud == sqmgrInviteAudience {
					audValid = true
					break
				}
			}

			if !audValid ||
				claims.Issuer != model.IssuerSqMGR ||
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

func (s *Server) postPoolTokenSquaresBulkEndpoint() http.HandlerFunc {
	type squareResult struct {
		SquareID int    `json:"squareId"`
		OK       bool   `json:"ok"`
		Error    string `json:"error,omitempty"`
	}

	type requestPayload struct {
		SquareIDs []int                 `json:"squareIds"`
		Action    string                `json:"action"`
		Claimant  string                `json:"claimant"`
		State     model.PoolSquareState `json:"state"`
		Note      string                `json:"note"`
	}

	type response struct {
		Results []squareResult `json:"results"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool, ok := poolFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}
		user, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		var req requestPayload
		if ok := s.parseJSONPayload(w, r, &req); !ok {
			return
		}

		if len(req.SquareIDs) == 0 {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("squareIds must not be empty"))
			return
		}

		switch req.Action {
		case "claim":
			if req.Claimant == "" {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("claimant is required for claim action"))
				return
			}
		case "unclaim":
			// no additional fields required
		case "set_state":
			if req.State != model.PoolSquareStateClaimed && req.State != model.PoolSquareStatePaidPartial && req.State != model.PoolSquareStatePaidFull {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("state must be claimed, paid-partial, or paid-full"))
				return
			}
		default:
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid action: %s", req.Action))
			return
		}

		results := make([]squareResult, 0, len(req.SquareIDs))
		for _, squareID := range req.SquareIDs {
			if squareID < 1 || squareID > pool.NumberOfSquares() {
				results = append(results, squareResult{
					SquareID: squareID,
					OK:       false,
					Error:    "invalid square ID",
				})
				continue
			}

			square, err := pool.SquareBySquareID(squareID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					results = append(results, squareResult{SquareID: squareID, OK: false, Error: "square not found"})
				} else {
					results = append(results, squareResult{SquareID: squareID, OK: false, Error: "internal error"})
				}
				continue
			}

			if pool.GridType() == model.GridTypeRoll100 && square.ParentID > 0 {
				results = append(results, squareResult{
					SquareID: squareID,
					OK:       false,
					Error:    "cannot directly edit a secondary square; edit the primary square instead",
				})
				continue
			}

			if req.Action == "claim" && square.State != model.PoolSquareStateUnclaimed {
				results = append(results, squareResult{SquareID: squareID, OK: false, Error: "already claimed"})
				continue
			}

			if req.Action == "set_state" && square.State == model.PoolSquareStateUnclaimed {
				results = append(results, squareResult{SquareID: squareID, OK: false, Error: "square must be claimed first"})
				continue
			}

			tx, err := s.model.DB.BeginTx(r.Context(), nil)
			if err != nil {
				results = append(results, squareResult{SquareID: squareID, OK: false, Error: "internal error"})
				continue
			}

			var saveErr error
			switch req.Action {
			case "claim":
				square.SetClaimant(req.Claimant)
				square.State = model.PoolSquareStateClaimed
				square.SetUserID(user.ID)
				saveErr = square.Save(r.Context(), tx, true, model.PoolSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       "admin: bulk claim",
				})
			case "unclaim":
				square.State = model.PoolSquareStateUnclaimed
				saveErr = square.Save(r.Context(), tx, true, model.PoolSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       "admin: bulk unclaim",
				})
				if saveErr == nil && pool.GridType() == model.GridTypeRoll100 {
					childSquares, childErr := square.ChildSquares(r.Context(), tx)
					if childErr != nil {
						saveErr = childErr
					} else {
						for _, child := range childSquares {
							child.State = model.PoolSquareStateUnclaimed
							if childErr = child.Save(r.Context(), tx, true, model.PoolSquareLog{
								RemoteAddr: r.RemoteAddr,
								Note:       fmt.Sprintf("admin: bulk unclaim (secondary of square %d)", square.SquareID),
							}); childErr != nil {
								saveErr = childErr
								break
							}
						}
					}
				}
			case "set_state":
				square.State = req.State
				setStateNote := fmt.Sprintf("admin: bulk set state to %s", req.State)
				if req.Note != "" {
					setStateNote = req.Note
				}
				saveErr = square.Save(r.Context(), tx, true, model.PoolSquareLog{
					RemoteAddr: r.RemoteAddr,
					Note:       setStateNote,
				})
			}

			if saveErr != nil {
				_ = tx.Rollback()
				errMsg := "internal error"
				if saveErr == model.ErrSquareAlreadyClaimed {
					errMsg = "already claimed"
				}
				results = append(results, squareResult{SquareID: squareID, OK: false, Error: errMsg})
				continue
			}

			if err := tx.Commit(); err != nil {
				results = append(results, squareResult{SquareID: squareID, OK: false, Error: "internal error"})
				continue
			}

			results = append(results, squareResult{SquareID: squareID, OK: true})
		}

		s.broker.Publish(pool.Token(), PoolEvent{Type: EventSquareUpdated})
		s.writeJSONResponse(w, http.StatusOK, response{Results: results})
	}
}

type poolResponse struct {
	*model.PoolJSON
	IsAdmin                  bool `json:"isAdmin"`
	CanChangeNumberSetConfig bool `json:"canChangeNumberSetConfig,omitempty"`
}
