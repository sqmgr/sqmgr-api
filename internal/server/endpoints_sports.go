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
