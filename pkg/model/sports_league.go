/*
Copyright (C) 2019 Tom Peters

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
	"database/sql/driver"
	"fmt"
)

// SportsLeague represents a sports league
type SportsLeague string

const (
	// SportsLeagueNFL is the National Football League
	SportsLeagueNFL SportsLeague = "nfl"
	// SportsLeagueNBA is the National Basketball Association
	SportsLeagueNBA SportsLeague = "nba"
	// SportsLeagueWNBA is the Women's National Basketball Association
	SportsLeagueWNBA SportsLeague = "wnba"
	// SportsLeagueNCAAB is NCAA Men's Basketball
	SportsLeagueNCAAB SportsLeague = "ncaab"
	// SportsLeagueNCAAF is NCAA Football
	SportsLeagueNCAAF SportsLeague = "ncaaf"
)

// SportsLeagueInfo contains metadata for a sports league
type SportsLeagueInfo struct {
	Key   SportsLeague `json:"key"`
	Label string       `json:"label"`
}

// validSportsLeagues contains all valid leagues
var validSportsLeagues = []SportsLeagueInfo{
	{Key: SportsLeagueNFL, Label: "NFL"},
	{Key: SportsLeagueNBA, Label: "NBA"},
	{Key: SportsLeagueWNBA, Label: "WNBA"},
	{Key: SportsLeagueNCAAB, Label: "NCAAB"},
	{Key: SportsLeagueNCAAF, Label: "NCAAF"},
}

// ValidSportsLeagues returns all valid sports leagues with metadata
func ValidSportsLeagues() []SportsLeagueInfo {
	return validSportsLeagues
}

// IsValidSportsLeague returns true if the league is valid
func IsValidSportsLeague(league string) bool {
	for _, l := range validSportsLeagues {
		if string(l.Key) == league {
			return true
		}
	}
	return false
}

// IsValid returns true if the league is valid
func (l SportsLeague) IsValid() bool {
	return IsValidSportsLeague(string(l))
}

// UsesHalves returns true if the league uses halves instead of quarters (e.g., NCAAB)
func (l SportsLeague) UsesHalves() bool {
	return l == SportsLeagueNCAAB
}

// Value implements driver.Valuer for database storage
func (l SportsLeague) Value() (driver.Value, error) {
	return string(l), nil
}

// Scan implements sql.Scanner for database retrieval
func (l *SportsLeague) Scan(value interface{}) error {
	if value == nil {
		return fmt.Errorf("cannot scan nil into SportsLeague")
	}
	switch v := value.(type) {
	case []byte:
		*l = SportsLeague(v)
	case string:
		*l = SportsLeague(v)
	default:
		return fmt.Errorf("cannot scan %T into SportsLeague", value)
	}
	return nil
}

// Deprecated aliases for backward compatibility during transition
type BDLLeague = SportsLeague

const (
	BDLLeagueNFL   = SportsLeagueNFL
	BDLLeagueNBA   = SportsLeagueNBA
	BDLLeagueWNBA  = SportsLeagueWNBA
	BDLLeagueNCAAB = SportsLeagueNCAAB
	BDLLeagueNCAAF = SportsLeagueNCAAF
)

type BDLLeagueInfo = SportsLeagueInfo

func ValidBDLLeagues() []BDLLeagueInfo {
	return ValidSportsLeagues()
}

func IsValidBDLLeague(league string) bool {
	return IsValidSportsLeague(league)
}
