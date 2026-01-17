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
	"net/http"
	"net/http/httptest"
	"testing"

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
	s := &Server{}

	// Create request and recorder
	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	rec := httptest.NewRecorder()

	// Add admin user to context
	adminUser := &model.User{IsAdmin: true}
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
	s := &Server{}

	// Create request and recorder
	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	rec := httptest.NewRecorder()

	// Add non-admin user to context
	nonAdminUser := &model.User{IsAdmin: false}
	ctx := context.WithValue(req.Context(), ctxUserKey, nonAdminUser)

	// Call the handler
	s.adminHandler(nextHandler).ServeHTTP(rec, req.WithContext(ctx))

	// Verify non-admin user is rejected
	g.Expect(nextHandlerCalled).Should(gomega.BeFalse())
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusForbidden))
}
