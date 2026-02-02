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
