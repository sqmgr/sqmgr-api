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
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

const defaultAdminPoolsLimit = 25
const maxAdminPoolsLimit = 100
const defaultAdminUsersLimit = 25
const maxAdminUsersLimit = 100
const defaultAdminEventsLimit = 25
const maxAdminEventsLimit = 100

// validStatsPeriods defines the valid period values for stats filtering
var validStatsPeriods = map[string]bool{
	"all":   true,
	"1h":    true,
	"24h":   true,
	"week":  true,
	"month": true,
	"year":  true,
}

// ensureUserEmail fetches and caches the email from Auth0 if needed
func (s *Server) ensureUserEmail(ctx context.Context, user *model.User) {
	if user.Store == model.UserStoreAuth0 && (user.Email == nil || *user.Email == "") {
		if s.auth0Client.IsConfigured() {
			if email, err := s.auth0Client.GetUserEmail(ctx, user.StoreID); err == nil && email != "" {
				if cacheErr := user.SetEmail(ctx, email); cacheErr != nil {
					logrus.WithError(cacheErr).Warn("could not cache user email")
				}
			}
		}
	}
}

// getAdminStatsEndpoint returns site-wide statistics
func (s *Server) getAdminStatsEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		period := r.FormValue("period")
		logrus.WithField("form period", period).Info("getting admin stats")
		if !validStatsPeriods[period] {
			period = "all"
		}

		logrus.WithField("set period", period).Info("getting admin stats")
		stats, err := s.model.GetAdminStats(r.Context(), period)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, stats)
	}
}

// getAdminPoolsEndpoint returns paginated list of all pools
func (s *Server) getAdminPoolsEndpoint() http.HandlerFunc {
	type response struct {
		Pools []*model.AdminPool `json:"pools"`
		Total int64              `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		search := r.FormValue("search")

		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit <= 0 {
			limit = defaultAdminPoolsLimit
		}
		if limit > maxAdminPoolsLimit {
			limit = maxAdminPoolsLimit
		}

		pools, err := s.model.GetAllPools(r.Context(), search, offset, limit)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		total, err := s.model.GetAllPoolsCount(r.Context(), search)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			Pools: pools,
			Total: total,
		})
	}
}

// postAdminPoolJoinEndpoint allows an admin to join any pool
func (s *Server) postAdminPoolJoinEndpoint() http.HandlerFunc {
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

		if err := user.JoinPool(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// getAdminUserEndpoint returns user profile and stats for admin view
func (s *Server) getAdminUserEndpoint() http.HandlerFunc {
	type userResponse struct {
		ID          int64           `json:"id"`
		Store       model.UserStore `json:"store"`
		StoreID     string          `json:"storeId"`
		Email       *string         `json:"email"`
		IsSiteAdmin bool            `json:"isSiteAdmin"`
		Created     string          `json:"created"`
	}
	type response struct {
		User  userResponse          `json:"user"`
		Stats *model.AdminUserStats `json:"stats"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		idStr := mux.Vars(r)["id"]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, nil)
			return
		}

		user, err := s.model.GetUserByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.ensureUserEmail(r.Context(), user)

		stats, err := s.model.GetUserStats(r.Context(), id)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			User: userResponse{
				ID:          user.ID,
				Store:       user.Store,
				StoreID:     user.StoreID,
				Email:       user.Email,
				IsSiteAdmin: user.IsSiteAdmin,
				Created:     user.Created.Format("2006-01-02T15:04:05Z07:00"),
			},
			Stats: stats,
		})
	}
}

// getAdminUserPoolsEndpoint returns paginated pools created by a specific user
func (s *Server) getAdminUserPoolsEndpoint() http.HandlerFunc {
	type response struct {
		Pools []*model.AdminPool `json:"pools"`
		Total int64              `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		idStr := mux.Vars(r)["id"]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, nil)
			return
		}

		// Verify user exists
		_, err = s.model.GetUserByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit <= 0 {
			limit = defaultAdminPoolsLimit
		}
		if limit > maxAdminPoolsLimit {
			limit = maxAdminPoolsLimit
		}

		includeArchived := r.FormValue("includeArchived") == "true"

		pools, err := s.model.GetPoolsByUserID(r.Context(), id, includeArchived, offset, limit)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		total, err := s.model.GetPoolsByUserIDCount(r.Context(), id, includeArchived)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			Pools: pools,
			Total: total,
		})
	}
}

// getAdminUsersEndpoint returns paginated list of all users
func (s *Server) getAdminUsersEndpoint() http.HandlerFunc {
	type response struct {
		Users []*model.AdminUser `json:"users"`
		Total int64              `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		search := r.FormValue("search")

		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit <= 0 {
			limit = defaultAdminUsersLimit
		}
		if limit > maxAdminUsersLimit {
			limit = maxAdminUsersLimit
		}

		sortBy := r.FormValue("sortBy")
		sortDir := r.FormValue("sortDir")

		users, err := s.model.GetAllUsers(r.Context(), search, offset, limit, sortBy, sortDir)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		total, err := s.model.GetAllUsersCount(r.Context(), search)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			Users: users,
			Total: total,
		})
	}
}

// getAdminEventsEndpoint returns paginated list of sports events with linked grids
func (s *Server) getAdminEventsEndpoint() http.HandlerFunc {
	type response struct {
		Events []*model.AdminLinkedEvent `json:"events"`
		Total  int64                     `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit <= 0 {
			limit = defaultAdminEventsLimit
		}
		if limit > maxAdminEventsLimit {
			limit = maxAdminEventsLimit
		}

		sortBy := r.FormValue("sortBy")
		sortDir := r.FormValue("sortDir")

		events, err := s.model.GetAdminLinkedEvents(r.Context(), offset, limit, sortBy, sortDir)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		total, err := s.model.GetAdminLinkedEventsCount(r.Context())
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			Events: events,
			Total:  total,
		})
	}
}

// getAdminEventGridsEndpoint returns grids linked to a specific sports event
func (s *Server) getAdminEventGridsEndpoint() http.HandlerFunc {
	type response struct {
		Grids []*model.AdminEventGrid `json:"grids"`
		Total int64                   `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		idStr := mux.Vars(r)["id"]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, nil)
			return
		}

		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit <= 0 {
			limit = defaultAdminEventsLimit
		}
		if limit > maxAdminEventsLimit {
			limit = maxAdminEventsLimit
		}

		grids, err := s.model.GetAdminEventGrids(r.Context(), id, offset, limit)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		total, err := s.model.GetAdminEventGridsCount(r.Context(), id)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			Grids: grids,
			Total: total,
		})
	}
}
