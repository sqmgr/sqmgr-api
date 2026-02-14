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

// periodToInterval converts a period string to a PostgreSQL interval
func periodToInterval(period string) string {
	switch period {
	case "1h":
		return "1 hour"
	case "24h":
		return "1 day"
	case "week":
		return "7 days"
	case "month":
		return "30 days"
	case "year":
		return "365 days"
	default:
		return ""
	}
}

// GetAdminStats returns site-wide statistics filtered by time period
// Supported periods: "24h", "week", "month", "year", "all" (default)
func (m *Model) GetAdminStats(ctx context.Context, period string) (*AdminStats, error) {
	stats := &AdminStats{}

	interval := periodToInterval(period)
	var timeFilter string
	if interval != "" {
		timeFilter = fmt.Sprintf(" WHERE created > NOW() - INTERVAL '%s'", interval)
	}

	// Total pools
	query := "SELECT COUNT(*) FROM pools"
	if timeFilter != "" {
		query += timeFilter
	}
	row := m.DB.QueryRowContext(ctx, query)
	if err := row.Scan(&stats.TotalPools); err != nil {
		return nil, fmt.Errorf("counting total pools: %w", err)
	}

	// Total users (non-guest)
	query = "SELECT COUNT(*) FROM users WHERE store = 'auth0'"
	if interval != "" {
		query += fmt.Sprintf(" AND created > NOW() - INTERVAL '%s'", interval)
	}
	row = m.DB.QueryRowContext(ctx, query)
	if err := row.Scan(&stats.TotalUsers); err != nil {
		return nil, fmt.Errorf("counting total users: %w", err)
	}

	// Guest users
	query = "SELECT COUNT(*) FROM users WHERE store = 'sqmgr'"
	if interval != "" {
		query += fmt.Sprintf(" AND created > NOW() - INTERVAL '%s'", interval)
	}
	row = m.DB.QueryRowContext(ctx, query)
	if err := row.Scan(&stats.GuestUsers); err != nil {
		return nil, fmt.Errorf("counting guest users: %w", err)
	}

	// Active pools
	query = "SELECT COUNT(*) FROM pools WHERE archived = false"
	if interval != "" {
		query += fmt.Sprintf(" AND created > NOW() - INTERVAL '%s'", interval)
	}
	row = m.DB.QueryRowContext(ctx, query)
	if err := row.Scan(&stats.ActivePools); err != nil {
		return nil, fmt.Errorf("counting active pools: %w", err)
	}

	// Claimed squares (state != 'unclaimed')
	query = "SELECT COUNT(*) FROM pool_squares WHERE state != 'unclaimed'"
	if interval != "" {
		query += fmt.Sprintf(" AND modified > NOW() - INTERVAL '%s'", interval)
	}
	row = m.DB.QueryRowContext(ctx, query)
	if err := row.Scan(&stats.ClaimedSquares); err != nil {
		return nil, fmt.Errorf("counting claimed squares: %w", err)
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

// AdminUser represents a user in the admin list
type AdminUser struct {
	ID          int64     `json:"id"`
	Store       UserStore `json:"store"`
	Email       *string   `json:"email"`
	IsAdmin     bool      `json:"isAdmin"`
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
			u.is_admin,
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
			&user.IsAdmin,
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
}

// GetAdminLinkedEvents returns sports events that have at least one active grid linked,
// with the count of linked grids, sorted and paginated
func (m *Model) GetAdminLinkedEvents(ctx context.Context, offset int64, limit int, sortBy string, sortDir string) ([]*AdminLinkedEvent, error) {
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

	query := `
		SELECT
			e.id, e.espn_id, e.league, e.name, e.home_team_id, e.away_team_id,
			e.event_date, e.status, e.status_detail, e.home_score, e.away_score,
			COUNT(g.id) AS grid_count
		FROM sports_events e
		INNER JOIN grids g ON g.sports_event_id = e.id AND g.state = 'active'
		GROUP BY e.id
		ORDER BY ` + orderColumn + ` ` + orderDir + `
		OFFSET $1
		LIMIT $2`

	rows, err := m.DB.QueryContext(ctx, query, offset, limit)
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

// GetAdminLinkedEventsCount returns the count of sports events with at least one active linked grid
func (m *Model) GetAdminLinkedEventsCount(ctx context.Context) (int64, error) {
	var count int64
	row := m.DB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT e.id)
		FROM sports_events e
		INNER JOIN grids g ON g.sports_event_id = e.id AND g.state = 'active'
	`)
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
			p.token, p.name, p.user_id, u.email, g.created
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
