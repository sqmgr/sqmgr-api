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

package sports

import (
	"time"
)

// League represents a sports league
type League string

const (
	LeagueNFL   League = "nfl"
	LeagueNBA   League = "nba"
	LeagueWNBA  League = "wnba"
	LeagueNCAAB League = "ncaab"
	LeagueNCAAF League = "ncaaf"
)

// AllLeagues returns all supported leagues
func AllLeagues() []League {
	return []League{LeagueNFL, LeagueNBA, LeagueWNBA, LeagueNCAAB, LeagueNCAAF}
}

// IsValid returns true if the league is valid
func (l League) IsValid() bool {
	switch l {
	case LeagueNFL, LeagueNBA, LeagueWNBA, LeagueNCAAB, LeagueNCAAF:
		return true
	}
	return false
}

// ESPNPath returns the ESPN API path for this league (sport/league)
func (l League) ESPNPath() string {
	switch l {
	case LeagueNFL:
		return "football/nfl"
	case LeagueNBA:
		return "basketball/nba"
	case LeagueWNBA:
		return "basketball/wnba"
	case LeagueNCAAB:
		return "basketball/mens-college-basketball"
	case LeagueNCAAF:
		return "football/college-football"
	default:
		return ""
	}
}

// Team represents a team from the ESPN API
type Team struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DisplayName    string `json:"displayName"`
	Abbreviation   string `json:"abbreviation"`
	Location       string `json:"location"`
	Color          string `json:"color"`          // Primary color hex (no #)
	AlternateColor string `json:"alternateColor"` // Secondary color hex (no #)
}

// EventStatus represents the status of a game
type EventStatus string

const (
	EventStatusScheduled  EventStatus = "scheduled"
	EventStatusInProgress EventStatus = "in_progress"
	EventStatusFinal      EventStatus = "final"
)

// Event represents a game/event
type Event struct {
	ID           string      // ESPN event ID (string format)
	Name         string      // Event name from ESPN (e.g., "Super Bowl LVIII")
	Date         time.Time   // Event date/time
	Status       EventStatus // Game status
	StatusDetail string      // Status description from ESPN (e.g., "Halftime", "End of 1st Quarter")
	Period       int         // Current period (0=not started, 1-4=quarters, 5+=OT)
	Clock        string      // Game clock display (e.g., "12:34", "5:00")
	Season       int         // Season year
	SeasonType   SeasonType  // Season type (preseason, regular, postseason)
	Week         *int        // Week number (NFL only)
	Venue        string      // Venue name

	HomeTeam      Team
	AwayTeam      Team
	HomeTeamScore *int
	AwayTeamScore *int

	// Quarter scores
	HomeQ1 *int
	HomeQ2 *int
	HomeQ3 *int
	HomeQ4 *int
	HomeOT *int
	AwayQ1 *int
	AwayQ2 *int
	AwayQ3 *int
	AwayQ4 *int
	AwayOT *int
}

// SeasonType represents the type of season
type SeasonType int

const (
	SeasonTypePreseason  SeasonType = 1
	SeasonTypeRegular    SeasonType = 2
	SeasonTypePostseason SeasonType = 3
)

// ScoreboardOptions holds options for fetching scoreboard data
type ScoreboardOptions struct {
	Date       string     // Date in YYYYMMDD format
	Week       int        // NFL week number (1-18 for regular season, 1-5 for postseason)
	Season     int        // Season year
	SeasonType SeasonType // Season type (1=preseason, 2=regular, 3=postseason)
}

// SeasonInfo holds information about a league's current or upcoming season
type SeasonInfo struct {
	Year      int       // Season year
	StartDate time.Time // Season start date
	EndDate   time.Time // Season end date
	Type      string    // Season type name (e.g., "Regular Season", "Postseason")
	InSeason  bool      // True if current date is within the season
}

// ESPN API response types

// espnTeamsResponse is the ESPN API response for teams endpoint
type espnTeamsResponse struct {
	Sports []struct {
		Leagues []struct {
			Teams []struct {
				Team espnTeam `json:"team"`
			} `json:"teams"`
		} `json:"leagues"`
	} `json:"sports"`
}

// espnTeam represents a team in ESPN API responses
type espnTeam struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DisplayName    string `json:"displayName"`
	Abbreviation   string `json:"abbreviation"`
	Location       string `json:"location"`
	Color          string `json:"color"`          // Primary color hex (no #)
	AlternateColor string `json:"alternateColor"` // Secondary color hex (no #)
}

// espnScoreboardResponse is the ESPN API response for scoreboard endpoint
type espnScoreboardResponse struct {
	Leagues []espnLeagueInfo `json:"leagues"`
	Events  []espnEvent      `json:"events"`
	Week    *struct {
		Number int `json:"number"`
	} `json:"week,omitempty"`
	Season *struct {
		Year int `json:"year"`
		Type int `json:"type"` // 1=preseason, 2=regular, 3=postseason
	} `json:"season,omitempty"`
}

// espnEvent represents an event in ESPN API responses
type espnEvent struct {
	ID           string            `json:"id"`
	Date         string            `json:"date"` // ISO 8601 format
	Name         string            `json:"name"`
	ShortName    string            `json:"shortName"`
	Season       espnSeason        `json:"season"`
	Week         *espnWeek         `json:"week,omitempty"`
	Competitions []espnCompetition `json:"competitions"`
	Status       espnStatus        `json:"status"`
}

type espnSeason struct {
	Year int `json:"year"`
	Type int `json:"type"`
}

type espnWeek struct {
	Number int `json:"number"`
}

// espnCompetition represents a competition (the actual game)
type espnCompetition struct {
	ID          string           `json:"id"`
	Date        string           `json:"date"`
	Venue       *espnVenue       `json:"venue,omitempty"`
	Competitors []espnCompetitor `json:"competitors"`
	Status      espnStatus       `json:"status"`
	Notes       []espnNote       `json:"notes,omitempty"`
}

type espnVenue struct {
	FullName string `json:"fullName"`
}

// espnNote represents a note/headline for special events (e.g., "Super Bowl LVIII")
type espnNote struct {
	Type     string `json:"type"`
	Headline string `json:"headline"`
}

// espnCompetitor represents a team in a competition
type espnCompetitor struct {
	ID         string          `json:"id"`
	HomeAway   string          `json:"homeAway"` // "home" or "away"
	Team       espnTeam        `json:"team"`
	Score      string          `json:"score"` // Score as string
	Linescores []espnLinescore `json:"linescores,omitempty"`
}

type espnLinescore struct {
	Value float64 `json:"value"`
}

// espnStatus represents game status
type espnStatus struct {
	Clock        float64        `json:"clock"`
	DisplayClock string         `json:"displayClock"`
	Period       int            `json:"period"`
	Type         espnStatusType `json:"type"`
}

type espnStatusType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`  // "STATUS_SCHEDULED", "STATUS_IN_PROGRESS", "STATUS_FINAL"
	State       string `json:"state"` // "pre", "in", "post"
	Completed   bool   `json:"completed"`
	Description string `json:"description"`
}

// espnLeagueSeason holds season info from the leagues array in scoreboard response
type espnLeagueSeason struct {
	Year      int    `json:"year"`
	StartDate string `json:"startDate"` // ISO 8601 format
	EndDate   string `json:"endDate"`   // ISO 8601 format
	Type      struct {
		ID   string `json:"id"`
		Type int    `json:"type"`
		Name string `json:"name"`
	} `json:"type"`
}

// espnLeagueInfo holds league info from the scoreboard response
type espnLeagueInfo struct {
	Season espnLeagueSeason `json:"season"`
	// Calendar is ignored - it has different structures per league
}

// espnTeamScheduleResponse is the ESPN API response for team schedule endpoint
type espnTeamScheduleResponse struct {
	Team   espnTeam               `json:"team"`
	Events []espnScheduleEvent    `json:"events"`
	Season espnTeamScheduleSeason `json:"season"`
}

// espnTeamScheduleSeason holds season info from team schedule response
type espnTeamScheduleSeason struct {
	Year        int    `json:"year"`
	DisplayName string `json:"displayName"`
}

// espnScheduleEvent represents an event in team schedule responses
type espnScheduleEvent struct {
	ID           string                    `json:"id"`
	Date         string                    `json:"date"`
	Name         string                    `json:"name"`
	ShortName    string                    `json:"shortName"`
	Season       espnSeason                `json:"season"`
	SeasonType   espnSeasonType            `json:"seasonType"`
	Week         *espnWeek                 `json:"week,omitempty"`
	Competitions []espnScheduleCompetition `json:"competitions"`
}

// espnSeasonType represents season type info
type espnSeasonType struct {
	ID   string `json:"id"`
	Type int    `json:"type"`
	Name string `json:"name"`
}

// espnScheduleCompetition represents a competition in team schedule
type espnScheduleCompetition struct {
	ID          string                   `json:"id"`
	Date        string                   `json:"date"`
	Venue       *espnVenue               `json:"venue,omitempty"`
	Competitors []espnScheduleCompetitor `json:"competitors"`
	Status      espnStatus               `json:"status"`
	Notes       []espnNote               `json:"notes,omitempty"`
}

// espnScheduleCompetitor represents a competitor in team schedule
type espnScheduleCompetitor struct {
	ID       string     `json:"id"`
	HomeAway string     `json:"homeAway"`
	Team     espnTeam   `json:"team"`
	Score    *espnScore `json:"score,omitempty"`
}

// espnScore represents a score object (used in team schedule)
type espnScore struct {
	Value        float64 `json:"value"`
	DisplayValue string  `json:"displayValue"`
}

// espnSummaryResponse is the ESPN API response for event summary endpoint
type espnSummaryResponse struct {
	Header espnSummaryHeader `json:"header"`
}

// espnSummaryHeader contains the header info from summary response
type espnSummaryHeader struct {
	ID           string                   `json:"id"`
	Season       espnSeason               `json:"season"`
	Competitions []espnSummaryCompetition `json:"competitions"`
}

// espnSummaryCompetition represents a competition in summary response
type espnSummaryCompetition struct {
	ID          string                  `json:"id"`
	Date        string                  `json:"date"`
	Venue       *espnVenue              `json:"venue,omitempty"`
	Competitors []espnSummaryCompetitor `json:"competitors"`
	Status      espnStatus              `json:"status"`
	Notes       []espnNote              `json:"notes,omitempty"`
}

// espnSummaryCompetitor represents a competitor in summary response
type espnSummaryCompetitor struct {
	ID         string                 `json:"id"`
	HomeAway   string                 `json:"homeAway"`
	Winner     bool                   `json:"winner"`
	Team       espnSummaryTeam        `json:"team"`
	Score      string                 `json:"score"`
	Linescores []espnSummaryLinescore `json:"linescores,omitempty"`
}

// espnSummaryTeam represents team info in summary response
type espnSummaryTeam struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DisplayName    string `json:"displayName"`
	Abbreviation   string `json:"abbreviation"`
	Location       string `json:"location"`
	Color          string `json:"color"`
	AlternateColor string `json:"alternateColor"`
}

// espnSummaryLinescore represents a period score in summary response
type espnSummaryLinescore struct {
	DisplayValue string `json:"displayValue"`
}
