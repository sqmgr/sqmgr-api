/*
Copyright (C) 2024 Tom Peters

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
	"fmt"
	"time"

	"github.com/lib/pq"
)

// GridNumberSet represents a set of numbers for a specific quarter/period
type GridNumberSet struct {
	model       *Model
	id          int64
	gridID      int64
	setType     NumberSetType
	homeNumbers []int
	awayNumbers []int
	manualDraw  bool
	created     time.Time
	modified    time.Time
}

// GridNumberSetJSON represents the JSON format for a number set
type GridNumberSetJSON struct {
	ID          int64         `json:"id"`
	SetType     NumberSetType `json:"setType"`
	HomeNumbers []int         `json:"homeNumbers"`
	AwayNumbers []int         `json:"awayNumbers"`
	ManualDraw  bool          `json:"manualDraw"`
	Created     time.Time     `json:"created"`
	Modified    time.Time     `json:"modified"`
}

// JSON returns the JSON representation of the grid number set
func (g *GridNumberSet) JSON() *GridNumberSetJSON {
	return &GridNumberSetJSON{
		ID:          g.id,
		SetType:     g.setType,
		HomeNumbers: g.homeNumbers,
		AwayNumbers: g.awayNumbers,
		ManualDraw:  g.manualDraw,
		Created:     g.created,
		Modified:    g.modified,
	}
}

// ID returns the ID
func (g *GridNumberSet) ID() int64 {
	return g.id
}

// SetType returns the set type
func (g *GridNumberSet) SetType() NumberSetType {
	return g.setType
}

// HomeNumbers returns the home team numbers
func (g *GridNumberSet) HomeNumbers() []int {
	return g.homeNumbers
}

// AwayNumbers returns the away team numbers
func (g *GridNumberSet) AwayNumbers() []int {
	return g.awayNumbers
}

// ManualDraw returns whether this was a manual draw
func (g *GridNumberSet) ManualDraw() bool {
	return g.manualDraw
}

// HasNumbers returns true if numbers have been drawn
func (g *GridNumberSet) HasNumbers() bool {
	return g.homeNumbers != nil && g.awayNumbers != nil
}

// SetNumbers sets the numbers manually
func (g *GridNumberSet) SetNumbers(homeNumbers, awayNumbers []int) error {
	if g.HasNumbers() {
		return ErrNumbersAlreadyDrawn
	}

	if !numbersAreValid(homeNumbers) || !numbersAreValid(awayNumbers) {
		return ErrNumbersAreInvalid
	}

	g.manualDraw = true
	g.homeNumbers = homeNumbers
	g.awayNumbers = awayNumbers

	return nil
}

// SelectRandomNumbers randomly assigns numbers
func (g *GridNumberSet) SelectRandomNumbers() error {
	if g.HasNumbers() {
		return ErrNumbersAlreadyDrawn
	}

	hNums, err := randomNumbers()
	if err != nil {
		return err
	}

	g.homeNumbers = hNums

	aNums, err := randomNumbers()
	if err != nil {
		return err
	}

	g.awayNumbers = aNums
	g.manualDraw = false

	return nil
}

// Save saves the grid number set to the database
func (g *GridNumberSet) Save(ctx context.Context, tx *sql.Tx) error {
	if g.id == 0 {
		// Insert new record
		const query = `
			INSERT INTO grid_number_sets (grid_id, set_type, home_numbers, away_numbers, manual_draw)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created, modified
		`
		var row *sql.Row
		if tx != nil {
			row = tx.QueryRowContext(ctx, query, g.gridID, g.setType, pq.Array(g.homeNumbers), pq.Array(g.awayNumbers), g.manualDraw)
		} else {
			row = g.model.DB.QueryRowContext(ctx, query, g.gridID, g.setType, pq.Array(g.homeNumbers), pq.Array(g.awayNumbers), g.manualDraw)
		}
		if err := row.Scan(&g.id, &g.created, &g.modified); err != nil {
			return fmt.Errorf("inserting grid number set: %w", err)
		}
	} else {
		// Update existing record
		const query = `
			UPDATE grid_number_sets
			SET home_numbers = $1, away_numbers = $2, manual_draw = $3, modified = (NOW() AT TIME ZONE 'utc')
			WHERE id = $4
		`
		var err error
		if tx != nil {
			_, err = tx.ExecContext(ctx, query, pq.Array(g.homeNumbers), pq.Array(g.awayNumbers), g.manualDraw, g.id)
		} else {
			_, err = g.model.DB.ExecContext(ctx, query, pq.Array(g.homeNumbers), pq.Array(g.awayNumbers), g.manualDraw, g.id)
		}
		if err != nil {
			return fmt.Errorf("updating grid number set: %w", err)
		}
	}
	return nil
}

// GridNumberSetsByGridID loads all number sets for a grid
func (m *Model) GridNumberSetsByGridID(ctx context.Context, gridID int64) (map[NumberSetType]*GridNumberSet, error) {
	const query = `
		SELECT id, grid_id, set_type, home_numbers, away_numbers, manual_draw, created, modified
		FROM grid_number_sets
		WHERE grid_id = $1
	`

	rows, err := m.DB.QueryContext(ctx, query, gridID)
	if err != nil {
		return nil, fmt.Errorf("querying grid number sets: %w", err)
	}
	defer rows.Close()

	result := make(map[NumberSetType]*GridNumberSet)
	for rows.Next() {
		gns, err := m.gridNumberSetByRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		result[gns.setType] = gns
	}

	return result, nil
}

func (m *Model) gridNumberSetByRow(scan scanFunc) (*GridNumberSet, error) {
	gns := &GridNumberSet{model: m}
	var homeNumbers, awayNumbers []sql.NullInt64

	if err := scan(&gns.id, &gns.gridID, &gns.setType, pq.Array(&homeNumbers), pq.Array(&awayNumbers), &gns.manualDraw, &gns.created, &gns.modified); err != nil {
		return nil, fmt.Errorf("scanning grid number set: %w", err)
	}

	if homeNumbers != nil {
		gns.homeNumbers = make([]int, len(homeNumbers))
		for i, val := range homeNumbers {
			gns.homeNumbers[i] = int(val.Int64)
		}
	}

	if awayNumbers != nil {
		gns.awayNumbers = make([]int, len(awayNumbers))
		for i, val := range awayNumbers {
			gns.awayNumbers[i] = int(val.Int64)
		}
	}

	gns.created = gns.created.In(locationNewYork)
	gns.modified = gns.modified.In(locationNewYork)

	return gns, nil
}

// NewGridNumberSet creates a new grid number set
func (m *Model) NewGridNumberSet(gridID int64, setType NumberSetType) *GridNumberSet {
	return &GridNumberSet{
		model:   m,
		gridID:  gridID,
		setType: setType,
	}
}
