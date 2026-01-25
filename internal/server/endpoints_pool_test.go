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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

func setupTestServerForPool(t *testing.T) (*Server, sqlmock.Sqlmock, *model.Model) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
	}

	s.Router.Path("/pool/{token}").Methods(http.MethodGet).Handler(s.getPoolTokenEndpoint())

	return s, mock, m
}

func poolColumns() []string {
	return []string{
		"id", "token", "user_id", "name", "grid_type", "number_set_config",
		"password_hash", "password_required", "open_access_on_lock", "locks",
		"created", "modified", "check_id", "archived",
	}
}

func gridColumns() []string {
	return []string{
		"id", "pool_id", "ord", "label", "home_team_name", "home_numbers",
		"away_team_name", "away_numbers", "event_date", "rollover", "state",
		"created", "modified", "manual_draw",
	}
}

func TestGetPoolTokenEndpoint_AdminReceivesCanChangeNumberSetConfig(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForPool(t)

	// User with ID 100 - embedded Model gives access to DB
	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	// Create pool via mock with user_id = 100 (same as user, so user is owner/admin)
	poolToken := "test-token-123"
	now := time.Now()

	// First query: load the pool for context
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	// Load pool for context
	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Since user is admin (owner - user.ID == pool.userID), CanChangeNumberSetConfig will be called
	// CanChangeNumberSetConfig calls pool.Grids which queries the database
	gridsRows := sqlmock.NewRows(gridColumns()) // Empty - no grids, so numbers haven't been drawn

	mock.ExpectQuery("SELECT .+ FROM grids WHERE pool_id = \\$1").
		WithArgs(int64(1), int64(0), 50).
		WillReturnRows(gridsRows)

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken, nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Admin should receive isAdmin = true and canChangeNumberSetConfig field
	g.Expect(result["isAdmin"]).Should(gomega.BeTrue())
	g.Expect(result).Should(gomega.HaveKey("canChangeNumberSetConfig"))
	g.Expect(result["canChangeNumberSetConfig"]).Should(gomega.BeTrue())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetPoolTokenEndpoint_NonAdminDoesNotReceiveCanChangeNumberSetConfig(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForPool(t)

	// User with ID 200 (different from pool owner) - embedded Model gives access to DB
	user := &model.User{
		Model: m,
		ID:    200,
		Store: model.UserStoreAuth0,
	}

	// Create pool via mock with user_id = 100 (user 200 is NOT owner)
	poolToken := "test-token-456"
	now := time.Now()

	// First query: load the pool for context
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	// Load pool for context
	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Since user (ID=200) is not owner (user_id=100), IsAdminOf will query pools_users
	// Return no rows to indicate user is not an admin
	mock.ExpectQuery("SELECT true FROM pools_users WHERE pool_id = \\$1 AND user_id = \\$2 AND is_admin").
		WithArgs(int64(1), int64(200)).
		WillReturnRows(sqlmock.NewRows([]string{"bool"})) // empty = not admin

	// Since user is NOT admin, CanChangeNumberSetConfig should NOT be called
	// No grids query expected

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken, nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Non-admin should receive isAdmin = false
	g.Expect(result["isAdmin"]).Should(gomega.BeFalse())

	// canChangeNumberSetConfig should NOT be present (omitempty in struct)
	_, hasKey := result["canChangeNumberSetConfig"]
	g.Expect(hasKey).Should(gomega.BeFalse(), "canChangeNumberSetConfig should not be present for non-admin")

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}
