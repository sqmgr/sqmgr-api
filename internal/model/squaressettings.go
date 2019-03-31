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
	SquaresID      int64
	homeTeamName   *string
	homeTeamColor1 *string
	homeTeamColor2 *string
	homeTeamColor3 *string
	awayTeamName   *string
	awayTeamColor1 *string
	awayTeamColor2 *string
	awayTeamColor3 *string
	Modified       *time.Time
}

// defaults for SquaresSettings
const (
	defaultHomeTeamName   = "Home Team"
	defaultHomeTeamColor1 = "#000"
	defaultHomeTeamColor2 = "#000"
	defaultHomeTeamColor3 = "#000"
	defaultAwayTeamName   = "Away Team"
	defaultAwayTeamColor1 = "#000"
	defaultAwayTeamColor2 = "#000"
	defaultAwayTeamColor3 = "#000"
)

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
		s.homeTeamName,
		s.homeTeamColor1,
		s.homeTeamColor2,
		s.homeTeamColor3,
		s.awayTeamName,
		s.awayTeamColor1,
		s.awayTeamColor2,
		s.awayTeamColor3,
		s.SquaresID,
	)

	return err
}

func (s *SquaresSettings) HomeTeamName() string {
	if s.homeTeamName != nil {
		return *s.homeTeamName
	}

	return defaultHomeTeamName
}

func (s *SquaresSettings) HomeTeamColor1() string {
	if s.homeTeamColor1 != nil {
		return *s.homeTeamColor1
	}

	return defaultHomeTeamColor1
}

func (s *SquaresSettings) HomeTeamColor2() string {
	if s.homeTeamColor2 != nil {
		return *s.homeTeamColor2
	}

	return defaultHomeTeamColor2
}

func (s *SquaresSettings) HomeTeamColor3() string {
	if s.homeTeamColor3 != nil {
		return *s.homeTeamColor3
	}

	return defaultHomeTeamColor3
}

func (s *SquaresSettings) AwayTeamName() string {
	if s.awayTeamName != nil {
		return *s.awayTeamName
	}

	return defaultAwayTeamName
}

func (s *SquaresSettings) AwayTeamColor1() string {
	if s.awayTeamColor1 != nil {
		return *s.awayTeamColor1
	}

	return defaultAwayTeamColor1
}

func (s *SquaresSettings) AwayTeamColor2() string {
	if s.awayTeamColor2 != nil {
		return *s.awayTeamColor2
	}

	return defaultAwayTeamColor2
}

func (s *SquaresSettings) AwayTeamColor3() string {
	if s.awayTeamColor3 != nil {
		return *s.awayTeamColor3
	}

	return defaultAwayTeamColor3
}

func (s *SquaresSettings) SetHomeTeamName(str string) {
	if len(str) == 0 {
		s.homeTeamName = nil
		return
	}

	s.homeTeamName = &str
}

func (s *SquaresSettings) SetHomeTeamColor1(str string) {
	if len(str) == 0 {
		s.homeTeamColor1 = nil
		return
	}

	s.homeTeamColor1 = &str
}

func (s *SquaresSettings) SetHomeTeamColor2(str string) {
	if len(str) == 0 {
		s.homeTeamColor2 = nil
		return
	}

	s.homeTeamColor2 = &str
}

func (s *SquaresSettings) SetHomeTeamColor3(str string) {
	if len(str) == 0 {
		s.homeTeamColor3 = nil
		return
	}

	s.homeTeamColor3 = &str
}

func (s *SquaresSettings) SetAwayTeamName(str string) {
	if len(str) == 0 {
		s.awayTeamName = nil
		return
	}

	s.awayTeamName = &str
}

func (s *SquaresSettings) SetAwayTeamColor1(str string) {
	if len(str) == 0 {
		s.awayTeamColor1 = nil
		return
	}

	s.awayTeamColor1 = &str
}

func (s *SquaresSettings) SetAwayTeamColor2(str string) {
	if len(str) == 0 {
		s.awayTeamColor2 = nil
		return
	}

	s.awayTeamColor2 = &str
}

func (s *SquaresSettings) SetAwayTeamColor3(str string) {
	if len(str) == 0 {
		s.awayTeamColor3 = nil
		return
	}

	s.awayTeamColor3 = &str
}
