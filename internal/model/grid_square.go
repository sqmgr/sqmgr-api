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
	"time"
)

// GridSquareState represents the state of an individual square within a given grid
type GridSquareState string

// constants for GridSquareState
const (
	GridSquareStateUnclaimed   GridSquareState = "unclaimed"
	GridSquareStateClaimed     GridSquareState = "claimed"
	GridSquareStatePaidPartial GridSquareState = "paid-partial"
	GridSquareStatePaidFull    GridSquareState = "paid-full"
)

// GridSquareStates are the valid states of a GridSquare
var GridSquareStates = []GridSquareState{
	GridSquareStateClaimed,
	GridSquareStatePaidPartial,
	GridSquareStatePaidFull,
	GridSquareStateUnclaimed,
}

// ErrSquareAlreadyClaimed is an error when a user tries to claim a square that has already been claimed.
var ErrSquareAlreadyClaimed = errors.New("square has already been claimed")

// ValidGridSquareStates contains a map of valid states.
var ValidGridSquareStates = map[GridSquareState]bool{}

func init() {
	for _, state := range GridSquareStates {
		ValidGridSquareStates[state] = true
	}
}

// IsValid will ensure that it's a valid state
func (g GridSquareState) IsValid() bool {
	_, ok := ValidGridSquareStates[g]
	return ok
}

// GridSquare is an individual square within a grid
type GridSquare struct {
	*Model
	ID       int64            `json:"-"`
	GridID   int64            `json:"-"`
	SquareID int              `json:"squareID"`
	State    GridSquareState  `json:"state"`
	Claimant string           `json:"claimant"`
	Modified time.Time        `json:"modified"`
	Logs     []*GridSquareLog `json:"logs,omitempty"`
}

// GridSquareLog represents an individual log entry for a grid square
type GridSquareLog struct {
	id           int64
	gridSquareID int64
	squareID     int
	UserID       int64
	state        GridSquareState
	claimant     string
	RemoteAddr   string
	Note         string
	created      time.Time
}

// SquareID is a getter for the square ID
func (g *GridSquareLog) SquareID() int {
	return g.squareID
}

// Claimant is a getter for the claimant
func (g *GridSquareLog) Claimant() string {
	return g.claimant
}

type gridSquareLogJSON struct {
	SquareID   int             `json:"squareID"`
	State      GridSquareState `json:"state"`
	Claimant   string          `json:"claimant"`
	RemoteAddr string          `json:"remoteAddr"`
	Note       string          `json:"note"`
	Created    time.Time       `json:"created"`
}

// MarshalJSON will custom marshal the JSON
func (g *GridSquareLog) MarshalJSON() ([]byte, error) {
	return json.Marshal(gridSquareLogJSON{
		SquareID:   g.SquareID(),
		State:      g.State(),
		Claimant:   g.Claimant(),
		RemoteAddr: g.RemoteAddr,
		Note:       g.Note,
		Created:    g.Created(),
	})
}

// Created is a getter for created
func (g *GridSquareLog) Created() time.Time {
	return g.created
}

// State is a getter for state
func (g *GridSquareLog) State() GridSquareState {
	return g.state
}

// GridSquareID is a getter for gridSquareID
func (g *GridSquareLog) GridSquareID() int64 {
	return g.gridSquareID
}

// ID is a getter for id
func (g *GridSquareLog) ID() int64 {
	return g.id
}

// Save will save the grid square and the associated log data to the database
func (g *GridSquare) Save(ctx context.Context, isAdmin bool, gridSquareLog GridSquareLog) error {
	var claimant *string
	if g.Claimant != "" {
		claimant = &g.Claimant
	}

	var userID *int64
	var remoteAddr *string

	if gridSquareLog.UserID > 0 {
		userID = &gridSquareLog.UserID
	}

	if gridSquareLog.RemoteAddr != "" {
		remoteAddr = &gridSquareLog.RemoteAddr
	}

	const query = "SELECT * FROM update_grid_square($1, $2, $3, $4, $5, $6, $7)"
	row := g.Model.db.QueryRowContext(ctx, query, g.ID, g.State, claimant, userID, remoteAddr, gridSquareLog.Note, isAdmin)

	var ok bool
	if err := row.Scan(&ok); err != nil {
		return err
	}

	if !ok {
		return ErrSquareAlreadyClaimed
	}

	return nil
}

func gridSquareLogByRow(scan scanFunc) (*GridSquareLog, error) {
	var l GridSquareLog
	var remoteAddr *string
	var userID *int64
	var claimant *string

	if err := scan(&l.id, &l.gridSquareID, &l.squareID, &userID, &l.state, &claimant, &remoteAddr, &l.Note, &l.created); err != nil {
		return nil, err
	}

	if userID != nil {
		l.UserID = *userID
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
func (g *GridSquare) LoadLogs(ctx context.Context) error {
	const query = `
		SELECT grid_squares_logs.id, grid_square_id, square_id, user_id, grid_squares_logs.state, grid_squares_logs.claimant, remote_addr, note, grid_squares_logs.created
		FROM grid_squares_logs
		INNER JOIN grid_squares ON grid_squares_logs.grid_square_id = grid_squares.id
		WHERE grid_square_id = $1 
		ORDER BY id DESC`
	rows, err := g.Model.db.QueryContext(ctx, query, g.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	logs := make([]*GridSquareLog, 0)
	for rows.Next() {
		l, err := gridSquareLogByRow(rows.Scan)
		if err != nil {
			return err
		}

		logs = append(logs, l)
	}

	g.Logs = logs
	return nil
}
