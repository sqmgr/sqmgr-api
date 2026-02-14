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

func TestGetUserSelfEndpoint_ReturnsUserInfo(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/user/self").Methods(http.MethodGet).Handler(s.getUserSelfEndpoint())

	email := "test@example.com"
	created := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	user := &model.User{
		ID:      123,
		Store:   model.UserStoreAuth0,
		StoreID: "auth0|abc123",
		IsAdmin: false,
		Email:   &email,
		Created: created,
	}

	req := httptest.NewRequest(http.MethodGet, "/user/self", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(result["id"]).Should(gomega.BeEquivalentTo(123))
	g.Expect(result["store_id"]).Should(gomega.Equal("auth0|abc123"))
	g.Expect(result["store"]).Should(gomega.Equal("auth0"))
	g.Expect(result["is_admin"]).Should(gomega.BeFalse())
	g.Expect(result["email"]).Should(gomega.Equal("test@example.com"))
	g.Expect(result["created"]).Should(gomega.Equal("2024-01-15T10:30:00Z"))
}

func TestGetUserSelfEndpoint_NilEmail(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/user/self").Methods(http.MethodGet).Handler(s.getUserSelfEndpoint())

	created := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	user := &model.User{
		ID:      456,
		Store:   model.UserStoreSqMGR,
		StoreID: "sqmgr|guest123",
		IsAdmin: false,
		Email:   nil,
		Created: created,
	}

	req := httptest.NewRequest(http.MethodGet, "/user/self", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(result["id"]).Should(gomega.BeEquivalentTo(456))
	g.Expect(result["store"]).Should(gomega.Equal("sqmgr"))
	g.Expect(result["email"]).Should(gomega.BeNil())
}

func TestGetUserSelfEndpoint_AdminUser(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/user/self").Methods(http.MethodGet).Handler(s.getUserSelfEndpoint())

	email := "admin@example.com"
	created := time.Now()

	user := &model.User{
		ID:      1,
		Store:   model.UserStoreAuth0,
		StoreID: "auth0|admin",
		IsAdmin: true,
		Email:   &email,
		Created: created,
	}

	req := httptest.NewRequest(http.MethodGet, "/user/self", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(result["is_admin"]).Should(gomega.BeTrue())
}

func TestUserHandler_MissingUserContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	nextHandlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHandlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	s.Router.Path("/user/{id:[0-9]+}").Methods(http.MethodGet).Handler(s.userHandler(nextHandler))

	req := httptest.NewRequest(http.MethodGet, "/user/123", nil)
	rec := httptest.NewRecorder()

	// No user in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(nextHandlerCalled).Should(gomega.BeFalse())
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestGetUserSelfEndpoint_MissingUserContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/user/self").Methods(http.MethodGet).Handler(s.getUserSelfEndpoint())

	req := httptest.NewRequest(http.MethodGet, "/user/self", nil)
	rec := httptest.NewRecorder()

	// No user in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestGetUserSelfStatsEndpoint_MissingUserContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/user/self/stats").Methods(http.MethodGet).Handler(s.getUserSelfStatsEndpoint())

	req := httptest.NewRequest(http.MethodGet, "/user/self/stats", nil)
	rec := httptest.NewRecorder()

	// No user in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestGetUserIDPoolMembershipEndpoint_MissingUserIDContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/user/{id:[0-9]+}/{membership}").Methods(http.MethodGet).Handler(s.getUserIDPoolMembershipEndpoint())

	req := httptest.NewRequest(http.MethodGet, "/user/123/own", nil)
	rec := httptest.NewRecorder()

	// No userID in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestDeleteUserIDPoolTokenEndpoint_MissingUserIDContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/user/{id:[0-9]+}/pool/{token}").Methods(http.MethodDelete).Handler(s.deleteUserIDPoolTokenEndpoint())

	req := httptest.NewRequest(http.MethodDelete, "/user/123/pool/test-token", nil)
	rec := httptest.NewRecorder()

	// No userID in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func TestPostUserIDGuestJWT_MissingUserIDContext(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/user/{id:[0-9]+}/guestjwt").Methods(http.MethodPost).Handler(s.postUserIDGuestJWT())

	req := httptest.NewRequest(http.MethodPost, "/user/123/guestjwt", nil)
	rec := httptest.NewRecorder()

	// No userID in context
	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))
}

func setupTestServerForUserStats(t *testing.T) (*Server, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	s := &Server{
		Router: mux.NewRouter(),
		model:  model.New(db),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/user/self/stats").Methods(http.MethodGet).Handler(s.getUserSelfStatsEndpoint())

	return s, mock
}

func TestGetUserSelfStatsEndpoint_ReturnsStats(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerForUserStats(t)

	user := &model.User{
		ID:    123,
		Store: model.UserStoreAuth0,
	}

	// Mock PoolsOwnedByUserIDCount with includeArchived=true (total)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools WHERE user_id = \\$1").
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// Mock PoolsOwnedByUserIDCount with includeArchived=false (active only)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools WHERE user_id = \\$1 AND archived = 'f'").
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	// Mock PoolsJoinedByUserIDCount
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools LEFT JOIN pools_users").
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

	req := httptest.NewRequest(http.MethodGet, "/user/self/stats", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]int64
	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(result["poolsCreated"]).Should(gomega.Equal(int64(5)))
	g.Expect(result["poolsJoined"]).Should(gomega.Equal(int64(7)))
	g.Expect(result["activePools"]).Should(gomega.Equal(int64(3)))
	g.Expect(result["archivedPools"]).Should(gomega.Equal(int64(2))) // 5 - 3 = 2

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetUserSelfStatsEndpoint_ZeroStats(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerForUserStats(t)

	user := &model.User{
		ID:    456,
		Store: model.UserStoreAuth0,
	}

	// Mock all counts returning 0
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools WHERE user_id = \\$1").
		WithArgs(int64(456)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools WHERE user_id = \\$1 AND archived = 'f'").
		WithArgs(int64(456)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools LEFT JOIN pools_users").
		WithArgs(int64(456)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	req := httptest.NewRequest(http.MethodGet, "/user/self/stats", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]int64
	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(result["poolsCreated"]).Should(gomega.Equal(int64(0)))
	g.Expect(result["poolsJoined"]).Should(gomega.Equal(int64(0)))
	g.Expect(result["activePools"]).Should(gomega.Equal(int64(0)))
	g.Expect(result["archivedPools"]).Should(gomega.Equal(int64(0)))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetUserSelfStatsEndpoint_AllPoolsArchived(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerForUserStats(t)

	user := &model.User{
		ID:    789,
		Store: model.UserStoreAuth0,
	}

	// Mock: 3 total pools, 0 active = all 3 archived
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools WHERE user_id = \\$1").
		WithArgs(int64(789)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools WHERE user_id = \\$1 AND archived = 'f'").
		WithArgs(int64(789)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools LEFT JOIN pools_users").
		WithArgs(int64(789)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	req := httptest.NewRequest(http.MethodGet, "/user/self/stats", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]int64
	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(result["poolsCreated"]).Should(gomega.Equal(int64(3)))
	g.Expect(result["poolsJoined"]).Should(gomega.Equal(int64(2)))
	g.Expect(result["activePools"]).Should(gomega.Equal(int64(0)))
	g.Expect(result["archivedPools"]).Should(gomega.Equal(int64(3)))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}
