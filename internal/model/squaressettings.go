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
	"time"
)

// SquaresSettings will contain various user-defined settings
type SquaresSettings struct {
	SquaresID      int64      `json:"-"`
	HomeTeamName   *string    `json:"homeTeamName"`
	HomeTeamColor1 *string    `json:"homeTeamColor1"`
	HomeTeamColor2 *string    `json:"homeTeamColor2"`
	HomeTeamColor3 *string    `json:"homeTeamColor3"`
	AwayTeamName   *string    `json:"awayTeamName"`
	AwayTeamColor1 *string    `json:"awayTeamColor1"`
	AwayTeamColor2 *string    `json:"awayTeamColor2"`
	AwayTeamColor3 *string    `json:"awayTeamColor3"`
	Modified       *time.Time `json:"-"`
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
			modified = (NOW() AT TIME ZONE 'utc')
		WHERE squares_id = $9
	`,
		s.HomeTeamName,
		s.HomeTeamColor1,
		s.HomeTeamColor2,
		s.HomeTeamColor3,
		s.AwayTeamName,
		s.AwayTeamColor1,
		s.AwayTeamColor2,
		s.AwayTeamColor3,
		s.SquaresID,
	)

	return err
}

// SetHomeTeamName will set the home team name
func (s *SquaresSettings) SetHomeTeamName(str string) {
	if len(str) == 0 {
		s.HomeTeamName = nil
		return
	}

	s.HomeTeamName = &str
}

// SetHomeTeamColor1 will set the home team color1
func (s *SquaresSettings) SetHomeTeamColor1(str string) {
	if len(str) == 0 {
		s.HomeTeamColor1 = nil
		return
	}

	s.HomeTeamColor1 = &str
}

// SetHomeTeamColor2 will set the home team color2
func (s *SquaresSettings) SetHomeTeamColor2(str string) {
	if len(str) == 0 {
		s.HomeTeamColor2 = nil
		return
	}

	s.HomeTeamColor2 = &str
}

// SetHomeTeamColor3 will set the home team color3
func (s *SquaresSettings) SetHomeTeamColor3(str string) {
	if len(str) == 0 {
		s.HomeTeamColor3 = nil
		return
	}

	s.HomeTeamColor3 = &str
}

// SetAwayTeamName will set the away team name
func (s *SquaresSettings) SetAwayTeamName(str string) {
	if len(str) == 0 {
		s.AwayTeamName = nil
		return
	}

	s.AwayTeamName = &str
}

// SetAwayTeamColor1 will set the away team color1
func (s *SquaresSettings) SetAwayTeamColor1(str string) {
	if len(str) == 0 {
		s.AwayTeamColor1 = nil
		return
	}

	s.AwayTeamColor1 = &str
}

// SetAwayTeamColor2 will set the away team color2
func (s *SquaresSettings) SetAwayTeamColor2(str string) {
	if len(str) == 0 {
		s.AwayTeamColor2 = nil
		return
	}

	s.AwayTeamColor2 = &str
}

// SetAwayTeamColor3 will set the away team color3
func (s *SquaresSettings) SetAwayTeamColor3(str string) {
	if len(str) == 0 {
		s.AwayTeamColor3 = nil
		return
	}

	s.AwayTeamColor3 = &str
}
