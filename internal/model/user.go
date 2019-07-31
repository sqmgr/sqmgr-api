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
	"database/sql"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"time"
)

// ErrUserExists is an error when the user already exists (when trying to create a new account)
var ErrUserExists = errors.New("model: user already exists")

// ErrUserNotFound is when the user is not found in the database
var ErrUserNotFound = errors.New("model: user not found")

// UserStore represents a store where users may reside
type UserStore string

// constants for the JWT "iss" (issuer) claim
const (
	IssuerAuth0 = "https://sqmgr.auth0.com/"
	IssuerSqMGR = "https://api.sqmgr.com/"
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
	ID      int64
	Store   UserStore
	StoreID string
	Created time.Time

	// not stored in the database
	Token *jwt.Token
}

// UserAction are a set of actions that a user can perform
type UserAction int

// UserAction constants
const (
	UserActionCreatePool UserAction = iota
)

// Can will return true if the user can do the action
func (u *User) Can(action UserAction) bool {
	switch action {
	case UserActionCreatePool:
		return u.Store == UserStoreAuth0
	}

	return false
}

// GetUser will get or create a record in the database based on the JWT issuer and store id
func (m *Model) GetUser(ctx context.Context, issuer, storeID string) (*User, error) {
	store, ok := issToStore[issuer]
	if !ok {
		return nil, fmt.Errorf("invalid issuer: %s", issuer)
	}

	row := m.db.QueryRowContext(ctx, "SELECT id, store, store_id, created FROM get_user($1, $2)", store, storeID)

	var u User
	u.Model = m
	if err := row.Scan(&u.ID, &u.Store, &u.StoreID, &u.Created); err != nil {
		return nil, err
	}

	return &u, nil
}

// JoinPool will link a user to a grid game.
func (u *User) JoinPool(ctx context.Context, p *Pool) error {
	_, err := u.db.ExecContext(ctx, "INSERT INTO pools_users (pool_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", p.id, u.ID)
	return err
}

// IsMemberOf will return true if the user belongs to the grid
func (u *User) IsMemberOf(ctx context.Context, p *Pool) (bool, error) {
	// user is the admin
	if u.ID == p.userID {
		return true, nil
	}

	// otherwise, check to see if they are a member

	row := u.db.QueryRowContext(ctx, "SELECT true FROM pools_users WHERE pool_id = $1 AND user_id = $2", p.id, u.ID)

	var ok bool
	if err := row.Scan(&ok); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, err
	}

	return ok, nil
}

// IsAdminOf will return true if the user is the admin of the grid
func (u *User) IsAdminOf(ctx context.Context, p *Pool) bool {
	return u.ID == p.userID
}
