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

package model

import (
	"context"
	"fmt"
)

// AdminStats holds site-wide statistics for admin dashboard
type AdminStats struct {
	TotalPools    int64 `json:"totalPools"`
	TotalUsers    int64 `json:"totalUsers"`
	GuestUsers    int64 `json:"guestUsers"`
	ActivePools   int64 `json:"activePools"`
	ArchivedPools int64 `json:"archivedPools"`
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

	// Archived pools
	query = "SELECT COUNT(*) FROM pools WHERE archived = true"
	if interval != "" {
		query += fmt.Sprintf(" AND created > NOW() - INTERVAL '%s'", interval)
	}
	row = m.DB.QueryRowContext(ctx, query)
	if err := row.Scan(&stats.ArchivedPools); err != nil {
		return nil, fmt.Errorf("counting archived pools: %w", err)
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
