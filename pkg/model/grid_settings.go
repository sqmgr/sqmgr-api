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
	"time"
	"unicode/utf8"
)

const (
	// NotesMaxLength is the maximum number of characters the notes can be
	NotesMaxLength = 500
)

// constants for default colors
const (
	DefaultHomeTeamColor1 = "#555555"
	DefaultHomeTeamColor2 = "#999999"
	DefaultAwayTeamColor1 = "#666666"
	DefaultAwayTeamColor2 = "#333333"
)

// GridSettings will contain various user-defined settings
// This object uses getters and setters to help guard against user input.
type GridSettings struct {
	gridID         int64
	homeTeamColor1 *string
	homeTeamColor2 *string
	awayTeamColor1 *string
	awayTeamColor2 *string
	notes          *string
	modified       *time.Time
}

// gridSettingsJSON is used for custom serialization
type gridSettingsJSON struct {
	HomeTeamColor1 string `json:"homeTeamColor1"`
	HomeTeamColor2 string `json:"homeTeamColor2"`
	AwayTeamColor1 string `json:"awayTeamColor1"`
	AwayTeamColor2 string `json:"awayTeamColor2"`
	Notes          string `json:"notes"`
}

// MarshalJSON adds custom JSON marshalling support
func (g GridSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(gridSettingsJSON{
		HomeTeamColor1: g.HomeTeamColor1(),
		HomeTeamColor2: g.HomeTeamColor2(),
		AwayTeamColor1: g.AwayTeamColor1(),
		AwayTeamColor2: g.AwayTeamColor2(),
		Notes:          g.Notes(),
	})
}

// Save will save the settings
func (g *GridSettings) Save(ctx context.Context, q Queryable) error {
	_, err := q.ExecContext(ctx, `
		UPDATE grid_settings SET
			home_team_color_1 = $1,
			home_team_color_2 = $2,
			away_team_color_1 = $3,
			away_team_color_2 = $4,
			notes = $5,
			modified = (NOW() AT TIME ZONE 'utc')
		WHERE grid_id = $6
	`,
		g.homeTeamColor1,
		g.homeTeamColor2,
		g.awayTeamColor1,
		g.awayTeamColor2,
		g.notes,
		g.gridID,
	)

	return err
}

// SetNotes will set the notes of the grid
func (g *GridSettings) SetNotes(str string) {
	if len(str) == 0 {
		g.notes = nil
		return
	}

	nRunes := utf8.RuneCountInString(str)
	if nRunes > NotesMaxLength {
		strChars := []rune(str)
		str = string(strChars[0:NotesMaxLength])
	}

	g.notes = &str
}

// Notes returns the notes
func (g *GridSettings) Notes() string {
	if g.notes == nil {
		return ""
	}

	return *g.notes
}

// SetHomeTeamColor1 is a setter for the home team primary color
func (g *GridSettings) SetHomeTeamColor1(color string) {
	if color == "" {
		g.homeTeamColor1 = nil
		return
	}

	g.homeTeamColor1 = &color
}

// SetHomeTeamColor2 is a setter for the home team secondary color
func (g *GridSettings) SetHomeTeamColor2(color string) {
	if color == "" {
		g.homeTeamColor2 = nil
		return
	}

	g.homeTeamColor2 = &color
}

// SetAwayTeamColor1 is a setter for the away team primary color
func (g *GridSettings) SetAwayTeamColor1(color string) {
	if color == "" {
		g.awayTeamColor1 = nil
		return
	}

	g.awayTeamColor1 = &color
}

// SetAwayTeamColor2 is a setter for the away team secondary color
func (g *GridSettings) SetAwayTeamColor2(color string) {
	if color == "" {
		g.awayTeamColor2 = nil
		return
	}

	g.awayTeamColor2 = &color
}

// HomeTeamColor1 is a getter for the home team primary color
func (g *GridSettings) HomeTeamColor1() string {
	if g.homeTeamColor1 == nil {
		return DefaultHomeTeamColor1
	}

	return *g.homeTeamColor1
}

// HomeTeamColor2 is a getter for the home team secondary color
func (g *GridSettings) HomeTeamColor2() string {
	if g.homeTeamColor2 == nil {
		return DefaultHomeTeamColor2
	}

	return *g.homeTeamColor2
}

// AwayTeamColor1 is a getter for the away team primary color
func (g *GridSettings) AwayTeamColor1() string {
	if g.awayTeamColor1 == nil {
		return DefaultAwayTeamColor1
	}

	return *g.awayTeamColor1
}

// AwayTeamColor2 is a getter for the away team secondary color
func (g *GridSettings) AwayTeamColor2() string {
	if g.awayTeamColor2 == nil {
		return DefaultAwayTeamColor2
	}

	return *g.awayTeamColor2
}
