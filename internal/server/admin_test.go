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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

func TestAdminHandler_AllowsAdmin(t *testing.T) {
	g := gomega.NewWithT(t)

	// Track whether the next handler was called
	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create a minimal server for testing
	s := &Server{broker: NewPoolBroker()}

	// Create request and recorder
	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	rec := httptest.NewRecorder()

	// Add admin user to context
	adminUser := &model.User{IsSiteAdmin: true}
	ctx := context.WithValue(req.Context(), ctxUserKey, adminUser)

	// Call the handler
	s.adminHandler(nextHandler).ServeHTTP(rec, req.WithContext(ctx))

	// Verify admin user passes through
	g.Expect(nextHandlerCalled).Should(gomega.BeTrue())
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
}

func TestAdminHandler_DeniesNonAdmin(t *testing.T) {
	g := gomega.NewWithT(t)

	// Track whether the next handler was called
	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create a minimal server for testing
	s := &Server{broker: NewPoolBroker()}

	// Create request and recorder
	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	rec := httptest.NewRecorder()

	// Add non-admin user to context
	nonAdminUser := &model.User{IsSiteAdmin: false}
	ctx := context.WithValue(req.Context(), ctxUserKey, nonAdminUser)

	// Call the handler
	s.adminHandler(nextHandler).ServeHTTP(rec, req.WithContext(ctx))

	// Verify non-admin user is rejected
	g.Expect(nextHandlerCalled).Should(gomega.BeFalse())
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusForbidden))
}

func TestAdminHandler_MissingUserContext(t *testing.T) {
	g := gomega.NewWithT(t)

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	s := &Server{broker: NewPoolBroker()}

	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	rec := httptest.NewRecorder()

	// No user in context
	s.adminHandler(nextHandler).ServeHTTP(rec, req)

	g.Expect(nextHandlerCalled).Should(gomega.BeFalse())
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestValidStatsPeriods(t *testing.T) {
	g := gomega.NewWithT(t)

	// Test valid periods
	g.Expect(validStatsPeriods["all"]).Should(gomega.BeTrue())
	g.Expect(validStatsPeriods["24h"]).Should(gomega.BeTrue())
	g.Expect(validStatsPeriods["week"]).Should(gomega.BeTrue())
	g.Expect(validStatsPeriods["month"]).Should(gomega.BeTrue())
	g.Expect(validStatsPeriods["year"]).Should(gomega.BeTrue())

	g.Expect(validStatsPeriods["1h"]).Should(gomega.BeTrue())
	g.Expect(validStatsPeriods["custom"]).Should(gomega.BeTrue())

	// Test invalid periods
	g.Expect(validStatsPeriods["invalid"]).Should(gomega.BeFalse())
	g.Expect(validStatsPeriods[""]).Should(gomega.BeFalse())
	g.Expect(validStatsPeriods["day"]).Should(gomega.BeFalse())
}

// statsRequest builds a GET /admin/stats request with the given query string
func statsRequest(query string) *http.Request {
	target := "/admin/stats"
	if query != "" {
		target += "?" + query
	}
	return httptest.NewRequest(http.MethodGet, target, nil)
}

func TestParseStatsFilter_Periods(t *testing.T) {
	g := gomega.NewWithT(t)

	// Every existing period is passed through untouched
	for _, period := range []string{"all", "1h", "24h", "week", "month", "year"} {
		filter, err := parseStatsFilter(statsRequest("period=" + period))
		g.Expect(err).Should(gomega.Succeed(), "period %q", period)
		g.Expect(filter.Period).Should(gomega.Equal(period), "period %q", period)
		g.Expect(filter.Start.IsZero()).Should(gomega.BeTrue(), "period %q", period)
		g.Expect(filter.End.IsZero()).Should(gomega.BeTrue(), "period %q", period)
	}

	// Unknown, missing, and empty periods fall back to "all"
	for _, query := range []string{"period=bogus", "period=", ""} {
		filter, err := parseStatsFilter(statsRequest(query))
		g.Expect(err).Should(gomega.Succeed(), "query %q", query)
		g.Expect(filter.Period).Should(gomega.Equal(model.StatsPeriodAll), "query %q", query)
	}

	// start and end are ignored unless the period is custom
	filter, err := parseStatsFilter(statsRequest("period=week&start=2024-01-01&end=2024-01-31"))
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(filter.Period).Should(gomega.Equal("week"))
	g.Expect(filter.Start.IsZero()).Should(gomega.BeTrue())
	g.Expect(filter.End.IsZero()).Should(gomega.BeTrue())

	// Garbage dates are also ignored unless the period is custom
	_, err = parseStatsFilter(statsRequest("period=month&start=nope&end=nope"))
	g.Expect(err).Should(gomega.Succeed())
}

func TestParseStatsFilter_Custom(t *testing.T) {
	g := gomega.NewWithT(t)

	filter, err := parseStatsFilter(statsRequest("period=custom&start=2024-01-01&end=2024-01-31"))
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(filter.Period).Should(gomega.Equal(model.StatsPeriodCustom))
	g.Expect(filter.Start).Should(gomega.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))
	g.Expect(filter.End).Should(gomega.Equal(time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)))

	// Same day for start and end is valid
	filter, err = parseStatsFilter(statsRequest("period=custom&start=2024-01-01&end=2024-01-01"))
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(filter.Start).Should(gomega.Equal(filter.End))
}

func TestParseStatsFilter_CustomErrors(t *testing.T) {
	g := gomega.NewWithT(t)

	tests := map[string]string{
		"missing both":     "period=custom",
		"missing start":    "period=custom&end=2024-01-31",
		"missing end":      "period=custom&start=2024-01-01",
		"empty start":      "period=custom&start=&end=2024-01-31",
		"empty end":        "period=custom&start=2024-01-01&end=",
		"malformed start":  "period=custom&start=01-01-2024&end=2024-01-31",
		"malformed end":    "period=custom&start=2024-01-01&end=not-a-date",
		"impossible date":  "period=custom&start=2024-13-45&end=2024-01-31",
		"timestamp start":  "period=custom&start=2024-01-01T00:00:00Z&end=2024-01-31",
		"start after end":  "period=custom&start=2024-02-01&end=2024-01-31",
		"start after by 1": "period=custom&start=2024-01-02&end=2024-01-01",
	}

	for name, query := range tests {
		_, err := parseStatsFilter(statsRequest(query))
		g.Expect(err).Should(gomega.HaveOccurred(), name)
	}
}

func TestGetAdminStatsEndpoint_BadRequests(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}
	s.Router.Path("/admin/stats").Methods(http.MethodGet).Handler(s.getAdminStatsEndpoint())

	// parseStatsFilter's own validation logic is exhaustively covered by
	// TestParseStatsFilter_CustomErrors; these two cases just prove the 400
	// wiring end-to-end through the actual HTTP handler.
	queries := []string{
		"period=custom", // missing start and end
		"period=custom&start=2024-02-01&end=2024-01-31", // start after end
	}

	for _, query := range queries {
		rec := httptest.NewRecorder()
		s.Router.ServeHTTP(rec, statsRequest(query))

		// Validation happens before the model is consulted, so no DB is needed
		g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest), query)
		g.Expect(rec.Body.String()).ShouldNot(gomega.BeEmpty(), query)
	}
}

func TestGetAdminUserEndpoint_InvalidID(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/admin/user/{id:[0-9]+}").Methods(http.MethodGet).Handler(s.getAdminUserEndpoint())

	// Test with non-numeric ID (should not match route)
	req := httptest.NewRequest(http.MethodGet, "/admin/user/abc", nil)
	rec := httptest.NewRecorder()

	adminUser := &model.User{IsSiteAdmin: true}
	ctx := context.WithValue(req.Context(), ctxUserKey, adminUser)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	// Route doesn't match, returns 404
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusNotFound))
}

func TestGetAdminUserPoolsEndpoint_InvalidID(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/admin/user/{id:[0-9]+}/pools").Methods(http.MethodGet).Handler(s.getAdminUserPoolsEndpoint())

	// Test with non-numeric ID (should not match route)
	req := httptest.NewRequest(http.MethodGet, "/admin/user/abc/pools", nil)
	rec := httptest.NewRecorder()

	adminUser := &model.User{IsSiteAdmin: true}
	ctx := context.WithValue(req.Context(), ctxUserKey, adminUser)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	// Route doesn't match, returns 404
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusNotFound))
}

func TestPostAdminPoolJoinEndpoint_MissingUserContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/admin/pool/{token}/join").Methods(http.MethodPost).Handler(s.postAdminPoolJoinEndpoint())

	req := httptest.NewRequest(http.MethodPost, "/admin/pool/test-token/join", nil)
	rec := httptest.NewRecorder()

	// No user in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestGetAdminUserPoolsEndpoint_DefaultPagination(t *testing.T) {
	g := gomega.NewWithT(t)

	// Verify default pagination values
	g.Expect(defaultAdminPoolsLimit).Should(gomega.Equal(25))
	g.Expect(maxAdminPoolsLimit).Should(gomega.Equal(100))
}
