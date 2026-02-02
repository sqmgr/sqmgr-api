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
	"strings"
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

func setupTestServerForDrawNumbers(t *testing.T) (*Server, sqlmock.Sqlmock, *model.Model) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	m := model.New(db)
	s := &Server{
		Router: mux.NewRouter(),
		model:  m,
	}

	s.Router.Path("/pool/{token}/grid/{id}").Methods(http.MethodPost).Handler(s.postPoolTokenGridIDEndpoint())

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
		"status", "period", "clock", "home_score", "away_score",
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
			"final", 4, "0:00", 28, 21,
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
			"final", 4, "0:00", 28, 21,
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
			"final", 4, "0:00", 28, 21,
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
