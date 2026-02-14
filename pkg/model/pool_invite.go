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
	"time"

	"github.com/sqmgr/sqmgr-api/pkg/tokengen"
)

const inviteTokenLen = 10

// PoolInvite represents an invite token for joining a pool
type PoolInvite struct {
	Token     string    `json:"token"`
	PoolID    int64     `json:"-"`
	CheckID   int       `json:"-"`
	ExpiresAt time.Time `json:"-"`
	Created   time.Time `json:"-"`
}

// NewPoolInvite creates a new invite token for a pool. It retries on token collision.
func (m *Model) NewPoolInvite(ctx context.Context, poolID int64, checkID int, ttl time.Duration) (*PoolInvite, error) {
	for i := 0; i <= maxRetries; i++ {
		token, err := tokengen.Generate(inviteTokenLen)
		if err != nil {
			return nil, fmt.Errorf("generating invite token: %w", err)
		}

		expiresAt := time.Now().Add(ttl)

		invite := &PoolInvite{}
		err = m.DB.QueryRowContext(ctx,
			`INSERT INTO pool_invites (token, pool_id, check_id, expires_at)
			 VALUES ($1, $2, $3, $4)
			 RETURNING token, pool_id, check_id, expires_at, created`,
			token, poolID, checkID, expiresAt,
		).Scan(&invite.Token, &invite.PoolID, &invite.CheckID, &invite.ExpiresAt, &invite.Created)

		if err != nil {
			// Token collision — retry
			continue
		}

		return invite, nil
	}

	return nil, ErrRetryLimitExceeded
}

// PoolInviteByToken looks up an invite by its short token
func (m *Model) PoolInviteByToken(ctx context.Context, token string) (*PoolInvite, error) {
	invite := &PoolInvite{}
	err := m.DB.QueryRowContext(ctx,
		`SELECT token, pool_id, check_id, expires_at, created
		 FROM pool_invites
		 WHERE token = $1`,
		token,
	).Scan(&invite.Token, &invite.PoolID, &invite.CheckID, &invite.ExpiresAt, &invite.Created)

	if err != nil {
		return nil, fmt.Errorf("looking up pool invite: %w", err)
	}

	return invite, nil
}

// ActiveInvite returns the current valid invite for the pool (matching check_id, not expired), or nil if none exists
func (p *Pool) ActiveInvite(ctx context.Context) (*PoolInvite, error) {
	invite := &PoolInvite{}
	err := p.model.DB.QueryRowContext(ctx,
		`SELECT token, pool_id, check_id, expires_at, created
		 FROM pool_invites
		 WHERE pool_id = $1 AND check_id = $2 AND expires_at > (NOW() AT TIME ZONE 'utc')
		 ORDER BY created DESC
		 LIMIT 1`,
		p.id, p.checkID,
	).Scan(&invite.Token, &invite.PoolID, &invite.CheckID, &invite.ExpiresAt, &invite.Created)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up active invite: %w", err)
	}

	return invite, nil
}
