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

package sportsync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"github.com/sqmgr/sqmgr-api/pkg/sports"
)

// fakeClient records calls and returns canned data.
type fakeClient struct {
	summary      *sports.Event
	summaryErr   error
	summaryCalls int
}

func (f *fakeClient) GetTeams(context.Context, sports.League) ([]sports.Team, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeClient) GetSeasonInfo(context.Context, sports.League) (*sports.SeasonInfo, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeClient) GetTeamSchedule(context.Context, sports.League, string, sports.SeasonType) ([]sports.Event, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeClient) GetScoreboard(context.Context, sports.League, sports.ScoreboardOptions) ([]sports.Event, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeClient) GetScoreboardForDateRange(context.Context, sports.League, time.Time, time.Time) ([]sports.Event, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeClient) GetEventSummary(context.Context, sports.League, string) (*sports.Event, error) {
	f.summaryCalls++
	return f.summary, f.summaryErr
}

var eventColumns = []string{
	"id", "espn_id", "league", "name", "home_team_id", "away_team_id", "event_date", "season", "week", "postseason", "venue",
	"status", "status_detail", "period", "clock", "home_score", "away_score",
	"home_q1", "home_q2", "home_q3", "home_q4", "home_ot",
	"away_q1", "away_q2", "away_q3", "away_q4", "away_ot",
	"created", "modified", "last_synced", "manual_override",
}

var teamColumns = []string{"id", "league", "name", "full_name", "abbreviation", "conference", "division", "location", "color", "alternate_color", "created", "modified"}

func eventRow(now time.Time, id int64, espnID, status string, homeScore, awayScore interface{}, override bool) *sqlmock.Rows {
	return sqlmock.NewRows(eventColumns).AddRow(
		id, espnID, "nfl", nil, "home", "away", now, 2026, nil, false, nil,
		status, nil, nil, nil, homeScore, awayScore,
		nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil,
		now, now, now, override,
	)
}

func newTestSyncer(t *testing.T, client Client, dryRun bool) (*Syncer, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	return New(model.New(db), client, Options{Logger: logrus.NewEntry(logger), DryRun: dryRun}), mock
}

func TestRefreshEvent_ManualOverride(t *testing.T) {
	g := gomega.NewWithT(t)
	client := &fakeClient{}
	syncer, mock := newTestSyncer(t, client, false)

	_, err := syncer.RefreshEvent(context.Background(), &model.SportsEvent{ID: 1, ESPNID: "401", League: model.SportsLeagueNFL, ManualOverride: true})
	g.Expect(err).Should(gomega.MatchError(ErrManualOverride))
	g.Expect(client.summaryCalls).Should(gomega.BeZero())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestRefreshEvent_NoESPNID(t *testing.T) {
	g := gomega.NewWithT(t)
	client := &fakeClient{}
	syncer, _ := newTestSyncer(t, client, false)

	_, err := syncer.RefreshEvent(context.Background(), &model.SportsEvent{ID: 1, League: model.SportsLeagueNFL})
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("no ESPN id")))
	g.Expect(client.summaryCalls).Should(gomega.BeZero())
}

func TestRefreshEvent_ESPNFailure(t *testing.T) {
	g := gomega.NewWithT(t)
	client := &fakeClient{summaryErr: errors.New("espn down")}
	syncer, _ := newTestSyncer(t, client, false)

	_, err := syncer.RefreshEvent(context.Background(), &model.SportsEvent{ID: 1, ESPNID: "401", League: model.SportsLeagueNFL})
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("espn down")))
	g.Expect(client.summaryCalls).Should(gomega.Equal(1))
}

func TestRefreshEvent_UpdatesAndReloads(t *testing.T) {
	g := gomega.NewWithT(t)
	now := time.Now()
	home := sports.Team{ID: "home", Name: "Home", DisplayName: "Home Team", Abbreviation: "HOM"}
	away := sports.Team{ID: "away", Name: "Away", DisplayName: "Away Team", Abbreviation: "AWY"}
	score := func(i int) *int { return &i }
	client := &fakeClient{summary: &sports.Event{
		ID: "401", Date: now, Status: sports.EventStatusFinal, Season: 2026,
		HomeTeam: home, AwayTeam: away, HomeTeamScore: score(24), AwayTeamScore: score(17),
	}}
	syncer, mock := newTestSyncer(t, client, false)

	// ProcessEvent: load existing, upsert both teams, upsert event, notify
	mock.ExpectQuery(`SELECT .+ FROM sports_events WHERE espn_id = \$1`).
		WithArgs("401").
		WillReturnRows(eventRow(now, 9, "401", "in_progress", 14, 10, false))
	mock.ExpectExec(`INSERT INTO sports_teams`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO sports_teams`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO sports_events`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(9)))
	mock.ExpectExec(`SELECT pg_notify\('sports_event_updated', \$1\)`).WithArgs("9").WillReturnResult(sqlmock.NewResult(0, 0))

	// Reload with teams
	mock.ExpectQuery(`SELECT .+ FROM sports_events WHERE id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(eventRow(now, 9, "401", "final", 24, 17, false))
	mock.ExpectQuery(`SELECT .+ FROM sports_teams WHERE id = \$1 AND league = \$2`).
		WithArgs("home", model.SportsLeagueNFL).
		WillReturnRows(sqlmock.NewRows(teamColumns).AddRow("home", "nfl", "Home", "Home Team", "HOM", nil, nil, nil, nil, nil, now, now))
	mock.ExpectQuery(`SELECT .+ FROM sports_teams WHERE id = \$1 AND league = \$2`).
		WithArgs("away", model.SportsLeagueNFL).
		WillReturnRows(sqlmock.NewRows(teamColumns).AddRow("away", "nfl", "Away", "Away Team", "AWY", nil, nil, nil, nil, nil, now, now))

	updated, err := syncer.RefreshEvent(context.Background(), &model.SportsEvent{ID: 9, ESPNID: "401", League: model.SportsLeagueNFL})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(updated.Status).Should(gomega.Equal(model.SportsEventStatusFinal))
	g.Expect(*updated.HomeScore).Should(gomega.Equal(24))
	g.Expect(updated.HomeTeam().FullName).Should(gomega.Equal("Home Team"))
	g.Expect(client.summaryCalls).Should(gomega.Equal(1))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestProcessEvent_DryRunTouchesNothing(t *testing.T) {
	g := gomega.NewWithT(t)
	syncer, mock := newTestSyncer(t, &fakeClient{}, true)

	err := syncer.ProcessEvent(context.Background(), model.SportsLeagueNFL, sports.Event{ID: "401"})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestProcessEvent_SkipsManualOverride(t *testing.T) {
	g := gomega.NewWithT(t)
	syncer, mock := newTestSyncer(t, &fakeClient{}, false)

	mock.ExpectQuery(`SELECT .+ FROM sports_events WHERE espn_id = \$1`).
		WithArgs("401").
		WillReturnRows(eventRow(time.Now(), 9, "401", "in_progress", 14, 10, true))

	err := syncer.ProcessEvent(context.Background(), model.SportsLeagueNFL, sports.Event{ID: "401", Status: sports.EventStatusFinal})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	// No team or event upsert happened
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestProcessEvent_NewEventSyncsGrids(t *testing.T) {
	g := gomega.NewWithT(t)
	syncer, mock := newTestSyncer(t, &fakeClient{}, false)

	mock.ExpectQuery(`SELECT .+ FROM sports_events WHERE espn_id = \$1`).
		WithArgs("402").
		WillReturnRows(sqlmock.NewRows(eventColumns))
	mock.ExpectExec(`INSERT INTO sports_teams`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO sports_teams`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO sports_events`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))
	// A brand new event has no listeners to notify but does sync grid names and colors
	mock.ExpectExec(`UPDATE grids g`).WithArgs(int64(10)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`UPDATE grid_settings gs`).WithArgs(int64(10)).WillReturnResult(sqlmock.NewResult(0, 0))

	err := syncer.ProcessEvent(context.Background(), model.SportsLeagueNFL, sports.Event{
		ID: "402", Date: time.Now(), Season: 2026,
		HomeTeam: sports.Team{ID: "h"}, AwayTeam: sports.Team{ID: "a"},
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestAllLeagues(t *testing.T) {
	g := gomega.NewWithT(t)
	g.Expect(AllLeagues()).Should(gomega.ConsistOf(
		model.SportsLeagueNFL, model.SportsLeagueNBA, model.SportsLeagueWNBA, model.SportsLeagueNCAAB, model.SportsLeagueNCAAF,
	))
}

func TestCompleteSyncLog_SurvivesCancelledContext(t *testing.T) {
	g := gomega.NewWithT(t)
	syncer, mock := newTestSyncer(t, &fakeClient{}, false)

	// The run's context has already expired, as it would after a timeout.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mock.ExpectQuery(`INSERT INTO sports_sync_log`).
		WithArgs(model.SportsSyncTypeScores, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "started_at"}).AddRow(int64(3), time.Now()))
	mock.ExpectExec(`UPDATE sports_sync_log`).
		WithArgs(7, false, "deadline exceeded", int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	syncLog := syncer.startSyncLog(context.Background(), model.SportsSyncTypeScores, nil, syncer.log)
	g.Expect(syncLog).ShouldNot(gomega.BeNil())
	syncer.completeSyncLog(ctx, syncLog, 7, errors.New("deadline exceeded"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}
