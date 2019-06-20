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
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"github.com/synacor/argon2id"
)

// NameMaxLength is the maximum length the pool name may be
const NameMaxLength = 50

// Pool is an individual pool board
// This object uses getters and setters to help guard against user input.
type Pool struct {
	model        *Model
	id           int64
	token        string
	userID       int64
	name         string
	gridType     GridType
	passwordHash string
	locks        time.Time
	created      time.Time
	modified     time.Time

	squares map[int]*PoolSquare
}

// IsLocked will return true if the locks date is in the past
func (p *Pool) IsLocked() bool {
	if p.locks.IsZero() {
		return false
	}

	return p.locks.Before(time.Now())
}

// Locks is a getter for locks
func (p *Pool) Locks() time.Time {
	return p.locks
}

// SetLocks is a setter for locks
func (p *Pool) SetLocks(locks time.Time) {
	p.locks = locks
}

// PoolWithID returns an empty pool object with only the ID set
func PoolWithID(id int64) *Pool {
	return &Pool{id: id}
}

type poolJSON struct {
	Token    string    `json:"token"`
	Name     string    `json:"name"`
	GridType GridType  `json:"gridType"`
	Locks    time.Time `json:"locks"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
}

// ID returns the id
func (p *Pool) ID() int64 {
	return p.id
}

// Token is a getter for the token
func (p *Pool) Token() string {
	return p.token
}

// Name is a getter for the name
func (p *Pool) Name() string {
	return p.name
}

// Created is a getter for the created date
func (p *Pool) Created() time.Time {
	return p.created
}

// Modified is a getter for the modified date
func (p *Pool) Modified() time.Time {
	return p.modified
}

// SetName is a setter for the name
func (p *Pool) SetName(name string) {
	if utf8.RuneCountInString(name) > NameMaxLength {
		name = string([]rune(name)[0:NameMaxLength])
	}

	p.name = name
}

// GridType is a getter for the grid type
func (p *Pool) GridType() GridType {
	return p.gridType
}

// SetGridType is a setter for the grid type
func (p *Pool) SetGridType(gridType GridType) {
	p.gridType = gridType
}

// MarshalJSON provides custom JSON marshalling
func (p *Pool) MarshalJSON() ([]byte, error) {
	return json.Marshal(poolJSON{
		Token:    p.token,
		Name:     p.name,
		Locks:    p.Locks(),
		GridType: p.gridType,
		Created:  p.created,
		Modified: p.modified,
	})
}

type executer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type scanFunc func(dest ...interface{}) error

func (m *Model) poolByRow(scan scanFunc) (*Pool, error) {
	s := Pool{model: m}
	var locks *time.Time
	if err := scan(&s.id, &s.token, &s.userID, &s.name, &s.gridType, &s.passwordHash, &locks, &s.created, &s.modified); err != nil {
		return nil, err
	}

	if locks != nil {
		s.locks = locks.In(locationNewYork)
	}

	// XXX: do we want the ability to let the user choose the time zone?
	s.created = s.created.In(locationNewYork)
	s.modified = s.modified.In(locationNewYork)

	return &s, nil
}

// PoolsJoinedByUser will return a collection of pools that the user joined
func (m *Model) PoolsJoinedByUser(ctx context.Context, u *User, offset, limit int) ([]*Pool, error) {
	const query = `
		SELECT pools.*
		FROM pools
		LEFT JOIN pools_users ON pools.id = pools_users.pool_id
		WHERE pools_users.user_id = $1
		ORDER BY pools.id DESC
		OFFSET $2
		LIMIT $3`

	return m.poolsByRows(m.db.QueryContext(ctx, query, u.ID, offset, limit))
}

// PoolsJoinedByUserCount will return a how many pools the user joined
func (m *Model) PoolsJoinedByUserCount(ctx context.Context, u *User) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM pools
		LEFT JOIN pools_users ON pools.id = pools_users.pool_id
		WHERE pools_users.user_id = $1`

	return m.poolsCount(m.db.QueryRowContext(ctx, query, u.ID))
}

// PoolsOwnedByUser will return a collection of pools that were created by the user
func (m *Model) PoolsOwnedByUser(ctx context.Context, u *User, offset, limit int) ([]*Pool, error) {
	const query = `
		SELECT *
		FROM pools
		WHERE user_id = $1
		ORDER BY pools.id DESC
		OFFSET $2
		LIMIT $3`

	return m.poolsByRows(m.db.QueryContext(ctx, query, u.ID, offset, limit))
}

// PoolsOwnedByUserCount will return how many pools were created by the user
func (m *Model) PoolsOwnedByUserCount(ctx context.Context, u *User) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM pools
		WHERE user_id = $1`

	return m.poolsCount(m.db.QueryRowContext(ctx, query, u.ID))
}

func (m *Model) poolsByRows(rows *sql.Rows, err error) ([]*Pool, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	collection := make([]*Pool, 0)
	for rows.Next() {
		pool, err := m.poolByRow(rows.Scan)
		if err != nil {
			return nil, err
		}

		collection = append(collection, pool)
	}

	return collection, nil
}

func (m *Model) poolsCount(row *sql.Row) (int64, error) {
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// PoolByToken will return the pools with the matching token
func (m *Model) PoolByToken(ctx context.Context, token string) (*Pool, error) {
	row := m.db.QueryRowContext(ctx, "SELECT * FROM pools WHERE token = $1", token)
	return m.poolByRow(row.Scan)
}

// PoolByID will return the pools with the matching ID
func (m *Model) PoolByID(id int64) (*Pool, error) {
	row := m.db.QueryRow("SELECT * FROM pools WHERE id = $1", id)
	return m.poolByRow(row.Scan)
}

// NewPool will save new pool into the database
func (m *Model) NewPool(ctx context.Context, userID int64, name string, gridType GridType, password string) (*Pool, error) {
	if err := IsValidGridType(string(gridType)); err != nil {
		return nil, err
	}

	token, err := m.NewToken()
	if err != nil {
		return nil, err
	}

	passwordHash, err := argon2id.DefaultHashPassword(password)
	if err != nil {
		return nil, err
	}

	const query = `
		SELECT *
		FROM new_pool($1, $2, $3, $4, $5, $6)
	`

	row := m.db.QueryRowContext(ctx, query, token, userID, name, gridType, passwordHash, gridType.Squares())

	pool, err := m.poolByRow(row.Scan)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// SetPassword will set a new password and ensures that it's properly hashed
func (p *Pool) SetPassword(password string) error {
	passwordHash, err := argon2id.DefaultHashPassword(password)
	if err != nil {
		return err
	}

	p.passwordHash = passwordHash
	return nil
}

// Save will save the pool
func (p *Pool) Save(ctx context.Context) error {
	const query = `UPDATE pools SET name = $1, grid_type = $2, password_hash = $3, locks = $4, modified = (NOW() AT TIME ZONE 'utc')  WHERE id = $5`

	var locks *time.Time
	if !p.locks.IsZero() {
		locksInUTC := p.locks.UTC()
		locks = &locksInUTC
	}

	_, err := p.model.db.ExecContext(ctx, query, p.name, p.gridType, p.passwordHash, locks, p.id)
	return err
}

// PasswordIsValid is will return true if the password matches
func (p *Pool) PasswordIsValid(password string) bool {
	if err := argon2id.Compare(p.passwordHash, password); err != nil {
		if err != argon2id.ErrMismatchedHashAndPassword {
			logrus.WithError(err).Error("could not check password")
		}

		return false
	}

	return true
}

// Squares will return the squares that belong to a pool. This method will lazily load the squares
func (p *Pool) Squares() (map[int]*PoolSquare, error) {
	if p.squares == nil {
		const query = `
		SELECT id, square_id, user_id, session_user_id, state, claimant, modified
		FROM pool_squares
		WHERE pool_id = $1
		ORDER BY square_id`

		rows, err := p.model.db.Query(query, p.id)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		squares := make(map[int]*PoolSquare)
		for rows.Next() {
			gs, err := p.squareByRow(rows.Scan)
			if err != nil {
				return nil, err
			}
			squares[gs.SquareID] = gs
		}

		p.squares = squares
	}

	return p.squares, nil
}

// SquareBySquareID will return a single square based on the square ID
func (p *Pool) SquareBySquareID(squareID int) (*PoolSquare, error) {
	const query = `
	SELECT id, square_id, user_id, session_user_id, state, claimant, modified
	FROM pool_squares
	WHERE pool_id = $1
		AND square_id = $2`

	row := p.model.db.QueryRow(query, p.id, squareID)
	return p.squareByRow(row.Scan)
}

func (p *Pool) squareByRow(scan scanFunc) (*PoolSquare, error) {
	gs := PoolSquare{
		Model:  p.model,
		PoolID: p.id,
	}

	var claimant *string
	var userID *int64
	var sessionUserID *string
	if err := scan(&gs.ID, &gs.SquareID, &userID, &sessionUserID, &gs.State, &claimant, &gs.Modified); err != nil {
		return nil, err
	}

	if claimant != nil {
		gs.Claimant = *claimant
	}

	if userID != nil {
		gs.userID = *userID
	}

	if sessionUserID != nil {
		gs.sessionUserID = *sessionUserID
	}

	gs.Modified = gs.Modified.In(locationNewYork)

	return &gs, nil
}

// Logs will return all pool square logs for the pool
func (p *Pool) Logs(ctx context.Context, offset, limit int) ([]*PoolSquareLog, error) {
	const query = `
		SELECT pool_squares_logs.id, pool_square_id, square_id, pool_squares_logs.user_id, pool_squares_logs.session_user_id, pool_squares_logs.state, pool_squares_logs.claimant, remote_addr, note, pool_squares_logs.created
		FROM pool_squares_logs
		INNER JOIN pool_squares ON pool_squares_logs.pool_square_id = pool_squares.id
		WHERE pool_squares.pool_id = $1
		ORDER BY pool_squares_logs.id DESC
		OFFSET $2
		LIMIT $3`
	rows, err := p.model.db.QueryContext(ctx, query, p.id, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]*PoolSquareLog, 0)
	for rows.Next() {
		l, err := poolSquareLogByRow(rows.Scan)
		if err != nil {
			return nil, err
		}

		logs = append(logs, l)
	}

	return logs, nil
}

// LogsCount will return how many logs exist for the given pool
func (p *Pool) LogsCount(ctx context.Context) (int64, error) {
	const query = `
		SELECT COUNT(pool_squares_logs.*)
		FROM pool_squares_logs
		INNER JOIN pool_squares ON pool_squares_logs.pool_square_id = pool_squares.id
		WHERE pool_squares.pool_id = $1`
	row := p.model.db.QueryRowContext(ctx, query, p.id)

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// DefaultGrid will return the default grid for the pool
func (p *Pool) DefaultGrid(ctx context.Context) (*Grid, error) {
	grids, err := p.Grids(ctx, 0, 1)
	if err != nil {
		return nil, err
	}

	if len(grids) != 1 {
		return nil, fmt.Errorf("expected only 1 grid to be returned for pool %s, but got %d", p.token, len(grids))
	}

	return grids[0], nil
}

// Grids returns all grids assigned to the pool. By default, this will only return "active" grids. Pass true to as the allStates
// argument to return grids with all states
func (p *Pool) Grids(ctx context.Context, offset, limit int, allStates ...bool) ([]*Grid, error) {
	activeOnly := len(allStates) == 0 || !allStates[0]
	stateClause := ""
	if activeOnly {
		stateClause = " AND state = 'active'"
	}

	query := `
SELECT *
FROM grids
WHERE pool_id = $1` + stateClause + `
ORDER BY ord, id
OFFSET $2
LIMIT $3
`

	rows, err := p.model.db.QueryContext(ctx, query, p.id, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grids := make([]*Grid, 0)
	for rows.Next() {
		grid, err := p.model.gridByRow(rows.Scan)
		if err != nil {
			return nil, err
		}

		grids = append(grids, grid)
	}

	return grids, nil
}

// NewGrid will create a new grid for the pool with some default settings
func (p *Pool) NewGrid(ctx context.Context) (*Grid, error) {
	const query = `SELECT * FROM new_grid($1)`
	row := p.model.db.QueryRowContext(ctx, query, p.id)
	return p.model.gridByRow(row.Scan)
}

// GridByID will return a grid by its ID and ensures that it belongs to the pool
func (p *Pool) GridByID(ctx context.Context, id int64) (*Grid, error) {
	const query = `SELECT * FROM grids WHERE id = $1 AND pool_id = $2 AND state = 'active'`
	row := p.model.db.QueryRowContext(ctx, query, id, p.id)
	return p.model.gridByRow(row.Scan)
}
