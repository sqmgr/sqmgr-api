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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"github.com/synacor/argon2id"
)

// setupTestServerWithMock creates a test server with a mocked database
func setupTestServerWithMock(t *testing.T) (*Server, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	s := &Server{
		Router: mux.NewRouter(),
		model:  model.New(db),
		broker: NewPoolBroker(),
	}

	// Register the public squares endpoint
	s.Router.Path("/pool/{token:[A-Za-z0-9_-]+}/squares/public").Methods(http.MethodGet).Handler(s.getPoolTokenSquaresPublicEndpoint())

	return s, mock
}

// poolColumns matches the order from pool.go poolColumns constant
var testPoolColumns = []string{
	"id", "token", "user_id", "name", "grid_type", "number_set_config", "password_hash",
	"password_required", "open_access_on_lock", "locks", "created", "modified",
	"check_id", "archived",
}

// squaresColumns matches the columns returned by Pool.Squares() query
var testSquaresColumns = []string{
	"id", "square_id", "parent_id", "user_id", "state", "claimant",
	"modified", "parent_square_id", "child_square_ids",
}

func TestGetPoolTokenSquaresPublicEndpoint_PoolNotFound(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerWithMock(t)

	// Mock pool lookup - returns no rows
	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs("nonexistent-token").
		WillReturnRows(sqlmock.NewRows(testPoolColumns))

	req := httptest.NewRequest(http.MethodGet, "/pool/nonexistent-token/squares/public", nil)
	rec := httptest.NewRecorder()

	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusNotFound))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetPoolTokenSquaresPublicEndpoint_NoPasswordRequired(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerWithMock(t)

	now := time.Now()

	// Mock pool lookup - passwordRequired=false
	poolRows := sqlmock.NewRows(testPoolColumns).AddRow(
		1,            // id
		"test-token", // token
		100,          // user_id
		"Test Pool",  // name
		"std100",     // grid_type
		"single",     // number_set_config
		"hash",       // password_hash
		false,        // password_required
		false,        // open_access_on_lock
		nil,          // locks (nil = not locked)
		now,          // created
		now,          // modified
		1,            // check_id
		false,        // archived
	)
	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs("test-token").
		WillReturnRows(poolRows)

	// Mock squares lookup - return empty squares
	squaresRows := sqlmock.NewRows(testSquaresColumns)
	mock.ExpectQuery("SELECT .+ FROM pool_squares").
		WithArgs(1). // pool_id
		WillReturnRows(squaresRows)

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token/squares/public", nil)
	rec := httptest.NewRecorder()

	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[int]*model.PoolSquareJSON
	err := json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(result).Should(gomega.BeEmpty())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetPoolTokenSquaresPublicEndpoint_PasswordRequired_NoCredentials(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerWithMock(t)

	now := time.Now()

	// Mock pool lookup - passwordRequired=true, unlocked
	poolRows := sqlmock.NewRows(testPoolColumns).AddRow(
		1,            // id
		"test-token", // token
		100,          // user_id
		"Test Pool",  // name
		"std100",     // grid_type
		"single",     // number_set_config
		"hash",       // password_hash
		true,         // password_required
		false,        // open_access_on_lock
		nil,          // locks (nil = not locked)
		now,          // created
		now,          // modified
		1,            // check_id
		false,        // archived
	)
	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs("test-token").
		WillReturnRows(poolRows)

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token/squares/public", nil)
	rec := httptest.NewRecorder()

	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnauthorized))
	g.Expect(rec.Header().Get("WWW-Authenticate")).Should(gomega.Equal(`Basic realm="Pool Access"`))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetPoolTokenSquaresPublicEndpoint_PasswordRequired_WrongPassword(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerWithMock(t)

	now := time.Now()

	// Generate a valid password hash for "correct-password"
	correctPasswordHash, err := argon2id.DefaultHashPassword("correct-password")
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Mock pool lookup - passwordRequired=true, unlocked
	poolRows := sqlmock.NewRows(testPoolColumns).AddRow(
		1,                   // id
		"test-token",        // token
		100,                 // user_id
		"Test Pool",         // name
		"std100",            // grid_type
		"single",            // number_set_config
		correctPasswordHash, // password_hash
		true,                // password_required
		false,               // open_access_on_lock
		nil,                 // locks (nil = not locked)
		now,                 // created
		now,                 // modified
		1,                   // check_id
		false,               // archived
	)
	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs("test-token").
		WillReturnRows(poolRows)

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token/squares/public", nil)
	req.SetBasicAuth("user", "wrong-password")
	rec := httptest.NewRecorder()

	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnauthorized))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetPoolTokenSquaresPublicEndpoint_PasswordRequired_CorrectPassword(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerWithMock(t)

	now := time.Now()

	// Generate a valid password hash for "correct-password"
	correctPasswordHash, err := argon2id.DefaultHashPassword("correct-password")
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Mock pool lookup - passwordRequired=true, unlocked
	poolRows := sqlmock.NewRows(testPoolColumns).AddRow(
		1,                   // id
		"test-token",        // token
		100,                 // user_id
		"Test Pool",         // name
		"std100",            // grid_type
		"single",            // number_set_config
		correctPasswordHash, // password_hash
		true,                // password_required
		false,               // open_access_on_lock
		nil,                 // locks (nil = not locked)
		now,                 // created
		now,                 // modified
		1,                   // check_id
		false,               // archived
	)
	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs("test-token").
		WillReturnRows(poolRows)

	// Mock squares lookup
	squaresRows := sqlmock.NewRows(testSquaresColumns).AddRow(
		1,           // id
		1,           // square_id
		nil,         // parent_id
		nil,         // user_id
		"unclaimed", // state
		nil,         // claimant
		now,         // modified
		nil,         // parent_square_id
		nil,         // child_square_ids
	)
	mock.ExpectQuery("SELECT .+ FROM pool_squares").
		WithArgs(1). // pool_id
		WillReturnRows(squaresRows)

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token/squares/public", nil)
	req.SetBasicAuth("user", "correct-password")
	rec := httptest.NewRecorder()

	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var result map[string]*model.PoolSquareJSON
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(result).Should(gomega.HaveLen(1))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetPoolTokenSquaresPublicEndpoint_LockedWithOpenAccess(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerWithMock(t)

	now := time.Now()
	lockedTime := now.Add(-time.Hour) // locked in the past

	// Mock pool lookup - passwordRequired=true, locked=true, openAccessOnLock=true
	poolRows := sqlmock.NewRows(testPoolColumns).AddRow(
		1,            // id
		"test-token", // token
		100,          // user_id
		"Test Pool",  // name
		"std100",     // grid_type
		"single",     // number_set_config
		"hash",       // password_hash
		true,         // password_required
		true,         // open_access_on_lock
		lockedTime,   // locks (in the past = locked)
		now,          // created
		now,          // modified
		1,            // check_id
		false,        // archived
	)
	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs("test-token").
		WillReturnRows(poolRows)

	// Mock squares lookup - no auth required due to open access
	squaresRows := sqlmock.NewRows(testSquaresColumns)
	mock.ExpectQuery("SELECT .+ FROM pool_squares").
		WithArgs(1). // pool_id
		WillReturnRows(squaresRows)

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token/squares/public", nil)
	// Note: no authentication provided
	rec := httptest.NewRecorder()

	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetPoolTokenSquaresPublicEndpoint_LockedNoOpenAccess(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := setupTestServerWithMock(t)

	now := time.Now()
	lockedTime := now.Add(-time.Hour) // locked in the past

	// Mock pool lookup - passwordRequired=true, locked=true, openAccessOnLock=false
	poolRows := sqlmock.NewRows(testPoolColumns).AddRow(
		1,            // id
		"test-token", // token
		100,          // user_id
		"Test Pool",  // name
		"std100",     // grid_type
		"single",     // number_set_config
		"hash",       // password_hash
		true,         // password_required
		false,        // open_access_on_lock
		lockedTime,   // locks (in the past = locked)
		now,          // created
		now,          // modified
		1,            // check_id
		false,        // archived
	)
	mock.ExpectQuery("SELECT .+ FROM pools WHERE token = \\$1").
		WithArgs("test-token").
		WillReturnRows(poolRows)

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token/squares/public", nil)
	rec := httptest.NewRecorder()

	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnauthorized))
	g.Expect(rec.Header().Get("WWW-Authenticate")).Should(gomega.Equal(`Basic realm="Pool Access"`))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

// TestPoolIsLockedBehavior verifies Pool.IsLocked() behavior with time settings
func TestPoolIsLockedBehavior(t *testing.T) {
	g := gomega.NewWithT(t)

	pool := &model.Pool{}

	// Zero time (default) means unlocked
	g.Expect(pool.IsLocked()).Should(gomega.BeFalse())

	// Future time means unlocked
	pool.SetLocks(time.Now().Add(time.Hour))
	g.Expect(pool.IsLocked()).Should(gomega.BeFalse())

	// Past time means locked
	pool.SetLocks(time.Now().Add(-time.Hour))
	g.Expect(pool.IsLocked()).Should(gomega.BeTrue())
}
