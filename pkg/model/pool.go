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
	"github.com/lib/pq"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"github.com/synacor/argon2id"
)

// NameMaxLength is the maximum length the pool name may be
const NameMaxLength = 50

// MaxGridsPerPool is the maximum number of grids we allow per pool
const MaxGridsPerPool = 50

// Pool is an individual pool board
// This object uses getters and setters to help guard against user input.
type Pool struct {
	model            *Model
	id               int64
	token            string
	userID           int64
	name             string
	gridType         GridType
	passwordHash     string
	checkID          int
	archived         bool
	passwordRequired bool
	openAccessOnLock bool
	locks            time.Time
	created          time.Time
	modified         time.Time

	squares map[int]*PoolSquare
}

// PasswordRequired returns whether the password is actually required to join the pool
func (p *Pool) PasswordRequired() bool {
	return p.passwordRequired
}

// SetPasswordRequired will set whether the password is required to join the pool
func (p *Pool) SetPasswordRequired(passwordRequired bool) {
	p.passwordRequired = passwordRequired
}

// OpenAccessOnLock returns whether a password is required
// when the pool has been locked.
func (p *Pool) OpenAccessOnLock() bool {
	return p.openAccessOnLock
}

// SetOpenAccessOnLock sets whether passwords are required
// when the pool is locked
func (p *Pool) SetOpenAccessOnLock(openAccess bool) {
	p.openAccessOnLock = openAccess
}

// Archived is the getter whether the pool has been archived
func (p *Pool) Archived() bool {
	return p.archived
}

// SetArchived is the setter for whether the pool has been archived
func (p *Pool) SetArchived(archived bool) {
	p.archived = archived
}

// CheckID will return the current check ID.
func (p *Pool) CheckID() int {
	return p.checkID
}

// IncrementCheckID will update the current check ID. A check ID can be changed
// if you want to prevent old JWT links from working.
func (p *Pool) IncrementCheckID() {
	p.checkID++
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

// PoolJSON represents an object that can be exposed to an end-user
type PoolJSON struct {
	Token            string    `json:"token"`
	Name             string    `json:"name"`
	GridType         GridType  `json:"gridType"`
	Archived         bool      `json:"archived"`
	PasswordRequired bool      `json:"passwordRequired"`
	OpenAccessOnLock bool      `json:"openAccessOnLock"`
	Locks            time.Time `json:"locks"`
	Created          time.Time `json:"created"`
	Modified         time.Time `json:"modified"`
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

// NumberOfSquares will return the number of squares in the pool
func (p *Pool) NumberOfSquares() int {
	return p.gridType.Squares()
}

// JSON returns JSON that can be sent to the front-end
func (p *Pool) JSON() *PoolJSON {
	return &PoolJSON{
		Token:            p.token,
		Name:             p.name,
		GridType:         p.gridType,
		Archived:         p.Archived(),
		PasswordRequired: p.PasswordRequired(),
		OpenAccessOnLock: p.OpenAccessOnLock(),
		Locks:            p.Locks(),
		Created:          p.created,
		Modified:         p.modified,
	}
}

func (m *Model) poolByRow(scan scanFunc) (*Pool, error) {
	pool := Pool{model: m}
	var locks *time.Time
	if err := scan(&pool.id, &pool.token, &pool.userID, &pool.name, &pool.gridType, &pool.passwordHash, &pool.passwordRequired, &pool.openAccessOnLock, &locks, &pool.created, &pool.modified, &pool.checkID, &pool.archived); err != nil {
		return nil, fmt.Errorf("scanning pool row: %w", err)
	}

	if locks != nil {
		pool.locks = locks.In(locationNewYork)
	}

	// XXX: do we want the ability to let the user choose the time zone?
	pool.created = pool.created.In(locationNewYork)
	pool.modified = pool.modified.In(locationNewYork)

	return &pool, nil
}

// PoolsJoinedByUserID will return a collection of pools that the user joined
func (m *Model) PoolsJoinedByUserID(ctx context.Context, userID int64, offset int64, limit int) ([]*Pool, error) {
	const query = `
		SELECT ` + poolColumns + `
		FROM pools
		LEFT JOIN pools_users ON pools.id = pools_users.pool_id
		WHERE pools_users.user_id = $1
		ORDER BY pools.id DESC
		OFFSET $2
		LIMIT $3`

	return m.poolsByRows(m.DB.QueryContext(ctx, query, userID, offset, limit))
}

// PoolsJoinedByUserIDCount will return a how many pools the user joined
func (m *Model) PoolsJoinedByUserIDCount(ctx context.Context, userID int64) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM pools
		LEFT JOIN pools_users ON pools.id = pools_users.pool_id
		WHERE pools_users.user_id = $1`

	return m.poolsCount(m.DB.QueryRowContext(ctx, query, userID))
}

// PoolsOwnedByUserID will return a collection of pools that were created by the user
func (m *Model) PoolsOwnedByUserID(ctx context.Context, userID int64, includeArchived bool, offset int64, limit int) ([]*Pool, error) {
	const baseQuery = `
		SELECT ` + poolColumns + `
		FROM pools
		WHERE user_id = $1%s
		ORDER BY pools.id DESC
		OFFSET $2
		LIMIT $3`

	var query string
	if includeArchived {
		query = fmt.Sprintf(baseQuery, "")
	} else {
		query = fmt.Sprintf(baseQuery, " AND archived = 'f'")
	}

	return m.poolsByRows(m.DB.QueryContext(ctx, query, userID, offset, limit))
}

// PoolsOwnedByUserIDCount will return how many pools were created by the user
func (m *Model) PoolsOwnedByUserIDCount(ctx context.Context, userID int64, includeArchived bool) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM pools
		WHERE user_id = $1`

	if !includeArchived {
		query += " AND archived = 'f'"
	}

	return m.poolsCount(m.DB.QueryRowContext(ctx, query, userID))
}

func (m *Model) poolsByRows(rows *sql.Rows, err error) ([]*Pool, error) {
	if err != nil {
		return nil, fmt.Errorf("querying pools: %w", err)
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
		return 0, fmt.Errorf("counting pools: %w", err)
	}

	return count, nil
}

// PoolByToken will return the pools with the matching token
func (m *Model) PoolByToken(ctx context.Context, token string) (*Pool, error) {
	row := m.DB.QueryRowContext(ctx, "SELECT "+poolColumns+" FROM pools WHERE token = $1", token)
	return m.poolByRow(row.Scan)
}

// PoolByID will return the pools with the matching ID
func (m *Model) PoolByID(id int64) (*Pool, error) {
	row := m.DB.QueryRow("SELECT "+poolColumns+" FROM pools WHERE id = $1", id)
	return m.poolByRow(row.Scan)
}

// NewPool will save new pool into the database
func (m *Model) NewPool(ctx context.Context, userID int64, name string, gridType GridType, password string) (*Pool, error) {
	if err := IsValidGridType(string(gridType)); err != nil {
		return nil, fmt.Errorf("validating grid type: %w", err)
	}

	token, err := m.NewToken()
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	passwordHash, err := argon2id.DefaultHashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	const query = `
		SELECT ` + poolColumns + `
		FROM new_pool($1, $2, $3, $4, $5, $6) AS pools
	`

	row := m.DB.QueryRowContext(ctx, query, token, userID, name, gridType, passwordHash, gridType.Squares())

	pool, err := m.poolByRow(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	return pool, nil
}

// SetPassword will set a new password and ensures that it's properly hashed
func (p *Pool) SetPassword(password string) error {
	passwordHash, err := argon2id.DefaultHashPassword(password)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	p.passwordHash = passwordHash
	return nil
}

// Save will save the pool
func (p *Pool) Save(ctx context.Context) error {
	const query = `
UPDATE pools
SET name = $1,
    grid_type = $2,
    password_hash = $3,
    locks = $4,
    check_id = $5,
    archived = $6,
    password_required = $7,
    open_access_on_lock = $8,
    modified = (NOW() AT TIME ZONE 'utc')
WHERE id = $9`

	var locks *time.Time
	if !p.locks.IsZero() {
		locksInUTC := p.locks.UTC()
		locks = &locksInUTC
	}

	_, err := p.model.DB.ExecContext(ctx, query, p.name, p.gridType, p.passwordHash, locks, p.checkID, p.archived, p.passwordRequired, p.openAccessOnLock, p.id)
	if err != nil {
		return fmt.Errorf("saving pool: %w", err)
	}
	return nil
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

// CheckIDIsValid will return true if the check IDs match.
// This is used to invalidate JWT links
func (p *Pool) CheckIDIsValid(check int) bool {
	return check == p.checkID
}

// Squares will return the squares that belong to a pool. This method will lazily load the squares
func (p *Pool) Squares() (map[int]*PoolSquare, error) {
	if p.squares == nil {
		const query = `
SELECT ps.id,
       ps.square_id,
       ps.parent_id,
       ps.user_id,
       ps.state,
       ps.claimant,
       ps.modified,
       ps2.square_id                                                                   AS parent_square_id,
       NULLIF(array_agg(ps3.square_id) FILTER (WHERE ps3.square_id IS NOT NULL), '{}') AS child_square_ids
FROM pool_squares ps
         LEFT JOIN pool_squares ps2 ON ps.parent_id = ps2.id
         LEFT JOIN pool_squares ps3 ON ps.id = ps3.parent_id -- bring in child squares
WHERE ps.pool_id = $1
GROUP BY ps.id,
         ps.square_id,
         ps.parent_id,
         ps.user_id,
         ps.state,
         ps.claimant,
         ps.modified,
         ps2.square_id
ORDER BY ps.square_id`

		rows, err := p.model.DB.Query(query, p.id)
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
	SELECT
	       ps.id,
	       ps.square_id,
	       ps.parent_id,
	       ps.user_id,
	       ps.state,
	       ps.claimant,
	       ps.modified,
	       ps2.square_id AS parent_square_id,
	       (SELECT array_agg(square_id) FROM pool_squares ps3 WHERE ps3.parent_id = ps.id) AS child_square_ids
	FROM pool_squares ps
	LEFT JOIN pool_squares ps2 ON ps.parent_id = ps2.id
	WHERE
	      ps.pool_id = $1 AND
	      ps.square_id = $2`

	row := p.model.DB.QueryRow(query, p.id, squareID)
	return p.squareByRow(row.Scan)
}

func (p *Pool) squareByRow(scan scanFunc) (*PoolSquare, error) {
	gs := PoolSquare{
		Model:  p.model,
		PoolID: p.id,
	}

	var claimant *string
	var userID *int64
	var parentID *int64
	var parentSquareID *int
	var childSquareIDs []sql.NullInt64
	if err := scan(&gs.ID, &gs.SquareID, &parentID, &userID, &gs.State, &claimant, &gs.Modified, &parentSquareID, pq.Array(&childSquareIDs)); err != nil {
		return nil, err
	}

	if claimant != nil {
		gs.claimant = *claimant
	}

	if userID != nil {
		gs.userID = *userID
	}

	if parentID != nil {
		gs.ParentID = *parentID
	}

	if parentSquareID != nil {
		gs.ParentSquareID = *parentSquareID
	}

	if childSquareIDs != nil {
		gs.ChildSquareIDs = make([]int8, len(childSquareIDs))
		for i, c := range childSquareIDs {
			gs.ChildSquareIDs[i] = int8(c.Int64)
		}
	}

	gs.Modified = gs.Modified.In(locationNewYork)

	return &gs, nil
}

// Logs will return all pool square logs for the pool
func (p *Pool) Logs(ctx context.Context, offset int64, limit int) ([]*PoolSquareLog, error) {
	const query = `
		SELECT pool_squares_logs.id, pool_square_id, square_id, pool_squares_logs.user_id, pool_squares_logs.state, pool_squares_logs.claimant, remote_addr, note, pool_squares_logs.created
		FROM pool_squares_logs
		INNER JOIN pool_squares ON pool_squares_logs.pool_square_id = pool_squares.id
		WHERE pool_squares.pool_id = $1
		ORDER BY pool_squares_logs.id DESC
		OFFSET $2
		LIMIT $3`
	rows, err := p.model.DB.QueryContext(ctx, query, p.id, offset, limit)
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
	row := p.model.DB.QueryRowContext(ctx, query, p.id)

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
func (p *Pool) Grids(ctx context.Context, offset int64, limit int, allStates ...bool) ([]*Grid, error) {
	activeOnly := len(allStates) == 0 || !allStates[0]
	stateClause := ""
	if activeOnly {
		stateClause = " AND state = 'active'"
	}

	query := `
SELECT ` + gridColumns + `
FROM grids
WHERE pool_id = $1` + stateClause + `
ORDER BY ord, id
OFFSET $2
LIMIT $3
`

	rows, err := p.model.DB.QueryContext(ctx, query, p.id, offset, limit)
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

// GridsCount returns the count of all grids assigned to the pool. By default, this will only return "active" grids. Pass true to as the allStates
// argument to return grids with all states
func (p *Pool) GridsCount(ctx context.Context, allStates ...bool) (int64, error) {
	activeOnly := len(allStates) == 0 || !allStates[0]
	stateClause := ""
	if activeOnly {
		stateClause = " AND state = 'active'"
	}

	query := `SELECT COUNT(*) FROM grids WHERE pool_id = $1` + stateClause

	row := p.model.DB.QueryRowContext(ctx, query, p.id)
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// SetGridsOrder will re-arrange the order of the grids
func (p *Pool) SetGridsOrder(ctx context.Context, gridIDs []int64) error {
	tx, err := p.model.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	rollback := func() {
		if err := tx.Rollback(); err != nil {
			logrus.WithError(err).Warn("could not rollback transaction")
		}
	}

	rows, err := tx.QueryContext(ctx, "SELECT id, ord FROM grids WHERE pool_id = $1", p.id)
	if err != nil {
		rollback()
		return err
	}
	defer rows.Close()

	id2ord := make(map[int64]int)
	for rows.Next() {
		var id int64
		var ord int
		if err := rows.Scan(&id, &ord); err != nil {
			rollback()
			return err
		}

		id2ord[id] = ord
	}

	// pool_id is present to ensure user has access (prevents us from having to check the owner of every grid)
	stmt, err := tx.PrepareContext(ctx, "UPDATE grids SET ord = $1, modified = (NOW() at time zone 'UTC') WHERE id = $2 AND pool_id = $3")
	if err != nil {
		return err
	}

	for ord, id := range gridIDs {
		l := logrus.WithFields(logrus.Fields{"pool_id": p.id, "grid_id": id})
		curOrd, ok := id2ord[id]
		if !ok {
			l.Warn("could not find grid in pool")
		}

		if ord != curOrd {
			result, err := stmt.ExecContext(ctx, ord, id, p.id)
			if err != nil {
				rollback()
				return err
			}

			rowsAffected, _ := result.RowsAffected()

			if rowsAffected == 0 {
				l.Warn("no rows affected")
				rollback()
				return errors.New("no rows affected")
			}
		}
	}

	return tx.Commit()
}

// NewGrid will create a new grid for the pool with some default settings
func (p *Pool) NewGrid() *Grid {
	return &Grid{
		model:    p.model,
		poolID:   p.id,
		settings: &GridSettings{},
	}
}

// GridByID will return a grid by its ID and ensures that it belongs to the pool
func (p *Pool) GridByID(ctx context.Context, id int64) (*Grid, error) {
	const query = `
	SELECT ` + gridColumns + `
	FROM
	     grids
	WHERE
	      id = $1 AND
	      pool_id = $2 AND
	      state = 'active'`
	row := p.model.DB.QueryRowContext(ctx, query, id, p.id)
	return p.model.gridByRow(row.Scan)
}

// RemoveAllMembers will boot all members from the pool
func (p *Pool) RemoveAllMembers(ctx context.Context) error {
	_, err := p.model.DB.ExecContext(ctx, "DELETE FROM pools_users WHERE pool_id = $1", p.ID())
	return err
}

const poolColumns = `
pools.id,
pools.token,
pools.user_id,
pools.name,
pools.grid_type,
pools.password_hash,
pools.password_required,
pools.open_access_on_lock,
pools.locks,
pools.created,
pools.modified,
pools.check_id,
pools.archived
`
