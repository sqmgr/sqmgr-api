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
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Analytics exposes read-only reporting queries intended for site admins.
// Every query runs against the Queryable it was built with, which is normally
// a READ ONLY transaction opened by RunReadOnly so that nothing executed here
// can ever modify data.
type Analytics struct {
	q Queryable
}

// NewAnalytics returns an Analytics bound to the given Queryable.
func NewAnalytics(q Queryable) *Analytics {
	return &Analytics{q: q}
}

// ErrPoolNotFound is returned when a pool token does not match any pool.
var ErrPoolNotFound = errors.New("model: pool not found")

// RunReadOnly opens a READ ONLY transaction, applies a statement timeout, and
// runs fn with an Analytics bound to that transaction. The transaction is
// always rolled back; PostgreSQL rejects any write attempted inside it.
func RunReadOnly(ctx context.Context, db *sql.DB, statementTimeout time.Duration, fn func(*Analytics) error) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("beginning read-only transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if statementTimeout > 0 {
		// SET does not accept bind parameters; the value is a formatted integer.
		stmt := fmt.Sprintf("SET LOCAL statement_timeout = %d", statementTimeout.Milliseconds())
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("setting statement timeout: %w", err)
		}
	}

	return fn(NewAnalytics(tx))
}

// DateRange bounds a timestamp column. Start is inclusive and End is
// exclusive; a zero value on either side leaves that side unbounded. Times
// should be UTC, matching the UTC wall-clock timestamps stored in the
// database.
type DateRange struct {
	Start time.Time
	End   time.Time
}

// IsZero reports whether the range is unbounded on both sides.
func (r DateRange) IsZero() bool {
	return r.Start.IsZero() && r.End.IsZero()
}

// queryArgs accumulates positional bind arguments and hands back the
// placeholder for each one.
type queryArgs []interface{}

func (a *queryArgs) add(v interface{}) string {
	*a = append(*a, v)
	return "$" + strconv.Itoa(len(*a))
}

// conditions returns SQL conditions restricting column to the range, binding
// the boundaries into args. Column names are always supplied by this package.
func (r DateRange) conditions(column string, args *queryArgs) []string {
	var conds []string
	if !r.Start.IsZero() {
		conds = append(conds, column+" >= "+args.add(r.Start))
	}
	if !r.End.IsZero() {
		conds = append(conds, column+" < "+args.add(r.End))
	}
	return conds
}

// whereClause renders conditions as a " WHERE ..." clause, or "" when empty.
func whereClause(conds []string) string {
	if len(conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(conds, " AND ")
}

// clampLimit applies a default and maximum to a caller-supplied limit.
func clampLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

// sortDirection normalizes a sort direction to ASC or DESC.
func sortDirection(dir, def string) string {
	switch strings.ToLower(dir) {
	case "asc":
		return "ASC"
	case "desc":
		return "DESC"
	}
	return def
}

// SiteStats holds site-wide counts, each restricted to a date range on the
// row's created timestamp (modified for squares, which is when they were
// last claimed or updated).
type SiteStats struct {
	Pools           int64 `json:"pools"`
	ActivePools     int64 `json:"activePools"`
	ArchivedPools   int64 `json:"archivedPools"`
	RegisteredUsers int64 `json:"registeredUsers"`
	GuestUsers      int64 `json:"guestUsers"`
	SiteAdmins      int64 `json:"siteAdmins"`
	Grids           int64 `json:"grids"`
	ClaimedSquares  int64 `json:"claimedSquares"`
	PoolMemberships int64 `json:"poolMemberships"`
}

// SiteStats returns site-wide counts for the given range.
func (a *Analytics) SiteStats(ctx context.Context, r DateRange) (*SiteStats, error) {
	stats := &SiteStats{}

	counts := []struct {
		label      string
		table      string
		timeColumn string
		filters    []string
		dests      []*int64
	}{
		{"pools", "pools", "created", []string{"TRUE", "archived = false", "archived = true"}, []*int64{&stats.Pools, &stats.ActivePools, &stats.ArchivedPools}},
		{"users", "users", "created", []string{"store = 'auth0'", "store = 'sqmgr'", "is_site_admin = true"}, []*int64{&stats.RegisteredUsers, &stats.GuestUsers, &stats.SiteAdmins}},
		{"grids", "grids", "created", []string{"state = 'active'"}, []*int64{&stats.Grids}},
		{"claimed squares", "pool_squares", "modified", []string{"state != 'unclaimed'"}, []*int64{&stats.ClaimedSquares}},
		{"pool memberships", "pools_users", "created", []string{"TRUE"}, []*int64{&stats.PoolMemberships}},
	}

	for _, c := range counts {
		var args queryArgs
		selects := make([]string, len(c.filters))
		for i, f := range c.filters {
			selects[i] = "COUNT(*) FILTER (WHERE " + f + ")"
		}
		query := "SELECT " + strings.Join(selects, ", ") + " FROM " + c.table + whereClause(r.conditions(c.timeColumn, &args))

		dests := make([]interface{}, len(c.dests))
		for i, d := range c.dests {
			dests[i] = d
		}
		if err := a.q.QueryRowContext(ctx, query, args...).Scan(dests...); err != nil {
			return nil, fmt.Errorf("counting %s: %w", c.label, err)
		}
	}

	return stats, nil
}

// Metrics that can be charted over time with TimeSeries.
const (
	MetricPoolsCreated      = "pools_created"
	MetricUsersRegistered   = "users_registered"
	MetricGuestUsersCreated = "guest_users_created"
	MetricSquaresClaimed    = "squares_claimed"
	MetricGridsCreated      = "grids_created"
	MetricPoolMembersJoined = "pool_members_joined"
)

// TimeSeriesMetrics lists every metric supported by TimeSeries.
var TimeSeriesMetrics = []string{
	MetricPoolsCreated,
	MetricUsersRegistered,
	MetricGuestUsersCreated,
	MetricSquaresClaimed,
	MetricGridsCreated,
	MetricPoolMembersJoined,
}

// Intervals that TimeSeries can bucket by.
const (
	IntervalDay   = "day"
	IntervalWeek  = "week"
	IntervalMonth = "month"
	IntervalYear  = "year"
)

// TimeSeriesIntervals lists every interval supported by TimeSeries.
var TimeSeriesIntervals = []string{IntervalDay, IntervalWeek, IntervalMonth, IntervalYear}

type timeSeriesSource struct {
	from       string
	timeColumn string
	condition  string
}

var timeSeriesSources = map[string]timeSeriesSource{
	MetricPoolsCreated:      {from: "pools", timeColumn: "created"},
	MetricUsersRegistered:   {from: "users", timeColumn: "created", condition: "store = 'auth0'"},
	MetricGuestUsersCreated: {from: "users", timeColumn: "created", condition: "store = 'sqmgr'"},
	MetricSquaresClaimed:    {from: "pool_squares_logs", timeColumn: "created", condition: "state = 'claimed'"},
	MetricGridsCreated:      {from: "grids", timeColumn: "created"},
	MetricPoolMembersJoined: {from: "pools_users", timeColumn: "created"},
}

// TimeSeriesPoint is one bucket of a time series. Period is the first day of
// the bucket formatted as YYYY-MM-DD.
type TimeSeriesPoint struct {
	Period string `json:"period"`
	Count  int64  `json:"count"`
}

// TimeSeries counts a metric per interval within the range. Buckets with no
// activity between the first and last bucket (or the range bounds, when set)
// are filled in with a zero count so the result can be charted directly.
func (a *Analytics) TimeSeries(ctx context.Context, metric, interval string, r DateRange) ([]TimeSeriesPoint, error) {
	src, ok := timeSeriesSources[metric]
	if !ok {
		return nil, fmt.Errorf("unknown metric %q; expected one of %s", metric, strings.Join(TimeSeriesMetrics, ", "))
	}
	if !isValidInterval(interval) {
		return nil, fmt.Errorf("unknown interval %q; expected one of %s", interval, strings.Join(TimeSeriesIntervals, ", "))
	}

	var args queryArgs
	conds := r.conditions(src.timeColumn, &args)
	if src.condition != "" {
		conds = append(conds, src.condition)
	}

	// interval was validated against the allowlist above and is inlined; it
	// is a literal, not user-controlled SQL.
	query := "SELECT date_trunc('" + interval + "', " + src.timeColumn + ") AS period, COUNT(*) FROM " + src.from +
		whereClause(conds) + " GROUP BY 1 ORDER BY 1"

	rows, err := a.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying %s time series: %w", metric, err)
	}
	defer rows.Close()

	type bucket struct {
		period time.Time
		count  int64
	}
	var buckets []bucket
	for rows.Next() {
		var b bucket
		if err := rows.Scan(&b.period, &b.count); err != nil {
			return nil, fmt.Errorf("scanning time series row: %w", err)
		}
		// The column is a zoneless UTC timestamp; pin the wall-clock date to
		// UTC so it keys consistently with the range bounds regardless of the
		// location the driver attached.
		y, m, d := b.period.Date()
		b.period = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		buckets = append(buckets, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading time series rows: %w", err)
	}

	counts := make(map[time.Time]int64, len(buckets))
	for _, b := range buckets {
		counts[b.period] = b.count
	}

	periods := periodsToFill(buckets, interval, r, func(i int) time.Time { return buckets[i].period })
	points := make([]TimeSeriesPoint, 0, len(periods))
	for _, p := range periods {
		points = append(points, TimeSeriesPoint{Period: p.Format("2006-01-02"), Count: counts[p]})
	}

	return points, nil
}

func isValidInterval(interval string) bool {
	for _, i := range TimeSeriesIntervals {
		if i == interval {
			return true
		}
	}
	return false
}

// maxFilledPeriods caps zero-filling so an unbounded daily series over many
// years cannot balloon the response. Above the cap only observed buckets are
// returned.
const maxFilledPeriods = 5000

// periodsToFill returns every period from the first bucket (or the truncated
// range start) through the last bucket (or the last period inside the range
// end). The buckets slice is only used for its length; period reads the i'th
// bucket's period.
func periodsToFill[T any](buckets []T, interval string, r DateRange, period func(int) time.Time) []time.Time {
	var lo, hi time.Time
	if len(buckets) > 0 {
		lo = period(0)
		hi = period(len(buckets) - 1)
	}
	if !r.Start.IsZero() {
		start := truncateToInterval(r.Start, interval)
		if lo.IsZero() || start.Before(lo) {
			lo = start
		}
	}
	if !r.End.IsZero() {
		// End is exclusive, so the last period is the one containing the
		// instant just before it. Timestamps are stored at microsecond
		// precision.
		end := truncateToInterval(r.End.Add(-time.Microsecond), interval)
		if hi.IsZero() || end.After(hi) {
			hi = end
		}
	}
	if lo.IsZero() || hi.IsZero() {
		return nil
	}

	var periods []time.Time
	for p := lo; !p.After(hi); p = nextPeriod(p, interval) {
		periods = append(periods, p)
		if len(periods) > maxFilledPeriods {
			observed := make([]time.Time, len(buckets))
			for i := range buckets {
				observed[i] = period(i)
			}
			return observed
		}
	}
	return periods
}

// truncateToInterval mirrors PostgreSQL's date_trunc for the supported
// intervals (weeks start on Monday) while preserving the time's location.
func truncateToInterval(t time.Time, interval string) time.Time {
	y, m, d := t.Date()
	loc := t.Location()
	switch interval {
	case IntervalWeek:
		day := time.Date(y, m, d, 0, 0, 0, 0, loc)
		return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
	case IntervalMonth:
		return time.Date(y, m, 1, 0, 0, 0, 0, loc)
	case IntervalYear:
		return time.Date(y, 1, 1, 0, 0, 0, 0, loc)
	default:
		return time.Date(y, m, d, 0, 0, 0, 0, loc)
	}
}

func nextPeriod(t time.Time, interval string) time.Time {
	switch interval {
	case IntervalWeek:
		return t.AddDate(0, 0, 7)
	case IntervalMonth:
		return t.AddDate(0, 1, 0)
	case IntervalYear:
		return t.AddDate(1, 0, 0)
	default:
		return t.AddDate(0, 0, 1)
	}
}

// PoolFillRateFilter restricts which pools PoolFillRates summarizes.
type PoolFillRateFilter struct {
	Created  DateRange
	GridType GridType
	Archived *bool
}

// PoolFillRates summarizes how full pools are: how many are completely
// claimed, untouched, or somewhere in between.
type PoolFillRates struct {
	Pools             int64   `json:"pools"`
	FullPools         int64   `json:"fullPools"`
	PartialPools      int64   `json:"partialPools"`
	EmptyPools        int64   `json:"emptyPools"`
	FullPoolPercent   float64 `json:"fullPoolPercent"`
	AverageFillPct    float64 `json:"averageFillPercent"`
	MedianFillPct     float64 `json:"medianFillPercent"`
	TotalSquares      int64   `json:"totalSquares"`
	ClaimedSquares    int64   `json:"claimedSquares"`
	PoolsUnder25Pct   int64   `json:"poolsUnder25Percent"`
	Pools25To49Pct    int64   `json:"pools25To49Percent"`
	Pools50To74Pct    int64   `json:"pools50To74Percent"`
	Pools75To99Pct    int64   `json:"pools75To99Percent"`
	PoolsWithNoClaims int64   `json:"poolsWithNoClaims"`
}

// PoolFillRates computes fill statistics across the pools matching the filter.
func (a *Analytics) PoolFillRates(ctx context.Context, f PoolFillRateFilter) (*PoolFillRates, error) {
	var args queryArgs
	conds := f.Created.conditions("p.created", &args)
	if f.GridType != "" {
		conds = append(conds, "p.grid_type = "+args.add(string(f.GridType)))
	}
	if f.Archived != nil {
		conds = append(conds, "p.archived = "+args.add(*f.Archived))
	}

	query := `
		WITH pool_fill AS (
			SELECT
				p.id,
				COUNT(ps.id) AS total,
				COUNT(ps.id) FILTER (WHERE ps.state != 'unclaimed') AS claimed
			FROM pools p
			LEFT JOIN pool_squares ps ON ps.pool_id = p.id` + whereClause(conds) + `
			GROUP BY p.id
		), fill_pct AS (
			SELECT total, claimed,
				CASE WHEN total > 0 THEN claimed * 100.0 / total ELSE 0 END AS pct
			FROM pool_fill
		)
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE total > 0 AND claimed = total),
			COUNT(*) FILTER (WHERE claimed > 0 AND claimed < total),
			COUNT(*) FILTER (WHERE claimed = 0),
			COALESCE(AVG(pct), 0),
			COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY pct::double precision), 0),
			COALESCE(SUM(total), 0),
			COALESCE(SUM(claimed), 0),
			COUNT(*) FILTER (WHERE pct > 0 AND pct < 25),
			COUNT(*) FILTER (WHERE pct >= 25 AND pct < 50),
			COUNT(*) FILTER (WHERE pct >= 50 AND pct < 75),
			COUNT(*) FILTER (WHERE pct >= 75 AND pct < 100)
		FROM fill_pct`

	rates := &PoolFillRates{}
	if err := a.q.QueryRowContext(ctx, query, args...).Scan(
		&rates.Pools, &rates.FullPools, &rates.PartialPools, &rates.EmptyPools,
		&rates.AverageFillPct, &rates.MedianFillPct, &rates.TotalSquares, &rates.ClaimedSquares,
		&rates.PoolsUnder25Pct, &rates.Pools25To49Pct, &rates.Pools50To74Pct, &rates.Pools75To99Pct,
	); err != nil {
		return nil, fmt.Errorf("computing pool fill rates: %w", err)
	}
	rates.PoolsWithNoClaims = rates.EmptyPools
	if rates.Pools > 0 {
		rates.FullPoolPercent = float64(rates.FullPools) * 100 / float64(rates.Pools)
	}

	return rates, nil
}

// BreakdownRow is one group of a PoolBreakdown.
type BreakdownRow struct {
	Value   string  `json:"value"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

type breakdownDimension struct {
	from       string
	expr       string
	timeColumn string
	condition  string
	describe   string
}

var breakdownDimensions = map[string]breakdownDimension{
	"grid_type":           {from: "pools p", expr: "p.grid_type", timeColumn: "p.created", describe: "pools by grid type"},
	"number_set_config":   {from: "pools p", expr: "p.number_set_config::text", timeColumn: "p.created", describe: "pools by number set configuration"},
	"archived":            {from: "pools p", expr: "p.archived::text", timeColumn: "p.created", describe: "pools by archived flag"},
	"password_required":   {from: "pools p", expr: "p.password_required::text", timeColumn: "p.created", describe: "pools by whether a password is required to join"},
	"open_access_on_lock": {from: "pools p", expr: "p.open_access_on_lock::text", timeColumn: "p.created", describe: "pools by whether they open up once locked"},
	"owner_store":         {from: "pools p INNER JOIN users u ON u.id = p.user_id", expr: "u.store::text", timeColumn: "p.created", describe: "pools by the owner's account type"},
	"league":              {from: "grids g LEFT JOIN sports_events e ON e.id = g.sports_event_id", expr: "COALESCE(e.league::text, 'none')", timeColumn: "g.created", condition: "g.state = 'active'", describe: "active grids by the league of their linked sports event ('none' when unlinked)"},
	"square_state":        {from: "pool_squares ps", expr: "ps.state::text", timeColumn: "ps.modified", describe: "squares by claim state (range applies to when the square was last modified)"},
}

// BreakdownDimensions lists the supported dimensions with a description of
// what each one counts, for documentation.
func BreakdownDimensions() map[string]string {
	out := make(map[string]string, len(breakdownDimensions))
	for k, v := range breakdownDimensions {
		out[k] = v.describe
	}
	return out
}

// PoolBreakdown groups records by a dimension and returns the count and
// percentage of the total in each group.
func (a *Analytics) PoolBreakdown(ctx context.Context, dimension string, r DateRange) ([]BreakdownRow, error) {
	dim, ok := breakdownDimensions[dimension]
	if !ok {
		names := make([]string, 0, len(breakdownDimensions))
		for k := range breakdownDimensions {
			names = append(names, k)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("unknown dimension %q; expected one of %s", dimension, strings.Join(names, ", "))
	}

	var args queryArgs
	conds := r.conditions(dim.timeColumn, &args)
	if dim.condition != "" {
		conds = append(conds, dim.condition)
	}

	query := "SELECT " + dim.expr + " AS value, COUNT(*) AS count FROM " + dim.from + whereClause(conds) +
		" GROUP BY 1 ORDER BY count DESC, value"

	rows, err := a.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying %s breakdown: %w", dimension, err)
	}
	defer rows.Close()

	result := make([]BreakdownRow, 0)
	var total int64
	for rows.Next() {
		var row BreakdownRow
		if err := rows.Scan(&row.Value, &row.Count); err != nil {
			return nil, fmt.Errorf("scanning breakdown row: %w", err)
		}
		total += row.Count
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading breakdown rows: %w", err)
	}

	if total > 0 {
		for i := range result {
			result[i].Percent = float64(result[i].Count) * 100 / float64(total)
		}
	}

	return result, nil
}

// EngagementSummary describes how users engaged with the site within a range.
type EngagementSummary struct {
	RegisteredUsersCreated int64   `json:"registeredUsersCreated"`
	GuestUsersCreated      int64   `json:"guestUsersCreated"`
	PoolsCreated           int64   `json:"poolsCreated"`
	PoolCreators           int64   `json:"poolCreators"`
	RepeatPoolCreators     int64   `json:"repeatPoolCreators"`
	ReturningPoolCreators  int64   `json:"returningPoolCreators"`
	PoolsWithAnyClaims     int64   `json:"poolsWithAnyClaims"`
	PoolsWithSportsEvent   int64   `json:"poolsWithSportsEvent"`
	AvgMembersPerPool      float64 `json:"avgMembersPerPool"`
	AvgGridsPerPool        float64 `json:"avgGridsPerPool"`
	ClaimEvents            int64   `json:"claimEvents"`
	ClaimEventsRegistered  int64   `json:"claimEventsByRegisteredUsers"`
	ClaimEventsGuest       int64   `json:"claimEventsByGuestUsers"`
	ClaimEventsAnonymous   int64   `json:"claimEventsAnonymous"`
	DistinctClaimers       int64   `json:"distinctClaimers"`
}

// EngagementSummary computes engagement figures for the range. The range
// applies to when each thing happened: user sign-ups, pool creation, and
// square claims respectively. ReturningPoolCreators is only meaningful when
// the range has a start and counts creators who also created a pool before it.
func (a *Analytics) EngagementSummary(ctx context.Context, r DateRange) (*EngagementSummary, error) {
	s := &EngagementSummary{}

	var args queryArgs
	query := "SELECT COUNT(*) FILTER (WHERE store = 'auth0'), COUNT(*) FILTER (WHERE store = 'sqmgr') FROM users" +
		whereClause(r.conditions("created", &args))
	if err := a.q.QueryRowContext(ctx, query, args...).Scan(&s.RegisteredUsersCreated, &s.GuestUsersCreated); err != nil {
		return nil, fmt.Errorf("counting new users: %w", err)
	}

	args = nil
	query = `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE n >= 2), COALESCE(SUM(n), 0)
		FROM (SELECT user_id, COUNT(*) AS n FROM pools` + whereClause(r.conditions("created", &args)) + ` GROUP BY user_id) c`
	if err := a.q.QueryRowContext(ctx, query, args...).Scan(&s.PoolCreators, &s.RepeatPoolCreators, &s.PoolsCreated); err != nil {
		return nil, fmt.Errorf("counting pool creators: %w", err)
	}

	if !r.Start.IsZero() {
		args = nil
		conds := r.conditions("p.created", &args)
		conds = append(conds, "EXISTS (SELECT 1 FROM pools o WHERE o.user_id = p.user_id AND o.created < "+args.add(r.Start)+")")
		query = "SELECT COUNT(DISTINCT p.user_id) FROM pools p" + whereClause(conds)
		if err := a.q.QueryRowContext(ctx, query, args...).Scan(&s.ReturningPoolCreators); err != nil {
			return nil, fmt.Errorf("counting returning pool creators: %w", err)
		}
	}

	args = nil
	query = `
		SELECT
			COALESCE(AVG(members), 0),
			COALESCE(AVG(grids), 0),
			COUNT(*) FILTER (WHERE linked > 0),
			COUNT(*) FILTER (WHERE claimed > 0)
		FROM (
			SELECT
				(SELECT COUNT(*) FROM pools_users pu WHERE pu.pool_id = p.id) AS members,
				(SELECT COUNT(*) FROM grids g WHERE g.pool_id = p.id AND g.state = 'active') AS grids,
				(SELECT COUNT(*) FROM grids g WHERE g.pool_id = p.id AND g.state = 'active' AND g.sports_event_id IS NOT NULL) AS linked,
				(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state != 'unclaimed') AS claimed
			FROM pools p` + whereClause(r.conditions("p.created", &args)) + `
		) s`
	if err := a.q.QueryRowContext(ctx, query, args...).Scan(&s.AvgMembersPerPool, &s.AvgGridsPerPool, &s.PoolsWithSportsEvent, &s.PoolsWithAnyClaims); err != nil {
		return nil, fmt.Errorf("summarizing pools: %w", err)
	}

	args = nil
	conds := r.conditions("l.created", &args)
	conds = append(conds, "l.state = 'claimed'")
	query = `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE u.store = 'auth0'),
			COUNT(*) FILTER (WHERE u.store = 'sqmgr'),
			COUNT(*) FILTER (WHERE l.user_id IS NULL),
			COUNT(DISTINCT l.user_id)
		FROM pool_squares_logs l
		LEFT JOIN users u ON u.id = l.user_id` + whereClause(conds)
	if err := a.q.QueryRowContext(ctx, query, args...).Scan(&s.ClaimEvents, &s.ClaimEventsRegistered, &s.ClaimEventsGuest, &s.ClaimEventsAnonymous, &s.DistinctClaimers); err != nil {
		return nil, fmt.Errorf("counting claims: %w", err)
	}

	return s, nil
}

// AnalyticsPool is a pool row enriched with membership and fill information.
type AnalyticsPool struct {
	Token           string          `json:"token"`
	Name            string          `json:"name"`
	GridType        GridType        `json:"gridType"`
	NumberSetConfig NumberSetConfig `json:"numberSetConfig"`
	Archived        bool            `json:"archived"`
	OwnerID         int64           `json:"ownerId"`
	OwnerEmail      *string         `json:"ownerEmail"`
	OwnerStore      string          `json:"ownerStore"`
	Locks           *time.Time      `json:"locks"`
	Created         time.Time       `json:"created"`
	MemberCount     int64           `json:"memberCount"`
	GridCount       int64           `json:"gridCount"`
	TotalSquares    int64           `json:"totalSquares"`
	ClaimedSquares  int64           `json:"claimedSquares"`
	FillPercent     float64         `json:"fillPercent"`
}

// PoolListFilter restricts and orders the pools returned by ListPools.
type PoolListFilter struct {
	// Search matches a substring of the pool name or the exact pool token.
	Search         string
	OwnerEmail     string
	Created        DateRange
	GridType       GridType
	Archived       *bool
	MinFillPercent *float64
	MaxFillPercent *float64
	SortBy         string
	SortDir        string
	Limit          int
	Offset         int64
}

// PoolList is a page of pools plus the total number matching the filter.
type PoolList struct {
	Pools []*AnalyticsPool `json:"pools"`
	Total int64            `json:"total"`
}

// Limits for list queries.
const (
	DefaultAnalyticsLimit = 25
	MaxAnalyticsLimit     = 500
)

// poolStatsQuery selects AnalyticsPool columns for pools matching the WHERE
// clause substituted for %s. Callers complete it with a FROM pool_fill clause
// and may filter and sort by any of the aliased columns.
const poolStatsQuery = `
	WITH pool_stats AS (
		SELECT
			p.id,
			p.token,
			p.name,
			p.grid_type,
			p.number_set_config,
			p.archived,
			p.user_id,
			u.email AS owner_email,
			COALESCE(u.store::text, '') AS owner_store,
			p.locks,
			p.created,
			(SELECT COUNT(*) FROM pools_users pu WHERE pu.pool_id = p.id) AS member_count,
			(SELECT COUNT(*) FROM grids g WHERE g.pool_id = p.id AND g.state = 'active') AS grid_count,
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id) AS total_squares,
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state != 'unclaimed') AS claimed_squares
		FROM pools p
		LEFT JOIN users u ON u.id = p.user_id%s
	), pool_fill AS (
		SELECT *, CASE WHEN total_squares > 0 THEN claimed_squares * 100.0 / total_squares ELSE 0 END AS fill_percent
		FROM pool_stats
	)
	SELECT
		id, token, name, grid_type, number_set_config, archived, user_id, owner_email, owner_store, locks, created,
		member_count, grid_count, total_squares, claimed_squares, fill_percent`

func scanAnalyticsPool(scan scanFunc, extra ...interface{}) (int64, *AnalyticsPool, error) {
	var id int64
	p := &AnalyticsPool{}
	dests := []interface{}{
		&id, &p.Token, &p.Name, &p.GridType, &p.NumberSetConfig, &p.Archived, &p.OwnerID, &p.OwnerEmail, &p.OwnerStore, &p.Locks, &p.Created,
		&p.MemberCount, &p.GridCount, &p.TotalSquares, &p.ClaimedSquares, &p.FillPercent,
	}
	dests = append(dests, extra...)
	if err := scan(dests...); err != nil {
		return 0, nil, err
	}
	return id, p, nil
}

var poolSortColumns = map[string]string{
	"created":         "created",
	"name":            "name",
	"member_count":    "member_count",
	"grid_count":      "grid_count",
	"total_squares":   "total_squares",
	"claimed_squares": "claimed_squares",
	"fill_percent":    "fill_percent",
}

// ListPools returns a page of pools matching the filter, newest first by
// default.
func (a *Analytics) ListPools(ctx context.Context, f PoolListFilter) (*PoolList, error) {
	var args queryArgs
	conds := f.Created.conditions("p.created", &args)
	if f.Search != "" {
		conds = append(conds, "(p.name ILIKE "+args.add("%"+f.Search+"%")+" OR p.token = "+args.add(f.Search)+")")
	}
	if f.OwnerEmail != "" {
		conds = append(conds, "u.email ILIKE "+args.add(f.OwnerEmail))
	}
	if f.GridType != "" {
		conds = append(conds, "p.grid_type = "+args.add(string(f.GridType)))
	}
	if f.Archived != nil {
		conds = append(conds, "p.archived = "+args.add(*f.Archived))
	}

	var outer []string
	if f.MinFillPercent != nil {
		outer = append(outer, "fill_percent >= "+args.add(*f.MinFillPercent))
	}
	if f.MaxFillPercent != nil {
		outer = append(outer, "fill_percent <= "+args.add(*f.MaxFillPercent))
	}

	orderColumn := "created"
	if col, ok := poolSortColumns[f.SortBy]; ok {
		orderColumn = col
	}
	orderDir := sortDirection(f.SortDir, "DESC")

	limit := clampLimit(f.Limit, DefaultAnalyticsLimit, MaxAnalyticsLimit)
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(poolStatsQuery, whereClause(conds)) + `, COUNT(*) OVER() AS total
	FROM pool_fill` + whereClause(outer) + `
	ORDER BY ` + orderColumn + ` ` + orderDir + `, id DESC
	OFFSET ` + args.add(offset) + ` LIMIT ` + args.add(limit)

	rows, err := a.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying pools: %w", err)
	}
	defer rows.Close()

	list := &PoolList{Pools: make([]*AnalyticsPool, 0)}
	for rows.Next() {
		var total int64
		_, p, err := scanAnalyticsPool(rows.Scan, &total)
		if err != nil {
			return nil, fmt.Errorf("scanning pool row: %w", err)
		}
		list.Total = total
		list.Pools = append(list.Pools, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading pool rows: %w", err)
	}

	return list, nil
}

// SquareStateCounts is the number of squares in a pool in each state.
type SquareStateCounts struct {
	Unclaimed   int64 `json:"unclaimed"`
	Claimed     int64 `json:"claimed"`
	PaidPartial int64 `json:"paidPartial"`
	PaidFull    int64 `json:"paidFull"`
}

// AnalyticsGrid is a grid within a pool, with its linked sports event if any.
type AnalyticsGrid struct {
	ID              int64      `json:"id"`
	Label           *string    `json:"label"`
	HomeTeamName    *string    `json:"homeTeamName"`
	AwayTeamName    *string    `json:"awayTeamName"`
	State           State      `json:"state"`
	EventDate       *time.Time `json:"eventDate"`
	ManualDraw      bool       `json:"manualDraw"`
	Rollover        bool       `json:"rollover"`
	Created         time.Time  `json:"created"`
	SportsEventID   *int64     `json:"sportsEventId"`
	SportsEventName *string    `json:"sportsEventName"`
	League          *string    `json:"league"`
}

// PoolDetails is everything an admin might want to know about one pool.
type PoolDetails struct {
	*AnalyticsPool
	PasswordRequired bool              `json:"passwordRequired"`
	OpenAccessOnLock bool              `json:"openAccessOnLock"`
	Modified         time.Time         `json:"modified"`
	SquareStates     SquareStateCounts `json:"squareStates"`
	ActiveInvites    int64             `json:"activeInvites"`
	Grids            []*AnalyticsGrid  `json:"grids"`
}

// PoolDetails returns details for the pool with the given token.
func (a *Analytics) PoolDetails(ctx context.Context, token string) (*PoolDetails, error) {
	query := fmt.Sprintf(poolStatsQuery, " WHERE p.token = $1") + `
	FROM pool_fill`
	id, pool, err := scanAnalyticsPool(a.q.QueryRowContext(ctx, query, token).Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPoolNotFound
		}
		return nil, fmt.Errorf("querying pool: %w", err)
	}

	d := &PoolDetails{AnalyticsPool: pool, Grids: make([]*AnalyticsGrid, 0)}

	if err := a.q.QueryRowContext(ctx, `
		SELECT
			p.password_required,
			p.open_access_on_lock,
			p.modified,
			(SELECT COUNT(*) FROM pool_invites i WHERE i.pool_id = p.id AND i.expires_at > (NOW() AT TIME ZONE 'utc')),
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state = 'unclaimed'),
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state = 'claimed'),
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state = 'paid-partial'),
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state = 'paid-full')
		FROM pools p
		WHERE p.id = $1`, id).Scan(
		&d.PasswordRequired, &d.OpenAccessOnLock, &d.Modified, &d.ActiveInvites,
		&d.SquareStates.Unclaimed, &d.SquareStates.Claimed, &d.SquareStates.PaidPartial, &d.SquareStates.PaidFull,
	); err != nil {
		return nil, fmt.Errorf("querying pool details: %w", err)
	}

	rows, err := a.q.QueryContext(ctx, `
		SELECT
			g.id, g.label, g.home_team_name, g.away_team_name, g.state, g.event_date, g.manual_draw, g.rollover, g.created,
			g.sports_event_id, e.name, e.league::text
		FROM grids g
		LEFT JOIN sports_events e ON e.id = g.sports_event_id
		WHERE g.pool_id = $1
		ORDER BY g.ord, g.id`, id)
	if err != nil {
		return nil, fmt.Errorf("querying pool grids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		g := &AnalyticsGrid{}
		if err := rows.Scan(
			&g.ID, &g.Label, &g.HomeTeamName, &g.AwayTeamName, &g.State, &g.EventDate, &g.ManualDraw, &g.Rollover, &g.Created,
			&g.SportsEventID, &g.SportsEventName, &g.League,
		); err != nil {
			return nil, fmt.Errorf("scanning grid row: %w", err)
		}
		d.Grids = append(d.Grids, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading grid rows: %w", err)
	}

	return d, nil
}

// poolIDByToken resolves a token to the pool's ID and owner ID.
func (a *Analytics) poolIDByToken(ctx context.Context, token string) (id, ownerID int64, err error) {
	err = a.q.QueryRowContext(ctx, "SELECT id, user_id FROM pools WHERE token = $1", token).Scan(&id, &ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrPoolNotFound
	}
	if err != nil {
		return 0, 0, fmt.Errorf("looking up pool: %w", err)
	}
	return id, ownerID, nil
}

// AnalyticsPoolSquare is one square of a pool along with who claimed it.
type AnalyticsPoolSquare struct {
	SquareID  int             `json:"squareId"`
	State     PoolSquareState `json:"state"`
	Claimant  *string         `json:"claimant"`
	UserID    *int64          `json:"userId"`
	UserStore *string         `json:"userStore"`
	UserEmail *string         `json:"userEmail"`
	Modified  time.Time       `json:"modified"`
}

// PoolSquares lists a pool's squares with the claimant and, when the claim
// was made by a registered user, their email address. Unclaimed squares are
// omitted unless includeUnclaimed is set.
func (a *Analytics) PoolSquares(ctx context.Context, token string, includeUnclaimed bool) ([]*AnalyticsPoolSquare, error) {
	poolID, _, err := a.poolIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT ps.square_id, ps.state, ps.claimant, ps.user_id, u.store::text, u.email, ps.modified
		FROM pool_squares ps
		LEFT JOIN users u ON u.id = ps.user_id
		WHERE ps.pool_id = $1`
	if !includeUnclaimed {
		query += " AND ps.state != 'unclaimed'"
	}
	query += " ORDER BY ps.square_id"

	rows, err := a.q.QueryContext(ctx, query, poolID)
	if err != nil {
		return nil, fmt.Errorf("querying pool squares: %w", err)
	}
	defer rows.Close()

	squares := make([]*AnalyticsPoolSquare, 0)
	for rows.Next() {
		s := &AnalyticsPoolSquare{}
		if err := rows.Scan(&s.SquareID, &s.State, &s.Claimant, &s.UserID, &s.UserStore, &s.UserEmail, &s.Modified); err != nil {
			return nil, fmt.Errorf("scanning pool square row: %w", err)
		}
		squares = append(squares, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading pool square rows: %w", err)
	}

	return squares, nil
}

// PoolMember is a user with access to a pool.
type PoolMember struct {
	UserID         int64      `json:"userId"`
	Store          UserStore  `json:"store"`
	Email          *string    `json:"email"`
	IsOwner        bool       `json:"isOwner"`
	IsManager      bool       `json:"isManager"`
	Joined         *time.Time `json:"joined"`
	SquaresClaimed int64      `json:"squaresClaimed"`
}

// PoolMembers lists everyone with access to a pool, owner first. The owner
// normally has no membership row, so they are included explicitly.
func (a *Analytics) PoolMembers(ctx context.Context, token string) ([]*PoolMember, error) {
	poolID, ownerID, err := a.poolIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	const query = `
		SELECT
			u.id, u.store, u.email, u.id = $1 AS is_owner, m.is_manager, m.joined,
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = $2 AND ps.user_id = u.id AND ps.state != 'unclaimed') AS squares_claimed
		FROM (
			SELECT pu.user_id, pu.is_manager, pu.created AS joined FROM pools_users pu WHERE pu.pool_id = $2
			UNION ALL
			SELECT $1::bigint, true, NULL::timestamp
			WHERE NOT EXISTS (SELECT 1 FROM pools_users pu WHERE pu.pool_id = $2 AND pu.user_id = $1)
		) m
		INNER JOIN users u ON u.id = m.user_id
		ORDER BY is_owner DESC, m.joined NULLS FIRST, u.id`

	rows, err := a.q.QueryContext(ctx, query, ownerID, poolID)
	if err != nil {
		return nil, fmt.Errorf("querying pool members: %w", err)
	}
	defer rows.Close()

	members := make([]*PoolMember, 0)
	for rows.Next() {
		m := &PoolMember{}
		if err := rows.Scan(&m.UserID, &m.Store, &m.Email, &m.IsOwner, &m.IsManager, &m.Joined, &m.SquaresClaimed); err != nil {
			return nil, fmt.Errorf("scanning pool member row: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading pool member rows: %w", err)
	}

	return members, nil
}

// PoolActivityEntry is one entry from a pool's square change log.
type PoolActivityEntry struct {
	ID         int64           `json:"id"`
	SquareID   int             `json:"squareId"`
	State      PoolSquareState `json:"state"`
	Claimant   *string         `json:"claimant"`
	UserID     *int64          `json:"userId"`
	UserStore  *string         `json:"userStore"`
	UserEmail  *string         `json:"userEmail"`
	Note       string          `json:"note"`
	RemoteAddr *string         `json:"remoteAddr"`
	Created    time.Time       `json:"created"`
}

// PoolActivity returns a page of the pool's square change log, newest first.
func (a *Analytics) PoolActivity(ctx context.Context, token string, offset int64, limit int) ([]*PoolActivityEntry, error) {
	poolID, _, err := a.poolIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if offset < 0 {
		offset = 0
	}
	limit = clampLimit(limit, DefaultAnalyticsLimit, MaxAnalyticsLimit)

	const query = `
		SELECT l.id, ps.square_id, l.state, l.claimant, l.user_id, u.store::text, u.email, l.note, l.remote_addr, l.created
		FROM pool_squares_logs l
		INNER JOIN pool_squares ps ON ps.id = l.pool_square_id
		LEFT JOIN users u ON u.id = l.user_id
		WHERE ps.pool_id = $1
		ORDER BY l.id DESC
		OFFSET $2 LIMIT $3`

	rows, err := a.q.QueryContext(ctx, query, poolID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("querying pool activity: %w", err)
	}
	defer rows.Close()

	entries := make([]*PoolActivityEntry, 0)
	for rows.Next() {
		e := &PoolActivityEntry{}
		if err := rows.Scan(&e.ID, &e.SquareID, &e.State, &e.Claimant, &e.UserID, &e.UserStore, &e.UserEmail, &e.Note, &e.RemoteAddr, &e.Created); err != nil {
			return nil, fmt.Errorf("scanning pool activity row: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading pool activity rows: %w", err)
	}

	return entries, nil
}

// PoolActivityCount returns the number of square log entries for a pool.
func (a *Analytics) PoolActivityCount(ctx context.Context, token string) (int64, error) {
	poolID, _, err := a.poolIDByToken(ctx, token)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := a.q.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM pool_squares_logs l
		INNER JOIN pool_squares ps ON ps.id = l.pool_square_id
		WHERE ps.pool_id = $1`, poolID).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting pool activity: %w", err)
	}

	return count, nil
}

// AnalyticsUser is a user row enriched with activity counts.
type AnalyticsUser struct {
	ID             int64     `json:"id"`
	Store          UserStore `json:"store"`
	Email          *string   `json:"email"`
	IsSiteAdmin    bool      `json:"isSiteAdmin"`
	Created        time.Time `json:"created"`
	PoolsOwned     int64     `json:"poolsOwned"`
	PoolsJoined    int64     `json:"poolsJoined"`
	SquaresClaimed int64     `json:"squaresClaimed"`
}

// UserListFilter restricts and orders the users returned by ListUsers.
type UserListFilter struct {
	Search  string
	Store   string
	Created DateRange
	SortBy  string
	SortDir string
	Limit   int
	Offset  int64
}

// UserList is a page of users plus the total number matching the filter.
type UserList struct {
	Users []*AnalyticsUser `json:"users"`
	Total int64            `json:"total"`
}

const userStatsColumns = `
	u.id, u.store, u.email, u.is_site_admin, u.created,
	(SELECT COUNT(*) FROM pools p WHERE p.user_id = u.id) AS pools_owned,
	(SELECT COUNT(*) FROM pools_users pu INNER JOIN pools p ON p.id = pu.pool_id WHERE pu.user_id = u.id AND p.user_id != u.id) AS pools_joined,
	(SELECT COUNT(*) FROM pool_squares ps WHERE ps.user_id = u.id AND ps.state != 'unclaimed') AS squares_claimed`

func scanAnalyticsUser(scan scanFunc, extra ...interface{}) (*AnalyticsUser, error) {
	u := &AnalyticsUser{}
	dests := []interface{}{&u.ID, &u.Store, &u.Email, &u.IsSiteAdmin, &u.Created, &u.PoolsOwned, &u.PoolsJoined, &u.SquaresClaimed}
	dests = append(dests, extra...)
	if err := scan(dests...); err != nil {
		return nil, err
	}
	return u, nil
}

var userSortColumns = map[string]string{
	"created":         "created",
	"id":              "id",
	"pools_owned":     "pools_owned",
	"pools_joined":    "pools_joined",
	"squares_claimed": "squares_claimed",
}

// ListUsers returns a page of users. Store may be "auth0" (registered users,
// the default), "sqmgr" (guest users), or "all".
func (a *Analytics) ListUsers(ctx context.Context, f UserListFilter) (*UserList, error) {
	var args queryArgs
	conds := f.Created.conditions("u.created", &args)

	switch f.Store {
	case "", string(UserStoreAuth0):
		conds = append(conds, "u.store = 'auth0'")
	case string(UserStoreSqMGR):
		conds = append(conds, "u.store = 'sqmgr'")
	case "all":
	default:
		return nil, fmt.Errorf("unknown store %q; expected auth0, sqmgr, or all", f.Store)
	}

	if f.Search != "" {
		conds = append(conds, "u.email ILIKE "+args.add("%"+f.Search+"%"))
	}

	orderColumn := "created"
	if col, ok := userSortColumns[f.SortBy]; ok {
		orderColumn = col
	}
	orderDir := sortDirection(f.SortDir, "DESC")

	limit := clampLimit(f.Limit, DefaultAnalyticsLimit, MaxAnalyticsLimit)
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	query := `SELECT * FROM (SELECT ` + userStatsColumns + `, COUNT(*) OVER() AS total FROM users u` + whereClause(conds) + `) s
		ORDER BY ` + orderColumn + ` ` + orderDir + `, id DESC
		OFFSET ` + args.add(offset) + ` LIMIT ` + args.add(limit)

	rows, err := a.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying users: %w", err)
	}
	defer rows.Close()

	list := &UserList{Users: make([]*AnalyticsUser, 0)}
	for rows.Next() {
		var total int64
		u, err := scanAnalyticsUser(rows.Scan, &total)
		if err != nil {
			return nil, fmt.Errorf("scanning user row: %w", err)
		}
		list.Total = total
		list.Users = append(list.Users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading user rows: %w", err)
	}

	return list, nil
}

// UserPoolSummary is a compact description of a pool a user owns or belongs to.
type UserPoolSummary struct {
	Token     string     `json:"token"`
	Name      string     `json:"name"`
	Archived  bool       `json:"archived"`
	Created   time.Time  `json:"created"`
	IsManager bool       `json:"isManager"`
	Joined    *time.Time `json:"joined"`
}

// UserDetails is a user's profile, activity counts, and pools.
type UserDetails struct {
	*AnalyticsUser
	StoreID            string             `json:"storeId"`
	ArchivedPoolsOwned int64              `json:"archivedPoolsOwned"`
	PoolsOwnedList     []*UserPoolSummary `json:"poolsOwnedList"`
	PoolsJoinedList    []*UserPoolSummary `json:"poolsJoinedList"`
}

// maxUserDetailPools caps how many pools UserDetails lists in each category.
const maxUserDetailPools = 50

// UserDetails looks a user up by ID (when id > 0) or by email address.
func (a *Analytics) UserDetails(ctx context.Context, id int64, email string) (*UserDetails, error) {
	var args queryArgs
	var cond string
	switch {
	case id > 0:
		cond = "u.id = " + args.add(id)
	case email != "":
		cond = "u.email ILIKE " + args.add(email)
	default:
		return nil, errors.New("either a user id or an email address is required")
	}

	query := "SELECT " + userStatsColumns + `,
		u.store_id,
		(SELECT COUNT(*) FROM pools p WHERE p.user_id = u.id AND p.archived = true)
		FROM users u WHERE ` + cond + " ORDER BY u.id LIMIT 1"

	d := &UserDetails{PoolsOwnedList: make([]*UserPoolSummary, 0), PoolsJoinedList: make([]*UserPoolSummary, 0)}
	u, err := scanAnalyticsUser(a.q.QueryRowContext(ctx, query, args...).Scan, &d.StoreID, &d.ArchivedPoolsOwned)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("querying user: %w", err)
	}
	d.AnalyticsUser = u

	d.PoolsOwnedList, err = a.userPools(ctx, `
		SELECT p.token, p.name, p.archived, p.created, true, NULL::timestamp
		FROM pools p WHERE p.user_id = $1 ORDER BY p.id DESC LIMIT $2`, u.ID)
	if err != nil {
		return nil, fmt.Errorf("querying pools owned: %w", err)
	}

	d.PoolsJoinedList, err = a.userPools(ctx, `
		SELECT p.token, p.name, p.archived, p.created, pu.is_manager, pu.created
		FROM pools_users pu INNER JOIN pools p ON p.id = pu.pool_id
		WHERE pu.user_id = $1 AND p.user_id != $1 ORDER BY pu.created DESC LIMIT $2`, u.ID)
	if err != nil {
		return nil, fmt.Errorf("querying pools joined: %w", err)
	}

	return d, nil
}

func (a *Analytics) userPools(ctx context.Context, query string, userID int64) ([]*UserPoolSummary, error) {
	rows, err := a.q.QueryContext(ctx, query, userID, maxUserDetailPools)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pools := make([]*UserPoolSummary, 0)
	for rows.Next() {
		p := &UserPoolSummary{}
		if err := rows.Scan(&p.Token, &p.Name, &p.Archived, &p.Created, &p.IsManager, &p.Joined); err != nil {
			return nil, err
		}
		pools = append(pools, p)
	}
	return pools, rows.Err()
}

// PoolCreator is a user ranked by the pools they created.
type PoolCreator struct {
	UserID         int64     `json:"userId"`
	Store          UserStore `json:"store"`
	Email          *string   `json:"email"`
	PoolsCreated   int64     `json:"poolsCreated"`
	SquaresClaimed int64     `json:"squaresClaimedInTheirPools"`
	Members        int64     `json:"membersAcrossTheirPools"`
}

// TopPoolCreators ranks users by pools created within the range.
func (a *Analytics) TopPoolCreators(ctx context.Context, r DateRange, limit int) ([]*PoolCreator, error) {
	var args queryArgs
	conds := r.conditions("p.created", &args)
	limit = clampLimit(limit, DefaultAnalyticsLimit, MaxAnalyticsLimit)

	query := `
		SELECT
			u.id, u.store, u.email,
			COUNT(p.id) AS pools_created,
			COALESCE(SUM((SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state != 'unclaimed')), 0),
			COALESCE(SUM((SELECT COUNT(*) FROM pools_users pu WHERE pu.pool_id = p.id)), 0)
		FROM pools p
		INNER JOIN users u ON u.id = p.user_id` + whereClause(conds) + `
		GROUP BY u.id
		ORDER BY pools_created DESC, u.id
		LIMIT ` + args.add(limit)

	rows, err := a.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying top pool creators: %w", err)
	}
	defer rows.Close()

	creators := make([]*PoolCreator, 0)
	for rows.Next() {
		c := &PoolCreator{}
		if err := rows.Scan(&c.UserID, &c.Store, &c.Email, &c.PoolsCreated, &c.SquaresClaimed, &c.Members); err != nil {
			return nil, fmt.Errorf("scanning pool creator row: %w", err)
		}
		creators = append(creators, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading pool creator rows: %w", err)
	}

	return creators, nil
}

// PopularEvent is a sports event ranked by how many grids are linked to it.
type PopularEvent struct {
	ID           int64     `json:"id"`
	League       string    `json:"league"`
	Name         *string   `json:"name"`
	HomeTeam     *string   `json:"homeTeam"`
	AwayTeam     *string   `json:"awayTeam"`
	EventDate    time.Time `json:"eventDate"`
	Status       string    `json:"status"`
	HomeScore    *int      `json:"homeScore"`
	AwayScore    *int      `json:"awayScore"`
	GridCount    int64     `json:"gridCount"`
	PoolCount    int64     `json:"poolCount"`
	ClaimedTotal int64     `json:"claimedSquaresInLinkedPools"`
}

// PopularEvents ranks sports events by linked active grids. The range applies
// to the event date.
func (a *Analytics) PopularEvents(ctx context.Context, league string, r DateRange, limit int) ([]*PopularEvent, error) {
	var args queryArgs
	conds := r.conditions("e.event_date", &args)
	if league != "" {
		if !IsValidSportsLeague(league) {
			return nil, fmt.Errorf("invalid league %q", league)
		}
		conds = append(conds, "e.league = "+args.add(league))
	}
	limit = clampLimit(limit, DefaultAnalyticsLimit, MaxAnalyticsLimit)

	query := `
		SELECT
			e.id, e.league::text, e.name, home.full_name, away.full_name, e.event_date, e.status, e.home_score, e.away_score,
			COUNT(g.id) AS grid_count,
			COUNT(DISTINCT g.pool_id) AS pool_count,
			COALESCE(SUM((SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = g.pool_id AND ps.state != 'unclaimed')), 0)
		FROM sports_events e
		INNER JOIN grids g ON g.sports_event_id = e.id AND g.state = 'active'
		LEFT JOIN sports_teams home ON home.id = e.home_team_id AND home.league = e.league
		LEFT JOIN sports_teams away ON away.id = e.away_team_id AND away.league = e.league` + whereClause(conds) + `
		GROUP BY e.id, home.full_name, away.full_name
		ORDER BY grid_count DESC, e.event_date DESC
		LIMIT ` + args.add(limit)

	rows, err := a.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying popular events: %w", err)
	}
	defer rows.Close()

	events := make([]*PopularEvent, 0)
	for rows.Next() {
		e := &PopularEvent{}
		if err := rows.Scan(
			&e.ID, &e.League, &e.Name, &e.HomeTeam, &e.AwayTeam, &e.EventDate, &e.Status, &e.HomeScore, &e.AwayScore,
			&e.GridCount, &e.PoolCount, &e.ClaimedTotal,
		); err != nil {
			return nil, fmt.Errorf("scanning popular event row: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading popular event rows: %w", err)
	}

	return events, nil
}

// SportsSyncRun is one run of the sports sync job.
type SportsSyncRun struct {
	ID               int64      `json:"id"`
	SyncType         string     `json:"syncType"`
	League           *string    `json:"league"`
	StartedAt        time.Time  `json:"startedAt"`
	CompletedAt      *time.Time `json:"completedAt"`
	RecordsProcessed *int       `json:"recordsProcessed"`
	ErrorMessage     *string    `json:"errorMessage"`
	Success          *bool      `json:"success"`
}

// SportsSyncRuns returns the most recent sports sync runs.
func (a *Analytics) SportsSyncRuns(ctx context.Context, limit int) ([]*SportsSyncRun, error) {
	return a.SportsSyncRunsByType(ctx, "", limit)
}

// SportsSyncRunsByType returns the most recent sports sync runs of the given
// type, newest first. An empty syncType returns every type.
func (a *Analytics) SportsSyncRunsByType(ctx context.Context, syncType string, limit int) ([]*SportsSyncRun, error) {
	limit = clampLimit(limit, DefaultAnalyticsLimit, MaxAnalyticsLimit)

	var args queryArgs
	var conds []string
	if syncType != "" {
		conds = append(conds, "sync_type = "+args.add(syncType))
	}

	query := `
		SELECT id, sync_type, league::text, started_at, completed_at, records_processed, error_message, success
		FROM sports_sync_log` + whereClause(conds) + `
		ORDER BY id DESC
		LIMIT ` + args.add(limit)

	rows, err := a.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying sports sync runs: %w", err)
	}
	defer rows.Close()

	return scanSportsSyncRuns(rows)
}

// LatestSportsSyncRuns returns the most recent run for every (sync type,
// league) pair, newest first.
func (a *Analytics) LatestSportsSyncRuns(ctx context.Context) ([]*SportsSyncRun, error) {
	rows, err := a.q.QueryContext(ctx, `
		SELECT DISTINCT ON (sync_type, league)
			id, sync_type, league::text, started_at, completed_at, records_processed, error_message, success
		FROM sports_sync_log
		ORDER BY sync_type, league, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("querying latest sports sync runs: %w", err)
	}
	defer rows.Close()

	runs, err := scanSportsSyncRuns(rows)
	if err != nil {
		return nil, err
	}

	sort.Slice(runs, func(i, j int) bool { return runs[i].ID > runs[j].ID })
	return runs, nil
}

// scanSportsSyncRuns reads rows of sync log columns into SportsSyncRun values.
func scanSportsSyncRuns(rows *sql.Rows) ([]*SportsSyncRun, error) {
	runs := make([]*SportsSyncRun, 0)
	for rows.Next() {
		run := &SportsSyncRun{}
		if err := rows.Scan(&run.ID, &run.SyncType, &run.League, &run.StartedAt, &run.CompletedAt, &run.RecordsProcessed, &run.ErrorMessage, &run.Success); err != nil {
			return nil, fmt.Errorf("scanning sports sync run row: %w", err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading sports sync run rows: %w", err)
	}

	return runs, nil
}

// SchemaColumn describes one column of a table.
type SchemaColumn struct {
	Name     string `json:"name"`
	DataType string `json:"dataType"`
	Nullable bool   `json:"nullable"`
}

// SchemaTable describes one table and its columns.
type SchemaTable struct {
	Name    string         `json:"name"`
	Columns []SchemaColumn `json:"columns"`
}

// SchemaEnum describes one enum type and its allowed values.
type SchemaEnum struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// Schema describes the public database schema.
type Schema struct {
	Tables []SchemaTable `json:"tables"`
	Enums  []SchemaEnum  `json:"enums"`
}

// Schema returns the tables, columns, and enum types in the public schema so
// callers can write their own queries.
func (a *Analytics) Schema(ctx context.Context) (*Schema, error) {
	rows, err := a.q.QueryContext(ctx, `
		SELECT c.table_name, c.column_name, c.udt_name, c.is_nullable = 'YES'
		FROM information_schema.columns c
		INNER JOIN information_schema.tables t ON t.table_schema = c.table_schema AND t.table_name = c.table_name
		WHERE c.table_schema = 'public' AND t.table_type = 'BASE TABLE' AND c.table_name != 'schema_migrations'
		ORDER BY c.table_name, c.ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("querying columns: %w", err)
	}
	defer rows.Close()

	schema := &Schema{Tables: make([]SchemaTable, 0), Enums: make([]SchemaEnum, 0)}
	for rows.Next() {
		var table string
		var col SchemaColumn
		if err := rows.Scan(&table, &col.Name, &col.DataType, &col.Nullable); err != nil {
			return nil, fmt.Errorf("scanning column row: %w", err)
		}
		if n := len(schema.Tables); n == 0 || schema.Tables[n-1].Name != table {
			schema.Tables = append(schema.Tables, SchemaTable{Name: table})
		}
		last := &schema.Tables[len(schema.Tables)-1]
		last.Columns = append(last.Columns, col)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading column rows: %w", err)
	}

	enumRows, err := a.q.QueryContext(ctx, `
		SELECT t.typname, e.enumlabel
		FROM pg_enum e
		INNER JOIN pg_type t ON t.oid = e.enumtypid
		ORDER BY t.typname, e.enumsortorder`)
	if err != nil {
		return nil, fmt.Errorf("querying enums: %w", err)
	}
	defer enumRows.Close()

	for enumRows.Next() {
		var name, value string
		if err := enumRows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("scanning enum row: %w", err)
		}
		if n := len(schema.Enums); n == 0 || schema.Enums[n-1].Name != name {
			schema.Enums = append(schema.Enums, SchemaEnum{Name: name})
		}
		last := &schema.Enums[len(schema.Enums)-1]
		last.Values = append(last.Values, value)
	}
	if err := enumRows.Err(); err != nil {
		return nil, fmt.Errorf("reading enum rows: %w", err)
	}

	return schema, nil
}

// QueryResult is the outcome of an ad hoc read-only query.
type QueryResult struct {
	Columns   []string        `json:"columns"`
	Rows      [][]interface{} `json:"rows"`
	RowCount  int             `json:"rowCount"`
	Truncated bool            `json:"truncated"`
}

// Limits for ad hoc queries.
const (
	DefaultQueryRows = 100
	MaxQueryRows     = 1000
)

// ErrInvalidQuery is returned when an ad hoc query is rejected before being run.
var ErrInvalidQuery = errors.New("model: invalid query")

// ErrQueryFailed is returned when PostgreSQL rejects an ad hoc query, for
// example because of a syntax error or an unknown table.
var ErrQueryFailed = errors.New("model: query failed")

// deniedQueryPattern matches identifiers that have no place in an analytics
// query: functions that sleep, block, signal other backends, touch the
// filesystem, or change session settings, plus the one column that must not
// be read. This is defense in depth on top of the READ ONLY transaction; the
// database role's privileges are the real boundary.
var deniedQueryPattern = regexp.MustCompile(`(?i)\b(pg_sleep|pg_sleep_for|pg_sleep_until|pg_terminate_backend|pg_cancel_backend|pg_advisory_lock|pg_advisory_xact_lock|pg_read_file|pg_read_binary_file|pg_ls_dir|pg_stat_file|lo_import|lo_export|dblink|set_config|pg_reload_conf|pg_notify|password_hash)\b`)

// escapedIdentifierPattern matches PostgreSQL Unicode-escaped identifiers and
// strings (U&"..." / U&'...') and C-style escaped strings (E'...'), which
// could spell a denied name without matching deniedQueryPattern. Analytics
// queries never need them.
var escapedIdentifierPattern = regexp.MustCompile(`(?i)\b[UE]&?['"]`)

// hiddenQueryColumns are stripped from ad hoc query results when they appear
// under their own name. Aliases and row constructors are not unwrapped, so
// deniedQueryPattern also rejects queries that mention the column.
var hiddenQueryColumns = map[string]bool{"password_hash": true}

// prepareQuery validates an ad hoc query and wraps it so that only a single
// SELECT-shaped statement can run.
func prepareQuery(query string) (string, error) {
	q := strings.TrimSpace(query)
	q = strings.TrimRight(q, "; \t\r\n")
	if q == "" {
		return "", fmt.Errorf("%w: query is empty", ErrInvalidQuery)
	}
	if strings.Contains(q, ";") {
		return "", fmt.Errorf("%w: only a single statement is allowed", ErrInvalidQuery)
	}
	if m := deniedQueryPattern.FindString(q); m != "" {
		return "", fmt.Errorf("%w: %s is not permitted", ErrInvalidQuery, m)
	}
	if escapedIdentifierPattern.MatchString(q) || strings.Contains(q, `\`) {
		return "", fmt.Errorf("%w: escaped identifiers and strings are not permitted", ErrInvalidQuery)
	}
	// Wrapping in a subquery means anything other than a SELECT/WITH/VALUES
	// is a syntax error, and the LIMIT bounds the result size.
	return "SELECT * FROM (\n" + q + "\n) AS mcp_query LIMIT $1", nil
}

// Query runs an ad hoc SELECT and returns at most maxRows rows. It relies on
// the surrounding READ ONLY transaction to guarantee nothing is modified.
func (a *Analytics) Query(ctx context.Context, query string, maxRows int) (*QueryResult, error) {
	wrapped, err := prepareQuery(query)
	if err != nil {
		return nil, err
	}
	maxRows = clampLimit(maxRows, DefaultQueryRows, MaxQueryRows)

	// Ask for one extra row to learn whether the result was truncated.
	rows, err := a.q.QueryContext(ctx, wrapped, maxRows+1)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQueryFailed, err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("reading columns: %w", err)
	}

	keep := make([]int, 0, len(columns))
	result := &QueryResult{Columns: make([]string, 0, len(columns)), Rows: make([][]interface{}, 0)}
	for i, c := range columns {
		if hiddenQueryColumns[c] {
			continue
		}
		keep = append(keep, i)
		result.Columns = append(result.Columns, c)
	}

	for rows.Next() {
		if len(result.Rows) == maxRows {
			result.Truncated = true
			break
		}
		values := make([]interface{}, len(columns))
		dests := make([]interface{}, len(columns))
		for i := range values {
			dests[i] = &values[i]
		}
		if err := rows.Scan(dests...); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		row := make([]interface{}, 0, len(keep))
		for _, i := range keep {
			row = append(row, normalizeQueryValue(values[i]))
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading rows: %w", err)
	}
	result.RowCount = len(result.Rows)

	return result, nil
}

// normalizeQueryValue converts driver values into JSON-friendly ones.
func normalizeQueryValue(v interface{}) interface{} {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}
