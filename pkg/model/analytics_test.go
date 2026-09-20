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

package model

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/onsi/gomega"
)

func TestDateRangeConditions(t *testing.T) {
	g := gomega.NewWithT(t)

	var args queryArgs
	g.Expect(DateRange{}.conditions("created", &args)).Should(gomega.BeEmpty())
	g.Expect(args).Should(gomega.BeEmpty())
	g.Expect(DateRange{}.IsZero()).Should(gomega.BeTrue())

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

	args = queryArgs{"existing"}
	conds := DateRange{Start: start, End: end}.conditions("p.created", &args)
	g.Expect(conds).Should(gomega.Equal([]string{"p.created >= $2", "p.created < $3"}))
	g.Expect(args).Should(gomega.Equal(queryArgs{"existing", start, end}))

	args = nil
	conds = DateRange{End: end}.conditions("created", &args)
	g.Expect(conds).Should(gomega.Equal([]string{"created < $1"}))
	g.Expect(args).Should(gomega.Equal(queryArgs{end}))

	g.Expect(whereClause(nil)).Should(gomega.BeEmpty())
	g.Expect(whereClause([]string{"a", "b"})).Should(gomega.Equal(" WHERE a AND b"))
}

func TestClampLimitAndSortDirection(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(clampLimit(0, 25, 500)).Should(gomega.Equal(25))
	g.Expect(clampLimit(-5, 25, 500)).Should(gomega.Equal(25))
	g.Expect(clampLimit(10, 25, 500)).Should(gomega.Equal(10))
	g.Expect(clampLimit(9999, 25, 500)).Should(gomega.Equal(500))

	g.Expect(sortDirection("asc", "DESC")).Should(gomega.Equal("ASC"))
	g.Expect(sortDirection("DESC", "ASC")).Should(gomega.Equal("DESC"))
	g.Expect(sortDirection("sideways", "DESC")).Should(gomega.Equal("DESC"))
}

func TestTruncateToInterval(t *testing.T) {
	g := gomega.NewWithT(t)

	// Wednesday 2024-03-13 15:04:05
	ts := time.Date(2024, 3, 13, 15, 4, 5, 0, time.UTC)

	g.Expect(truncateToInterval(ts, IntervalDay)).Should(gomega.Equal(time.Date(2024, 3, 13, 0, 0, 0, 0, time.UTC)))
	g.Expect(truncateToInterval(ts, IntervalWeek)).Should(gomega.Equal(time.Date(2024, 3, 11, 0, 0, 0, 0, time.UTC)), "weeks start on Monday")
	g.Expect(truncateToInterval(ts, IntervalMonth)).Should(gomega.Equal(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)))
	g.Expect(truncateToInterval(ts, IntervalYear)).Should(gomega.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))

	// A Sunday belongs to the week that started the previous Monday
	sunday := time.Date(2024, 3, 17, 23, 0, 0, 0, time.UTC)
	g.Expect(truncateToInterval(sunday, IntervalWeek)).Should(gomega.Equal(time.Date(2024, 3, 11, 0, 0, 0, 0, time.UTC)))
	// A Monday is its own week start
	monday := time.Date(2024, 3, 18, 0, 0, 0, 0, time.UTC)
	g.Expect(truncateToInterval(monday, IntervalWeek)).Should(gomega.Equal(monday))

	g.Expect(nextPeriod(monday, IntervalDay)).Should(gomega.Equal(monday.AddDate(0, 0, 1)))
	g.Expect(nextPeriod(monday, IntervalWeek)).Should(gomega.Equal(monday.AddDate(0, 0, 7)))
	g.Expect(nextPeriod(monday, IntervalMonth)).Should(gomega.Equal(monday.AddDate(0, 1, 0)))
	g.Expect(nextPeriod(monday, IntervalYear)).Should(gomega.Equal(monday.AddDate(1, 0, 0)))
}

func TestPeriodsToFill(t *testing.T) {
	g := gomega.NewWithT(t)

	jan := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	apr := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	buckets := []time.Time{jan, apr}
	period := func(i int) time.Time { return buckets[i] }

	// No range: fill between the first and last observed buckets
	periods := periodsToFill(buckets, IntervalMonth, DateRange{}, period)
	g.Expect(periods).Should(gomega.Equal([]time.Time{
		jan, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), apr,
	}))

	// A range extends the fill on both sides; the exclusive end lands in June
	r := DateRange{Start: time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC), End: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)}
	periods = periodsToFill(buckets, IntervalMonth, r, period)
	g.Expect(periods).Should(gomega.HaveLen(7))
	g.Expect(periods[0]).Should(gomega.Equal(time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)))
	g.Expect(periods[6]).Should(gomega.Equal(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)))

	// No buckets and no range yields nothing; no buckets but a range fills the range
	g.Expect(periodsToFill([]time.Time{}, IntervalMonth, DateRange{}, period)).Should(gomega.BeEmpty())
	g.Expect(periodsToFill([]time.Time{}, IntervalDay, DateRange{Start: jan, End: jan.AddDate(0, 0, 3)}, period)).Should(gomega.HaveLen(3))

	// Above the fill cap only observed buckets come back
	far := []time.Time{jan, jan.AddDate(30, 0, 0)}
	periods = periodsToFill(far, IntervalDay, DateRange{}, func(i int) time.Time { return far[i] })
	g.Expect(periods).Should(gomega.Equal(far))
}

func TestPrepareQuery(t *testing.T) {
	g := gomega.NewWithT(t)

	wrapped, err := prepareQuery("  SELECT id FROM pools ; \n")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(wrapped).Should(gomega.Equal("SELECT * FROM (\nSELECT id FROM pools\n) AS mcp_query LIMIT $1"))

	for _, bad := range []string{
		"",
		"   ;  ",
		"SELECT 1; DELETE FROM pools",
		"SELECT pg_sleep(10)",
		"select PG_TERMINATE_BACKEND(1)",
		"SELECT set_config('statement_timeout', '0', true)",
		"SELECT * FROM dblink('...')",
		"SELECT password_hash AS h FROM pools",
		`SELECT U&"pg\005fread\005ffile"('/etc/passwd')`,
		`SELECT u&'\0041' FROM pools`,
		`SELECT E'\x70' FROM pools`,
		`SELECT "pg_read\_file"('x')`,
	} {
		_, err := prepareQuery(bad)
		g.Expect(err).Should(gomega.HaveOccurred(), "query %q", bad)
		g.Expect(errors.Is(err, ErrInvalidQuery)).Should(gomega.BeTrue(), "query %q", bad)
	}

	// Identifiers that merely contain a denied name are fine, as are ordinary
	// quoted identifiers and strings
	_, err = prepareQuery("SELECT my_pg_sleep_count FROM stats")
	g.Expect(err).Should(gomega.Succeed())
	_, err = prepareQuery(`SELECT "name", 'Sue''s pool' AS label FROM pools WHERE name = 'Super Bowl'`)
	g.Expect(err).Should(gomega.Succeed())
}

func TestAnalyticsValidation(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	db, _, err := sqlmock.New()
	g.Expect(err).Should(gomega.Succeed())
	defer db.Close()
	a := NewAnalytics(db)

	_, err = a.TimeSeries(ctx, "nope", IntervalMonth, DateRange{})
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("unknown metric")))

	_, err = a.TimeSeries(ctx, MetricPoolsCreated, "fortnight", DateRange{})
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("unknown interval")))

	_, err = a.PoolBreakdown(ctx, "color", DateRange{})
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("unknown dimension")))

	_, err = a.ListUsers(ctx, UserListFilter{Store: "ldap"})
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("unknown store")))

	_, err = a.UserDetails(ctx, 0, "")
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("user id or an email")))

	_, err = a.PopularEvents(ctx, "xfl", DateRange{}, 10)
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("invalid league")))

	_, err = a.Query(ctx, "SELECT 1; SELECT 2", 10)
	g.Expect(errors.Is(err, ErrInvalidQuery)).Should(gomega.BeTrue())

	g.Expect(BreakdownDimensions()).Should(gomega.HaveKey("grid_type"))
	g.Expect(BreakdownDimensions()).Should(gomega.HaveLen(len(breakdownDimensions)))
}

func TestRunReadOnly(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	g.Expect(err).Should(gomega.Succeed())
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = 30000")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1")).WillReturnRows(sqlmock.NewRows([]string{"one"}).AddRow(1))
	mock.ExpectRollback()

	var got int
	err = RunReadOnly(ctx, db, 30*time.Second, func(a *Analytics) error {
		return a.q.QueryRowContext(ctx, "SELECT 1").Scan(&got)
	})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(got).Should(gomega.Equal(1))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	// Errors from fn propagate and the transaction is still rolled back
	mock.ExpectBegin()
	mock.ExpectRollback()
	boom := errors.New("boom")
	err = RunReadOnly(ctx, db, 0, func(a *Analytics) error { return boom })
	g.Expect(errors.Is(err, boom)).Should(gomega.BeTrue())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	// A failure to begin is reported
	mock.ExpectBegin().WillReturnError(errors.New("no conn"))
	err = RunReadOnly(ctx, db, 0, func(a *Analytics) error { return nil })
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("beginning read-only transaction")))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestAnalyticsQueryWithMock(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	g.Expect(err).Should(gomega.Succeed())
	defer db.Close()
	a := NewAnalytics(db)

	rows := sqlmock.NewRows([]string{"token", "password_hash", "name"}).
		AddRow("abc", "secret", []byte("Pool A")).
		AddRow("def", "secret", []byte("Pool B")).
		AddRow("ghi", "secret", []byte("Pool C"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM (\nSELECT * FROM pools\n) AS mcp_query LIMIT $1")).
		WithArgs(3).
		WillReturnRows(rows)

	result, err := a.Query(ctx, "SELECT * FROM pools;", 2)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(result.Columns).Should(gomega.Equal([]string{"token", "name"}), "password_hash is hidden even when selected implicitly")
	g.Expect(result.RowCount).Should(gomega.Equal(2))
	g.Expect(result.Truncated).Should(gomega.BeTrue())
	g.Expect(result.Rows).Should(gomega.Equal([][]interface{}{{"abc", "Pool A"}, {"def", "Pool B"}}))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	// Default limit is applied when maxRows is zero, and a short result is not truncated
	mock.ExpectQuery(regexp.QuoteMeta("LIMIT $1")).WithArgs(DefaultQueryRows + 1).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(1)))
	result, err = a.Query(ctx, "SELECT 1 AS n", 0)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(result.Truncated).Should(gomega.BeFalse())
	g.Expect(result.RowCount).Should(gomega.Equal(1))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	// Database errors are wrapped
	mock.ExpectQuery(regexp.QuoteMeta("LIMIT $1")).WillReturnError(errors.New("syntax error"))
	_, err = a.Query(ctx, "SELECT nope", 5)
	g.Expect(errors.Is(err, ErrQueryFailed)).Should(gomega.BeTrue())
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("syntax error")))
}

// seedAnalyticsData creates a registered owner, a guest, a 25-square pool with
// two claimed squares (one by the owner, one by the guest), a second empty
// pool, and a pool joined by the guest. It returns the owner, guest, and the
// pools in that order.
func seedAnalyticsData(t *testing.T, m *Model, ctx context.Context) (*User, *User, *Pool, *Pool) {
	t.Helper()
	g := gomega.NewWithT(t)

	owner, err := m.GetUser(ctx, IssuerAuth0, "auth0|analytics-"+randString())
	g.Expect(err).Should(gomega.Succeed())
	email := "analytics-" + randString() + "@example.com"
	g.Expect(owner.SetEmail(ctx, email)).Should(gomega.Succeed())

	guest, err := m.GetUser(ctx, IssuerSqMGR, "guest-"+randString())
	g.Expect(err).Should(gomega.Succeed())

	pool, err := m.NewPool(ctx, owner.ID, "Analytics Pool "+randString(), GridTypeStd25, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(guest.JoinPool(ctx, pool)).Should(gomega.Succeed())

	claim := func(squareID int, u *User, claimant string) {
		sq, err := pool.SquareBySquareID(squareID)
		g.Expect(err).Should(gomega.Succeed())
		sq.claimant = claimant
		sq.State = PoolSquareStateClaimed
		sq.SetUserID(u.ID)
		g.Expect(sq.Save(ctx, m.DB, true, PoolSquareLog{Note: "test claim", RemoteAddr: "127.0.0.1"})).Should(gomega.Succeed())
	}
	claim(1, owner, "Owner")
	claim(2, guest, "Guest")

	empty, err := m.NewPool(ctx, owner.ID, "Empty Analytics Pool "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	return owner, guest, pool, empty
}

func TestAnalyticsIntegration(t *testing.T) {
	ensureIntegration(t)
	m := New(getDB())
	ctx := context.Background()

	owner, guest, pool, empty := seedAnalyticsData(t, m, ctx)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	recent := DateRange{Start: today.AddDate(0, 0, -1), End: today.AddDate(0, 0, 2)}

	run := func(t *testing.T, fn func(g *gomega.WithT, a *Analytics)) {
		t.Helper()
		g := gomega.NewWithT(t)
		err := RunReadOnly(ctx, m.DB, 10*time.Second, func(a *Analytics) error {
			fn(g, a)
			return nil
		})
		g.Expect(err).Should(gomega.Succeed())
	}

	t.Run("rejects writes", func(t *testing.T) {
		g := gomega.NewWithT(t)
		err := RunReadOnly(ctx, m.DB, 10*time.Second, func(a *Analytics) error {
			_, err := a.q.ExecContext(ctx, "UPDATE pools SET name = 'hacked' WHERE id = $1", pool.ID())
			return err
		})
		g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("read-only transaction")))

		// The wrapped ad hoc query cannot smuggle a write either
		err = RunReadOnly(ctx, m.DB, 10*time.Second, func(a *Analytics) error {
			_, err := a.Query(ctx, "DELETE FROM pools WHERE id = "+itoa(pool.ID()), 10)
			return err
		})
		g.Expect(err).Should(gomega.HaveOccurred())

		reloaded, err := m.PoolByToken(ctx, pool.Token())
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(reloaded.Name()).Should(gomega.Equal(pool.Name()))
	})

	t.Run("site stats", func(t *testing.T) {
		run(t, func(g *gomega.WithT, a *Analytics) {
			stats, err := a.SiteStats(ctx, DateRange{})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(stats.Pools).Should(gomega.BeNumerically(">=", 2))
			g.Expect(stats.ActivePools + stats.ArchivedPools).Should(gomega.Equal(stats.Pools))
			g.Expect(stats.RegisteredUsers).Should(gomega.BeNumerically(">=", 1))
			g.Expect(stats.GuestUsers).Should(gomega.BeNumerically(">=", 1))
			g.Expect(stats.ClaimedSquares).Should(gomega.BeNumerically(">=", 2))
			g.Expect(stats.PoolMemberships).Should(gomega.BeNumerically(">=", 1))

			none, err := a.SiteStats(ctx, DateRange{Start: today.AddDate(50, 0, 0)})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(*none).Should(gomega.Equal(SiteStats{}))
		})
	})

	t.Run("time series", func(t *testing.T) {
		run(t, func(g *gomega.WithT, a *Analytics) {
			for _, metric := range TimeSeriesMetrics {
				points, err := a.TimeSeries(ctx, metric, IntervalDay, recent)
				g.Expect(err).Should(gomega.Succeed(), metric)
				g.Expect(points).Should(gomega.HaveLen(3), metric)
			}

			points, err := a.TimeSeries(ctx, MetricPoolsCreated, IntervalDay, recent)
			g.Expect(err).Should(gomega.Succeed())
			var total int64
			for _, p := range points {
				total += p.Count
			}
			g.Expect(total).Should(gomega.BeNumerically(">=", 2))
			g.Expect(points[0].Period).Should(gomega.Equal(today.AddDate(0, 0, -1).Format("2006-01-02")))

			monthly, err := a.TimeSeries(ctx, MetricSquaresClaimed, IntervalMonth, DateRange{})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(monthly).ShouldNot(gomega.BeEmpty())
		})
	})

	t.Run("fill rates", func(t *testing.T) {
		run(t, func(g *gomega.WithT, a *Analytics) {
			rates, err := a.PoolFillRates(ctx, PoolFillRateFilter{Created: recent})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(rates.Pools).Should(gomega.BeNumerically(">=", 2))
			g.Expect(rates.FullPools + rates.PartialPools + rates.EmptyPools).Should(gomega.Equal(rates.Pools))
			g.Expect(rates.PartialPools).Should(gomega.BeNumerically(">=", 1))
			g.Expect(rates.EmptyPools).Should(gomega.BeNumerically(">=", 1))
			g.Expect(rates.ClaimedSquares).Should(gomega.BeNumerically(">=", 2))

			archived := false
			only25, err := a.PoolFillRates(ctx, PoolFillRateFilter{Created: recent, GridType: GridTypeStd25, Archived: &archived})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(only25.Pools).Should(gomega.BeNumerically(">=", 1))
			g.Expect(only25.Pools).Should(gomega.BeNumerically("<=", rates.Pools))
		})
	})

	t.Run("breakdown", func(t *testing.T) {
		run(t, func(g *gomega.WithT, a *Analytics) {
			for dimension := range breakdownDimensions {
				rows, err := a.PoolBreakdown(ctx, dimension, recent)
				g.Expect(err).Should(gomega.Succeed(), dimension)
				var pct float64
				for _, r := range rows {
					pct += r.Percent
				}
				if len(rows) > 0 {
					g.Expect(pct).Should(gomega.BeNumerically("~", 100, 0.01), dimension)
				}
			}

			rows, err := a.PoolBreakdown(ctx, "grid_type", recent)
			g.Expect(err).Should(gomega.Succeed())
			values := make([]string, 0, len(rows))
			for _, r := range rows {
				values = append(values, r.Value)
			}
			g.Expect(values).Should(gomega.ContainElements("std25", "std100"))
		})
	})

	t.Run("engagement", func(t *testing.T) {
		run(t, func(g *gomega.WithT, a *Analytics) {
			s, err := a.EngagementSummary(ctx, recent)
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(s.RegisteredUsersCreated).Should(gomega.BeNumerically(">=", 1))
			g.Expect(s.GuestUsersCreated).Should(gomega.BeNumerically(">=", 1))
			g.Expect(s.PoolsCreated).Should(gomega.BeNumerically(">=", 2))
			g.Expect(s.PoolCreators).Should(gomega.BeNumerically(">=", 1))
			g.Expect(s.RepeatPoolCreators).Should(gomega.BeNumerically(">=", 1))
			g.Expect(s.PoolsWithAnyClaims).Should(gomega.BeNumerically(">=", 1))
			g.Expect(s.ClaimEvents).Should(gomega.BeNumerically(">=", 2))
			g.Expect(s.ClaimEventsRegistered).Should(gomega.BeNumerically(">=", 1))
			g.Expect(s.ClaimEventsGuest).Should(gomega.BeNumerically(">=", 1))
			g.Expect(s.DistinctClaimers).Should(gomega.BeNumerically(">=", 2))

			all, err := a.EngagementSummary(ctx, DateRange{})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(all.ReturningPoolCreators).Should(gomega.BeZero(), "only computed when the range has a start")
		})
	})

	t.Run("list pools", func(t *testing.T) {
		run(t, func(g *gomega.WithT, a *Analytics) {
			list, err := a.ListPools(ctx, PoolListFilter{Search: pool.Name()})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(list.Total).Should(gomega.Equal(int64(1)))
			g.Expect(list.Pools).Should(gomega.HaveLen(1))
			p := list.Pools[0]
			g.Expect(p.Token).Should(gomega.Equal(pool.Token()))
			g.Expect(p.OwnerID).Should(gomega.Equal(owner.ID))
			g.Expect(*p.OwnerEmail).Should(gomega.Equal(*owner.Email))
			g.Expect(p.OwnerStore).Should(gomega.Equal("auth0"))
			g.Expect(p.MemberCount).Should(gomega.Equal(int64(1)))
			g.Expect(p.GridCount).Should(gomega.Equal(int64(1)))
			g.Expect(p.TotalSquares).Should(gomega.Equal(int64(25)))
			g.Expect(p.ClaimedSquares).Should(gomega.Equal(int64(2)))
			g.Expect(p.FillPercent).Should(gomega.Equal(float64(8)))

			minFill := 1.0
			list, err = a.ListPools(ctx, PoolListFilter{OwnerEmail: *owner.Email, MinFillPercent: &minFill, SortBy: "fill_percent", SortDir: "asc"})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(list.Total).Should(gomega.Equal(int64(1)))
			g.Expect(list.Pools[0].Token).Should(gomega.Equal(pool.Token()))

			maxFill := 0.0
			list, err = a.ListPools(ctx, PoolListFilter{OwnerEmail: *owner.Email, MaxFillPercent: &maxFill})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(list.Pools).Should(gomega.HaveLen(1))
			g.Expect(list.Pools[0].Token).Should(gomega.Equal(empty.Token()))

			list, err = a.ListPools(ctx, PoolListFilter{OwnerEmail: *owner.Email, GridType: GridTypeStd100, Created: recent, Limit: 1})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(list.Total).Should(gomega.Equal(int64(1)))
			g.Expect(list.Pools[0].Token).Should(gomega.Equal(empty.Token()))

			list, err = a.ListPools(ctx, PoolListFilter{OwnerEmail: *owner.Email, Offset: 5})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(list.Pools).Should(gomega.BeEmpty())

			// The owner filter is a case-insensitive substring match, so a
			// fragment of the address (without the domain) still finds the
			// owner's pools.
			fragment := strings.ToUpper(strings.TrimSuffix(*owner.Email, "@example.com"))
			list, err = a.ListPools(ctx, PoolListFilter{OwnerEmail: fragment})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(list.Total).Should(gomega.Equal(int64(2)))
			for _, p := range list.Pools {
				g.Expect(p.OwnerID).Should(gomega.Equal(owner.ID))
			}

			list, err = a.ListPools(ctx, PoolListFilter{OwnerEmail: "nobody-" + randString()})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(list.Total).Should(gomega.BeZero())
		})
	})

	t.Run("pool details, squares, members, activity", func(t *testing.T) {
		run(t, func(g *gomega.WithT, a *Analytics) {
			d, err := a.PoolDetails(ctx, pool.Token())
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(d.Token).Should(gomega.Equal(pool.Token()))
			g.Expect(d.PasswordRequired).Should(gomega.BeTrue())
			g.Expect(d.SquareStates).Should(gomega.Equal(SquareStateCounts{Unclaimed: 23, Claimed: 2}))
			g.Expect(d.Grids).Should(gomega.HaveLen(1))
			g.Expect(d.Grids[0].State).Should(gomega.Equal(Active))
			g.Expect(d.Grids[0].SportsEventID).Should(gomega.BeNil())

			_, err = a.PoolDetails(ctx, "does-not-exist")
			g.Expect(errors.Is(err, ErrPoolNotFound)).Should(gomega.BeTrue())

			squares, err := a.PoolSquares(ctx, pool.Token(), false)
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(squares).Should(gomega.HaveLen(2))
			g.Expect(squares[0].SquareID).Should(gomega.Equal(1))
			g.Expect(*squares[0].Claimant).Should(gomega.Equal("Owner"))
			g.Expect(*squares[0].UserID).Should(gomega.Equal(owner.ID))
			g.Expect(*squares[0].UserEmail).Should(gomega.Equal(*owner.Email))
			g.Expect(*squares[0].UserStore).Should(gomega.Equal("auth0"))
			g.Expect(squares[1].UserEmail).Should(gomega.BeNil(), "guests have no email")
			g.Expect(*squares[1].UserStore).Should(gomega.Equal("sqmgr"))

			all, err := a.PoolSquares(ctx, pool.Token(), true)
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(all).Should(gomega.HaveLen(25))

			_, err = a.PoolSquares(ctx, "does-not-exist", true)
			g.Expect(errors.Is(err, ErrPoolNotFound)).Should(gomega.BeTrue())

			members, err := a.PoolMembers(ctx, pool.Token())
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(members).Should(gomega.HaveLen(2))
			g.Expect(members[0].UserID).Should(gomega.Equal(owner.ID))
			g.Expect(members[0].IsOwner).Should(gomega.BeTrue())
			g.Expect(members[0].IsManager).Should(gomega.BeTrue())
			g.Expect(members[0].SquaresClaimed).Should(gomega.Equal(int64(1)))
			g.Expect(members[1].UserID).Should(gomega.Equal(guest.ID))
			g.Expect(members[1].IsOwner).Should(gomega.BeFalse())
			g.Expect(members[1].Joined).ShouldNot(gomega.BeNil())
			g.Expect(members[1].SquaresClaimed).Should(gomega.Equal(int64(1)))

			activity, err := a.PoolActivity(ctx, pool.Token(), 0, 10)
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(activity).Should(gomega.HaveLen(2))
			g.Expect(activity[0].SquareID).Should(gomega.Equal(2), "newest first")
			g.Expect(activity[0].Note).Should(gomega.Equal("test claim"))
			g.Expect(*activity[1].UserEmail).Should(gomega.Equal(*owner.Email))

			page, err := a.PoolActivity(ctx, pool.Token(), 1, 10)
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(page).Should(gomega.HaveLen(1))
		})
	})

	t.Run("users", func(t *testing.T) {
		run(t, func(g *gomega.WithT, a *Analytics) {
			list, err := a.ListUsers(ctx, UserListFilter{Search: *owner.Email})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(list.Total).Should(gomega.Equal(int64(1)))
			u := list.Users[0]
			g.Expect(u.ID).Should(gomega.Equal(owner.ID))
			g.Expect(u.PoolsOwned).Should(gomega.Equal(int64(2)))
			g.Expect(u.PoolsJoined).Should(gomega.BeZero())
			g.Expect(u.SquaresClaimed).Should(gomega.Equal(int64(1)))

			guests, err := a.ListUsers(ctx, UserListFilter{Store: "sqmgr", Created: recent, SortBy: "pools_joined", Limit: 500})
			g.Expect(err).Should(gomega.Succeed())
			var found *AnalyticsUser
			for _, gu := range guests.Users {
				if gu.ID == guest.ID {
					found = gu
				}
			}
			g.Expect(found).ShouldNot(gomega.BeNil())
			g.Expect(found.PoolsJoined).Should(gomega.Equal(int64(1)))
			g.Expect(found.Store).Should(gomega.Equal(UserStoreSqMGR))

			all, err := a.ListUsers(ctx, UserListFilter{Store: "all", Created: recent, Limit: 500})
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(all.Total).Should(gomega.BeNumerically(">=", 2))

			d, err := a.UserDetails(ctx, 0, *owner.Email)
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(d.ID).Should(gomega.Equal(owner.ID))
			g.Expect(d.StoreID).Should(gomega.Equal(owner.StoreID))
			g.Expect(d.PoolsOwnedList).Should(gomega.HaveLen(2))
			g.Expect(d.PoolsOwnedList[0].Token).Should(gomega.Equal(empty.Token()), "newest first")
			g.Expect(d.PoolsJoinedList).Should(gomega.BeEmpty())

			gd, err := a.UserDetails(ctx, guest.ID, "")
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(gd.PoolsJoinedList).Should(gomega.HaveLen(1))
			g.Expect(gd.PoolsJoinedList[0].Token).Should(gomega.Equal(pool.Token()))
			g.Expect(gd.PoolsJoinedList[0].Joined).ShouldNot(gomega.BeNil())

			_, err = a.UserDetails(ctx, 0, "nobody-"+randString()+"@example.com")
			g.Expect(errors.Is(err, ErrUserNotFound)).Should(gomega.BeTrue())

			creators, err := a.TopPoolCreators(ctx, recent, 500)
			g.Expect(err).Should(gomega.Succeed())
			var creator *PoolCreator
			for _, c := range creators {
				if c.UserID == owner.ID {
					creator = c
				}
			}
			g.Expect(creator).ShouldNot(gomega.BeNil())
			g.Expect(creator.PoolsCreated).Should(gomega.Equal(int64(2)))
			g.Expect(creator.SquaresClaimed).Should(gomega.Equal(int64(2)))
			g.Expect(creator.Members).Should(gomega.Equal(int64(1)))
		})
	})

	t.Run("popular events and sync runs", func(t *testing.T) {
		event, linkedPool, _ := createTestGridLinkedToEvent(t, m, ctx, "Analytics Home", "Analytics Away", "ff0000", "0000ff")

		run(t, func(g *gomega.WithT, a *Analytics) {
			events, err := a.PopularEvents(ctx, string(SportsLeagueNFL), DateRange{}, 500)
			g.Expect(err).Should(gomega.Succeed())
			var found *PopularEvent
			for _, e := range events {
				if e.ID == event.ID {
					found = e
				}
			}
			g.Expect(found).ShouldNot(gomega.BeNil())
			g.Expect(found.GridCount).Should(gomega.Equal(int64(1)))
			g.Expect(found.PoolCount).Should(gomega.Equal(int64(1)))
			g.Expect(*found.HomeTeam).Should(gomega.Equal("Analytics Home"))

			d, err := a.PoolDetails(ctx, linkedPool.Token())
			g.Expect(err).Should(gomega.Succeed())
			var linked *AnalyticsGrid
			for _, gr := range d.Grids {
				if gr.SportsEventID != nil && *gr.SportsEventID == event.ID {
					linked = gr
				}
			}
			g.Expect(linked).ShouldNot(gomega.BeNil())
			g.Expect(*linked.League).Should(gomega.Equal("nfl"))

			runs, err := a.SportsSyncRuns(ctx, 5)
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(len(runs)).Should(gomega.BeNumerically("<=", 5))
		})
	})

	t.Run("schema and ad hoc query", func(t *testing.T) {
		run(t, func(g *gomega.WithT, a *Analytics) {
			schema, err := a.Schema(ctx)
			g.Expect(err).Should(gomega.Succeed())
			tables := make(map[string][]SchemaColumn)
			for _, tbl := range schema.Tables {
				tables[tbl.Name] = tbl.Columns
			}
			g.Expect(tables).Should(gomega.HaveKey("pools"))
			g.Expect(tables).Should(gomega.HaveKey("pool_squares"))
			g.Expect(tables).ShouldNot(gomega.HaveKey("schema_migrations"))
			g.Expect(tables["pools"][0].Name).Should(gomega.Equal("id"))
			enums := make(map[string][]string)
			for _, e := range schema.Enums {
				enums[e.Name] = e.Values
			}
			g.Expect(enums["square_states"]).Should(gomega.Equal([]string{"unclaimed", "claimed", "paid-partial", "paid-full"}))

			result, err := a.Query(ctx, "SELECT p.*, COUNT(ps.id) AS claimed FROM pools p JOIN pool_squares ps ON ps.pool_id = p.id AND ps.state != 'unclaimed' WHERE p.id = "+itoa(pool.ID())+" GROUP BY p.id;", 10)
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(result.Columns).ShouldNot(gomega.ContainElement("password_hash"))
			g.Expect(result.Columns).Should(gomega.ContainElements("token", "name", "claimed"))
			g.Expect(result.Rows[0][len(result.Columns)-1]).Should(gomega.BeEquivalentTo(2))

			_, err = a.Query(ctx, "SELECT password_hash AS h FROM pools", 10)
			g.Expect(errors.Is(err, ErrInvalidQuery)).Should(gomega.BeTrue())

			result, err = a.Query(ctx, "SELECT p.token, COUNT(ps.id) AS claimed FROM pools p JOIN pool_squares ps ON ps.pool_id = p.id AND ps.state != 'unclaimed' WHERE p.id = "+itoa(pool.ID())+" GROUP BY p.id", 10)
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(result.Columns).Should(gomega.Equal([]string{"token", "claimed"}))
			g.Expect(result.Rows).Should(gomega.HaveLen(1))
			g.Expect(result.Rows[0][0]).Should(gomega.Equal(pool.Token()))
			g.Expect(result.Rows[0][1]).Should(gomega.BeEquivalentTo(2))
			g.Expect(result.Truncated).Should(gomega.BeFalse())

			result, err = a.Query(ctx, "WITH ids AS (SELECT id FROM pools ORDER BY id) SELECT id FROM ids", 1)
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(result.RowCount).Should(gomega.Equal(1))
			g.Expect(result.Truncated).Should(gomega.BeTrue())

			_, err = a.Query(ctx, "SELECT * FROM no_such_table", 10)
			g.Expect(errors.Is(err, ErrQueryFailed)).Should(gomega.BeTrue())
		})
	})
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
