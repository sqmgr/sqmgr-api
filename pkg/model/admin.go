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
	"fmt"
	"strconv"
	"strings"
	"time"
)

// AdminStats holds site-wide statistics for admin dashboard
type AdminStats struct {
	TotalPools     int64 `json:"totalPools"`
	TotalUsers     int64 `json:"totalUsers"`
	GuestUsers     int64 `json:"guestUsers"`
	ActivePools    int64 `json:"activePools"`
	ClaimedSquares int64 `json:"claimedSquares"`
}

// AdminPool represents a pool in the admin list with additional metadata
type AdminPool struct {
	Token           string          `json:"token"`
	Name            string          `json:"name"`
	GridType        GridType        `json:"gridType"`
	NumberSetConfig NumberSetConfig `json:"numberSetConfig"`
	Archived        bool            `json:"archived"`
	OwnerID         int64           `json:"ownerId"`
	OwnerEmail      *string         `json:"ownerEmail"`
	OwnerStore      string          `json:"ownerStore"`
	MemberCount     int64           `json:"memberCount"`
	GridCount       int64           `json:"gridCount"`
	ClaimedCount    int64           `json:"claimedCount"`
	Created         string          `json:"created"`
}

// Supported values for StatsFilter.Period. Any other value (including the empty
// string) is treated the same as StatsPeriodAll, i.e. no time filtering.
const (
	StatsPeriodAll    = "all"
	StatsPeriodHour   = "1h"
	StatsPeriodDay    = "24h"
	StatsPeriodWeek   = "week"
	StatsPeriodMonth  = "month"
	StatsPeriodYear   = "year"
	StatsPeriodCustom = "custom"
)

// StatsFilter describes how admin stats should be restricted by time. When
// Period is StatsPeriodCustom, Start and End bound the range; End is inclusive
// of the entire day it names. For every other period the range is derived from
// the current time.
type StatsFilter struct {
	Period string
	Start  time.Time
	End    time.Time
}

// periodToInterval converts a period string to a PostgreSQL interval
func periodToInterval(period string) string {
	switch period {
	case StatsPeriodHour:
		return "1 hour"
	case StatsPeriodDay:
		return "1 day"
	case StatsPeriodWeek:
		return "7 days"
	case StatsPeriodMonth:
		return "30 days"
	case StatsPeriodYear:
		return "365 days"
	default:
		return ""
	}
}

// condition returns the SQL condition (without a leading WHERE or AND) that
// restricts the given timestamp column to the filter's range, along with any
// bind arguments. An empty condition means no time filtering should be applied.
// The column name is supplied by callers within this package and is never
// user-controlled; user-supplied dates are always bound as parameters.
//
// The placeholders are hardcoded to $1/$2 because this is the only
// parameterized condition statsCount combines into a query; staticCondition
// values passed to statsCount are always literal, args-free SQL fragments. If
// statsCount is ever changed to accept a staticCondition with its own bind
// arguments, these placeholder numbers (and the args returned here) will need
// to be renumbered to account for it.
func (f StatsFilter) condition(column string) (string, []interface{}) {
	if f.Period == StatsPeriodCustom {
		// The end date is inclusive, so compare against the start of the
		// following day.
		return fmt.Sprintf("%s >= $1 AND %s < $2", column, column),
			[]interface{}{f.Start, f.End.AddDate(0, 0, 1)}
	}

	if interval := periodToInterval(f.Period); interval != "" {
		return fmt.Sprintf("%s > NOW() - INTERVAL '%s'", column, interval), nil
	}

	return "", nil
}

// statsCount runs a COUNT(*) against table, combining an optional static
// condition with the filter's time condition on timeColumn.
func (m *Model) statsCount(ctx context.Context, table, staticCondition, timeColumn string, filter StatsFilter) (int64, error) {
	conditions := make([]string, 0, 2)
	if staticCondition != "" {
		conditions = append(conditions, staticCondition)
	}

	timeCondition, args := filter.condition(timeColumn)
	if timeCondition != "" {
		conditions = append(conditions, timeCondition)
	}

	query := "SELECT COUNT(*) FROM " + table
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int64
	if err := m.DB.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// GetAdminStats returns site-wide statistics filtered by time period
func (m *Model) GetAdminStats(ctx context.Context, filter StatsFilter) (*AdminStats, error) {
	stats := &AdminStats{}

	counts := []struct {
		label           string
		table           string
		staticCondition string
		timeColumn      string
		dest            *int64
	}{
		{"total pools", "pools", "", "created", &stats.TotalPools},
		{"total users", "users", "store = 'auth0'", "created", &stats.TotalUsers},
		{"guest users", "users", "store = 'sqmgr'", "created", &stats.GuestUsers},
		{"active pools", "pools", "archived = false", "created", &stats.ActivePools},
		{"claimed squares", "pool_squares", "state != 'unclaimed'", "modified", &stats.ClaimedSquares},
	}

	for _, c := range counts {
		count, err := m.statsCount(ctx, c.table, c.staticCondition, c.timeColumn, filter)
		if err != nil {
			return nil, fmt.Errorf("counting %s: %w", c.label, err)
		}
		*c.dest = count
	}

	return stats, nil
}

// GetAllPools returns all pools with optional search, pagination
func (m *Model) GetAllPools(ctx context.Context, search string, offset int64, limit int) ([]*AdminPool, error) {
	baseQuery := `
		SELECT
			p.token,
			p.name,
			p.grid_type,
			p.number_set_config,
			p.archived,
			p.user_id,
			u.email,
			u.store,
			(SELECT COUNT(*) FROM pools_users pu WHERE pu.pool_id = p.id) as member_count,
			(SELECT COUNT(*) FROM grids g WHERE g.pool_id = p.id AND g.state = 'active') as grid_count,
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state != 'unclaimed') as claimed_count,
			p.created
		FROM pools p
		LEFT JOIN users u ON u.id = p.user_id
		%s
		ORDER BY p.id DESC
		OFFSET $%d
		LIMIT $%d`

	if search != "" {
		query := fmt.Sprintf(baseQuery, "WHERE p.name ILIKE $1", 2, 3)
		rowsResult, queryErr := m.DB.QueryContext(ctx, query, "%"+search+"%", offset, limit)
		if queryErr != nil {
			return nil, fmt.Errorf("querying pools with search: %w", queryErr)
		}
		defer rowsResult.Close()

		return scanAdminPools(rowsResult)
	}

	query := fmt.Sprintf(baseQuery, "", 1, 2)
	rowsResult, err := m.DB.QueryContext(ctx, query, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("querying pools: %w", err)
	}
	defer rowsResult.Close()

	return scanAdminPools(rowsResult)
}

// scanAdminPools scans rows into AdminPool slice
func scanAdminPools(rows interface {
	Next() bool
	Scan(...interface{}) error
}) ([]*AdminPool, error) {
	pools := make([]*AdminPool, 0)
	for rows.Next() {
		pool := &AdminPool{}
		if err := rows.Scan(
			&pool.Token,
			&pool.Name,
			&pool.GridType,
			&pool.NumberSetConfig,
			&pool.Archived,
			&pool.OwnerID,
			&pool.OwnerEmail,
			&pool.OwnerStore,
			&pool.MemberCount,
			&pool.GridCount,
			&pool.ClaimedCount,
			&pool.Created,
		); err != nil {
			return nil, fmt.Errorf("scanning pool row: %w", err)
		}
		pools = append(pools, pool)
	}
	return pools, nil
}

// GetAllPoolsCount returns count of all pools with optional search
func (m *Model) GetAllPoolsCount(ctx context.Context, search string) (int64, error) {
	var count int64
	var row interface{ Scan(...interface{}) error }

	if search != "" {
		row = m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM pools WHERE name ILIKE $1", "%"+search+"%")
	} else {
		row = m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM pools")
	}

	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("counting pools: %w", err)
	}

	return count, nil
}

// AdminUserStats holds statistics for a specific user
type AdminUserStats struct {
	PoolsCreated  int64 `json:"poolsCreated"`
	PoolsJoined   int64 `json:"poolsJoined"`
	ActivePools   int64 `json:"activePools"`
	ArchivedPools int64 `json:"archivedPools"`
}

// GetUserStats returns statistics for a specific user
func (m *Model) GetUserStats(ctx context.Context, userID int64) (*AdminUserStats, error) {
	stats := &AdminUserStats{}

	// Pools created
	row := m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM pools WHERE user_id = $1", userID)
	if err := row.Scan(&stats.PoolsCreated); err != nil {
		return nil, fmt.Errorf("counting pools created: %w", err)
	}

	// Pools joined (excluding owned pools)
	row = m.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pools_users pu
		JOIN pools p ON p.id = pu.pool_id
		WHERE pu.user_id = $1 AND p.user_id != $1
	`, userID)
	if err := row.Scan(&stats.PoolsJoined); err != nil {
		return nil, fmt.Errorf("counting pools joined: %w", err)
	}

	// Active pools created
	row = m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM pools WHERE user_id = $1 AND archived = false", userID)
	if err := row.Scan(&stats.ActivePools); err != nil {
		return nil, fmt.Errorf("counting active pools: %w", err)
	}

	// Archived pools created
	row = m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM pools WHERE user_id = $1 AND archived = true", userID)
	if err := row.Scan(&stats.ArchivedPools); err != nil {
		return nil, fmt.Errorf("counting archived pools: %w", err)
	}

	return stats, nil
}

// GetPoolsByUserID returns pools created by a specific user
func (m *Model) GetPoolsByUserID(ctx context.Context, userID int64, includeArchived bool, offset int64, limit int) ([]*AdminPool, error) {
	baseQuery := `
		SELECT
			p.token,
			p.name,
			p.grid_type,
			p.number_set_config,
			p.archived,
			p.user_id,
			u.email,
			u.store,
			(SELECT COUNT(*) FROM pools_users pu WHERE pu.pool_id = p.id) as member_count,
			(SELECT COUNT(*) FROM grids g WHERE g.pool_id = p.id AND g.state = 'active') as grid_count,
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state != 'unclaimed') as claimed_count,
			p.created
		FROM pools p
		LEFT JOIN users u ON u.id = p.user_id
		WHERE p.user_id = $1 %s
		ORDER BY p.id DESC
		OFFSET $2
		LIMIT $3`

	archivedFilter := "AND p.archived = false"
	if includeArchived {
		archivedFilter = ""
	}

	query := fmt.Sprintf(baseQuery, archivedFilter)
	rowsResult, err := m.DB.QueryContext(ctx, query, userID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("querying user pools: %w", err)
	}
	defer rowsResult.Close()

	return scanAdminPools(rowsResult)
}

// GetPoolsByUserIDCount returns count of pools created by a specific user
func (m *Model) GetPoolsByUserIDCount(ctx context.Context, userID int64, includeArchived bool) (int64, error) {
	var count int64
	var row interface{ Scan(...interface{}) error }

	if includeArchived {
		row = m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM pools WHERE user_id = $1", userID)
	} else {
		row = m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM pools WHERE user_id = $1 AND archived = false", userID)
	}

	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("counting user pools: %w", err)
	}

	return count, nil
}

// GetJoinedPoolsByUserID returns pools a specific user is a member of but does not own,
// ordered by most recently joined
func (m *Model) GetJoinedPoolsByUserID(ctx context.Context, userID int64, includeArchived bool, offset int64, limit int) ([]*AdminPool, error) {
	baseQuery := `
		SELECT
			p.token,
			p.name,
			p.grid_type,
			p.number_set_config,
			p.archived,
			p.user_id,
			u.email,
			u.store,
			(SELECT COUNT(*) FROM pools_users pu2 WHERE pu2.pool_id = p.id) as member_count,
			(SELECT COUNT(*) FROM grids g WHERE g.pool_id = p.id AND g.state = 'active') as grid_count,
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state != 'unclaimed') as claimed_count,
			p.created
		FROM pools_users pu
		JOIN pools p ON p.id = pu.pool_id
		LEFT JOIN users u ON u.id = p.user_id
		WHERE pu.user_id = $1 AND p.user_id != $1 %s
		ORDER BY pu.created DESC, p.id DESC
		OFFSET $2
		LIMIT $3`

	archivedFilter := "AND p.archived = false"
	if includeArchived {
		archivedFilter = ""
	}

	query := fmt.Sprintf(baseQuery, archivedFilter)
	rowsResult, err := m.DB.QueryContext(ctx, query, userID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("querying user joined pools: %w", err)
	}
	defer rowsResult.Close()

	return scanAdminPools(rowsResult)
}

// GetJoinedPoolsByUserIDCount returns the count of pools a specific user is a member of but does not own
func (m *Model) GetJoinedPoolsByUserIDCount(ctx context.Context, userID int64, includeArchived bool) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM pools_users pu
		JOIN pools p ON p.id = pu.pool_id
		WHERE pu.user_id = $1 AND p.user_id != $1`
	if !includeArchived {
		query += " AND p.archived = false"
	}

	var count int64
	if err := m.DB.QueryRowContext(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting user joined pools: %w", err)
	}

	return count, nil
}

// AdminUser represents a user in the admin list
type AdminUser struct {
	ID          int64     `json:"id"`
	Store       UserStore `json:"store"`
	Email       *string   `json:"email"`
	IsSiteAdmin bool      `json:"isSiteAdmin"`
	PoolsOwned  int64     `json:"poolsOwned"`
	PoolsJoined int64     `json:"poolsJoined"`
	Created     string    `json:"created"`
}

// GetAllUsers returns all users with optional search, pagination, and sorting
func (m *Model) GetAllUsers(ctx context.Context, search string, offset int64, limit int, sortBy string, sortDir string) ([]*AdminUser, error) {
	// Validate sortBy to prevent SQL injection
	validSortColumns := map[string]string{
		"poolsOwned":  "pools_owned",
		"poolsJoined": "pools_joined",
		"created":     "u.created",
		"id":          "u.id",
	}

	orderColumn := "u.id"
	if col, ok := validSortColumns[sortBy]; ok {
		orderColumn = col
	}

	orderDir := "DESC"
	if sortDir == "asc" {
		orderDir = "ASC"
	}

	baseQuery := `
		SELECT
			u.id,
			u.store,
			u.email,
			u.is_site_admin,
			(SELECT COUNT(*) FROM pools p WHERE p.user_id = u.id) as pools_owned,
			(SELECT COUNT(*) FROM pools_users pu JOIN pools p ON p.id = pu.pool_id WHERE pu.user_id = u.id AND p.user_id != u.id) as pools_joined,
			u.created
		FROM users u
		%s
		ORDER BY ` + orderColumn + ` ` + orderDir + `
		OFFSET $%d
		LIMIT $%d`

	if search != "" {
		query := fmt.Sprintf(baseQuery, "WHERE u.store = 'auth0' AND u.email ILIKE $1", 2, 3)
		rowsResult, queryErr := m.DB.QueryContext(ctx, query, "%"+search+"%", offset, limit)
		if queryErr != nil {
			return nil, fmt.Errorf("querying users with search: %w", queryErr)
		}
		defer rowsResult.Close()

		return scanAdminUsers(rowsResult)
	}

	query := fmt.Sprintf(baseQuery, "WHERE u.store = 'auth0'", 1, 2)
	rowsResult, err := m.DB.QueryContext(ctx, query, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("querying users: %w", err)
	}
	defer rowsResult.Close()

	return scanAdminUsers(rowsResult)
}

// scanAdminUsers scans rows into AdminUser slice
func scanAdminUsers(rows interface {
	Next() bool
	Scan(...interface{}) error
}) ([]*AdminUser, error) {
	users := make([]*AdminUser, 0)
	for rows.Next() {
		user := &AdminUser{}
		if err := rows.Scan(
			&user.ID,
			&user.Store,
			&user.Email,
			&user.IsSiteAdmin,
			&user.PoolsOwned,
			&user.PoolsJoined,
			&user.Created,
		); err != nil {
			return nil, fmt.Errorf("scanning user row: %w", err)
		}
		users = append(users, user)
	}
	return users, nil
}

// GetAllUsersCount returns count of all users with optional search
func (m *Model) GetAllUsersCount(ctx context.Context, search string) (int64, error) {
	var count int64
	var row interface{ Scan(...interface{}) error }

	if search != "" {
		row = m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE store = 'auth0' AND email ILIKE $1", "%"+search+"%")
	} else {
		row = m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE store = 'auth0'")
	}

	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}

	return count, nil
}

// AdminLinkedEvent represents a sports event that has at least one grid linked to it
type AdminLinkedEvent struct {
	ID           int64             `json:"id"`
	ESPNID       string            `json:"espnId,omitempty"`
	League       SportsLeague      `json:"league"`
	Name         string            `json:"name,omitempty"`
	HomeTeamID   string            `json:"homeTeamId"`
	AwayTeamID   string            `json:"awayTeamId"`
	EventDate    time.Time         `json:"eventDate"`
	Status       SportsEventStatus `json:"status"`
	StatusDetail string            `json:"statusDetail,omitempty"`
	HomeScore    *int              `json:"homeScore,omitempty"`
	AwayScore    *int              `json:"awayScore,omitempty"`
	HomeTeam     *SportsTeamJSON   `json:"homeTeam,omitempty"`
	AwayTeam     *SportsTeamJSON   `json:"awayTeam,omitempty"`
	GridCount    int64             `json:"gridCount"`
}

// AdminEventGrid represents a grid linked to a sports event
type AdminEventGrid struct {
	GridID       int64     `json:"gridId"`
	GridName     string    `json:"gridName"`
	GridState    string    `json:"gridState"`
	PoolToken    string    `json:"poolToken"`
	PoolName     string    `json:"poolName"`
	CreatorID    int64     `json:"creatorId"`
	CreatorEmail *string   `json:"creatorEmail"`
	Created      time.Time `json:"created"`
	// ClaimedSquares is the number of squares in the grid's pool that are not unclaimed.
	ClaimedSquares int64 `json:"claimedSquares"`
	// TotalSquares is the number of squares in the grid's pool.
	TotalSquares int64 `json:"totalSquares"`
}

// AdminLinkedEventsFilter restricts which linked events GetAdminLinkedEvents and
// GetAdminLinkedEventsCount return. A zero value for any field means that field
// does not restrict the results.
//
// Start and End are calendar days; End is inclusive of the entire day it names.
// Because sports_events.event_date is a zoneless TIMESTAMP holding UTC
// wall-clock time, callers must supply Start and End as UTC midnight (as
// time.Parse produces when the input carries no zone) so that the bound
// values line up with the stored dates.
type AdminLinkedEventsFilter struct {
	League SportsLeague
	Status SportsEventStatus
	Start  time.Time
	End    time.Time
}

// conditions returns the SQL conditions (without a leading WHERE or AND) that
// apply the filter to the sports_events table aliased as "e", along with the
// bind arguments they reference. Placeholders are numbered from $1, so callers
// must append their own arguments after these.
func (f AdminLinkedEventsFilter) conditions() ([]string, []interface{}) {
	var conditions []string
	var args []interface{}

	if f.League != "" {
		args = append(args, string(f.League))
		conditions = append(conditions, fmt.Sprintf("e.league = $%d", len(args)))
	}

	if f.Status != "" {
		args = append(args, string(f.Status))
		conditions = append(conditions, fmt.Sprintf("e.status = $%d", len(args)))
	}

	if !f.Start.IsZero() {
		args = append(args, f.Start)
		conditions = append(conditions, fmt.Sprintf("e.event_date >= $%d", len(args)))
	}

	if !f.End.IsZero() {
		// The end date is inclusive, so compare against the start of the
		// following day.
		args = append(args, f.End.AddDate(0, 0, 1))
		conditions = append(conditions, fmt.Sprintf("e.event_date < $%d", len(args)))
	}

	return conditions, args
}

// whereClause renders the filter as a complete " WHERE ..." clause (with a
// leading space) or an empty string when the filter is unrestricted.
func (f AdminLinkedEventsFilter) whereClause() (string, []interface{}) {
	conditions, args := f.conditions()
	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

// GetAdminLinkedEvents returns sports events that have at least one active grid linked,
// with the count of linked grids, filtered, sorted and paginated
func (m *Model) GetAdminLinkedEvents(ctx context.Context, filter AdminLinkedEventsFilter, offset int64, limit int, sortBy string, sortDir string) ([]*AdminLinkedEvent, error) {
	// Validate sort column
	validSortColumns := map[string]string{
		"eventDate": "e.event_date",
		"gridCount": "grid_count",
	}
	orderColumn := "e.event_date"
	if col, ok := validSortColumns[sortBy]; ok {
		orderColumn = col
	}

	orderDir := "DESC"
	if sortDir == "asc" {
		orderDir = "ASC"
	}

	where, args := filter.whereClause()
	offsetArg := len(args) + 1
	limitArg := len(args) + 2
	args = append(args, offset, limit)

	query := `
		SELECT
			e.id, e.espn_id, e.league, e.name, e.home_team_id, e.away_team_id,
			e.event_date, e.status, e.status_detail, e.home_score, e.away_score,
			COUNT(g.id) AS grid_count
		FROM sports_events e
		INNER JOIN grids g ON g.sports_event_id = e.id AND g.state = 'active'` + where + `
		GROUP BY e.id
		ORDER BY ` + orderColumn + ` ` + orderDir + `
		OFFSET $` + strconv.Itoa(offsetArg) + `
		LIMIT $` + strconv.Itoa(limitArg)

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying linked events: %w", err)
	}
	defer rows.Close()

	// Scan rows and build SportsEvent objects for team loading
	var events []*AdminLinkedEvent
	var sportsEvents []*SportsEvent
	for rows.Next() {
		ale := &AdminLinkedEvent{}
		se := &SportsEvent{model: m}
		var name, statusDetail *string
		if err := rows.Scan(
			&ale.ID, &ale.ESPNID, &ale.League, &name, &ale.HomeTeamID, &ale.AwayTeamID,
			&ale.EventDate, &ale.Status, &statusDetail, &ale.HomeScore, &ale.AwayScore,
			&ale.GridCount,
		); err != nil {
			return nil, fmt.Errorf("scanning linked event row: %w", err)
		}
		if name != nil {
			ale.Name = *name
		}
		if statusDetail != nil {
			ale.StatusDetail = *statusDetail
		}
		// Build a SportsEvent for team loading
		se.ID = ale.ID
		se.League = ale.League
		se.HomeTeamID = ale.HomeTeamID
		se.AwayTeamID = ale.AwayTeamID
		sportsEvents = append(sportsEvents, se)
		events = append(events, ale)
	}

	// Batch load teams
	if err := m.LoadTeamsForSportsEvents(ctx, sportsEvents); err != nil {
		return nil, fmt.Errorf("loading teams for linked events: %w", err)
	}

	// Assign team JSON to results
	for i, se := range sportsEvents {
		if se.homeTeam != nil {
			events[i].HomeTeam = se.homeTeam.JSON()
		}
		if se.awayTeam != nil {
			events[i].AwayTeam = se.awayTeam.JSON()
		}
	}

	return events, nil
}

// GetAdminLinkedEventsCount returns the count of sports events with at least one
// active linked grid that match the filter
func (m *Model) GetAdminLinkedEventsCount(ctx context.Context, filter AdminLinkedEventsFilter) (int64, error) {
	where, args := filter.whereClause()

	var count int64
	row := m.DB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT e.id)
		FROM sports_events e
		INNER JOIN grids g ON g.sports_event_id = e.id AND g.state = 'active'`+where, args...)
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("counting linked events: %w", err)
	}
	return count, nil
}

// GetAdminEventGrids returns the grids linked to a specific sports event with pagination
func (m *Model) GetAdminEventGrids(ctx context.Context, eventID int64, offset int64, limit int) ([]*AdminEventGrid, error) {
	const query = `
		SELECT
			g.id, g.label, g.home_team_name, g.away_team_name, g.state,
			p.token, p.name, p.user_id, u.email, g.created,
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state != 'unclaimed') AS claimed_squares,
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id) AS total_squares
		FROM grids g
		INNER JOIN pools p ON p.id = g.pool_id
		LEFT JOIN users u ON u.id = p.user_id
		WHERE g.sports_event_id = $1 AND g.state = 'active'
		ORDER BY g.created DESC
		OFFSET $2
		LIMIT $3`

	rows, err := m.DB.QueryContext(ctx, query, eventID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("querying event grids: %w", err)
	}
	defer rows.Close()

	grids := make([]*AdminEventGrid, 0)
	for rows.Next() {
		grid := &AdminEventGrid{}
		var label, homeTeamName, awayTeamName *string
		if err := rows.Scan(
			&grid.GridID, &label, &homeTeamName, &awayTeamName, &grid.GridState,
			&grid.PoolToken, &grid.PoolName, &grid.CreatorID, &grid.CreatorEmail, &grid.Created,
			&grid.ClaimedSquares, &grid.TotalSquares,
		); err != nil {
			return nil, fmt.Errorf("scanning event grid row: %w", err)
		}

		// Build grid name using same logic as Grid.Name()
		away := ""
		if awayTeamName != nil {
			away = *awayTeamName
		}
		home := ""
		if homeTeamName != nil {
			home = *homeTeamName
		}
		vs := fmt.Sprintf("%s vs. %s", away, home)
		if label != nil {
			grid.GridName = fmt.Sprintf("%s: %s", *label, vs)
		} else {
			grid.GridName = vs
		}

		grids = append(grids, grid)
	}

	return grids, nil
}

// GetAdminEventGridsCount returns the count of active grids linked to a specific sports event
func (m *Model) GetAdminEventGridsCount(ctx context.Context, eventID int64) (int64, error) {
	var count int64
	row := m.DB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM grids
		WHERE sports_event_id = $1 AND state = 'active'
	`, eventID)
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("counting event grids: %w", err)
	}
	return count, nil
}
