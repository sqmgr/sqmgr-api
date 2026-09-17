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
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"github.com/sqmgr/sqmgr-api/pkg/sports"
	"github.com/sqmgr/sqmgr-api/pkg/sportsync"
)

// newAdminTestServer builds a server with a mocked database and every admin
// feature route registered behind a middleware that injects a site admin.
func newAdminTestServer(t *testing.T) (*Server, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	s := &Server{
		Router:   mux.NewRouter(),
		model:    model.New(db),
		broker:   NewPoolBroker(),
		syncRuns: &syncRunner{},
	}

	admin := &model.User{ID: 7, IsSiteAdmin: true}
	r := s.Router.NewRoute().Subrouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), ctxUserKey, admin)))
		})
	})
	r.Path("/admin/pool/{token:[A-Za-z0-9_-]+}/action").Methods(http.MethodPost).Handler(s.postAdminPoolActionEndpoint())
	r.Path("/admin/events/{id:[0-9]+}/refresh").Methods(http.MethodPost).Handler(s.postAdminEventRefreshEndpoint())
	r.Path("/admin/events/{id:[0-9]+}/override").Methods(http.MethodPost).Handler(s.postAdminEventOverrideEndpoint())
	r.Path("/admin/events/{id:[0-9]+}/override").Methods(http.MethodDelete).Handler(s.deleteAdminEventOverrideEndpoint())
	r.Path("/admin/sports/sync").Methods(http.MethodPost).Handler(s.postAdminSportsSyncEndpoint())
	r.Path("/admin/sports/sync-runs").Methods(http.MethodGet).Handler(s.getAdminSportsSyncRunsEndpoint())
	r.Path("/admin/audit").Methods(http.MethodGet).Handler(s.getAdminAuditEndpoint())
	r.Path("/admin/analytics/timeseries").Methods(http.MethodGet).Handler(s.getAdminTimeSeriesEndpoint())
	r.Path("/admin/analytics/breakdown").Methods(http.MethodGet).Handler(s.getAdminBreakdownEndpoint())
	r.Path("/admin/analytics/fill-rates").Methods(http.MethodGet).Handler(s.getAdminFillRatesEndpoint())
	r.Path("/admin/pools").Methods(http.MethodGet).Handler(s.getAdminPoolsEndpoint())

	return s, mock
}

func jsonRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestParseAdminPoolsFilter(t *testing.T) {
	g := gomega.NewWithT(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/pools?search=abc&ownerEmail=me@x.y&gridType=std25&archived=true&start=2026-01-01&end=2026-01-31&minFill=10&maxFill=90&sortBy=fill_percent&sortDir=asc&offset=50&limit=10", nil)
	filter, err := parseAdminPoolsFilter(req)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(filter.Search).Should(gomega.Equal("abc"))
	g.Expect(filter.OwnerEmail).Should(gomega.Equal("me@x.y"))
	g.Expect(filter.GridType).Should(gomega.Equal(model.GridTypeStd25))
	g.Expect(*filter.Archived).Should(gomega.BeTrue())
	g.Expect(filter.Created.Start).Should(gomega.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	// The end day is inclusive
	g.Expect(filter.Created.End).Should(gomega.Equal(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)))
	g.Expect(*filter.MinFillPercent).Should(gomega.Equal(10.0))
	g.Expect(*filter.MaxFillPercent).Should(gomega.Equal(90.0))
	g.Expect(filter.SortBy).Should(gomega.Equal("fill_percent"))
	g.Expect(filter.SortDir).Should(gomega.Equal("asc"))
	g.Expect(filter.Offset).Should(gomega.Equal(int64(50)))
	g.Expect(filter.Limit).Should(gomega.Equal(10))

	// Defaults and clamping
	filter, err = parseAdminPoolsFilter(httptest.NewRequest(http.MethodGet, "/admin/pools?limit=5000", nil))
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(filter.Limit).Should(gomega.Equal(maxAdminPoolsLimit))
	g.Expect(filter.Archived).Should(gomega.BeNil())
	g.Expect(filter.Created.IsZero()).Should(gomega.BeTrue())

	filter, err = parseAdminPoolsFilter(httptest.NewRequest(http.MethodGet, "/admin/pools", nil))
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(filter.Limit).Should(gomega.Equal(defaultAdminPoolsLimit))

	for _, query := range []string{
		"gridType=std1000",
		"archived=maybe",
		"start=yesterday",
		"minFill=101",
		"maxFill=abc",
		"minFill=60&maxFill=40",
		"start=2026-02-01&end=2026-01-01",
	} {
		_, err := parseAdminPoolsFilter(httptest.NewRequest(http.MethodGet, "/admin/pools?"+query, nil))
		g.Expect(err).Should(gomega.HaveOccurred(), query)
	}
}

func TestAdminPoolsEndpoint_BadRequest(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/pools?archived=nope", nil))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestEventOverridePayloadToOverride(t *testing.T) {
	g := gomega.NewWithT(t)
	n := func(i int) *int { return &i }
	parse := func(body string) eventOverridePayload {
		var p eventOverridePayload
		g.Expect(json.Unmarshal([]byte(body), &p)).Should(gomega.Succeed(), body)
		return p
	}

	current := &model.SportsEvent{
		HomeScore: n(14), AwayScore: n(7),
		HomeQ1: n(7), HomeQ2: n(7), HomeOT: n(3),
		AwayQ1: n(0), AwayQ2: n(7), AwayOT: n(0),
	}

	p := parse(`{"status":"final","homeScore":24,"awayScore":17,` +
		`"homeQuarters":[7,10,null,7],"awayQuarters":[3,7,7,0],"homeOT":0,"awayOT":null}`)
	g.Expect(p.validate()).Should(gomega.Succeed())
	o := p.toOverride(current)
	g.Expect(o.Status).Should(gomega.Equal(model.SportsEventStatusFinal))
	g.Expect(o.HomeScore).Should(gomega.Equal(n(24)))
	g.Expect(o.HomeQ2).Should(gomega.Equal(n(10)))
	// a null quarter clears it
	g.Expect(o.HomeQ3).Should(gomega.BeNil())
	g.Expect(o.AwayQ4).Should(gomega.Equal(n(0)))
	g.Expect(o.HomeOT).Should(gomega.Equal(n(0)))
	// an explicit null clears a score the event currently has
	g.Expect(o.AwayOT).Should(gomega.BeNil())

	// Anything omitted keeps the event's current value
	p = parse(`{"status":"in_progress"}`)
	g.Expect(p.validate()).Should(gomega.Succeed())
	o = p.toOverride(current)
	g.Expect(o.HomeScore).Should(gomega.Equal(n(14)))
	g.Expect(o.AwayScore).Should(gomega.Equal(n(7)))
	g.Expect([]*int{o.HomeQ1, o.HomeQ2, o.HomeQ3, o.HomeQ4, o.HomeOT}).Should(gomega.Equal([]*int{n(7), n(7), nil, nil, n(3)}))
	g.Expect([]*int{o.AwayQ1, o.AwayQ2, o.AwayQ3, o.AwayQ4, o.AwayOT}).Should(gomega.Equal([]*int{n(0), n(7), nil, nil, n(0)}))

	// Each team's quarters are kept or replaced independently
	o = parse(`{"status":"final","awayQuarters":[null,null,null,null]}`).toOverride(current)
	g.Expect([]*int{o.HomeQ1, o.HomeQ2}).Should(gomega.Equal([]*int{n(7), n(7)}))
	g.Expect([]*int{o.AwayQ1, o.AwayQ2, o.AwayQ3, o.AwayQ4}).Should(gomega.Equal([]*int{nil, nil, nil, nil}))

	for name, body := range map[string]string{
		"bad status":       `{"status":"postponed"}`,
		"negative score":   `{"status":"final","homeScore":-1}`,
		"negative ot":      `{"status":"final","awayOT":-3}`,
		"short quarters":   `{"status":"final","homeQuarters":[1,2]}`,
		"negative quarter": `{"status":"final","awayQuarters":[1,-2,3,4]}`,
	} {
		g.Expect(parse(body).validate()).ShouldNot(gomega.Succeed(), name)
	}

	// A score that is not an integer fails to decode
	var bad eventOverridePayload
	g.Expect(json.Unmarshal([]byte(`{"status":"final","homeOT":"three"}`), &bad)).ShouldNot(gomega.Succeed())
}

func TestEventScoreDetails(t *testing.T) {
	g := gomega.NewWithT(t)
	n := func(i int) *int { return &i }

	data, err := json.Marshal(eventScoreDetails(&model.SportsEvent{
		Status: model.SportsEventStatusFinal, HomeScore: n(24), AwayScore: n(17),
		HomeQ1: n(7), AwayQ4: n(0), HomeOT: n(3),
	}))
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(string(data)).Should(gomega.MatchJSON(`{
		"status": "final", "homeScore": 24, "awayScore": 17,
		"homeQuarters": [7, null, null, null], "awayQuarters": [null, null, null, 0],
		"homeOT": 3, "awayOT": null
	}`))
}

func TestSyncRunner(t *testing.T) {
	g := gomega.NewWithT(t)
	r := &syncRunner{}

	g.Expect(r.status()).Should(gomega.BeNil())
	g.Expect(r.start(SyncRunStatus{SyncType: model.SportsSyncTypeScores})).Should(gomega.BeTrue())
	g.Expect(r.start(SyncRunStatus{SyncType: model.SportsSyncTypeTeams})).Should(gomega.BeFalse())

	status := r.status()
	g.Expect(status).ShouldNot(gomega.BeNil())
	g.Expect(status.SyncType).Should(gomega.Equal(model.SportsSyncTypeScores))

	// The returned status is a copy
	status.SyncType = model.SportsSyncTypeTeams
	g.Expect(r.status().SyncType).Should(gomega.Equal(model.SportsSyncTypeScores))

	r.finish()
	g.Expect(r.status()).Should(gomega.BeNil())
	g.Expect(r.start(SyncRunStatus{SyncType: model.SportsSyncTypeTeams})).Should(gomega.BeTrue())
}

func TestPostAdminSportsSyncEndpoint_Validation(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	for name, body := range map[string]string{
		"bad type":   `{"syncType":"everything"}`,
		"bad league": `{"syncType":"scores","league":"mls"}`,
	} {
		rec := httptest.NewRecorder()
		s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/sports/sync", body))
		g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest), name)
	}

	// Wrong content type
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/sports/sync", strings.NewReader(`{}`)))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnsupportedMediaType))

	// No syncer configured
	rec = httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/sports/sync", `{"syncType":"scores"}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusServiceUnavailable))
	g.Expect(s.syncRuns.status()).Should(gomega.BeNil())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetAdminSportsSyncRunsEndpoint(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/sports/sync-runs?syncType=bogus", nil))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectExec(`SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`FROM sports_sync_log WHERE sync_type = \$1 ORDER BY id DESC LIMIT \$2`).
		WithArgs("scores", maxSyncRunsLimit).
		WillReturnRows(sqlmock.NewRows([]string{"id", "sync_type", "league", "started_at", "completed_at", "records_processed", "error_message", "success"}).
			AddRow(int64(1), "scores", nil, now, now, 4, nil, true))
	mock.ExpectRollback()

	rec = httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/sports/sync-runs?syncType=scores&limit=999", nil))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var body struct {
		Runs []model.SportsSyncRun `json:"runs"`
	}
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &body)).Should(gomega.Succeed())
	g.Expect(body.Runs).Should(gomega.HaveLen(1))
	g.Expect(body.Runs[0].SyncType).Should(gomega.Equal("scores"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetAdminAuditEndpoint(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	for _, query := range []string{"action=user.delete", "targetType=user"} {
		rec := httptest.NewRecorder()
		s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit?"+query, nil))
		g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest), query)
	}

	now := time.Now()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM admin_audit_log a WHERE a.action = \$1 AND a.target_type = \$2 AND a.target_id = \$3`).
		WithArgs("pool.lock", "pool", "tok").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`FROM admin_audit_log a LEFT JOIN users u`).
		WithArgs("pool.lock", "pool", "tok", int64(25), 25).
		WillReturnRows(sqlmock.NewRows([]string{"id", "admin_user_id", "email", "action", "target_type", "target_id", "target_label", "details", "reason", "created"}).
			AddRow(int64(1), int64(7), "a@b.c", "pool.lock", "pool", "tok", "Pool", []byte(`{"previouslyLocked":false}`), nil, now))

	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/audit?action=pool.lock&targetType=pool&targetId=tok&offset=25", nil))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var body struct {
		Entries []model.AdminAuditEntry `json:"entries"`
		Total   int64                   `json:"total"`
	}
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &body)).Should(gomega.Succeed())
	g.Expect(body.Total).Should(gomega.Equal(int64(1)))
	g.Expect(body.Entries).Should(gomega.HaveLen(1))
	g.Expect(body.Entries[0].Details).Should(gomega.HaveKeyWithValue("previouslyLocked", false))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestAdminAnalyticsEndpoints_BadRequests(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	for _, target := range []string{
		"/admin/analytics/timeseries",
		"/admin/analytics/timeseries?metric=pools_created&interval=hour",
		"/admin/analytics/timeseries?metric=pools_created&start=nope",
		"/admin/analytics/breakdown?dimension=color",
		"/admin/analytics/fill-rates?gridType=std7",
		"/admin/analytics/fill-rates?archived=sometimes",
		"/admin/analytics/fill-rates?start=2026-02-01&end=2026-01-01",
	} {
		rec := httptest.NewRecorder()
		s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest), target)
	}

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestGetAdminTimeSeriesEndpoint(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	mock.ExpectBegin()
	mock.ExpectExec(`SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"period", "count"}).
		AddRow(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), int64(3)))
	mock.ExpectRollback()

	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/analytics/timeseries?metric=pools_created&interval=day&start=2026-09-01&end=2026-09-02", nil))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var body struct {
		Metric   string `json:"metric"`
		Interval string `json:"interval"`
		Points   []struct {
			Period string `json:"period"`
			Count  int64  `json:"count"`
		} `json:"points"`
	}
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &body)).Should(gomega.Succeed())
	g.Expect(body.Metric).Should(gomega.Equal("pools_created"))
	g.Expect(body.Interval).Should(gomega.Equal("day"))
	g.Expect(body.Points).Should(gomega.HaveLen(2))
	g.Expect(body.Points[0].Count).Should(gomega.Equal(int64(3)))
	g.Expect(body.Points[1].Count).Should(gomega.BeZero())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func adminTestPoolRow(token string) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows(testPoolColumns).AddRow(
		int64(10), token, int64(1), "Test Pool", "std100", "standard", "hash",
		true, false, nil, now, now, 0, false,
	)
}

func TestPostAdminPoolActionEndpoint_Validation(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	// Unknown pool
	mock.ExpectQuery(`FROM pools WHERE token = \$1`).WithArgs("missing").WillReturnRows(sqlmock.NewRows(testPoolColumns))
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/pool/missing/action", `{"action":"archive"}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusNotFound))

	// Unknown action
	mock.ExpectQuery(`FROM pools WHERE token = \$1`).WithArgs("tok").WillReturnRows(adminTestPoolRow("tok"))
	rec = httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/pool/tok/action", `{"action":"delete"}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))

	// Short password
	mock.ExpectQuery(`FROM pools WHERE token = \$1`).WithArgs("tok").WillReturnRows(adminTestPoolRow("tok"))
	rec = httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/pool/tok/action", `{"action":"resetPassword","password":"abc"}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))
	g.Expect(rec.Body.String()).Should(gomega.ContainSubstring("validationErrors"))

	// Transfer without a user
	mock.ExpectQuery(`FROM pools WHERE token = \$1`).WithArgs("tok").WillReturnRows(adminTestPoolRow("tok"))
	rec = httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/pool/tok/action", `{"action":"transferOwnership"}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))

	// Transfer to the current owner
	mock.ExpectQuery(`FROM pools WHERE token = \$1`).WithArgs("tok").WillReturnRows(adminTestPoolRow("tok"))
	mock.ExpectQuery(`FROM users WHERE id = \$1`).WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "store", "store_id", "is_site_admin", "email", "created"}).AddRow(int64(1), "auth0", "auth0|1", false, nil, time.Now()))
	rec = httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/pool/tok/action", `{"action":"transferOwnership","userId":1}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))

	// Transfer to an unknown user
	mock.ExpectQuery(`FROM pools WHERE token = \$1`).WithArgs("tok").WillReturnRows(adminTestPoolRow("tok"))
	mock.ExpectQuery(`FROM users WHERE id = \$1`).WithArgs(int64(99)).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	rec = httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/pool/tok/action", `{"action":"transferOwnership","userId":99}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusNotFound))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostAdminPoolActionEndpoint_ArchiveRecordsAudit(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	mock.ExpectQuery(`FROM pools WHERE token = \$1`).WithArgs("tok").WillReturnRows(adminTestPoolRow("tok"))
	mock.ExpectExec(`UPDATE pools SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO admin_audit_log`).
		WithArgs(int64(7), "pool.archive", "pool", "tok", "Test Pool", `{"previous":false}`, "spam").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/pool/tok/action", `{"action":"archive","reason":"spam"}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusNoContent))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostAdminPoolActionEndpoint_RevokeInvites(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	mock.ExpectQuery(`FROM pools WHERE token = \$1`).WithArgs("tok").WillReturnRows(adminTestPoolRow("tok"))
	mock.ExpectExec(`UPDATE pool_invites SET expires_at`).WithArgs(int64(10)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery(`INSERT INTO admin_audit_log`).
		WithArgs(int64(7), "pool.revokeInvites", "pool", "tok", "Test Pool", `{"revoked":2}`, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/pool/tok/action", `{"action":"revokeInvites"}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusNoContent))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func adminTestEventRows(id int64, override bool) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows(sportsEventColumns()).AddRow(
		id, "401", "nfl", "Big Game", "home", "away", now, 2026, nil, false, nil,
		"in_progress", nil, 2, "5:00", 14, 7,
		7, 7, nil, nil, nil,
		0, 7, nil, nil, nil,
		now, now, now, override,
	)
}

func expectAdminEventLoad(mock sqlmock.Sqlmock, id int64, override bool) {
	now := time.Now()
	mock.ExpectQuery(`FROM sports_events WHERE id = \$1`).WithArgs(id).WillReturnRows(adminTestEventRows(id, override))
	mock.ExpectQuery(`FROM sports_teams WHERE id = \$1 AND league = \$2`).WithArgs("home", model.SportsLeagueNFL).
		WillReturnRows(sqlmock.NewRows(sportsTeamColumns()).AddRow("home", "nfl", "Home", "Home Team", "HOM", nil, nil, nil, nil, nil, now, now))
	mock.ExpectQuery(`FROM sports_teams WHERE id = \$1 AND league = \$2`).WithArgs("away", model.SportsLeagueNFL).
		WillReturnRows(sqlmock.NewRows(sportsTeamColumns()).AddRow("away", "nfl", "Away", "Away Team", "AWY", nil, nil, nil, nil, nil, now, now))
}

func TestPostAdminEventRefreshEndpoint_Override(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	// Unknown event
	mock.ExpectQuery(`FROM sports_events WHERE id = \$1`).WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows(sportsEventColumns()))
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/events/5/refresh", nil))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusNotFound))

	// Overridden event is not refreshed
	expectAdminEventLoad(mock, 6, true)
	rec = httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/events/6/refresh", nil))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusConflict))

	// No syncer configured
	expectAdminEventLoad(mock, 6, false)
	rec = httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/events/6/refresh", nil))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusServiceUnavailable))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPostAdminEventOverrideEndpoint(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	// Invalid payload is rejected before touching the database
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/events/6/override", `{"status":"cancelled"}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusBadRequest))

	// The payload omits the quarter and overtime scores, so the event's
	// existing ones are written back rather than being cleared.
	expectAdminEventLoad(mock, 6, false)
	mock.ExpectExec(`UPDATE sports_events SET status = \$1`).
		WithArgs("final", 24, 17, 7, 7, nil, nil, nil, 0, 7, nil, nil, nil, int64(6), true, false).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`SELECT pg_notify`).WithArgs("6").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`INSERT INTO admin_audit_log`).
		WithArgs(int64(7), "event.override", "event", "6", "Big Game", sqlmock.AnyArg(), "feed stuck").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	rec = httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/events/6/override", `{"status":"final","homeScore":24,"awayScore":17,"reason":"feed stuck"}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	var body model.SportsEventJSON
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &body)).Should(gomega.Succeed())
	g.Expect(body.ManualOverride).Should(gomega.BeTrue())
	g.Expect(body.Status).Should(gomega.Equal(model.SportsEventStatusFinal))
	g.Expect(*body.HomeScore).Should(gomega.Equal(24))
	g.Expect(*body.HomeQ2).Should(gomega.Equal(7))
	g.Expect(body.HomeTeam.FullName).Should(gomega.Equal("Home Team"))

	// Explicit nulls clear scores, and only the team whose quarters are sent is changed
	expectAdminEventLoad(mock, 6, false)
	mock.ExpectExec(`UPDATE sports_events SET status = \$1`).
		WithArgs("final", 24, 17, nil, nil, nil, nil, nil, 0, 7, nil, nil, nil, int64(6), true, false).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`SELECT pg_notify`).WithArgs("6").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`INSERT INTO admin_audit_log`).
		WithArgs(int64(7), "event.override", "event", "6", "Big Game", sqlmock.AnyArg(), nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	rec = httptest.NewRecorder()
	s.Router.ServeHTTP(rec, jsonRequest(http.MethodPost, "/admin/events/6/override",
		`{"status":"final","homeScore":24,"awayScore":17,"homeQuarters":[null,null,null,null],"homeOT":null}`))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestDeleteAdminEventOverrideEndpoint(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	expectAdminEventLoad(mock, 6, true)
	mock.ExpectExec(`UPDATE sports_events SET manual_override = false`).WithArgs(int64(6)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO admin_audit_log`).
		WithArgs(int64(7), "event.clearOverride", "event", "6", "Big Game", nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/admin/events/6/override", nil))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusNoContent))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestEventLabel(t *testing.T) {
	g := gomega.NewWithT(t)

	name := "Super Bowl"
	g.Expect(eventLabel(&model.SportsEvent{Name: &name})).Should(gomega.Equal("Super Bowl"))
	g.Expect(eventLabel(&model.SportsEvent{HomeTeamID: "h", AwayTeamID: "a"})).Should(gomega.Equal("a at h"))

	event := &model.SportsEvent{HomeTeamID: "h", AwayTeamID: "a"}
	event.SetHomeTeam(&model.SportsTeam{FullName: "Home Team"})
	event.SetAwayTeam(&model.SportsTeam{FullName: "Away Team"})
	g.Expect(eventLabel(event)).Should(gomega.Equal("Away Team at Home Team"))
}

func TestRecordAudit_SurvivesCancelledRequestContext(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	// The client disconnected after the action committed
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mock.ExpectQuery(`INSERT INTO admin_audit_log`).
		WithArgs(int64(7), "pool.lock", "pool", "tok", nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	s.recordAudit(ctx, &model.User{ID: 7}, model.AdminAuditRecord{
		Action:     model.AdminAuditPoolLock,
		TargetType: model.AdminAuditTargetPool,
		TargetID:   "tok",
	})

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

// panickingClient blows up on the first ESPN call, standing in for an
// unexpected response shape.
type panickingClient struct{}

func (panickingClient) GetTeams(context.Context, sports.League) ([]sports.Team, error) {
	panic("unexpected ESPN payload")
}
func (panickingClient) GetSeasonInfo(context.Context, sports.League) (*sports.SeasonInfo, error) {
	panic("unexpected ESPN payload")
}
func (panickingClient) GetTeamSchedule(context.Context, sports.League, string, sports.SeasonType) ([]sports.Event, error) {
	panic("unexpected ESPN payload")
}
func (panickingClient) GetScoreboard(context.Context, sports.League, sports.ScoreboardOptions) ([]sports.Event, error) {
	panic("unexpected ESPN payload")
}
func (panickingClient) GetScoreboardForDateRange(context.Context, sports.League, time.Time, time.Time) ([]sports.Event, error) {
	panic("unexpected ESPN payload")
}
func (panickingClient) GetEventSummary(context.Context, sports.League, string) (*sports.Event, error) {
	panic("unexpected ESPN payload")
}

func TestRunManualSync_RecoversFromPanic(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newAdminTestServer(t)

	quiet := logrus.New()
	quiet.SetLevel(logrus.PanicLevel)
	s.syncer = sportsync.New(s.model, panickingClient{}, sportsync.Options{Logger: logrus.NewEntry(quiet)})

	// The teams sync opens a log row before calling ESPN
	mock.ExpectQuery(`INSERT INTO sports_sync_log`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "started_at"}).AddRow(int64(1), time.Now()))

	g.Expect(s.syncRuns.start(SyncRunStatus{SyncType: model.SportsSyncTypeTeams})).Should(gomega.BeTrue())

	g.Expect(func() {
		s.runManualSync(model.SportsSyncTypeTeams, []model.SportsLeague{model.SportsLeagueNFL})
	}).ShouldNot(gomega.Panic())

	// The runner is released so the next manual sync can start
	g.Expect(s.syncRuns.status()).Should(gomega.BeNil())
}
