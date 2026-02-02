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
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

const defaultSportsEventsLimit = 50
const maxSportsEventsLimit = 500
const minSearchLength = 2

// getSportsEventsEndpoint returns sports events with optional filters
func (s *Server) getSportsEventsEndpoint() http.HandlerFunc {
	type response struct {
		Events []*model.SportsEventJSON `json:"events"`
		Total  int64                    `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		leagueStr := r.FormValue("league")
		if leagueStr == "" {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("missing required field: league"))
			return
		}

		if !model.IsValidSportsLeague(leagueStr) {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("invalid league"))
			return
		}

		league := model.SportsLeague(leagueStr)
		status := r.FormValue("status")
		search := r.FormValue("search")

		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit <= 0 {
			limit = defaultSportsEventsLimit
		}
		if limit > maxSportsEventsLimit {
			limit = maxSportsEventsLimit
		}

		var events []*model.SportsEvent
		var total int64
		var err error

		// If search term is provided (min 2 chars), use search function
		if len(search) >= minSearchLength {
			events, total, err = s.model.SearchSportsEvents(r.Context(), league, status, search, offset, limit)
		} else if status == "scheduled" {
			events, err = s.model.UpcomingSportsEvents(r.Context(), league, limit)
			total = int64(len(events))
		} else if status == "scheduled,in_progress" || status == "in_progress,scheduled" {
			events, total, err = s.model.LinkableSportsEventsWithTotal(r.Context(), league, offset, limit)
		} else if strings.Contains(status, ",") {
			// For other comma-separated values, reject for now
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("unsupported status combination"))
			return
		} else {
			events, err = s.model.SportsEventsByLeague(r.Context(), league, status, limit)
			total = int64(len(events))
		}

		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		// Load teams for all events
		if err := s.model.LoadTeamsForSportsEvents(r.Context(), events); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		eventJSONs := make([]*model.SportsEventJSON, len(events))
		for i, e := range events {
			eventJSONs[i] = e.JSON()
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			Events: eventJSONs,
			Total:  total,
		})
	}
}

// getSportsEventEndpoint returns a single sports event by ID
func (s *Server) getSportsEventEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := mux.Vars(r)["id"]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("invalid event ID"))
			return
		}

		event, err := s.model.SportsEventByIDWithTeams(r.Context(), id)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if event == nil {
			s.writeErrorResponse(w, http.StatusNotFound, nil)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, event.JSON())
	}
}

// getSportsTeamsEndpoint returns sports teams for a league
func (s *Server) getSportsTeamsEndpoint() http.HandlerFunc {
	type response struct {
		Teams []*model.SportsTeamJSON `json:"teams"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		leagueStr := r.FormValue("league")
		if leagueStr == "" {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("missing required field: league"))
			return
		}

		if !model.IsValidSportsLeague(leagueStr) {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("invalid league"))
			return
		}

		league := model.SportsLeague(leagueStr)

		teams, err := s.model.SportsTeamsByLeague(r.Context(), league)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		teamJSONs := make([]*model.SportsTeamJSON, len(teams))
		for i, t := range teams {
			teamJSONs[i] = t.JSON()
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			Teams: teamJSONs,
		})
	}
}

// getSportsLeaguesEndpoint returns all supported sports leagues
func (s *Server) getSportsLeaguesEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.writeJSONResponse(w, http.StatusOK, model.ValidSportsLeagues())
	}
}
