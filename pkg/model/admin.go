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
	Token        string   `json:"token"`
	Name         string   `json:"name"`
	GridType     GridType `json:"gridType"`
	Archived     bool     `json:"archived"`
	OwnerID      int64    `json:"ownerId"`
	MemberCount  int64    `json:"memberCount"`
	GridCount    int64    `json:"gridCount"`
	ClaimedCount int64    `json:"claimedCount"`
	Created      string   `json:"created"`
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
			p.archived,
			p.user_id,
			(SELECT COUNT(*) FROM pools_users pu WHERE pu.pool_id = p.id) as member_count,
			(SELECT COUNT(*) FROM grids g WHERE g.pool_id = p.id AND g.state = 'active') as grid_count,
			(SELECT COUNT(*) FROM pool_squares ps WHERE ps.pool_id = p.id AND ps.state != 'unclaimed') as claimed_count,
			p.created
		FROM pools p
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
			&pool.Archived,
			&pool.OwnerID,
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
