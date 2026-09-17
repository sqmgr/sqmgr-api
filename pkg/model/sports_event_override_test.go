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

package model

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/onsi/gomega"
)

func TestSportsEventApplyOverride(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)
	period := 3
	clock := "5:00"
	detail := "3rd Quarter"
	event := &SportsEvent{model: m, ID: 55, Status: SportsEventStatusInProgress, Period: &period, Clock: &clock, StatusDetail: &detail}

	mock.ExpectExec(`UPDATE sports_events SET status = \$1, home_score = \$2, away_score = \$3, .+ manual_override = true, .+ WHERE id = \$14`).
		WithArgs("final", 24, 17, 7, 10, 0, 7, nil, 3, 7, 7, 0, nil, int64(55), true, false).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`SELECT pg_notify\('sports_event_updated', \$1\)`).
		WithArgs("55").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = event.ApplyOverride(context.Background(), SportsEventOverride{
		Status:    SportsEventStatusFinal,
		HomeScore: intPtr(24), AwayScore: intPtr(17),
		HomeQ1: intPtr(7), HomeQ2: intPtr(10), HomeQ3: intPtr(0), HomeQ4: intPtr(7),
		AwayQ1: intPtr(3), AwayQ2: intPtr(7), AwayQ3: intPtr(7), AwayQ4: intPtr(0),
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	g.Expect(event.ManualOverride).Should(gomega.BeTrue())
	g.Expect(event.Status).Should(gomega.Equal(SportsEventStatusFinal))
	g.Expect(*event.HomeScore).Should(gomega.Equal(24))
	g.Expect(*event.AwayQ3).Should(gomega.Equal(7))
	g.Expect(event.HomeOT).Should(gomega.BeNil())
	// A final event has no period or clock and a "Final" status detail
	g.Expect(event.Period).Should(gomega.BeNil())
	g.Expect(event.Clock).Should(gomega.BeNil())
	g.Expect(*event.StatusDetail).Should(gomega.Equal("Final"))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestSportsEventApplyOverride_InProgressKeepsClock(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)
	period := 2
	clock := "1:00"
	detail := "Final"
	event := &SportsEvent{model: m, ID: 56, Status: SportsEventStatusFinal, Period: &period, Clock: &clock, StatusDetail: &detail}

	mock.ExpectExec(`UPDATE sports_events SET status = \$1`).
		WithArgs("in_progress", 10, 3, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, int64(56), false, true).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`SELECT pg_notify`).WithArgs("56").WillReturnResult(sqlmock.NewResult(0, 0))

	err = event.ApplyOverride(context.Background(), SportsEventOverride{Status: SportsEventStatusInProgress, HomeScore: intPtr(10), AwayScore: intPtr(3)})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(*event.Period).Should(gomega.Equal(2))
	g.Expect(*event.Clock).Should(gomega.Equal("1:00"))
	// ESPN's old status detail no longer applies
	g.Expect(event.StatusDetail).Should(gomega.BeNil())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestSportsEventApplyOverride_InvalidStatus(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	event := &SportsEvent{model: New(db), ID: 1}
	err = event.ApplyOverride(context.Background(), SportsEventOverride{Status: "postponed"})
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("invalid status")))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestSportsEventClearOverride(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	event := &SportsEvent{model: New(db), ID: 9, ManualOverride: true}

	mock.ExpectExec(`UPDATE sports_events SET manual_override = false, modified = \(NOW\(\) AT TIME ZONE 'utc'\) WHERE id = \$1`).
		WithArgs(int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	g.Expect(event.ClearOverride(context.Background())).Should(gomega.Succeed())
	g.Expect(event.ManualOverride).Should(gomega.BeFalse())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestStaleLinkedEvents(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)
	now := time.Now()

	mock.ExpectQuery(`SELECT e.id, e.league, e.name, ht.full_name, at.full_name, e.event_date, e.status, COUNT\(g.id\), e.last_synced FROM sports_events e INNER JOIN grids g ON g.sports_event_id = e.id AND g.state = 'active' .+ WHERE NOT e.manual_override AND \( \(e.status = 'in_progress' AND e.last_synced < \(NOW\(\) AT TIME ZONE 'utc'\) - INTERVAL '30 minutes'\) OR \(e.status = 'scheduled' AND e.event_date < \(NOW\(\) AT TIME ZONE 'utc'\) - INTERVAL '3 hours'\) \) GROUP BY`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "league", "name", "home", "away", "event_date", "status", "count", "last_synced"}).
			AddRow(int64(3), "nfl", nil, "Home Team", "Away Team", now, "in_progress", int64(2), now.Add(-time.Hour)))

	events, err := m.StaleLinkedEvents(context.Background())
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(events).Should(gomega.HaveLen(1))
	g.Expect(events[0].ID).Should(gomega.Equal(int64(3)))
	g.Expect(events[0].League).Should(gomega.Equal(SportsLeagueNFL))
	g.Expect(*events[0].HomeTeam).Should(gomega.Equal("Home Team"))
	g.Expect(events[0].GridCount).Should(gomega.Equal(int64(2)))
	g.Expect(events[0].Status).Should(gomega.Equal(SportsEventStatusInProgress))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestSportsEventOverrideIntegration(t *testing.T) {
	ensureIntegration(t)
	g := gomega.NewWithT(t)

	m := New(getDB())
	ctx := context.Background()

	home := &SportsTeam{ID: "override-home-" + randString(), League: SportsLeagueNFL, Name: "Home", FullName: "Home Team", Abbreviation: "HOM"}
	away := &SportsTeam{ID: "override-away-" + randString(), League: SportsLeagueNFL, Name: "Away", FullName: "Away Team", Abbreviation: "AWY"}
	g.Expect(m.UpsertSportsTeam(ctx, nil, home)).Should(gomega.Succeed())
	g.Expect(m.UpsertSportsTeam(ctx, nil, away)).Should(gomega.Succeed())

	event := m.NewSportsEvent()
	event.ESPNID = "override-" + randString()
	event.League = SportsLeagueNFL
	event.HomeTeamID = home.ID
	event.AwayTeamID = away.ID
	event.EventDate = time.Now().Add(-48 * time.Hour)
	event.Season = 2026
	event.Status = SportsEventStatusInProgress
	g.Expect(m.UpsertSportsEvent(ctx, nil, event)).Should(gomega.Succeed())

	g.Expect(event.ApplyOverride(ctx, SportsEventOverride{Status: SportsEventStatusInProgress, HomeScore: intPtr(21), AwayScore: intPtr(14)})).Should(gomega.Succeed())

	loaded, err := m.SportsEventByID(ctx, event.ID)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(loaded.ManualOverride).Should(gomega.BeTrue())
	g.Expect(*loaded.HomeScore).Should(gomega.Equal(21))
	g.Expect(*loaded.AwayScore).Should(gomega.Equal(14))
	g.Expect(loaded.JSON().ManualOverride).Should(gomega.BeTrue())

	// The sync job's stale finalization leaves overridden events alone even
	// though this one is two days old and still in progress.
	_, err = m.FinalizeStaleEvents(ctx)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	loaded, err = m.SportsEventByID(ctx, event.ID)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(loaded.Status).Should(gomega.Equal(SportsEventStatusInProgress))
	g.Expect(*loaded.HomeScore).Should(gomega.Equal(21))

	g.Expect(loaded.ClearOverride(ctx)).Should(gomega.Succeed())
	_, err = m.FinalizeStaleEvents(ctx)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	loaded, err = m.SportsEventByID(ctx, event.ID)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(loaded.ManualOverride).Should(gomega.BeFalse())
	g.Expect(loaded.Status).Should(gomega.Equal(SportsEventStatusFinal))
}
