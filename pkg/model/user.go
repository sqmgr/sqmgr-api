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

	"github.com/golang-jwt/jwt/v5"
)

// ErrUserExists is an error when the user already exists (when trying to create a new account)
var ErrUserExists = errors.New("model: user already exists")

// ErrUserNotFound is when the user is not found in the database
var ErrUserNotFound = errors.New("model: user not found")

// UserStore represents a store where users may reside
type UserStore string

// constants for the JWT "iss" (issuer) claim
const (
	IssuerAuth0    = "https://sqmgr.auth0.com/"
	IssuerSqMGR    = "https://api.sqmgr.com/"
	ClaimNamespace = "https://sqmgr.com"
)

// constants for UserStore
const (
	UserStoreSqMGR UserStore = "sqmgr"
	UserStoreAuth0 UserStore = "auth0"
)

var issToStore = map[string]UserStore{
	IssuerAuth0: UserStoreAuth0,
	IssuerSqMGR: UserStoreSqMGR,
}

// User represents a SqMGR user
type User struct {
	*Model
	ID          int64
	Store       UserStore
	StoreID     string
	IsSiteAdmin bool
	Email       *string
	Created     time.Time

	// not stored in the database
	Token *jwt.Token
}

// Permission is a user capability
type Permission int

// UserAction constants
const (
	PermissionCreatePool Permission = iota
)

// HasPermission will return true if the user can do the action
func (u *User) HasPermission(action Permission) bool {
	switch action {
	case PermissionCreatePool:
		return u.Store == UserStoreAuth0
	}

	return false
}

func (m *Model) userByRow(row *sql.Row) (*User, error) {
	var u User
	u.Model = m
	if err := row.Scan(&u.ID, &u.Store, &u.StoreID, &u.IsSiteAdmin, &u.Email, &u.Created); err != nil {
		return nil, fmt.Errorf("scanning user row: %w", err)
	}

	return &u, nil
}

// GetUserByID will return a user by its ID.
func (m *Model) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row := m.DB.QueryRowContext(ctx, "SELECT id, store, store_id, is_site_admin, email, created FROM users WHERE id = $1", id)
	return m.userByRow(row)
}

// GetUser will get or create a record in the database based on the JWT issuer and store id
func (m *Model) GetUser(ctx context.Context, issuer, storeID string) (*User, error) {
	store, ok := issToStore[issuer]
	if !ok {
		return nil, fmt.Errorf("invalid issuer: %s", issuer)
	}

	row := m.DB.QueryRowContext(ctx, "SELECT id, store, store_id, is_site_admin, email, created FROM get_user($1, $2)", store, storeID)
	return m.userByRow(row)
}

// JoinPool will link a user to a pool.
func (u *User) JoinPool(ctx context.Context, p *Pool) error {
	// no-op
	if isManager, err := u.IsManagerOf(ctx, p); err != nil {
		return fmt.Errorf("checking manager status: %w", err)
	} else if isManager {
		return nil
	}

	if _, err := u.DB.ExecContext(ctx, "INSERT INTO pools_users (pool_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", p.id, u.ID); err != nil {
		return fmt.Errorf("inserting pool user: %w", err)
	}
	return nil
}

// LeavePool will unlink a user from a pool
func (u *User) LeavePool(ctx context.Context, p *Pool) error {
	_, err := u.DB.ExecContext(ctx, "DELETE FROM pools_users WHERE pool_id = $1 AND user_id = $2", p.ID(), u.ID)
	if err != nil {
		return fmt.Errorf("deleting pool user: %w", err)
	}

	return nil
}

// IsMemberOf will return true if the user belongs to the grid
func (u *User) IsMemberOf(ctx context.Context, p *Pool) (bool, error) {
	// user is the admin
	if u.ID == p.userID {
		return true, nil
	}

	// otherwise, check to see if they are a member

	row := u.DB.QueryRowContext(ctx, "SELECT true FROM pools_users WHERE pool_id = $1 AND user_id = $2", p.id, u.ID)

	var ok bool
	if err := row.Scan(&ok); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("checking pool membership: %w", err)
	}

	return ok, nil
}

// SetManagerOf will set the user as a manager in the pool. Note: this user must
// already be a member
func (u *User) SetManagerOf(ctx context.Context, p *Pool, isManager bool) error {
	_, err := u.DB.ExecContext(ctx, `
UPDATE
    pools_users
SET
    is_manager = $1,
    modified = (NOW() AT TIME ZONE 'UTC')
WHERE
	pool_id = $2 AND
  	user_id = $3`, isManager, p.ID(), u.ID)
	if err != nil {
		return fmt.Errorf("updating manager status: %w", err)
	}

	return nil
}

// HasManagerVisibility returns true if the user should see manager-level details
// for a pool. This includes pool managers and site admins. Site admins get
// read-only visibility but not write authority over pools they don't own;
// write operations use IsManagerOf directly.
func (u *User) HasManagerVisibility(ctx context.Context, p *Pool) (bool, error) {
	if u.IsSiteAdmin {
		return true, nil
	}
	return u.IsManagerOf(ctx, p)
}

// IsManagerOf will return true if the user is a manager of the pool
func (u *User) IsManagerOf(ctx context.Context, p *Pool) (bool, error) {
	if u.ID == p.userID {
		return true, nil
	}

	// otherwise, check to see if they are a manager

	row := u.DB.QueryRowContext(ctx, "SELECT true FROM pools_users WHERE pool_id = $1 AND user_id = $2 AND is_manager", p.id, u.ID)

	var ok bool
	if err := row.Scan(&ok); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("checking manager status: %w", err)
	}

	return ok, nil
}

// PoolsCreatedWithin will return the number of pools a user has created within a given duration period
func (u *User) PoolsCreatedWithin(ctx context.Context, within time.Duration) (int, error) {
	const query = "SELECT COUNT(*) FROM pools WHERE user_id = $1 AND created > NOW() - INTERVAL '1 microsecond' * $2"
	row := u.DB.QueryRowContext(ctx, query, u.ID, within/time.Microsecond)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("counting pools created within duration: %w", err)
	}

	return count, nil
}

// SetEmail updates the user's email in the database
func (u *User) SetEmail(ctx context.Context, email string) error {
	_, err := u.DB.ExecContext(ctx, "UPDATE users SET email = $1 WHERE id = $2", email, u.ID)
	if err != nil {
		return fmt.Errorf("updating user email: %w", err)
	}
	u.Email = &email
	return nil
}
