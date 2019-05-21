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
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

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
	ID            int64 `json:"-"`
	PoolID        int64 `json:"-"`
	userID        int64
	sessionUserID string
	SquareID      int              `json:"squareID"`
	State         PoolSquareState  `json:"state"`
	Claimant      string           `json:"claimant"`
	Modified      time.Time        `json:"modified"`
	Logs          []*PoolSquareLog `json:"logs,omitempty"`
}

type poolSquareJSON struct {
	OpaqueUserID string           `json:"opaqueUserID"`
	SquareID     int              `json:"squareID"`
	State        PoolSquareState  `json:"state"`
	Claimant     string           `json:"claimant"`
	Modified     time.Time        `json:"modified"`
	Logs         []*PoolSquareLog `json:"logs,omitempty"`
}

// MarshalJSON will custom JSON encode a PoolSquare
func (p *PoolSquare) MarshalJSON() ([]byte, error) {
	oid, err := opaqueID(p.UserIdentifier())
	if err != nil {
		return nil, err
	}

	return json.Marshal(poolSquareJSON{
		OpaqueUserID: oid,
		SquareID:     p.SquareID,
		State:        p.State,
		Claimant:     p.Claimant,
		Modified:     p.Modified,
		Logs:         p.Logs,
	})
}

// PoolSquareLog represents an individual log entry for a pool square
type PoolSquareLog struct {
	id            int64
	poolSquareID  int64
	squareID      int
	userID        int64
	sessionUserID string
	state         PoolSquareState
	claimant      string
	RemoteAddr    string
	Note          string
	created       time.Time
}

// SetUserIdentifier will allow you to set either the userID (int64) or the sessionUserID (string)
func (p *PoolSquare) SetUserIdentifier(uid interface{}) {
	switch val := uid.(type) {
	case int64:
		p.userID = val
	case string:
		p.sessionUserID = val
	default:
		panic(fmt.Sprintf("invalid userID type %T", uid))
	}
}

// UserIdentifier will return the appropriate ID
func (p *PoolSquare) UserIdentifier() interface{} {
	if p.userID > 0 {
		return p.userID
	}

	return p.sessionUserID
}

// SquareID is a getter for the square ID
func (p *PoolSquareLog) SquareID() int {
	return p.squareID
}

// Claimant is a getter for the claimant
func (p *PoolSquareLog) Claimant() string {
	return p.claimant
}

type poolSquareLogJSON struct {
	SquareID   int             `json:"squareID"`
	State      PoolSquareState `json:"state"`
	Claimant   string          `json:"claimant"`
	RemoteAddr string          `json:"remoteAddr"`
	Note       string          `json:"note"`
	Created    time.Time       `json:"created"`
}

// MarshalJSON will custom marshal the JSON
func (p *PoolSquareLog) MarshalJSON() ([]byte, error) {
	return json.Marshal(poolSquareLogJSON{
		SquareID:   p.SquareID(),
		State:      p.State(),
		Claimant:   p.Claimant(),
		RemoteAddr: p.RemoteAddr,
		Note:       p.Note,
		Created:    p.Created(),
	})
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

// Save will save the pool square and the associated log data to the database
func (p *PoolSquare) Save(ctx context.Context, isAdmin bool, poolSquareLog PoolSquareLog) error {
	var claimant *string
	if p.Claimant != "" {
		claimant = &p.Claimant
	}

	var userID *int64
	var sessionUserID *string
	var remoteAddr *string

	if p.userID > 0 {
		userID = &p.userID
	}

	if p.sessionUserID != "" {
		sessionUserID = &p.sessionUserID
	}

	if poolSquareLog.RemoteAddr != "" {
		remoteAddr = &poolSquareLog.RemoteAddr
	}

	const query = "SELECT * FROM update_pool_square($1, $2, $3, $4, $5, $6, $7, $8)"
	row := p.Model.db.QueryRowContext(ctx, query, p.ID, p.State, claimant, userID, sessionUserID, remoteAddr, poolSquareLog.Note, isAdmin)

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
	var sessionUserID *string
	var claimant *string

	if err := scan(&l.id, &l.poolSquareID, &l.squareID, &userID, &sessionUserID, &l.state, &claimant, &remoteAddr, &l.Note, &l.created); err != nil {
		return nil, err
	}

	if userID != nil {
		l.userID = *userID
	}

	if sessionUserID != nil {
		l.sessionUserID = *sessionUserID
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
		SELECT pool_squares_logs.id, pool_square_id, square_id, pool_squares_logs.user_id, pool_squares_logs.session_user_id, pool_squares_logs.state, pool_squares_logs.claimant, remote_addr, note, pool_squares_logs.created
		FROM pool_squares_logs
		INNER JOIN pool_squares ON pool_squares_logs.pool_square_id = pool_squares.id
		WHERE pool_square_id = $1 
		ORDER BY id DESC`
	rows, err := p.Model.db.QueryContext(ctx, query, p.ID)
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
