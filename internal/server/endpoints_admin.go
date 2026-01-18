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
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

const defaultAdminPoolsLimit = 25
const maxAdminPoolsLimit = 100

// validStatsPeriods defines the valid period values for stats filtering
var validStatsPeriods = map[string]bool{
	"all":   true,
	"1h":    true,
	"24h":   true,
	"week":  true,
	"month": true,
	"year":  true,
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

		if err := user.JoinPool(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
