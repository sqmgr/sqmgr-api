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
	"errors"
	"time"
	"unicode/utf8"
)

// ClaimantMaxLength is the maximum number of characters allowed in a claimant name
const ClaimantMaxLength = 30

// PoolSquareState represents the state of an individual square within a given pool
type PoolSquareState string

// constants for PoolSquareState
const (
	PoolSquareStateUnclaimed   PoolSquareState = "unclaimed"
	PoolSquareStateClaimed     PoolSquareState = "claimed"
	PoolSquareStatePaidPartial PoolSquareState = "paid-partial"
	PoolSquareStatePaidFull    PoolSquareState = "paid-full"
)

// PoolSquareStates are the valid states of a PoolSquare
var PoolSquareStates = []PoolSquareState{
	PoolSquareStateClaimed,
	PoolSquareStatePaidPartial,
	PoolSquareStatePaidFull,
	PoolSquareStateUnclaimed,
}

// ErrSquareAlreadyClaimed is an error when a user tries to claim a square that has already been claimed.
var ErrSquareAlreadyClaimed = errors.New("square has already been claimed")

// ValidPoolSquareStates contains a map of valid states.
var ValidPoolSquareStates = map[PoolSquareState]bool{}

func init() {
	for _, state := range PoolSquareStates {
		ValidPoolSquareStates[state] = true
	}
}

// IsValid will ensure that it's a valid state
func (g PoolSquareState) IsValid() bool {
	_, ok := ValidPoolSquareStates[g]
	return ok
}

// PoolSquare is an individual square within a pool
type PoolSquare struct {
	*Model
	ID             int64 `json:"-"`
	ParentID       int64 `json:"-"`
	PoolID         int64 `json:"-"`
	userID         int64
	SquareID       int             `json:"-"`
	ParentSquareID int             `json:"-"`
	ChildSquareIDs []int8          `json:"-"`
	State          PoolSquareState `json:"-"`
	claimant       string
	Modified       time.Time        `json:"-"`
	Logs           []*PoolSquareLog `json:"-"`
}

// FIXME - remove the above json tags once we validate it's no longer necessary

// Claimant returns the claimant
func (p *PoolSquare) Claimant() string {
	return p.claimant
}

// SetClaimant will set the claimant and clamp the length to at most N runes.
func (p *PoolSquare) SetClaimant(claimant string) {
	if utf8.RuneCountInString(claimant) > ClaimantMaxLength {
		claimant = string([]rune(claimant)[0:ClaimantMaxLength])
	}

	p.claimant = claimant
}

// UserID is a getter
func (p *PoolSquare) UserID() int64 {
	return p.userID
}

// SetUserID is a setter
func (p *PoolSquare) SetUserID(userID int64) {
	p.userID = userID
}

// PoolSquareJSON represents JSON that can be sent to the front-end
type PoolSquareJSON struct {
	UserID         int64            `json:"userId"`
	SquareID       int              `json:"squareId"`
	ParentSquareID int              `json:"parentSquareId"`
	ChildSquareIDs []int8           `json:"childSquareIds"`
	State          PoolSquareState  `json:"state"`
	Claimant       string           `json:"claimant"`
	Modified       time.Time        `json:"modified"`
	Logs           []*PoolSquareLog `json:"logs,omitempty"`
}

// JSON will custom JSON encode a PoolSquare
func (p *PoolSquare) JSON() *PoolSquareJSON {
	return &PoolSquareJSON{
		UserID:         p.userID,
		SquareID:       p.SquareID,
		ParentSquareID: p.ParentSquareID,
		ChildSquareIDs: p.ChildSquareIDs,
		State:          p.State,
		Claimant:       p.Claimant(),
		Modified:       p.Modified,
		Logs:           p.Logs,
	}
}

// PoolSquareLog represents an individual log entry for a pool square
type PoolSquareLog struct {
	id           int64
	poolSquareID int64
	squareID     int
	userID       int64
	state        PoolSquareState
	claimant     string
	RemoteAddr   string
	Note         string
	created      time.Time
}

// SquareID is a getter for the square ID
func (p *PoolSquareLog) SquareID() int {
	return p.squareID
}

// Claimant is a getter for the claimant
func (p *PoolSquareLog) Claimant() string {
	return p.claimant
}

// PoolSquareLogJSON returns data safe for a user to see
type PoolSquareLogJSON struct {
	SquareID int             `json:"squareID"`
	State    PoolSquareState `json:"state"`
	Claimant string          `json:"claimant"`
	Note     string          `json:"note"`
	Created  time.Time       `json:"created"`
}

// JSON will return data safe for the front-end
func (p *PoolSquareLog) JSON() *PoolSquareLogJSON {
	return &PoolSquareLogJSON{
		SquareID: p.SquareID(),
		State:    p.State(),
		Claimant: p.Claimant(),
		Note:     p.Note,
		Created:  p.Created(),
	}
}

// MarshalJSON will return data safe for the front-end
func (p *PoolSquareLog) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.JSON())
}

// Created is a getter for created
func (p *PoolSquareLog) Created() time.Time {
	return p.created
}

// State is a getter for state
func (p *PoolSquareLog) State() PoolSquareState {
	return p.state
}

// PoolSquareID is a getter for poolSquareID
func (p *PoolSquareLog) PoolSquareID() int64 {
	return p.poolSquareID
}

// ID is a getter for id
func (p *PoolSquareLog) ID() int64 {
	return p.id
}

// SetParentSquare will set the parent square
func (p *PoolSquare) SetParentSquare(ctx context.Context, tx *sql.Tx, square *PoolSquare) error {
	_, err := tx.ExecContext(ctx, "UPDATE pool_squares SET parent_id = $1 WHERE id = $2", square.ID, p.ID)
	return err
}

// Save will save the pool square and the associated log data to the database
func (p *PoolSquare) Save(ctx context.Context, dbFn Queryable, isAdmin bool, poolSquareLog PoolSquareLog) error {
	var claimant *string
	if p.claimant != "" {
		claimant = &p.claimant
	}

	var userID *int64
	var remoteAddr *string

	if p.userID > 0 {
		userID = &p.userID
	}

	if poolSquareLog.RemoteAddr != "" {
		ip := ipFromRemoteAddr(poolSquareLog.RemoteAddr)
		remoteAddr = &ip
	}

	const query = "SELECT * FROM update_pool_square($1, $2, $3, $4, $5, $6, $7)"
	row := dbFn.QueryRowContext(ctx, query, p.ID, p.State, claimant, userID, remoteAddr, poolSquareLog.Note, isAdmin)

	var ok bool
	if err := row.Scan(&ok); err != nil {
		return err
	}

	if !ok {
		return ErrSquareAlreadyClaimed
	}

	return nil
}

func poolSquareLogByRow(scan scanFunc) (*PoolSquareLog, error) {
	var l PoolSquareLog
	var remoteAddr *string
	var userID *int64
	var claimant *string

	if err := scan(&l.id, &l.poolSquareID, &l.squareID, &userID, &l.state, &claimant, &remoteAddr, &l.Note, &l.created); err != nil {
		return nil, err
	}

	if userID != nil {
		l.userID = *userID
	}

	if remoteAddr != nil {
		l.RemoteAddr = *remoteAddr
	}

	if claimant != nil {
		l.claimant = *claimant
	}

	l.created = l.created.In(locationNewYork)

	return &l, nil
}

// LoadLogs will load the logs for the given square
func (p *PoolSquare) LoadLogs(ctx context.Context) error {
	const query = `
		SELECT
		       pool_squares_logs.id,
		       pool_square_id,
		       square_id,
		       pool_squares_logs.user_id,
		       pool_squares_logs.state,
		       pool_squares_logs.claimant,
		       remote_addr, note,
		       pool_squares_logs.created
		FROM
		     pool_squares_logs
		INNER JOIN
		         pool_squares ON pool_squares_logs.pool_square_id = pool_squares.id
		WHERE
		      pool_square_id = $1 
		ORDER BY
		         id DESC`
	rows, err := p.Model.DB.QueryContext(ctx, query, p.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	logs := make([]*PoolSquareLog, 0)
	for rows.Next() {
		l, err := poolSquareLogByRow(rows.Scan)
		if err != nil {
			return err
		}

		logs = append(logs, l)
	}

	p.Logs = logs
	return nil
}

// ChildSquares returns the children of the current square
func (p *PoolSquare) ChildSquares(ctx context.Context, q Queryable) ([]*PoolSquare, error) {
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
		FROM
			pool_squares ps
		LEFT JOIN
			pool_squares ps2 ON ps.parent_id = ps2.id
		WHERE
			ps.parent_id = $1
		ORDER BY
			ps.square_id`
	rows, err := q.QueryContext(ctx, query, p.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// this is a little hacky. it's done to be able to call squareByRow()
	pool := &Pool{model: p.Model, id: p.PoolID}

	squares := make([]*PoolSquare, 0)
	for rows.Next() {
		square, err := pool.squareByRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		squares = append(squares, square)
	}

	return squares, nil
}
