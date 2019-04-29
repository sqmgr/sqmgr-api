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

// NotesMaxLength is the maximum number of characters the notes can be
const NotesMaxLength = 500

// SquaresSettings will contain various user-defined settings
type SquaresSettings struct {
	SquaresID      int64   `json:"-"`
	HomeTeamName   *string `json:"homeTeamName"`
	HomeTeamColor1 *string `json:"homeTeamColor1"`
	HomeTeamColor2 *string `json:"homeTeamColor2"`
	HomeTeamColor3 *string `json:"homeTeamColor3"`
	AwayTeamName   *string `json:"awayTeamName"`
	AwayTeamColor1 *string `json:"awayTeamColor1"`
	AwayTeamColor2 *string `json:"awayTeamColor2"`
	AwayTeamColor3 *string `json:"awayTeamColor3"`
	notes          *string
	Modified       *time.Time `json:"-"`
}

type squaresSettingsJSON struct {
	HomeTeamName   *string `json:"homeTeamName"`
	HomeTeamColor1 *string `json:"homeTeamColor1"`
	HomeTeamColor2 *string `json:"homeTeamColor2"`
	HomeTeamColor3 *string `json:"homeTeamColor3"`
	AwayTeamName   *string `json:"awayTeamName"`
	AwayTeamColor1 *string `json:"awayTeamColor1"`
	AwayTeamColor2 *string `json:"awayTeamColor2"`
	AwayTeamColor3 *string `json:"awayTeamColor3"`
	Notes          *string `json:"notes"`
}

// MarshalJSON adds custom JSON marshalling support
func (s *SquaresSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(squaresSettingsJSON{
		HomeTeamName:   s.HomeTeamName,
		HomeTeamColor1: s.HomeTeamColor1,
		HomeTeamColor2: s.HomeTeamColor2,
		HomeTeamColor3: s.HomeTeamColor3,
		AwayTeamName:   s.AwayTeamName,
		AwayTeamColor1: s.AwayTeamColor1,
		AwayTeamColor2: s.AwayTeamColor2,
		AwayTeamColor3: s.AwayTeamColor3,
		Notes:          s.notes,
	})
}

// Save will save the settings
func (s *SquaresSettings) Save(q executer) error {
	_, err := q.Exec(`
		UPDATE squares_settings SET
			home_team_name = $1,
			home_team_color_1 = $2,
			home_team_color_2 = $3,
			home_team_color_3 = $4,
			away_team_name = $5,
			away_team_color_1 = $6,
			away_team_color_2 = $7,
			away_team_color_3 = $8,
			notes = $9,
			modified = (NOW() AT TIME ZONE 'utc')
		WHERE squares_id = $10
	`,
		s.HomeTeamName,
		s.HomeTeamColor1,
		s.HomeTeamColor2,
		s.HomeTeamColor3,
		s.AwayTeamName,
		s.AwayTeamColor1,
		s.AwayTeamColor2,
		s.AwayTeamColor3,
		s.notes,
		s.SquaresID,
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
