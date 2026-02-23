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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
	"github.com/sqmgr/sqmgr-api/pkg/auth0"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

func TestPoolHandler_MissingUserContext(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
		broker: NewPoolBroker(),
	}

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	s.Router.Path("/pool/{token}").Methods(http.MethodGet).Handler(s.poolHandler(nextHandler))

	poolToken := "test-missing-user"
	now := time.Now()

	// Pool loads fine
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken, nil)
	rec := httptest.NewRecorder()

	// No user in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
	g.Expect(nextCalled).Should(gomega.BeFalse())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPoolGridHandler_MissingPoolContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	s.Router.Path("/pool/{token}/grid/{id}").Methods(http.MethodGet).Handler(s.poolGridHandler(nextHandler))

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token/grid/1", nil)
	rec := httptest.NewRecorder()

	// No pool in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
	g.Expect(nextCalled).Should(gomega.BeFalse())
}

func TestPoolGridSquareManagerHandler_MissingPoolContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	s.Router.Path("/pool/{token}/grid/{id}/square/{square_id}").Methods(http.MethodPost).Handler(s.poolGridSquareManagerHandler(nextHandler))

	req := httptest.NewRequest(http.MethodPost, "/pool/test-token/grid/1/square/5", nil)
	rec := httptest.NewRecorder()

	// No pool in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
	g.Expect(nextCalled).Should(gomega.BeFalse())
}

func TestPoolGridSquareManagerHandler_MissingUserContext(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
		broker: NewPoolBroker(),
	}

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	s.Router.Path("/pool/{token}/grid/{id}/square/{square_id}").Methods(http.MethodPost).Handler(s.poolGridSquareManagerHandler(nextHandler))

	poolToken := "test-missing-user-sq"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	if err != nil {
		t.Fatalf("failed to load pool: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/grid/1/square/5", nil)
	rec := httptest.NewRecorder()

	// Pool in context but no user
	ctx := context.WithValue(req.Context(), ctxPoolKey, poolForContext)
	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
	g.Expect(nextCalled).Should(gomega.BeFalse())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPoolManagerHandler_MissingPoolContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	s.Router.Path("/pool/{token}/test").Methods(http.MethodGet).Handler(s.poolManagerHandler(nextHandler))

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token/test", nil)
	rec := httptest.NewRecorder()

	// No pool in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
	g.Expect(nextCalled).Should(gomega.BeFalse())
}

func TestPoolManagerHandler_MissingUserContext(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
		broker: NewPoolBroker(),
	}

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	s.Router.Path("/pool/{token}/test").Methods(http.MethodGet).Handler(s.poolManagerHandler(nextHandler))

	poolToken := "test-admin-missing-user"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	if err != nil {
		t.Fatalf("failed to load pool: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken+"/test", nil)
	rec := httptest.NewRecorder()

	// Pool in context but no user
	ctx := context.WithValue(req.Context(), ctxPoolKey, poolForContext)
	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
	g.Expect(nextCalled).Should(gomega.BeFalse())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPoolManagerHandler_NonManagerGetsForbidden(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
		broker: NewPoolBroker(),
	}

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	s.Router.Path("/pool/{token}/test").Methods(http.MethodGet).Handler(s.poolManagerHandler(nextHandler))

	poolToken := "test-admin-nonadmin"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	if err != nil {
		t.Fatalf("failed to load pool: %v", err)
	}

	// User 200 is not the owner (100), so IsManagerOf queries DB
	mock.ExpectQuery("SELECT true FROM pools_users WHERE pool_id = \\$1 AND user_id = \\$2 AND is_manager").
		WithArgs(int64(1), int64(200)).
		WillReturnRows(sqlmock.NewRows([]string{"bool"}))

	user := &model.User{Model: m, ID: 200, Store: model.UserStoreAuth0}

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken+"/test", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)
	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusForbidden))
	g.Expect(nextCalled).Should(gomega.BeFalse())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetPoolTokenEndpoint_MissingUserContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token}").Methods(http.MethodGet).Handler(s.getPoolTokenEndpoint())

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token", nil)
	rec := httptest.NewRecorder()

	// No user in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestGetPoolTokenEndpoint_MissingPoolContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token}").Methods(http.MethodGet).Handler(s.getPoolTokenEndpoint())

	user := &model.User{ID: 100, Store: model.UserStoreAuth0}

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	// User in context but no pool
	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestGetPoolTokenSquareEndpoint_MissingPoolContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token}/square").Methods(http.MethodGet).Handler(s.getPoolTokenSquareEndpoint())

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token/square", nil)
	rec := httptest.NewRecorder()

	// No pool in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestPostPoolEndpoint_MissingUserContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool").Methods(http.MethodPost).Handler(s.postPoolEndpoint())

	req := httptest.NewRequest(http.MethodPost, "/pool", nil)
	rec := httptest.NewRecorder()

	// No user in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestPostPoolTokenMemberEndpoint_MissingUserContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token}/member").Methods(http.MethodPost).Handler(s.postPoolTokenMemberEndpoint())

	req := httptest.NewRequest(http.MethodPost, "/pool/test-token/member", nil)
	rec := httptest.NewRecorder()

	// No user in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestAnnotationEndpoint_MissingGridContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token}/grid/{id}/square/{square_id}/annotation").Methods(http.MethodPost).Handler(s.postPoolTokenGridIDSquareSquareIDAnnotationEndpoint())

	req := httptest.NewRequest(http.MethodPost, "/pool/test-token/grid/1/square/5/annotation", nil)
	rec := httptest.NewRecorder()

	// No grid in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestDeleteAnnotationEndpoint_MissingGridContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token}/grid/{id}/square/{square_id}/annotation").Methods(http.MethodDelete).Handler(s.deletePoolTokenGridIDSquareSquareIDAnnotationEndpoint())

	req := httptest.NewRequest(http.MethodDelete, "/pool/test-token/grid/1/square/5/annotation", nil)
	rec := httptest.NewRecorder()

	// No grid in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func setupTestServerForPool(t *testing.T) (*Server, sqlmock.Sqlmock, *model.Model) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
		broker: NewPoolBroker(),
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
		"created", "modified", "manual_draw", "sports_event_id", "payout_config",
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

	// Admin should receive hasManagerVisibility = true and canChangeNumberSetConfig field
	g.Expect(result["hasManagerVisibility"]).Should(gomega.BeTrue())
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

	// Since user (ID=200) is not owner (user_id=100), IsManagerOf will query pools_users
	// Return no rows to indicate user is not a manager
	mock.ExpectQuery("SELECT true FROM pools_users WHERE pool_id = \\$1 AND user_id = \\$2 AND is_manager").
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

	// Non-manager should receive hasManagerVisibility = false
	g.Expect(result["hasManagerVisibility"]).Should(gomega.BeFalse())

	// canChangeNumberSetConfig should NOT be present (omitempty in struct)
	_, hasKey := result["canChangeNumberSetConfig"]
	g.Expect(hasKey).Should(gomega.BeFalse(), "canChangeNumberSetConfig should not be present for non-admin")

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func setupTestServerForDrawNumbers(t *testing.T) (*Server, sqlmock.Sqlmock, *model.Model) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token}/grid/{id}").Methods(http.MethodPost).Handler(s.poolManagerHandler(s.postPoolTokenGridIDEndpoint()))

	return s, mock, m
}

func gridSettingsColumns() []string {
	return []string{
		"grid_id", "home_team_color_1", "home_team_color_2",
		"away_team_color_1", "away_team_color_2",
		"notes", "branding_image_url", "branding_image_alt", "modified",
	}
}

func TestDrawNumbers_LocksPoolByDefault(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForDrawNumbers(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-token-draw-1"
	now := time.Now()

	// Pool is NOT locked (locks = nil)
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Create a grid with no numbers drawn
	gridRows := sqlmock.NewRows(gridColumns()).
		AddRow(1, int64(1), 0, "Game 1", "Home Team", nil, "Away Team", nil, now, false, "active", now, now, false, nil, nil)

	mock.ExpectQuery("SELECT .+ FROM grids WHERE id = \\$1 AND pool_id = \\$2").
		WithArgs(int64(1), int64(1)).
		WillReturnRows(gridRows)

	// LoadSettings
	settingsRows := sqlmock.NewRows(gridSettingsColumns()).
		AddRow(int64(1), "#000000", "#FFFFFF", "#FF0000", "#00FF00", "", "", "", now)

	mock.ExpectQuery("SELECT .+ FROM grid_settings WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(settingsRows)

	// LoadAnnotations
	annotationsRows := sqlmock.NewRows([]string{"grid_id", "square_id", "annotation", "icon"})
	mock.ExpectQuery("SELECT .+ FROM grid_annotations WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(annotationsRows)

	// Grid save (uses transaction)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE grid_settings SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE grids SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// Pool save (for locking)
	mock.ExpectExec("UPDATE pools SET").WillReturnResult(sqlmock.NewResult(0, 1))

	body := `{"action": "drawNumbers"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/grid/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Response should include poolLocks
	g.Expect(result).Should(gomega.HaveKey("poolLocks"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestDrawNumbers_DoesNotLockPoolWhenLockPoolFalse(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForDrawNumbers(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-token-draw-2"
	now := time.Now()

	// Pool is NOT locked (locks = nil)
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Create a grid with no numbers drawn
	gridRows := sqlmock.NewRows(gridColumns()).
		AddRow(1, int64(1), 0, "Game 1", "Home Team", nil, "Away Team", nil, now, false, "active", now, now, false, nil, nil)

	mock.ExpectQuery("SELECT .+ FROM grids WHERE id = \\$1 AND pool_id = \\$2").
		WithArgs(int64(1), int64(1)).
		WillReturnRows(gridRows)

	// LoadSettings
	settingsRows := sqlmock.NewRows(gridSettingsColumns()).
		AddRow(int64(1), "#000000", "#FFFFFF", "#FF0000", "#00FF00", "", "", "", now)

	mock.ExpectQuery("SELECT .+ FROM grid_settings WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(settingsRows)

	// LoadAnnotations
	annotationsRows := sqlmock.NewRows([]string{"grid_id", "square_id", "annotation", "icon"})
	mock.ExpectQuery("SELECT .+ FROM grid_annotations WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(annotationsRows)

	// Grid save (uses transaction)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE grid_settings SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE grids SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// NO pool save expected since lockPool = false

	body := `{"action": "drawNumbers", "data": {"lockPool": false}}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/grid/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Response should include poolLocks (with zero value since not locked)
	g.Expect(result).Should(gomega.HaveKey("poolLocks"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestDrawNumbers_DoesNotLockAlreadyLockedPool(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForDrawNumbers(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-token-draw-3"
	now := time.Now()
	pastLocks := now.Add(-24 * time.Hour) // Already locked in the past

	// Pool IS already locked
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, pastLocks, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Create a grid with no numbers drawn
	gridRows := sqlmock.NewRows(gridColumns()).
		AddRow(1, int64(1), 0, "Game 1", "Home Team", nil, "Away Team", nil, now, false, "active", now, now, false, nil, nil)

	mock.ExpectQuery("SELECT .+ FROM grids WHERE id = \\$1 AND pool_id = \\$2").
		WithArgs(int64(1), int64(1)).
		WillReturnRows(gridRows)

	// LoadSettings
	settingsRows := sqlmock.NewRows(gridSettingsColumns()).
		AddRow(int64(1), "#000000", "#FFFFFF", "#FF0000", "#00FF00", "", "", "", now)

	mock.ExpectQuery("SELECT .+ FROM grid_settings WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(settingsRows)

	// LoadAnnotations
	annotationsRows := sqlmock.NewRows([]string{"grid_id", "square_id", "annotation", "icon"})
	mock.ExpectQuery("SELECT .+ FROM grid_annotations WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(annotationsRows)

	// Grid save (uses transaction)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE grid_settings SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE grids SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// NO pool save expected since pool is already locked

	body := `{"action": "drawNumbers", "data": {"lockPool": true}}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/grid/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Response should include poolLocks
	g.Expect(result).Should(gomega.HaveKey("poolLocks"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func sportsEventColumns() []string {
	return []string{
		"id", "espn_id", "league", "name", "home_team_id", "away_team_id", "event_date", "season", "week", "postseason", "venue",
		"status", "status_detail", "period", "clock", "home_score", "away_score",
		"home_q1", "home_q2", "home_q3", "home_q4", "home_ot",
		"away_q1", "away_q2", "away_q3", "away_q4", "away_ot",
		"created", "modified", "last_synced",
	}
}

func sportsTeamColumns() []string {
	return []string{
		"id", "league", "name", "full_name", "abbreviation", "conference", "division", "location", "color", "alternate_color", "created", "modified",
	}
}

func setupTestServerForPoolHandler(t *testing.T) (*Server, sqlmock.Sqlmock, *model.Model, *bool) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
		broker: NewPoolBroker(),
	}

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	s.Router.Path("/pool/{token}").Methods(http.MethodGet).Handler(s.poolHandler(nextHandler))

	return s, mock, m, &nextCalled
}

func TestPoolHandler_SiteAdminBypassesPasswordProtectedPool(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m, nextCalled := setupTestServerForPoolHandler(t)

	// Site admin user (different ID from pool owner)
	user := &model.User{
		Model:       m,
		ID:          999,
		Store:       model.UserStoreAuth0,
		IsSiteAdmin: true,
	}

	poolToken := "test-admin-bypass"
	now := time.Now()

	// Password-protected pool owned by user 100
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Protected Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	// No membership or join queries should be issued for site admin

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken, nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	g.Expect(*nextCalled).Should(gomega.BeTrue(), "next handler should have been called")
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPoolHandler_SiteAdminNotAutoJoined(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m, nextCalled := setupTestServerForPoolHandler(t)

	// Site admin user
	user := &model.User{
		Model:       m,
		ID:          999,
		Store:       model.UserStoreAuth0,
		IsSiteAdmin: true,
	}

	poolToken := "test-admin-no-join"
	now := time.Now()

	// Open pool (password_required = false), so non-admins would be auto-joined
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Open Pool", "std100", "standard", "hash", false, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	// No JoinPool query should be issued for site admin

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken, nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	g.Expect(*nextCalled).Should(gomega.BeTrue(), "next handler should have been called")
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPoolHandler_NonAdminNonMemberDenied(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m, nextCalled := setupTestServerForPoolHandler(t)

	// Regular user (not site admin, not pool owner)
	user := &model.User{
		Model:       m,
		ID:          200,
		Store:       model.UserStoreAuth0,
		IsSiteAdmin: false,
	}

	poolToken := "test-non-member-denied"
	now := time.Now()

	// Password-protected pool owned by user 100
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Protected Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	// IsMemberOf: user 200 != pool owner 100, so it queries pools_users
	mock.ExpectQuery("SELECT true FROM pools_users WHERE pool_id = \\$1 AND user_id = \\$2").
		WithArgs(int64(1), int64(200)).
		WillReturnRows(sqlmock.NewRows([]string{"bool"})) // empty = not a member

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken, nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusForbidden))
	g.Expect(*nextCalled).Should(gomega.BeFalse(), "next handler should NOT have been called")
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestSaveGrid_BlocksChangingFinalLinkedEvent(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForDrawNumbers(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-token-final-event"
	now := time.Now()
	bdlEventID := int64(12345)

	// Create pool
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Create a grid linked to a BDL event
	gridRows := sqlmock.NewRows(gridColumns()).
		AddRow(1, int64(1), 0, "Game 1", "Home Team", nil, "Away Team", nil, now, false, "active", now, now, false, bdlEventID, nil)

	mock.ExpectQuery("SELECT .+ FROM grids WHERE id = \\$1 AND pool_id = \\$2").
		WithArgs(int64(1), int64(1)).
		WillReturnRows(gridRows)

	// LoadSettings
	settingsRows := sqlmock.NewRows(gridSettingsColumns()).
		AddRow(int64(1), "#000000", "#FFFFFF", "#FF0000", "#00FF00", "", "", "", now)

	mock.ExpectQuery("SELECT .+ FROM grid_settings WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(settingsRows)

	// LoadAnnotations
	annotationsRows := sqlmock.NewRows([]string{"grid_id", "square_id", "annotation", "icon"})
	mock.ExpectQuery("SELECT .+ FROM grid_annotations WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(annotationsRows)

	// LoadSportsEvent - returns an event with status "final"
	eventRows := sqlmock.NewRows(sportsEventColumns()).
		AddRow(bdlEventID, "401547417", "nfl", "Chiefs vs 49ers", "1", "2", now, 2025, 10, false, "Stadium",
			"final", "Final", 4, "0:00", 28, 21,
			7, 7, 7, 7, nil,
			7, 7, 7, 0, nil,
			now, now, now)

	mock.ExpectQuery("SELECT .+ FROM sports_events WHERE id = \\$1").
		WithArgs(bdlEventID).
		WillReturnRows(eventRows)

	// LoadTeams for the event (home team)
	homeTeamRows := sqlmock.NewRows(sportsTeamColumns()).
		AddRow("1", "nfl", "Chiefs", "Kansas City Chiefs", "KC", "AFC", "West", "Kansas City", "E31837", "FFB612", now, now)
	mock.ExpectQuery("SELECT .+ FROM sports_teams WHERE id = \\$1 AND league = \\$2").
		WithArgs("1", model.SportsLeagueNFL).
		WillReturnRows(homeTeamRows)

	// LoadTeams for the event (away team)
	awayTeamRows := sqlmock.NewRows(sportsTeamColumns()).
		AddRow("2", "nfl", "Bills", "Buffalo Bills", "BUF", "AFC", "East", "Buffalo", "00338D", "C60C30", now, now)
	mock.ExpectQuery("SELECT .+ FROM sports_teams WHERE id = \\$1 AND league = \\$2").
		WithArgs("2", model.SportsLeagueNFL).
		WillReturnRows(awayTeamRows)

	// Try to unlink the event (set bdlEventId to null)
	body := `{"action": "save", "data": {"eventDate": "2025-01-15", "label": "Game 1", "homeTeamName": "Home", "awayTeamName": "Away", "bdlEventId": null}}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/grid/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	// Should be rejected with 400 Bad Request
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(result["error"]).Should(gomega.Equal("Cannot change linked event after the game has ended"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestSaveGrid_BlocksChangingToAnotherEventWhenFinal(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForDrawNumbers(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-token-final-event-2"
	now := time.Now()
	bdlEventID := int64(12345)

	// Create pool
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Create a grid linked to a BDL event
	gridRows := sqlmock.NewRows(gridColumns()).
		AddRow(1, int64(1), 0, "Game 1", "Home Team", nil, "Away Team", nil, now, false, "active", now, now, false, bdlEventID, nil)

	mock.ExpectQuery("SELECT .+ FROM grids WHERE id = \\$1 AND pool_id = \\$2").
		WithArgs(int64(1), int64(1)).
		WillReturnRows(gridRows)

	// LoadSettings
	settingsRows := sqlmock.NewRows(gridSettingsColumns()).
		AddRow(int64(1), "#000000", "#FFFFFF", "#FF0000", "#00FF00", "", "", "", now)

	mock.ExpectQuery("SELECT .+ FROM grid_settings WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(settingsRows)

	// LoadAnnotations
	annotationsRows := sqlmock.NewRows([]string{"grid_id", "square_id", "annotation", "icon"})
	mock.ExpectQuery("SELECT .+ FROM grid_annotations WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(annotationsRows)

	// LoadSportsEvent - returns an event with status "final"
	eventRows := sqlmock.NewRows(sportsEventColumns()).
		AddRow(bdlEventID, "401547417", "nfl", "Chiefs vs 49ers", "1", "2", now, 2025, 10, false, "Stadium",
			"final", "Final", 4, "0:00", 28, 21,
			7, 7, 7, 7, nil,
			7, 7, 7, 0, nil,
			now, now, now)

	mock.ExpectQuery("SELECT .+ FROM sports_events WHERE id = \\$1").
		WithArgs(bdlEventID).
		WillReturnRows(eventRows)

	// LoadTeams for the event (home team)
	homeTeamRows := sqlmock.NewRows(sportsTeamColumns()).
		AddRow("1", "nfl", "Chiefs", "Kansas City Chiefs", "KC", "AFC", "West", "Kansas City", "E31837", "FFB612", now, now)
	mock.ExpectQuery("SELECT .+ FROM sports_teams WHERE id = \\$1 AND league = \\$2").
		WithArgs("1", model.SportsLeagueNFL).
		WillReturnRows(homeTeamRows)

	// LoadTeams for the event (away team)
	awayTeamRows := sqlmock.NewRows(sportsTeamColumns()).
		AddRow("2", "nfl", "Bills", "Buffalo Bills", "BUF", "AFC", "East", "Buffalo", "00338D", "C60C30", now, now)
	mock.ExpectQuery("SELECT .+ FROM sports_teams WHERE id = \\$1 AND league = \\$2").
		WithArgs("2", model.SportsLeagueNFL).
		WillReturnRows(awayTeamRows)

	// Try to change to a different event
	body := `{"action": "save", "data": {"eventDate": "2025-01-15", "label": "Game 1", "homeTeamName": "Home", "awayTeamName": "Away", "bdlEventId": 99999}}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/grid/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	// Should be rejected with 400 Bad Request
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(result["error"]).Should(gomega.Equal("Cannot change linked event after the game has ended"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func setupTestServerForInviteToken(t *testing.T) (*Server, sqlmock.Sqlmock, *model.Model) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token}/invitetoken").Methods(http.MethodGet).Handler(s.poolManagerHandler(s.getPoolTokenInviteTokenEndpoint()))

	return s, mock, m
}

func TestGetPoolTokenInviteToken_CreatesNewInvite(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForInviteToken(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-invite-1"
	now := time.Now()

	// Load pool for context
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// ActiveInvite returns no rows (no active invite)
	mock.ExpectQuery("SELECT .+ FROM pool_invites WHERE pool_id = \\$1 AND check_id = \\$2").
		WithArgs(int64(1), 0).
		WillReturnRows(sqlmock.NewRows([]string{"token", "pool_id", "check_id", "expires_at", "created"}))

	// NewPoolInvite inserts
	inviteRows := sqlmock.NewRows([]string{"token", "pool_id", "check_id", "expires_at", "created"}).
		AddRow("s8Kj2mXqAb", int64(1), 0, now.Add(time.Hour*24*365), now)

	mock.ExpectQuery("INSERT INTO pool_invites").
		WithArgs(sqlmock.AnyArg(), int64(1), 0, sqlmock.AnyArg()).
		WillReturnRows(inviteRows)

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken+"/invitetoken", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(result).Should(gomega.HaveKey("token"))
	g.Expect(result["token"]).Should(gomega.Equal("s8Kj2mXqAb"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetPoolTokenInviteToken_ReusesActiveInvite(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForInviteToken(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-invite-2"
	now := time.Now()

	// Load pool for context
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// ActiveInvite returns an existing valid invite
	inviteRows := sqlmock.NewRows([]string{"token", "pool_id", "check_id", "expires_at", "created"}).
		AddRow("existingTkn", int64(1), 0, now.Add(time.Hour*24*365), now)

	mock.ExpectQuery("SELECT .+ FROM pool_invites WHERE pool_id = \\$1 AND check_id = \\$2").
		WithArgs(int64(1), 0).
		WillReturnRows(inviteRows)

	// Should NOT insert a new invite

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken+"/invitetoken", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(result["token"]).Should(gomega.Equal("existingTkn"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetPoolTokenInviteToken_NonAdminForbidden(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForInviteToken(t)

	user := &model.User{
		Model: m,
		ID:    200,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-invite-3"
	now := time.Now()

	// Load pool for context (owner is 100, user is 200)
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// IsManagerOf query - user 200 is not manager
	mock.ExpectQuery("SELECT true FROM pools_users WHERE pool_id = \\$1 AND user_id = \\$2 AND is_manager").
		WithArgs(int64(1), int64(200)).
		WillReturnRows(sqlmock.NewRows([]string{"bool"}))

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken+"/invitetoken", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusForbidden))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func setupTestServerForJoinPool(t *testing.T) (*Server, sqlmock.Sqlmock, *model.Model) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token}/member").Methods(http.MethodPost).Handler(s.postPoolTokenMemberEndpoint())

	return s, mock, m
}

func TestPostPoolTokenMember_JoinWithInviteToken(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForJoinPool(t)

	user := &model.User{
		Model: m,
		ID:    200,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-join-1"
	now := time.Now()

	// PoolByToken query
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	// PoolInviteByToken query
	inviteRows := sqlmock.NewRows([]string{"token", "pool_id", "check_id", "expires_at", "created"}).
		AddRow("validToken", int64(1), 0, now.Add(time.Hour*24), now)

	mock.ExpectQuery("SELECT .+ FROM pool_invites WHERE token = \\$1").
		WithArgs("validToken").
		WillReturnRows(inviteRows)

	// JoinPool: first checks IsManagerOf (user 200 != owner 100, so queries DB)
	mock.ExpectQuery("SELECT true FROM pools_users WHERE pool_id = \\$1 AND user_id = \\$2 AND is_manager").
		WithArgs(int64(1), int64(200)).
		WillReturnRows(sqlmock.NewRows([]string{"bool"})) // not admin

	// Then inserts into pools_users
	mock.ExpectExec("INSERT INTO pools_users").
		WithArgs(int64(1), int64(200)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"invite": "validToken"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/member", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusNoContent))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenMember_InvalidInviteToken(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForJoinPool(t)

	user := &model.User{
		Model: m,
		ID:    200,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-join-2"
	now := time.Now()

	// PoolByToken query
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	// PoolInviteByToken returns not found
	mock.ExpectQuery("SELECT .+ FROM pool_invites WHERE token = \\$1").
		WithArgs("badToken").
		WillReturnError(sql.ErrNoRows)

	body := `{"invite": "badToken"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/member", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenMember_ExpiredInviteToken(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForJoinPool(t)

	user := &model.User{
		Model: m,
		ID:    200,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-join-3"
	now := time.Now()

	// PoolByToken query
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	// PoolInviteByToken returns an expired invite
	inviteRows := sqlmock.NewRows([]string{"token", "pool_id", "check_id", "expires_at", "created"}).
		AddRow("expiredTkn", int64(1), 0, now.Add(-time.Hour), now.Add(-2*time.Hour))

	mock.ExpectQuery("SELECT .+ FROM pool_invites WHERE token = \\$1").
		WithArgs("expiredTkn").
		WillReturnRows(inviteRows)

	body := `{"invite": "expiredTkn"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/member", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenMember_InviteTokenWrongPool(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForJoinPool(t)

	user := &model.User{
		Model: m,
		ID:    200,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-join-4"
	now := time.Now()

	// PoolByToken query - pool ID is 1
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	// PoolInviteByToken returns an invite for a different pool (pool_id=999)
	inviteRows := sqlmock.NewRows([]string{"token", "pool_id", "check_id", "expires_at", "created"}).
		AddRow("wrongPool", int64(999), 0, now.Add(time.Hour*24), now)

	mock.ExpectQuery("SELECT .+ FROM pool_invites WHERE token = \\$1").
		WithArgs("wrongPool").
		WillReturnRows(inviteRows)

	body := `{"invite": "wrongPool"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/member", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func squareColumns() []string {
	return []string{
		"id", "square_id", "parent_id", "user_id", "state", "claimant", "modified",
		"parent_square_id", "child_square_ids",
	}
}

func setupTestServerForSquareUpdate(t *testing.T) (*Server, sqlmock.Sqlmock, *model.Model) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token}/square/{id}").Methods(http.MethodPost).Handler(s.postPoolTokenSquareIDEndpoint())

	return s, mock, m
}

func TestAdminUnclaim_AlsoUnclaimsSecondarySquare(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForSquareUpdate(t)

	// Admin user (same ID as pool owner)
	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-admin-unclaim-1"
	now := time.Now()

	// Load pool (roll100 type, owner is user 100)
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "roll100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// SquareBySquareID: primary square 1 (claimed, with child square 2)
	primarySquareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, int64(200), "claimed", "Player1", now, nil, "{2}")

	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(primarySquareRows)

	// Begin transaction for admin action
	mock.ExpectBegin()

	// Save primary square as unclaimed (update_pool_square)
	// Note: Go code passes claimant/userID as-is; the SQL function clears them when state=unclaimed
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStateUnclaimed, "Player1", int64(200), sqlmock.AnyArg(), "admin unclaim", true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))

	// ChildSquares query for the primary square
	childRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(11), 2, int64(10), int64(200), "claimed", "Player1", now, 1, nil)

	mock.ExpectQuery("SELECT .+ FROM\\s+pool_squares ps").
		WithArgs(int64(10)).
		WillReturnRows(childRows)

	// Save child square as unclaimed
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(11), model.PoolSquareStateUnclaimed, "Player1", int64(200), sqlmock.AnyArg(), sqlmock.AnyArg(), true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))

	// Commit
	mock.ExpectCommit()

	// LoadLogs for the primary square (admin response includes logs)
	mock.ExpectQuery("SELECT .+ pool_squares_logs").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "pool_square_id", "square_id", "user_id", "state", "claimant", "remote_addr", "note", "created",
		}))

	body := `{"state": "unclaimed", "note": "admin unclaim"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/square/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestAdminUnclaim_OnlyUnclaimsTargetedSquareChildren(t *testing.T) {
	// Given a claimant has two sets of squares:
	// Set 1: square 1 (primary) -> square 2 (secondary)
	// Set 2: square 3 (primary) -> square 4 (secondary)
	// When admin unclaims square 1, only squares 1 and 2 are unclaimed.
	// Squares 3 and 4 remain claimed.
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForSquareUpdate(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-admin-unclaim-2"
	now := time.Now()

	// Load pool (roll100 type, owner is user 100)
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "roll100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// SquareBySquareID: primary square 1 (claimed, child is square 2)
	primarySquareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, int64(200), "claimed", "Player1", now, nil, "{2}")

	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(primarySquareRows)

	// Begin transaction
	mock.ExpectBegin()

	// Save primary square 1 as unclaimed
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStateUnclaimed, "Player1", int64(200), sqlmock.AnyArg(), "admin unclaim set 1", true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))

	// ChildSquares: only returns square 2 (secondary of square 1)
	// Square 4 is secondary of square 3, NOT of square 1, so it won't appear
	childRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(11), 2, int64(10), int64(200), "claimed", "Player1", now, 1, nil)

	mock.ExpectQuery("SELECT .+ FROM\\s+pool_squares ps").
		WithArgs(int64(10)).
		WillReturnRows(childRows)

	// Save child square 2 as unclaimed
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(11), model.PoolSquareStateUnclaimed, "Player1", int64(200), sqlmock.AnyArg(), sqlmock.AnyArg(), true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))

	// Commit - no queries for square 3 or 4
	mock.ExpectCommit()

	// LoadLogs
	mock.ExpectQuery("SELECT .+ pool_squares_logs").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "pool_square_id", "square_id", "user_id", "state", "claimant", "remote_addr", "note", "created",
		}))

	body := `{"state": "unclaimed", "note": "admin unclaim set 1"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/square/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	// ExpectationsWereMet verifies that no unexpected queries happened (e.g., no update for squares 3 or 4)
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestAdminStateChange_NonUnclaimDoesNotAffectChildren(t *testing.T) {
	// When admin changes state to something other than unclaimed (e.g., paid-full),
	// child squares should NOT be affected.
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForSquareUpdate(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-admin-paid-1"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "roll100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// SquareBySquareID: primary square 1 (claimed, with child)
	primarySquareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, int64(200), "claimed", "Player1", now, nil, "{2}")

	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(primarySquareRows)

	// Begin transaction
	mock.ExpectBegin()

	// Save primary square as paid-full (NOT unclaimed)
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStatePaidFull, "Player1", int64(200), sqlmock.AnyArg(), "marked paid", true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))

	// NO ChildSquares query expected since state is not unclaimed
	// Commit
	mock.ExpectCommit()

	// LoadLogs
	mock.ExpectQuery("SELECT .+ pool_squares_logs").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "pool_square_id", "square_id", "user_id", "state", "claimant", "remote_addr", "note", "created",
		}))

	body := `{"state": "paid-full", "note": "marked paid"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/square/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestAdminRename_BlocksRenameOfSecondarySquare(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForSquareUpdate(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-admin-rename-secondary"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "roll100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// SquareBySquareID: secondary square (has parent_id set)
	secondarySquareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(11), 2, int64(10), int64(200), "claimed", "Player1", now, 1, nil)

	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 2).
		WillReturnRows(secondarySquareRows)

	body := `{"claimant": "NewName", "rename": true}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/square/2", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(result["error"]).Should(gomega.ContainSubstring("secondary square"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestAdminRename_AlsoRenamesSecondarySquare(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForSquareUpdate(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-admin-rename-primary"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "roll100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// SquareBySquareID: primary square 1 (claimed, with child square 2)
	primarySquareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, int64(200), "claimed", "OldName", now, nil, "{2}")

	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(primarySquareRows)

	// Begin transaction for rename
	mock.ExpectBegin()

	// Save primary square with new claimant
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStateClaimed, "NewName", int64(200), sqlmock.AnyArg(), "admin: changed claimant from OldName", true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))

	// ChildSquares query
	childRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(11), 2, int64(10), int64(200), "claimed", "OldName", now, 1, nil)

	mock.ExpectQuery("SELECT .+ FROM\\s+pool_squares ps").
		WithArgs(int64(10)).
		WillReturnRows(childRows)

	// Save child square with new claimant
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(11), model.PoolSquareStateClaimed, "NewName", int64(200), sqlmock.AnyArg(), sqlmock.AnyArg(), true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))

	// Commit
	mock.ExpectCommit()

	// LoadLogs
	mock.ExpectQuery("SELECT .+ pool_squares_logs").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "pool_square_id", "square_id", "user_id", "state", "claimant", "remote_addr", "note", "created",
		}))

	body := `{"claimant": "NewName", "rename": true}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/square/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestSaveGrid_AllowsKeepingSameEventWhenFinal(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForDrawNumbers(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-token-final-event-3"
	now := time.Now()
	bdlEventID := int64(12345)

	// Create pool
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Create a grid linked to a BDL event
	gridRows := sqlmock.NewRows(gridColumns()).
		AddRow(1, int64(1), 0, "Game 1", "Home Team", nil, "Away Team", nil, now, false, "active", now, now, false, bdlEventID, nil)

	mock.ExpectQuery("SELECT .+ FROM grids WHERE id = \\$1 AND pool_id = \\$2").
		WithArgs(int64(1), int64(1)).
		WillReturnRows(gridRows)

	// LoadSettings
	settingsRows := sqlmock.NewRows(gridSettingsColumns()).
		AddRow(int64(1), "#000000", "#FFFFFF", "#FF0000", "#00FF00", "", "", "", now)

	mock.ExpectQuery("SELECT .+ FROM grid_settings WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(settingsRows)

	// LoadAnnotations
	annotationsRows := sqlmock.NewRows([]string{"grid_id", "square_id", "annotation", "icon"})
	mock.ExpectQuery("SELECT .+ FROM grid_annotations WHERE grid_id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(annotationsRows)

	// LoadSportsEvent - returns an event with status "final"
	eventRows := sqlmock.NewRows(sportsEventColumns()).
		AddRow(bdlEventID, "401547417", "nfl", "Chiefs vs 49ers", "1", "2", now, 2025, 10, false, "Stadium",
			"final", "Final", 4, "0:00", 28, 21,
			7, 7, 7, 7, nil,
			7, 7, 7, 0, nil,
			now, now, now)

	mock.ExpectQuery("SELECT .+ FROM sports_events WHERE id = \\$1").
		WithArgs(bdlEventID).
		WillReturnRows(eventRows)

	// LoadTeams for the event (home team)
	homeTeamRows := sqlmock.NewRows(sportsTeamColumns()).
		AddRow("1", "nfl", "Chiefs", "Kansas City Chiefs", "KC", "AFC", "West", "Kansas City", "E31837", "FFB612", now, now)
	mock.ExpectQuery("SELECT .+ FROM sports_teams WHERE id = \\$1 AND league = \\$2").
		WithArgs("1", model.SportsLeagueNFL).
		WillReturnRows(homeTeamRows)

	// LoadTeams for the event (away team)
	awayTeamRows := sqlmock.NewRows(sportsTeamColumns()).
		AddRow("2", "nfl", "Bills", "Buffalo Bills", "BUF", "AFC", "East", "Buffalo", "00338D", "C60C30", now, now)
	mock.ExpectQuery("SELECT .+ FROM sports_teams WHERE id = \\$1 AND league = \\$2").
		WithArgs("2", model.SportsLeagueNFL).
		WillReturnRows(awayTeamRows)

	// Grid save (should be allowed since keeping same event)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE grid_settings SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE grids SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// Save with the same bdlEventId (should be allowed)
	body := `{"action": "save", "data": {"eventDate": "2025-01-15", "label": "Game 1", "homeTeamName": "Home", "awayTeamName": "Away", "bdlEventId": 12345}}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/grid/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	// Should succeed
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusAccepted))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func setupTestServerForBulkSquares(t *testing.T) (*Server, sqlmock.Sqlmock, *model.Model) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token}/squares/bulk").Methods(http.MethodPost).Handler(s.poolManagerHandler(s.postPoolTokenSquaresBulkEndpoint()))

	return s, mock, m
}

func TestPostPoolTokenSquaresBulk_NonAdminGetsForbidden(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForBulkSquares(t)

	// User with ID 200 (not the pool owner)
	user := &model.User{
		Model: m,
		ID:    200,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-bulk-nonadmin"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// IsManagerOf query — returns empty (not manager)
	mock.ExpectQuery("SELECT true FROM pools_users WHERE pool_id = \\$1 AND user_id = \\$2 AND is_manager").
		WithArgs(int64(1), int64(200)).
		WillReturnRows(sqlmock.NewRows([]string{"bool"}))

	body := `{"squareIds": [1], "action": "claim", "claimant": "Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/squares/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusForbidden))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenSquaresBulk_InvalidActionReturnsBadRequest(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForBulkSquares(t)

	// Admin user (same ID as pool owner)
	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-bulk-invalid-action"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	body := `{"squareIds": [1], "action": "invalid_action"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/squares/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenSquaresBulk_ClaimSucceeds(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForBulkSquares(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-bulk-claim"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// SquareBySquareID for square 1 — unclaimed
	squareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, nil, "unclaimed", nil, now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(squareRows)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStateClaimed, "Alice", int64(100), sqlmock.AnyArg(), "admin: bulk claim", true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))
	mock.ExpectCommit()

	body := `{"squareIds": [1], "action": "claim", "claimant": "Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/squares/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(result).Should(gomega.HaveKey("results"))

	results := result["results"].([]interface{})
	g.Expect(results).Should(gomega.HaveLen(1))
	first := results[0].(map[string]interface{})
	g.Expect(first["ok"]).Should(gomega.BeTrue())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenSquaresBulk_UnclaimSucceeds(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForBulkSquares(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-bulk-unclaim"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Square 1 — claimed
	squareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, int64(200), "claimed", "Bob", now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(squareRows)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStateUnclaimed, "Bob", int64(200), sqlmock.AnyArg(), "admin: bulk unclaim", true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))
	mock.ExpectCommit()

	body := `{"squareIds": [1], "action": "unclaim"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/squares/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	results := result["results"].([]interface{})
	g.Expect(results).Should(gomega.HaveLen(1))
	g.Expect(results[0].(map[string]interface{})["ok"]).Should(gomega.BeTrue())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenSquaresBulk_SetStateSucceeds(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForBulkSquares(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-bulk-setstate"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Square 1 — claimed
	squareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, int64(200), "claimed", "Carol", now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(squareRows)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStatePaidFull, "Carol", int64(200), sqlmock.AnyArg(), "admin: bulk set state to paid-full", true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))
	mock.ExpectCommit()

	body := `{"squareIds": [1], "action": "set_state", "state": "paid-full"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/squares/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	results := result["results"].([]interface{})
	g.Expect(results).Should(gomega.HaveLen(1))
	g.Expect(results[0].(map[string]interface{})["ok"]).Should(gomega.BeTrue())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenSquaresBulk_AlreadyClaimedReturnsPartialError(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForBulkSquares(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-bulk-partial"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Square 1 — unclaimed → will be claimed
	square1Rows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, nil, "unclaimed", nil, now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(square1Rows)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStateClaimed, "Alice", int64(100), sqlmock.AnyArg(), "admin: bulk claim", true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))
	mock.ExpectCommit()

	// Square 3 — already claimed → returns partial error (no DB ops beyond the select)
	square3Rows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(12), 3, nil, int64(200), "claimed", "Bob", now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 3).
		WillReturnRows(square3Rows)

	body := `{"squareIds": [1, 3], "action": "claim", "claimant": "Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/squares/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	results := result["results"].([]interface{})
	g.Expect(results).Should(gomega.HaveLen(2))

	first := results[0].(map[string]interface{})
	g.Expect(first["ok"]).Should(gomega.BeTrue())
	g.Expect(first["squareId"]).Should(gomega.BeEquivalentTo(1))

	second := results[1].(map[string]interface{})
	g.Expect(second["ok"]).Should(gomega.BeFalse())
	g.Expect(second["squareId"]).Should(gomega.BeEquivalentTo(3))
	g.Expect(second["error"]).Should(gomega.Equal("already claimed"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestAdminSetState_CannotChangeStateOfUnclaimedSquare(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForSquareUpdate(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-admin-setstate-unclaimed"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Square is unclaimed
	squareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, nil, "unclaimed", nil, now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(squareRows)

	body := `{"state": "paid-full"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/square/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenSquaresBulk_SetStateFailsForUnclaimedSquare(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForBulkSquares(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-bulk-setstate-unclaimed"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Square 1 — claimed → succeeds
	square1Rows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, int64(100), "claimed", "Alice", now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(square1Rows)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStatePaidFull, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))
	mock.ExpectCommit()

	// Square 2 — unclaimed → returns partial error (no DB ops beyond the select)
	square2Rows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(11), 2, nil, nil, "unclaimed", nil, now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 2).
		WillReturnRows(square2Rows)

	body := `{"squareIds": [1, 2], "action": "set_state", "state": "paid-full"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/squares/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	results := result["results"].([]interface{})
	g.Expect(results).Should(gomega.HaveLen(2))

	first := results[0].(map[string]interface{})
	g.Expect(first["ok"]).Should(gomega.BeTrue())
	g.Expect(first["squareId"]).Should(gomega.BeEquivalentTo(1))

	second := results[1].(map[string]interface{})
	g.Expect(second["ok"]).Should(gomega.BeFalse())
	g.Expect(second["squareId"]).Should(gomega.BeEquivalentTo(2))
	g.Expect(second["error"]).Should(gomega.Equal("square must be claimed first"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenSquaresBulk_SetStateToClaimedSucceeds(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForBulkSquares(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-bulk-setstate-claimed"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Square 1 — paid-full, being reverted to claimed
	squareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, int64(200), "paid-full", "Dave", now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(squareRows)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStateClaimed, "Dave", int64(200), sqlmock.AnyArg(), "admin: bulk set state to claimed", true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))
	mock.ExpectCommit()

	body := `{"squareIds": [1], "action": "set_state", "state": "claimed"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/squares/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	results := result["results"].([]interface{})
	g.Expect(results).Should(gomega.HaveLen(1))
	g.Expect(results[0].(map[string]interface{})["ok"]).Should(gomega.BeTrue())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenSquaresBulk_SetStateWithNoteUsesProvidedNote(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForBulkSquares(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-bulk-setstate-note"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Square 1 — claimed, setting to paid-full with a custom note
	squareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, int64(200), "claimed", "Eve", now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(squareRows)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStatePaidFull, "Eve", int64(200), sqlmock.AnyArg(), "cash received", true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))
	mock.ExpectCommit()

	body := `{"squareIds": [1], "action": "set_state", "state": "paid-full", "note": "cash received"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/squares/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	results := result["results"].([]interface{})
	g.Expect(results).Should(gomega.HaveLen(1))
	g.Expect(results[0].(map[string]interface{})["ok"]).Should(gomega.BeTrue())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenSquaresBulk_SecondarySquareReturnsError(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForBulkSquares(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-bulk-secondary"
	now := time.Now()

	// roll100 pool
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "roll100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Square 1 — secondary square (parent_id = 5)
	squareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, int64(5), nil, "unclaimed", nil, now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(squareRows)

	body := `{"squareIds": [1], "action": "claim", "claimant": "Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/squares/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	results := result["results"].([]interface{})
	g.Expect(results).Should(gomega.HaveLen(1))
	first := results[0].(map[string]interface{})
	g.Expect(first["ok"]).Should(gomega.BeFalse())
	g.Expect(first["error"]).Should(gomega.Equal("cannot directly edit a secondary square; edit the primary square instead"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostPoolTokenSquaresBulk_UnclaimRoll100AlsoUnclainsSecondary(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForBulkSquares(t)

	user := &model.User{
		Model: m,
		ID:    100,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-bulk-unclaim-roll100"
	now := time.Now()

	// roll100 pool
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "roll100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Primary square (square_id=1, db id=10) — claimed, no parent_id
	squareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(10), 1, nil, int64(200), "claimed", "Bob", now, nil, nil)
	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 1).
		WillReturnRows(squareRows)

	mock.ExpectBegin()

	// Save primary as unclaimed
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(10), model.PoolSquareStateUnclaimed, "Bob", int64(200), sqlmock.AnyArg(), "admin: bulk unclaim", true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))

	// ChildSquares query — secondary square (db id=11, square_id=2, parent_id=10)
	childRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(11), 2, int64(10), int64(200), "claimed", "Bob", now, 1, nil)
	mock.ExpectQuery("SELECT .+ FROM\\s+pool_squares ps").
		WithArgs(int64(10)).
		WillReturnRows(childRows)

	// Save secondary as unclaimed
	mock.ExpectQuery("SELECT \\* FROM update_pool_square").
		WithArgs(int64(11), model.PoolSquareStateUnclaimed, "Bob", int64(200), sqlmock.AnyArg(), sqlmock.AnyArg(), true).
		WillReturnRows(sqlmock.NewRows([]string{"ok"}).AddRow(true))

	mock.ExpectCommit()

	body := `{"squareIds": [1], "action": "unclaim"}`
	req := httptest.NewRequest(http.MethodPost, "/pool/"+poolToken+"/squares/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	results := result["results"].([]interface{})
	g.Expect(results).Should(gomega.HaveLen(1))
	g.Expect(results[0].(map[string]interface{})["ok"]).Should(gomega.BeTrue())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func setupTestServerForSquareDetail(t *testing.T) (*Server, sqlmock.Sqlmock, *model.Model) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router:      mux.NewRouter(),
		model:       m,
		broker:      NewPoolBroker(),
		auth0Client: auth0.NewClient(auth0.Config{}),
	}

	s.Router.Path("/pool/{token}/square/{id}").Methods(http.MethodGet).Handler(s.getPoolTokenSquareIDEndpoint())

	return s, mock, m
}

func TestGetSquareDetail_SiteAdminSeesLogsAndUserInfo(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForSquareDetail(t)

	// Site admin user (NOT the pool owner)
	user := &model.User{
		Model:       m,
		ID:          999,
		Store:       model.UserStoreAuth0,
		IsSiteAdmin: true,
	}

	poolToken := "test-site-admin-square"
	now := time.Now()

	// Load pool (owner is user 100, NOT user 999)
	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// SquareBySquareID: square 5 claimed by user 300
	squareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(50), 5, nil, int64(300), "claimed", "Player1", now, nil, nil)

	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 5).
		WillReturnRows(squareRows)

	// HasManagerVisibility short-circuits for site admins (user.IsSiteAdmin == true),
	// so no pools_users query is issued.

	// Site admin triggers LoadLogs
	mock.ExpectQuery("SELECT .+ pool_squares_logs").
		WithArgs(int64(50)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "pool_square_id", "square_id", "user_id", "state", "claimant", "remote_addr", "note", "created",
		}))

	// Site admin triggers GetUserByID for userInfo (square has userID 300)
	email := "player1@example.com"
	mock.ExpectQuery("SELECT .+ FROM users WHERE id = \\$1").
		WithArgs(int64(300)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "store", "store_id", "is_site_admin", "email", "created"}).
			AddRow(int64(300), model.UserStoreAuth0, "auth0|300", false, &email, now))

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken+"/square/5", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Logs were loaded (verified by mock expectations below) but omitted from JSON since empty.
	// Site admin should see userInfo
	g.Expect(result).Should(gomega.HaveKey("userInfo"))
	userInfo := result["userInfo"].(map[string]interface{})
	g.Expect(userInfo["userType"]).Should(gomega.Equal("registered"))
	g.Expect(userInfo["email"]).Should(gomega.Equal("player1@example.com"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetSquareDetail_NonAdminDoesNotSeeLogs(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForSquareDetail(t)

	// Regular user (NOT pool owner, NOT site admin)
	user := &model.User{
		Model: m,
		ID:    200,
		Store: model.UserStoreAuth0,
	}

	poolToken := "test-nonadmin-square"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// SquareBySquareID
	squareRows := sqlmock.NewRows(squareColumns()).
		AddRow(int64(50), 5, nil, int64(0), "unclaimed", "", now, nil, nil)

	mock.ExpectQuery("SELECT .+ FROM pool_squares ps").
		WithArgs(int64(1), 5).
		WillReturnRows(squareRows)

	// IsManagerOf check: not owner, not pool manager
	mock.ExpectQuery("SELECT true FROM pools_users WHERE pool_id = \\$1 AND user_id = \\$2 AND is_manager").
		WithArgs(int64(1), int64(200)).
		WillReturnRows(sqlmock.NewRows([]string{"bool"}))

	// No LoadLogs or GetUserByID expected

	req := httptest.NewRequest(http.MethodGet, "/pool/"+poolToken+"/square/5", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)
	ctx = context.WithValue(ctx, ctxPoolKey, poolForContext)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Non-admin should NOT see logs or userInfo
	_, hasLogs := result["logs"]
	g.Expect(hasLogs).Should(gomega.BeFalse(), "logs should not be present for non-admin")
	_, hasUserInfo := result["userInfo"]
	g.Expect(hasUserInfo).Should(gomega.BeFalse(), "userInfo should not be present for non-admin")

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetPoolTokenEndpoint_SiteAdminGetsManagerVisibilityTrue(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock, m := setupTestServerForPool(t)

	// Site admin user (NOT the pool owner)
	user := &model.User{
		Model:       m,
		ID:          999,
		Store:       model.UserStoreAuth0,
		IsSiteAdmin: true,
	}

	poolToken := "test-site-admin-pool"
	now := time.Now()

	poolRows := sqlmock.NewRows(poolColumns()).
		AddRow(1, poolToken, int64(100), "Test Pool", "std100", "standard", "hash", true, false, nil, now, now, 0, false)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs(poolToken).
		WillReturnRows(poolRows)

	poolForContext, err := s.model.PoolByToken(context.Background(), poolToken)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// IsManagerOf is called first: site admin (ID=999) is not pool owner (user_id=100),
	// so it queries pools_users and finds no rows.
	mock.ExpectQuery("SELECT true FROM pools_users WHERE pool_id = \\$1 AND user_id = \\$2 AND is_manager").
		WithArgs(int64(1), int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{"bool"})) // empty = not pool manager

	// Since user.IsSiteAdmin is true, hasManagerVisibility is true, so CanChangeNumberSetConfig is called
	gridsRows := sqlmock.NewRows(gridColumns())
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

	// Site admin should receive hasManagerVisibility = true (read-only visibility)
	g.Expect(result["hasManagerVisibility"]).Should(gomega.BeTrue())
	// Site admin should NOT be pool admin
	g.Expect(result["isPoolManager"]).Should(gomega.BeFalse())
	// Site admin should also receive canChangeNumberSetConfig
	g.Expect(result).Should(gomega.HaveKey("canChangeNumberSetConfig"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}
