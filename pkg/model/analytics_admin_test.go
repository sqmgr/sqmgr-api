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

func TestPoolActivityCount(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	a := NewAnalytics(db)

	mock.ExpectQuery(`SELECT id, user_id FROM pools WHERE token = \$1`).
		WithArgs("tok").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}).AddRow(int64(4), int64(1)))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM pool_squares_logs l INNER JOIN pool_squares ps ON ps.id = l.pool_square_id WHERE ps.pool_id = \$1`).
		WithArgs(int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(17)))

	count, err := a.PoolActivityCount(context.Background(), "tok")
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(count).Should(gomega.Equal(int64(17)))

	// Unknown pool
	mock.ExpectQuery(`SELECT id, user_id FROM pools WHERE token = \$1`).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}))
	_, err = a.PoolActivityCount(context.Background(), "missing")
	g.Expect(err).Should(gomega.MatchError(ErrPoolNotFound))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestSportsSyncRunsByType(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	a := NewAnalytics(db)
	now := time.Now()
	columns := []string{"id", "sync_type", "league", "started_at", "completed_at", "records_processed", "error_message", "success"}

	mock.ExpectQuery(`SELECT id, sync_type, league::text, .+ FROM sports_sync_log WHERE sync_type = \$1 ORDER BY id DESC LIMIT \$2`).
		WithArgs("scores", 10).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(int64(5), "scores", nil, now, now, 3, nil, true))

	runs, err := a.SportsSyncRunsByType(context.Background(), "scores", 10)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(runs).Should(gomega.HaveLen(1))
	g.Expect(runs[0].SyncType).Should(gomega.Equal("scores"))
	g.Expect(runs[0].League).Should(gomega.BeNil())

	// No type filter goes through the same path with a clamped default limit
	mock.ExpectQuery(`SELECT id, sync_type, league::text, .+ FROM sports_sync_log ORDER BY id DESC LIMIT \$1`).
		WithArgs(DefaultAnalyticsLimit).
		WillReturnRows(sqlmock.NewRows(columns))
	runs, err = a.SportsSyncRuns(context.Background(), 0)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(runs).Should(gomega.BeEmpty())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestLatestSportsSyncRuns(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	a := NewAnalytics(db)
	now := time.Now()
	columns := []string{"id", "sync_type", "league", "started_at", "completed_at", "records_processed", "error_message", "success"}

	// DISTINCT ON orders by type and league; the result is re-sorted newest first
	mock.ExpectQuery(`SELECT DISTINCT ON \(sync_type, league\) id, sync_type, league::text, .+ FROM sports_sync_log ORDER BY sync_type, league, id DESC`).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(int64(2), "schedule", "nfl", now, now, 100, nil, true).
			AddRow(int64(9), "scores", nil, now, nil, nil, nil, nil).
			AddRow(int64(4), "teams", "nba", now, now, 0, "boom", false))

	runs, err := a.LatestSportsSyncRuns(context.Background())
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(runs).Should(gomega.HaveLen(3))
	g.Expect(runs[0].ID).Should(gomega.Equal(int64(9)))
	g.Expect(runs[1].ID).Should(gomega.Equal(int64(4)))
	g.Expect(*runs[1].ErrorMessage).Should(gomega.Equal("boom"))
	g.Expect(runs[2].ID).Should(gomega.Equal(int64(2)))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestListPoolsSearchMatchesToken(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	a := NewAnalytics(db)

	mock.ExpectQuery(`WHERE \(p.name ILIKE \$1 OR p.token = \$2\)`).
		WithArgs("%abc%", "abc", int64(0), DefaultAnalyticsLimit).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	list, err := a.ListPools(context.Background(), PoolListFilter{Search: "abc"})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(list.Pools).Should(gomega.BeEmpty())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}
