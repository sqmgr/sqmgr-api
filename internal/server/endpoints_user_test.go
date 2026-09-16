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
		ID:          123,
		Store:       model.UserStoreAuth0,
		StoreID:     "auth0|abc123",
		IsSiteAdmin: false,
		Email:       &email,
		Created:     created,
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
	g.Expect(result["is_site_admin"]).Should(gomega.BeFalse())
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
		ID:          456,
		Store:       model.UserStoreSqMGR,
		StoreID:     "sqmgr|guest123",
		IsSiteAdmin: false,
		Email:       nil,
		Created:     created,
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
		ID:          1,
		Store:       model.UserStoreAuth0,
		StoreID:     "auth0|admin",
		IsSiteAdmin: true,
		Email:       &email,
		Created:     created,
	}

	req := httptest.NewRequest(http.MethodGet, "/user/self", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), ctxUserKey, user)

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(result["is_site_admin"]).Should(gomega.BeTrue())
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

func setupTestServerForPoolMembership(t *testing.T) (*Server, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	s := &Server{
		Router: mux.NewRouter(),
		model:  model.New(db),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/user/{id:[0-9]+}/pool/{membership:(?:own|belong)}").Methods(http.MethodGet).Handler(s.getUserIDPoolMembershipEndpoint())

	return s, mock
}

func poolMembershipRows() *sqlmock.Rows {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	return sqlmock.NewRows([]string{
		"id", "token", "user_id", "name", "grid_type", "number_set_config", "password_hash",
		"password_required", "open_access_on_lock", "locks", "created", "modified", "check_id", "archived",
	}).AddRow(int64(1), "tok-1", int64(123), "Super Bowl Squares", "std100", "standard", "hash", true, false, nil, now, now, int64(0), false)
}

func TestGetUserIDPoolMembershipEndpoint_OwnWithSearch(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerForPoolMembership(t)

	// search is trimmed and passed as an ILIKE pattern after user_id, offset and limit
	mock.ExpectQuery("SELECT .+ FROM pools WHERE user_id = \\$1 AND archived = 'f' AND pools.name ILIKE \\$4 ORDER BY pools.id DESC OFFSET \\$2 LIMIT \\$3").
		WithArgs(int64(123), int64(0), 10, "%super%").
		WillReturnRows(poolMembershipRows())

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools WHERE user_id = \\$1 AND archived = 'f' AND pools.name ILIKE \\$2").
		WithArgs(int64(123), "%super%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	req := httptest.NewRequest(http.MethodGet, "/user/123/pool/own?search=%20super%20", nil)
	rec := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), ctxUserIDKey, int64(123))

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result struct {
		Pools []struct {
			Name string `json:"name"`
		} `json:"pools"`
		Total int64 `json:"total"`
	}
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &result)).Should(gomega.Succeed())
	g.Expect(result.Total).Should(gomega.Equal(int64(1)))
	g.Expect(result.Pools).Should(gomega.HaveLen(1))
	g.Expect(result.Pools[0].Name).Should(gomega.Equal("Super Bowl Squares"))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetUserIDPoolMembershipEndpoint_OwnIncludeArchivedWithSearch(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerForPoolMembership(t)

	mock.ExpectQuery("SELECT .+ FROM pools WHERE user_id = \\$1 AND pools.name ILIKE \\$4 ORDER BY pools.id DESC OFFSET \\$2 LIMIT \\$3").
		WithArgs(int64(123), int64(10), 10, "%bowl%").
		WillReturnRows(poolMembershipRows())

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools WHERE user_id = \\$1 AND pools.name ILIKE \\$2").
		WithArgs(int64(123), "%bowl%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(11))

	req := httptest.NewRequest(http.MethodGet, "/user/123/pool/own?search=bowl&includeArchived=true&offset=10", nil)
	rec := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), ctxUserIDKey, int64(123))

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetUserIDPoolMembershipEndpoint_OwnWithoutSearch(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerForPoolMembership(t)

	// a blank search must not add an ILIKE clause or an extra argument
	mock.ExpectQuery("SELECT .+ FROM pools WHERE user_id = \\$1 AND archived = 'f' ORDER BY pools.id DESC OFFSET \\$2 LIMIT \\$3").
		WithArgs(int64(123), int64(0), 10).
		WillReturnRows(poolMembershipRows())

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools WHERE user_id = \\$1 AND archived = 'f'$").
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	req := httptest.NewRequest(http.MethodGet, "/user/123/pool/own?search=%20%20", nil)
	rec := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), ctxUserIDKey, int64(123))

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetUserIDPoolMembershipEndpoint_BelongWithSearch(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerForPoolMembership(t)

	mock.ExpectQuery("SELECT .+ FROM pools LEFT JOIN pools_users ON pools.id = pools_users.pool_id WHERE pools_users.user_id = \\$1 AND pools.name ILIKE \\$4 ORDER BY pools.id DESC OFFSET \\$2 LIMIT \\$3").
		WithArgs(int64(123), int64(0), 10, "%squares%").
		WillReturnRows(poolMembershipRows())

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools LEFT JOIN pools_users ON pools.id = pools_users.pool_id WHERE pools_users.user_id = \\$1 AND pools.name ILIKE \\$2").
		WithArgs(int64(123), "%squares%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	req := httptest.NewRequest(http.MethodGet, "/user/123/pool/belong?search=squares", nil)
	rec := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), ctxUserIDKey, int64(123))

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result struct {
		Total int64 `json:"total"`
	}
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &result)).Should(gomega.Succeed())
	g.Expect(result.Total).Should(gomega.Equal(int64(1)))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetUserIDPoolMembershipEndpoint_BelongWithoutSearch(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerForPoolMembership(t)

	mock.ExpectQuery("SELECT .+ FROM pools LEFT JOIN pools_users ON pools.id = pools_users.pool_id WHERE pools_users.user_id = \\$1 ORDER BY pools.id DESC OFFSET \\$2 LIMIT \\$3").
		WithArgs(int64(123), int64(0), 10).
		WillReturnRows(poolMembershipRows())

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM pools LEFT JOIN pools_users ON pools.id = pools_users.pool_id WHERE pools_users.user_id = \\$1$").
		WithArgs(int64(123)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	req := httptest.NewRequest(http.MethodGet, "/user/123/pool/belong", nil)
	rec := httptest.NewRecorder()
	ctx := context.WithValue(req.Context(), ctxUserIDKey, int64(123))

	s.Router.ServeHTTP(rec, req.WithContext(ctx))

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}
