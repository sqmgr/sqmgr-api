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
	"encoding/json"
	"time"
	"unicode/utf8"
)

const (
	// NotesMaxLength is the maximum number of characters the notes can be
	NotesMaxLength = 500

	// TeamNameMaxLength is the maximum length of the team names
	TeamNameMaxLength = 50
)

// constants for default colors
const (
	DefaultHomeTeamName   = "Home Team"
	DefaultHomeTeamColor1 = "#00338d"
	DefaultHomeTeamColor2 = "#c60c30"
	DefaultAwayTeamName   = "Away Team"
	DefaultAwayTeamColor1 = "#f3d03e"
	DefaultAwayTeamColor2 = "#003087"
)

// SquaresSettings will contain various user-defined settings
// This object uses getters and setters to help guard against user input.
type SquaresSettings struct {
	squaresID      int64
	homeTeamName   *string
	homeTeamColor1 *string
	homeTeamColor2 *string
	awayTeamName   *string
	awayTeamColor1 *string
	awayTeamColor2 *string
	notes          *string
	modified       *time.Time
}

// squaresSettingsJSON is used for custom serialization
type squaresSettingsJSON struct {
	HomeTeamName   string `json:"homeTeamName"`
	HomeTeamColor1 string `json:"homeTeamColor1"`
	HomeTeamColor2 string `json:"homeTeamColor2"`
	AwayTeamName   string `json:"awayTeamName"`
	AwayTeamColor1 string `json:"awayTeamColor1"`
	AwayTeamColor2 string `json:"awayTeamColor2"`
	Notes          string `json:"notes"`
}

// MarshalJSON adds custom JSON marshalling support
func (s SquaresSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(squaresSettingsJSON{
		HomeTeamName:   s.HomeTeamName(),
		HomeTeamColor1: s.HomeTeamColor1(),
		HomeTeamColor2: s.HomeTeamColor2(),
		AwayTeamName:   s.AwayTeamName(),
		AwayTeamColor1: s.AwayTeamColor1(),
		AwayTeamColor2: s.AwayTeamColor2(),
		Notes:          s.Notes(),
	})
}

// Save will save the settings
func (s *SquaresSettings) Save(q executer) error {
	_, err := q.Exec(`
		UPDATE squares_settings SET
			home_team_name = $1,
			home_team_color_1 = $2,
			home_team_color_2 = $3,
			away_team_name = $4,
			away_team_color_1 = $5,
			away_team_color_2 = $6,
			notes = $7,
			modified = (NOW() AT TIME ZONE 'utc')
		WHERE squares_id = $8
	`,
		s.homeTeamName,
		s.homeTeamColor1,
		s.homeTeamColor2,
		s.awayTeamName,
		s.awayTeamColor1,
		s.awayTeamColor2,
		s.notes,
		s.squaresID,
	)

	return err
}

// SetNotes will set the notes of the squares
func (s *SquaresSettings) SetNotes(str string) {
	if len(str) == 0 {
		s.notes = nil
		return
	}

	nRunes := utf8.RuneCountInString(str)
	if nRunes > NotesMaxLength {
		strChars := []rune(str)
		str = string(strChars[0:NotesMaxLength])
	}

	s.notes = &str
}

// Notes returns the notes
func (s *SquaresSettings) Notes() string {
	if s.notes == nil {
		return ""
	}

	return *s.notes
}

// SetHomeTeamName is a setter for the home team name
func (s *SquaresSettings) SetHomeTeamName(name string) {
	if name == "" {
		s.homeTeamName = nil
		return
	}

	if utf8.RuneCountInString(name) > TeamNameMaxLength {
		name = string([]rune(name)[0:TeamNameMaxLength])
	}

	s.homeTeamName = &name
}

// SetHomeTeamColor1 is a setter for the home team primary color
func (s *SquaresSettings) SetHomeTeamColor1(color string) {
	if color == "" {
		s.homeTeamColor1 = nil
		return
	}

	s.homeTeamColor1 = &color
}

// SetHomeTeamColor2 is a setter for the home team secondary color
func (s *SquaresSettings) SetHomeTeamColor2(color string) {
	if color == "" {
		s.homeTeamColor2 = nil
		return
	}

	s.homeTeamColor2 = &color
}

// SetAwayTeamName is a setter for the home team name
func (s *SquaresSettings) SetAwayTeamName(name string) {
	if name == "" {
		s.awayTeamName = nil
		return
	}

	if utf8.RuneCountInString(name) > TeamNameMaxLength {
		name = string([]rune(name)[0:TeamNameMaxLength])
	}

	s.awayTeamName = &name
}

// SetAwayTeamColor1 is a setter for the away team primary color
func (s *SquaresSettings) SetAwayTeamColor1(color string) {
	if color == "" {
		s.awayTeamColor1 = nil
		return
	}

	s.awayTeamColor1 = &color
}

// SetAwayTeamColor2 is a setter for the away team secondary color
func (s *SquaresSettings) SetAwayTeamColor2(color string) {
	if color == "" {
		s.awayTeamColor2 = nil
		return
	}

	s.awayTeamColor2 = &color
}

// HomeTeamName is a getter for the home team name
func (s *SquaresSettings) HomeTeamName() string {
	if s.homeTeamName == nil {
		return DefaultHomeTeamName
	}

	return *s.homeTeamName
}

// HomeTeamColor1 is a getter for the home team primary color
func (s *SquaresSettings) HomeTeamColor1() string {
	if s.homeTeamColor1 == nil {
		return DefaultHomeTeamColor1
	}

	return *s.homeTeamColor1
}

// HomeTeamColor2 is a getter for the home team secondary color
func (s *SquaresSettings) HomeTeamColor2() string {
	if s.homeTeamColor2 == nil {
		return DefaultHomeTeamColor2
	}

	return *s.homeTeamColor2
}

// AwayTeamName is a getter for the home team name
func (s *SquaresSettings) AwayTeamName() string {
	if s.awayTeamName == nil {
		return DefaultAwayTeamName
	}

	return *s.awayTeamName
}

// AwayTeamColor1 is a getter for the away team primary color
func (s *SquaresSettings) AwayTeamColor1() string {
	if s.awayTeamColor1 == nil {
		return DefaultAwayTeamColor1
	}

	return *s.awayTeamColor1
}

// AwayTeamColor2 is a getter for the away team secondary color
func (s *SquaresSettings) AwayTeamColor2() string {
	if s.awayTeamColor2 == nil {
		return DefaultAwayTeamColor2
	}

	return *s.awayTeamColor2
}
