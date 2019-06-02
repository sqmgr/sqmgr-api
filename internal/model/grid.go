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
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"math/big"
	"time"
	"unicode/utf8"
)

// TeamNameMaxLength is the maximum length of the team names
const TeamNameMaxLength = 50

const (
	defaultHomeTeamName = "Home Team"
	defaultAwayTeamName = "Away Team"
)

// ErrNumbersAlreadyDrawn happens when SelectRandomNumbers() is called multiple times
var ErrNumbersAlreadyDrawn = errors.New("error: numbers have already been drawn")

// Grid represents a single grid from a pool. A pool may contain more than one grid.
type Grid struct {
	model *Model

	id           int64
	poolID       int64
	ord          int
	homeTeamName *string
	homeNumbers  []int
	awayTeamName *string
	awayNumbers  []int
	eventDate    time.Time
	created      time.Time
	modified     time.Time

	settings *GridSettings
}

type gridJSON struct {
	ID           int64         `json:"id"`
	Name         string        `json:"name"`
	HomeTeamName string        `json:"homeTeamName"`
	HomeNumbers  []int         `json:"homeNumbers"`
	AwayTeamName string        `json:"awayTeamName"`
	AwayNumbers  []int         `json:"awayNumbers"`
	EventDate    time.Time     `json:"eventDate"`
	Created      time.Time     `json:"created"`
	Modified     time.Time     `json:"modified"`
	Settings     *GridSettings `json:"settings"`
}

// MarshalJSON will marshal the JSON using a custom marshaller
func (g *Grid) MarshalJSON() ([]byte, error) {
	return json.Marshal(gridJSON{
		ID:           g.ID(),
		Name:         g.Name(),
		HomeTeamName: g.HomeTeamName(),
		HomeNumbers:  g.HomeNumbers(),
		AwayTeamName: g.AwayTeamName(),
		AwayNumbers:  g.AwayNumbers(),
		EventDate:    g.EventDate(),
		Created:      g.Created(),
		Modified:     g.modified,
		Settings:     g.settings,
	})
}

// SetEventDate is a setter for the event date
func (g *Grid) SetEventDate(eventDate time.Time) {
	g.eventDate = eventDate
}

// AwayTeamName is a getter for the away team name
func (g *Grid) AwayTeamName() string {
	if g.awayTeamName == nil {
		return defaultAwayTeamName
	}

	return *g.awayTeamName
}

// SetAwayTeamName is the setter for the away team name
func (g *Grid) SetAwayTeamName(awayTeamName string) {
	if awayTeamName == "" {
		g.awayTeamName = nil
		return
	}

	if utf8.RuneCountInString(awayTeamName) > TeamNameMaxLength {
		awayTeamName = string([]rune(awayTeamName)[0:TeamNameMaxLength])
	}

	g.awayTeamName = &awayTeamName
}

// HomeTeamName is a getter for the home team name
func (g *Grid) HomeTeamName() string {
	if g.homeTeamName == nil {
		return defaultHomeTeamName
	}

	return *g.homeTeamName
}

// SetHomeTeamName is a setter for the home team name
func (g *Grid) SetHomeTeamName(homeTeamName string) {
	if homeTeamName == "" {
		g.homeTeamName = nil
		return
	}

	if utf8.RuneCountInString(homeTeamName) > TeamNameMaxLength {
		homeTeamName = string([]rune(homeTeamName)[0:TeamNameMaxLength])
	}

	g.homeTeamName = &homeTeamName
}

// ID returns the grid ID
func (g *Grid) ID() int64 {
	return g.id
}

// Created returns the created timestamp
func (g *Grid) Created() time.Time {
	return g.created
}

// EventDate returns the date of the event
func (g *Grid) EventDate() time.Time {
	return g.eventDate
}

// AwayNumbers returns the numbers to be used for the away team
func (g *Grid) AwayNumbers() []int {
	return g.awayNumbers
}

// HomeNumbers returns the numbers to be used for the home team
func (g *Grid) HomeNumbers() []int {
	return g.homeNumbers
}

// Save will save the grid. It will also save any dependent objects
func (g *Grid) Save(ctx context.Context) error {
	const query = `
		UPDATE grids
		SET ord = $1,
		    home_team_name = $2,
			home_numbers = $3,
		    away_team_name = $4,
			away_numbers = $5,
			event_date = $6,
			modified = (now() at time zone 'utc')
		WHERE id = $7
	`

	tx, err := g.model.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if g.settings != nil {
		if err := g.settings.Save(ctx, tx); err != nil {
			if err2 := tx.Rollback(); err2 != nil {
				return fmt.Errorf("error found: %#v. Another error found when trying to rollback: %#v", err, err2)
			}

			return err
		}
	}

	var eventDate *time.Time
	if !g.eventDate.IsZero() {
		eventDate = &g.eventDate
	}

	if _, err := tx.ExecContext(ctx, query, g.ord, g.homeTeamName, pq.Array(g.homeNumbers), g.awayTeamName, pq.Array(g.awayNumbers), eventDate, g.id); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			return fmt.Errorf("error found: %#v. Another error found when trying to rollback: %#v", err, err2)
		}

		return err
	}

	return tx.Commit()
}

// Settings will return the settings
func (g *Grid) Settings() *GridSettings {
	return g.settings
}

// Name returns the name of the grid
func (g *Grid) Name() string {
	return fmt.Sprintf("%s vs. %s", g.AwayTeamName(), g.HomeTeamName())
}

// SelectRandomNumbers will select random numbers for the home and away team
func (g *Grid) SelectRandomNumbers() error {
	if g.homeNumbers != nil || g.awayNumbers != nil {
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

	return nil
}

// LoadSettings will load the settings
func (g *Grid) LoadSettings(ctx context.Context) error {
	row := g.model.db.QueryRowContext(ctx, `
		SELECT grid_id,
			   home_team_color_1, home_team_color_2,
			   away_team_color_1, away_team_color_2,
			   notes, modified
		FROM grid_settings
		WHERE grid_id = $1
	`, g.id)

	if g.settings == nil {
		g.settings = &GridSettings{}
	}

	return row.Scan(
		&g.settings.gridID,
		&g.settings.homeTeamColor1,
		&g.settings.homeTeamColor2,
		&g.settings.awayTeamColor1,
		&g.settings.awayTeamColor2,
		&g.settings.notes,
		&g.settings.modified,
	)
}

func randomNumbers() ([]int, error) {
	nums := make([]int, 10)
	for i := range nums {
		nums[i] = i
	}

	for i := len(nums) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return nil, err
		}
		j := int(jBig.Int64())

		nums[i], nums[j] = nums[j], nums[i]
	}

	return nums, nil
}

func (m *Model) gridByRow(scan scanFunc) (*Grid, error) {
	grid := &Grid{model: m}

	var homeNumbers, awayNumbers []sql.NullInt64
	var eventDate *time.Time

	if err := scan(&grid.id, &grid.poolID, &grid.ord, &grid.homeTeamName, pq.Array(&homeNumbers), &grid.awayTeamName, pq.Array(&awayNumbers), &eventDate, &grid.created, &grid.modified); err != nil {
		return nil, err
	}

	if homeNumbers != nil {
		grid.homeNumbers = make([]int, len(homeNumbers))
		for i, val := range homeNumbers {
			grid.homeNumbers[i] = int(val.Int64)
		}
	}

	if awayNumbers != nil {
		grid.awayNumbers = make([]int, len(awayNumbers))
		for i, val := range awayNumbers {
			grid.awayNumbers[i] = int(val.Int64)
		}
	}

	if eventDate != nil {
		grid.eventDate = *eventDate
	}

	grid.modified = grid.modified.In(locationNewYork)
	grid.created = grid.created.In(locationNewYork)

	return grid, nil
}
