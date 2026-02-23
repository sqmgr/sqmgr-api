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

	// Test invalid periods
	g.Expect(validStatsPeriods["invalid"]).Should(gomega.BeFalse())
	g.Expect(validStatsPeriods[""]).Should(gomega.BeFalse())
	g.Expect(validStatsPeriods["day"]).Should(gomega.BeFalse())
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
