/*
Copyright (C) 2026 Tom Peters

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
	"fmt"
	"net/http"
	"strconv"

	"github.com/sqmgr/sqmgr-api/pkg/model"
)

const defaultTopCreatorsLimit = 10
const maxTopCreatorsLimit = 100

// validTimeSeriesMetrics is the set of metrics accepted by the timeseries endpoint
var validTimeSeriesMetrics = map[string]bool{
	"pools_created":       true,
	"users_registered":    true,
	"guest_users_created": true,
	"squares_claimed":     true,
	"grids_created":       true,
	"pool_members_joined": true,
}

// validTimeSeriesIntervals is the set of bucket sizes accepted by the timeseries endpoint
var validTimeSeriesIntervals = map[string]bool{
	"day":   true,
	"week":  true,
	"month": true,
	"year":  true,
}

// queryDateRange parses the start and end query parameters into a DateRange.
// A parse failure is written as a 400 and false is returned.
func (s *Server) queryDateRange(w http.ResponseWriter, r *http.Request) (model.DateRange, bool) {
	dr, err := parseDateRange(r.FormValue("start"), r.FormValue("end"))
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err)
		return dr, false
	}
	return dr, true
}

// getAdminTimeSeriesEndpoint returns a metric counted per time bucket
func (s *Server) getAdminTimeSeriesEndpoint() http.HandlerFunc {
	type response struct {
		Metric   string                  `json:"metric"`
		Interval string                  `json:"interval"`
		Points   []model.TimeSeriesPoint `json:"points"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		metric := r.FormValue("metric")
		if !validTimeSeriesMetrics[metric] {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid metric %q", metric))
			return
		}

		interval := r.FormValue("interval")
		if interval == "" {
			interval = "day"
		}
		if !validTimeSeriesIntervals[interval] {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid interval %q", interval))
			return
		}

		dr, ok := s.queryDateRange(w, r)
		if !ok {
			return
		}

		points, err := queryAnalytics(s, r.Context(), func(a *model.Analytics) ([]model.TimeSeriesPoint, error) {
			return a.TimeSeries(r.Context(), metric, interval, dr)
		})
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
		if points == nil {
			points = []model.TimeSeriesPoint{}
		}

		s.writeJSONResponse(w, http.StatusOK, response{Metric: metric, Interval: interval, Points: points})
	}
}

// getAdminFillRatesEndpoint returns how full pools get
func (s *Server) getAdminFillRatesEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dr, ok := s.queryDateRange(w, r)
		if !ok {
			return
		}

		filter := model.PoolFillRateFilter{Created: dr}

		if gridType := r.FormValue("gridType"); gridType != "" {
			if err := model.IsValidGridType(gridType); err != nil {
				s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid grid type %q", gridType))
				return
			}
			filter.GridType = model.GridType(gridType)
		}

		archived, ok := s.queryOptionalBool(w, r, "archived")
		if !ok {
			return
		}
		filter.Archived = archived

		rates, err := queryAnalytics(s, r.Context(), func(a *model.Analytics) (*model.PoolFillRates, error) {
			return a.PoolFillRates(r.Context(), filter)
		})
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, rates)
	}
}

// queryOptionalBool parses an optional true/false query parameter. A bad value
// is written as a 400 and false is returned as the second value.
func (s *Server) queryOptionalBool(w http.ResponseWriter, r *http.Request, name string) (*bool, bool) {
	value := r.FormValue(name)
	if value == "" {
		return nil, true
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid %s %q; expected true or false", name, value))
		return nil, false
	}
	return &b, true
}

// getAdminBreakdownEndpoint groups pools, grids, or squares by one dimension
func (s *Server) getAdminBreakdownEndpoint() http.HandlerFunc {
	type response struct {
		Dimension string               `json:"dimension"`
		Rows      []model.BreakdownRow `json:"rows"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		dimension := r.FormValue("dimension")
		if _, ok := model.BreakdownDimensions()[dimension]; !ok {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid dimension %q", dimension))
			return
		}

		dr, ok := s.queryDateRange(w, r)
		if !ok {
			return
		}

		rows, err := queryAnalytics(s, r.Context(), func(a *model.Analytics) ([]model.BreakdownRow, error) {
			return a.PoolBreakdown(r.Context(), dimension, dr)
		})
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
		if rows == nil {
			rows = []model.BreakdownRow{}
		}

		s.writeJSONResponse(w, http.StatusOK, response{Dimension: dimension, Rows: rows})
	}
}

// getAdminEngagementEndpoint returns the engagement summary for a date range
func (s *Server) getAdminEngagementEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dr, ok := s.queryDateRange(w, r)
		if !ok {
			return
		}

		summary, err := queryAnalytics(s, r.Context(), func(a *model.Analytics) (*model.EngagementSummary, error) {
			return a.EngagementSummary(r.Context(), dr)
		})
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, summary)
	}
}

// getAdminTopCreatorsEndpoint ranks users by pools created in a date range
func (s *Server) getAdminTopCreatorsEndpoint() http.HandlerFunc {
	type response struct {
		Creators []*model.PoolCreator `json:"creators"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		dr, ok := s.queryDateRange(w, r)
		if !ok {
			return
		}

		_, limit := parsePagination(r, defaultTopCreatorsLimit, maxTopCreatorsLimit)

		creators, err := queryAnalytics(s, r.Context(), func(a *model.Analytics) ([]*model.PoolCreator, error) {
			return a.TopPoolCreators(r.Context(), dr, limit)
		})
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
		if creators == nil {
			creators = []*model.PoolCreator{}
		}

		s.writeJSONResponse(w, http.StatusOK, response{Creators: creators})
	}
}

// parseAdminPoolsFilter builds the pool list filter from the request's query
// parameters. An error is returned for values that are present but invalid.
func parseAdminPoolsFilter(r *http.Request) (model.PoolListFilter, error) {
	filter := model.PoolListFilter{
		Search:     r.FormValue("search"),
		OwnerEmail: r.FormValue("ownerEmail"),
		SortBy:     r.FormValue("sortBy"),
		SortDir:    r.FormValue("sortDir"),
	}

	dr, err := parseDateRange(r.FormValue("start"), r.FormValue("end"))
	if err != nil {
		return filter, err
	}
	filter.Created = dr

	if gridType := r.FormValue("gridType"); gridType != "" {
		if err := model.IsValidGridType(gridType); err != nil {
			return filter, fmt.Errorf("invalid grid type %q", gridType)
		}
		filter.GridType = model.GridType(gridType)
	}

	if archived := r.FormValue("archived"); archived != "" {
		b, err := strconv.ParseBool(archived)
		if err != nil {
			return filter, fmt.Errorf("invalid archived %q; expected true or false", archived)
		}
		filter.Archived = &b
	}

	for _, bound := range []struct {
		name string
		dest **float64
	}{
		{"minFill", &filter.MinFillPercent},
		{"maxFill", &filter.MaxFillPercent},
	} {
		value := r.FormValue(bound.name)
		if value == "" {
			continue
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil || f < 0 || f > 100 {
			return filter, fmt.Errorf("invalid %s %q; expected a number from 0 to 100", bound.name, value)
		}
		*bound.dest = &f
	}

	if filter.MinFillPercent != nil && filter.MaxFillPercent != nil && *filter.MinFillPercent > *filter.MaxFillPercent {
		return filter, errors.New("minFill cannot be greater than maxFill")
	}

	filter.Offset, filter.Limit = parsePagination(r, defaultAdminPoolsLimit, maxAdminPoolsLimit)

	return filter, nil
}
