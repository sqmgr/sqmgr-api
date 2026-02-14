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
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SportsEventStatus represents the status of a sports event
type SportsEventStatus string

const (
	// SportsEventStatusScheduled means the game hasn't started
	SportsEventStatusScheduled SportsEventStatus = "scheduled"
	// SportsEventStatusInProgress means the game is currently being played
	SportsEventStatusInProgress SportsEventStatus = "in_progress"
	// SportsEventStatusFinal means the game has ended
	SportsEventStatusFinal SportsEventStatus = "final"
)

// SportsEvent represents a cached event/game from the sports API
type SportsEvent struct {
	model *Model

	ID         int64  // Internal database ID
	ESPNID     string // ESPN's string-based event ID
	League     SportsLeague
	Name       *string // Event name from ESPN (e.g., "Super Bowl LVIII")
	HomeTeamID string
	AwayTeamID string
	EventDate  time.Time
	Season     int
	Week       *int
	Postseason bool
	Venue      *string

	// Game status
	Status       SportsEventStatus
	StatusDetail *string // Status description (e.g., "Halftime", "End of 1st Quarter")
	Period       *int
	Clock        *string // Game clock display (e.g., "12:34", "5:00")

	// Scores
	HomeScore *int
	AwayScore *int
	HomeQ1    *int
	HomeQ2    *int
	HomeQ3    *int
	HomeQ4    *int
	HomeOT    *int
	AwayQ1    *int
	AwayQ2    *int
	AwayQ3    *int
	AwayQ4    *int
	AwayOT    *int

	// Metadata
	Created    time.Time
	Modified   time.Time
	LastSynced time.Time

	// Loaded relationships
	homeTeam *SportsTeam
	awayTeam *SportsTeam
}

// SportsEventJSON represents event data for JSON serialization
type SportsEventJSON struct {
	ID           int64             `json:"id"`
	ESPNID       string            `json:"espnId,omitempty"`
	League       SportsLeague      `json:"league"`
	Name         string            `json:"name,omitempty"`
	HomeTeamID   string            `json:"homeTeamId"`
	AwayTeamID   string            `json:"awayTeamId"`
	EventDate    time.Time         `json:"eventDate"`
	Season       int               `json:"season"`
	Week         *int              `json:"week,omitempty"`
	Postseason   bool              `json:"postseason"`
	Venue        string            `json:"venue,omitempty"`
	Status       SportsEventStatus `json:"status"`
	StatusDetail string            `json:"statusDetail,omitempty"`
	Period       *int              `json:"period,omitempty"`
	Clock        string            `json:"clock,omitempty"`
	HomeScore    *int              `json:"homeScore,omitempty"`
	AwayScore    *int              `json:"awayScore,omitempty"`
	HomeQ1       *int              `json:"homeQ1,omitempty"`
	HomeQ2       *int              `json:"homeQ2,omitempty"`
	HomeQ3       *int              `json:"homeQ3,omitempty"`
	HomeQ4       *int              `json:"homeQ4,omitempty"`
	HomeOT       *int              `json:"homeOT,omitempty"`
	AwayQ1       *int              `json:"awayQ1,omitempty"`
	AwayQ2       *int              `json:"awayQ2,omitempty"`
	AwayQ3       *int              `json:"awayQ3,omitempty"`
	AwayQ4       *int              `json:"awayQ4,omitempty"`
	AwayOT       *int              `json:"awayOT,omitempty"`
	HomeTeam     *SportsTeamJSON   `json:"homeTeam,omitempty"`
	AwayTeam     *SportsTeamJSON   `json:"awayTeam,omitempty"`
	LastSynced   time.Time         `json:"lastSynced"`
}

// JSON returns the JSON representation of the event
func (e *SportsEvent) JSON() *SportsEventJSON {
	json := &SportsEventJSON{
		ID:         e.ID,
		ESPNID:     e.ESPNID,
		League:     e.League,
		HomeTeamID: e.HomeTeamID,
		AwayTeamID: e.AwayTeamID,
		EventDate:  e.EventDate,
		Season:     e.Season,
		Week:       e.Week,
		Postseason: e.Postseason,
		Status:     e.Status,
		Period:     e.Period,
		HomeScore:  e.HomeScore,
		AwayScore:  e.AwayScore,
		HomeQ1:     e.HomeQ1,
		HomeQ2:     e.HomeQ2,
		HomeQ3:     e.HomeQ3,
		HomeQ4:     e.HomeQ4,
		HomeOT:     e.HomeOT,
		AwayQ1:     e.AwayQ1,
		AwayQ2:     e.AwayQ2,
		AwayQ3:     e.AwayQ3,
		AwayQ4:     e.AwayQ4,
		AwayOT:     e.AwayOT,
		LastSynced: e.LastSynced,
	}
	if e.Name != nil {
		json.Name = *e.Name
	}
	if e.Venue != nil {
		json.Venue = *e.Venue
	}
	if e.Clock != nil {
		json.Clock = *e.Clock
	}
	if e.StatusDetail != nil {
		json.StatusDetail = *e.StatusDetail
	}
	if e.homeTeam != nil {
		json.HomeTeam = e.homeTeam.JSON()
	}
	if e.awayTeam != nil {
		json.AwayTeam = e.awayTeam.JSON()
	}
	return json
}

// HomeTeam returns the loaded home team
func (e *SportsEvent) HomeTeam() *SportsTeam {
	return e.homeTeam
}

// AwayTeam returns the loaded away team
func (e *SportsEvent) AwayTeam() *SportsTeam {
	return e.awayTeam
}

// SetHomeTeam sets the home team
func (e *SportsEvent) SetHomeTeam(team *SportsTeam) {
	e.homeTeam = team
}

// SetAwayTeam sets the away team
func (e *SportsEvent) SetAwayTeam(team *SportsTeam) {
	e.awayTeam = team
}

// HomeHalfScore returns the home team's score at halftime (Q1+Q2)
func (e *SportsEvent) HomeHalfScore() *int {
	if e.HomeQ1 == nil || e.HomeQ2 == nil {
		return nil
	}
	sum := *e.HomeQ1 + *e.HomeQ2
	return &sum
}

// AwayHalfScore returns the away team's score at halftime (Q1+Q2)
func (e *SportsEvent) AwayHalfScore() *int {
	if e.AwayQ1 == nil || e.AwayQ2 == nil {
		return nil
	}
	sum := *e.AwayQ1 + *e.AwayQ2
	return &sum
}

// HomeQ3CumulativeScore returns the home team's cumulative score through Q3 (Q1+Q2+Q3)
func (e *SportsEvent) HomeQ3CumulativeScore() *int {
	if e.HomeQ1 == nil || e.HomeQ2 == nil || e.HomeQ3 == nil {
		return nil
	}
	sum := *e.HomeQ1 + *e.HomeQ2 + *e.HomeQ3
	return &sum
}

// AwayQ3CumulativeScore returns the away team's cumulative score through Q3 (Q1+Q2+Q3)
func (e *SportsEvent) AwayQ3CumulativeScore() *int {
	if e.AwayQ1 == nil || e.AwayQ2 == nil || e.AwayQ3 == nil {
		return nil
	}
	sum := *e.AwayQ1 + *e.AwayQ2 + *e.AwayQ3
	return &sum
}

// IsPeriodComplete checks if a scoring period is complete based on game status and current period
func (e *SportsEvent) IsPeriodComplete(setType NumberSetType) bool {
	isFinal := e.Status == SportsEventStatusFinal
	period := 0
	if e.Period != nil {
		period = *e.Period
	}

	detail := ""
	if e.StatusDetail != nil {
		detail = strings.ToLower(*e.StatusDetail)
	}
	atEndOfPeriod := strings.Contains(detail, "end of")
	atHalftime := strings.Contains(detail, "halftime")

	switch setType {
	case NumberSetTypeQ1:
		return isFinal || period >= 2 || (period == 1 && atEndOfPeriod)
	case NumberSetTypeQ2, NumberSetTypeHalf:
		if e.League.UsesHalves() {
			return isFinal || period >= 2 || atHalftime || (period == 1 && atEndOfPeriod)
		}
		return isFinal || period >= 3 || atHalftime || (period == 2 && atEndOfPeriod)
	case NumberSetTypeQ3:
		return isFinal || period >= 4 || (period == 3 && atEndOfPeriod)
	case NumberSetTypeQ4, NumberSetTypeFinal, NumberSetTypeAll:
		return isFinal
	}
	return false
}

// ScoreForPeriod returns the home and away scores for a given number set type
func (e *SportsEvent) ScoreForPeriod(setType NumberSetType) (*int, *int) {
	switch setType {
	case NumberSetTypeQ1:
		return e.HomeQ1, e.AwayQ1
	case NumberSetTypeHalf:
		if e.League.UsesHalves() {
			return e.HomeQ1, e.AwayQ1
		}
		return e.HomeHalfScore(), e.AwayHalfScore()
	case NumberSetTypeQ2:
		return e.HomeHalfScore(), e.AwayHalfScore()
	case NumberSetTypeQ3:
		return e.HomeQ3CumulativeScore(), e.AwayQ3CumulativeScore()
	case NumberSetTypeQ4, NumberSetTypeFinal, NumberSetTypeAll:
		return e.HomeScore, e.AwayScore
	}
	return nil, nil
}

const sportsEventColumns = `
	id, espn_id, league, name, home_team_id, away_team_id, event_date, season, week, postseason, venue,
	status, status_detail, period, clock, home_score, away_score,
	home_q1, home_q2, home_q3, home_q4, home_ot,
	away_q1, away_q2, away_q3, away_q4, away_ot,
	created, modified, last_synced`

// sportsEventColumnsWithPrefix is for use in JOIN queries where table alias is needed
const sportsEventColumnsWithPrefix = `
	e.id, e.espn_id, e.league, e.name, e.home_team_id, e.away_team_id, e.event_date, e.season, e.week, e.postseason, e.venue,
	e.status, e.status_detail, e.period, e.clock, e.home_score, e.away_score,
	e.home_q1, e.home_q2, e.home_q3, e.home_q4, e.home_ot,
	e.away_q1, e.away_q2, e.away_q3, e.away_q4, e.away_ot,
	e.created, e.modified, e.last_synced`

func (m *Model) sportsEventByRow(scan scanFunc) (*SportsEvent, error) {
	event := &SportsEvent{model: m}
	if err := scan(
		&event.ID,
		&event.ESPNID,
		&event.League,
		&event.Name,
		&event.HomeTeamID,
		&event.AwayTeamID,
		&event.EventDate,
		&event.Season,
		&event.Week,
		&event.Postseason,
		&event.Venue,
		&event.Status,
		&event.StatusDetail,
		&event.Period,
		&event.Clock,
		&event.HomeScore,
		&event.AwayScore,
		&event.HomeQ1,
		&event.HomeQ2,
		&event.HomeQ3,
		&event.HomeQ4,
		&event.HomeOT,
		&event.AwayQ1,
		&event.AwayQ2,
		&event.AwayQ3,
		&event.AwayQ4,
		&event.AwayOT,
		&event.Created,
		&event.Modified,
		&event.LastSynced,
	); err != nil {
		return nil, err
	}
	return event, nil
}

// SportsEventByID returns an event by its internal database ID
func (m *Model) SportsEventByID(ctx context.Context, id int64) (*SportsEvent, error) {
	const query = `SELECT ` + sportsEventColumns + ` FROM sports_events WHERE id = $1`
	row := m.DB.QueryRowContext(ctx, query, id)
	event, err := m.sportsEventByRow(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return event, nil
}

// SportsEventByESPNID returns an event by its ESPN ID
func (m *Model) SportsEventByESPNID(ctx context.Context, espnID string) (*SportsEvent, error) {
	const query = `SELECT ` + sportsEventColumns + ` FROM sports_events WHERE espn_id = $1`
	row := m.DB.QueryRowContext(ctx, query, espnID)
	event, err := m.sportsEventByRow(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return event, nil
}

// SportsEventByIDWithTeams returns an event with its teams loaded
func (m *Model) SportsEventByIDWithTeams(ctx context.Context, id int64) (*SportsEvent, error) {
	event, err := m.SportsEventByID(ctx, id)
	if err != nil || event == nil {
		return event, err
	}

	if err := event.LoadTeams(ctx); err != nil {
		return nil, err
	}

	return event, nil
}

// LoadTeams loads the home and away teams for the event
func (e *SportsEvent) LoadTeams(ctx context.Context) error {
	homeTeam, err := e.model.SportsTeamByID(ctx, e.HomeTeamID, e.League)
	if err != nil {
		return err
	}
	e.homeTeam = homeTeam

	awayTeam, err := e.model.SportsTeamByID(ctx, e.AwayTeamID, e.League)
	if err != nil {
		return err
	}
	e.awayTeam = awayTeam

	return nil
}

// SportsEventsByLeague returns events for a given league with optional filters
func (m *Model) SportsEventsByLeague(ctx context.Context, league SportsLeague, status string, limit int) ([]*SportsEvent, error) {
	query := `SELECT ` + sportsEventColumns + ` FROM sports_events WHERE league = $1`
	args := []interface{}{league}
	argCount := 1

	if status != "" {
		argCount++
		query += ` AND status = $` + string(rune('0'+argCount))
		args = append(args, status)
	}

	query += ` ORDER BY event_date ASC`

	if limit > 0 {
		argCount++
		query += ` LIMIT $` + string(rune('0'+argCount))
		args = append(args, limit)
	}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*SportsEvent
	for rows.Next() {
		event, err := m.sportsEventByRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// UpcomingSportsEvents returns upcoming (scheduled) events for a league
func (m *Model) UpcomingSportsEvents(ctx context.Context, league SportsLeague, limit int) ([]*SportsEvent, error) {
	const query = `
		SELECT ` + sportsEventColumns + `
		FROM sports_events
		WHERE league = $1 AND status = 'scheduled' AND event_date >= NOW()
		ORDER BY event_date ASC
		LIMIT $2
	`
	rows, err := m.DB.QueryContext(ctx, query, league, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*SportsEvent
	for rows.Next() {
		event, err := m.sportsEventByRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// LinkableSportsEvents returns events that can be linked to a grid (scheduled or in_progress)
func (m *Model) LinkableSportsEvents(ctx context.Context, league SportsLeague, limit int) ([]*SportsEvent, error) {
	const query = `
		SELECT ` + sportsEventColumns + `
		FROM sports_events
		WHERE league = $1 AND status IN ('scheduled', 'in_progress')
		ORDER BY event_date ASC
		LIMIT $2
	`
	rows, err := m.DB.QueryContext(ctx, query, league, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*SportsEvent
	for rows.Next() {
		event, err := m.sportsEventByRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// InProgressSportsEvents returns all events currently in progress
func (m *Model) InProgressSportsEvents(ctx context.Context) ([]*SportsEvent, error) {
	const query = `
		SELECT ` + sportsEventColumns + `
		FROM sports_events
		WHERE status = 'in_progress'
		ORDER BY event_date ASC
	`
	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*SportsEvent
	for rows.Next() {
		event, err := m.sportsEventByRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// EventsNeedingScoreUpdate returns events that may need score updates
func (m *Model) EventsNeedingScoreUpdate(ctx context.Context) ([]*SportsEvent, error) {
	const query = `
		SELECT ` + sportsEventColumns + `
		FROM sports_events
		WHERE (status = 'in_progress' AND event_date >= NOW() - INTERVAL '1 day')
		   OR (status = 'scheduled' AND event_date BETWEEN NOW() AND NOW() + INTERVAL '2 hours')
		   OR (status != 'final' AND event_date >= NOW() - INTERVAL '1 day' AND event_date < NOW())
		ORDER BY event_date ASC
	`
	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*SportsEvent
	for rows.Next() {
		event, err := m.sportsEventByRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// UpsertSportsEvent inserts or updates a sports event by ESPN ID
func (m *Model) UpsertSportsEvent(ctx context.Context, q Queryable, event *SportsEvent) error {
	if q == nil {
		q = m.DB
	}

	const query = `
		INSERT INTO sports_events (
			espn_id, league, name, home_team_id, away_team_id, event_date, season, week, postseason, venue,
			status, status_detail, period, clock, home_score, away_score,
			home_q1, home_q2, home_q3, home_q4, home_ot,
			away_q1, away_q2, away_q3, away_q4, away_ot,
			created, modified, last_synced
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21,
			$22, $23, $24, $25, $26,
			(NOW() AT TIME ZONE 'utc'), (NOW() AT TIME ZONE 'utc'), (NOW() AT TIME ZONE 'utc')
		)
		ON CONFLICT (espn_id) WHERE espn_id IS NOT NULL DO UPDATE SET
			league = EXCLUDED.league,
			name = EXCLUDED.name,
			home_team_id = EXCLUDED.home_team_id,
			away_team_id = EXCLUDED.away_team_id,
			event_date = EXCLUDED.event_date,
			season = EXCLUDED.season,
			week = EXCLUDED.week,
			postseason = EXCLUDED.postseason,
			venue = EXCLUDED.venue,
			status = EXCLUDED.status,
			status_detail = EXCLUDED.status_detail,
			period = EXCLUDED.period,
			clock = EXCLUDED.clock,
			home_score = EXCLUDED.home_score,
			away_score = EXCLUDED.away_score,
			home_q1 = EXCLUDED.home_q1,
			home_q2 = EXCLUDED.home_q2,
			home_q3 = EXCLUDED.home_q3,
			home_q4 = EXCLUDED.home_q4,
			home_ot = EXCLUDED.home_ot,
			away_q1 = EXCLUDED.away_q1,
			away_q2 = EXCLUDED.away_q2,
			away_q3 = EXCLUDED.away_q3,
			away_q4 = EXCLUDED.away_q4,
			away_ot = EXCLUDED.away_ot,
			modified = (NOW() AT TIME ZONE 'utc'),
			last_synced = (NOW() AT TIME ZONE 'utc')
		RETURNING id
	`

	err := q.QueryRowContext(ctx, query,
		event.ESPNID,
		event.League,
		event.Name,
		event.HomeTeamID,
		event.AwayTeamID,
		event.EventDate,
		event.Season,
		event.Week,
		event.Postseason,
		event.Venue,
		event.Status,
		event.StatusDetail,
		event.Period,
		event.Clock,
		event.HomeScore,
		event.AwayScore,
		event.HomeQ1,
		event.HomeQ2,
		event.HomeQ3,
		event.HomeQ4,
		event.HomeOT,
		event.AwayQ1,
		event.AwayQ2,
		event.AwayQ3,
		event.AwayQ4,
		event.AwayOT,
	).Scan(&event.ID)
	return err
}

// FinalizeStaleEvents sets any non-final events with an event_date older than the
// score update lookback window (1 day) to final. This catches events that fell out
// of the EventsNeedingScoreUpdate window while still in progress.
func (m *Model) FinalizeStaleEvents(ctx context.Context) (int64, error) {
	const query = `
		UPDATE sports_events
		SET status = 'final',
		    modified = (NOW() AT TIME ZONE 'utc'),
		    home_score = NULL,
		    away_score = NULL,
		    home_q1 = NULL,
		    home_q2 = NULL,
		    home_q3 = NULL,
		    home_q4 = NULL,
		    home_ot = NULL,
		    away_q1 = NULL,
		    away_q2 = NULL,
		    away_q3 = NULL,
		    away_q4 = NULL,
		    away_ot = NULL,
		    period = NULL,
		    clock = NULL,
		    status_detail = NULL
		WHERE status != 'final'
		  AND event_date < NOW() - INTERVAL '1 day'
	`
	result, err := m.DB.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// NewSportsEvent creates a new SportsEvent instance
func (m *Model) NewSportsEvent() *SportsEvent {
	return &SportsEvent{
		model:  m,
		Status: SportsEventStatusScheduled,
	}
}

// SportsEventCount returns the count of events optionally filtered by league
func (m *Model) SportsEventCount(ctx context.Context, league SportsLeague) (int, error) {
	var count int
	var err error

	if league == "" {
		err = m.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sports_events`).Scan(&count)
	} else {
		err = m.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sports_events WHERE league = $1`, league).Scan(&count)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

// sportsTeamKey represents a unique team identifier (id + league)
type sportsTeamKey struct {
	ID     string
	League SportsLeague
}

// SearchSportsEvents searches for events by team name with pagination
// It joins to sports_teams and searches via ILIKE on name, full_name, and abbreviation
func (m *Model) SearchSportsEvents(ctx context.Context, league SportsLeague, status string, search string, offset int64, limit int) ([]*SportsEvent, int64, error) {
	searchPattern := "%" + search + "%"

	// Build the base query with JOINs
	baseQuery := `
		FROM sports_events e
		INNER JOIN sports_teams ht ON ht.id = e.home_team_id AND ht.league = e.league
		INNER JOIN sports_teams at ON at.id = e.away_team_id AND at.league = e.league
		WHERE e.league = $1
		  AND e.status IN ('scheduled', 'in_progress')
		  AND (ht.name ILIKE $2 OR ht.full_name ILIKE $2 OR ht.abbreviation ILIKE $2
		    OR at.name ILIKE $2 OR at.full_name ILIKE $2 OR at.abbreviation ILIKE $2)`

	// Count query
	countQuery := `SELECT COUNT(*) ` + baseQuery
	var total int64
	if err := m.DB.QueryRowContext(ctx, countQuery, league, searchPattern).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query with pagination
	dataQuery := `SELECT ` + sportsEventColumnsWithPrefix + ` ` + baseQuery + `
		ORDER BY e.event_date ASC
		OFFSET $3 LIMIT $4`

	rows, err := m.DB.QueryContext(ctx, dataQuery, league, searchPattern, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*SportsEvent
	for rows.Next() {
		event, err := m.sportsEventByRow(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// LinkableSportsEventsWithTotal returns linkable events with a total count for pagination
func (m *Model) LinkableSportsEventsWithTotal(ctx context.Context, league SportsLeague, offset int64, limit int) ([]*SportsEvent, int64, error) {
	// Count query
	const countQuery = `
		SELECT COUNT(*)
		FROM sports_events
		WHERE league = $1 AND status IN ('scheduled', 'in_progress')
	`
	var total int64
	if err := m.DB.QueryRowContext(ctx, countQuery, league).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query
	const query = `
		SELECT ` + sportsEventColumns + `
		FROM sports_events
		WHERE league = $1 AND status IN ('scheduled', 'in_progress')
		ORDER BY event_date ASC
		OFFSET $2 LIMIT $3
	`
	rows, err := m.DB.QueryContext(ctx, query, league, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*SportsEvent
	for rows.Next() {
		event, err := m.sportsEventByRow(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// LoadTeamsForSportsEvents loads teams for a slice of events
func (m *Model) LoadTeamsForSportsEvents(ctx context.Context, events []*SportsEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Collect unique team keys (id + league)
	teamKeys := make(map[sportsTeamKey]bool)
	for _, e := range events {
		teamKeys[sportsTeamKey{ID: e.HomeTeamID, League: e.League}] = true
		teamKeys[sportsTeamKey{ID: e.AwayTeamID, League: e.League}] = true
	}

	// Fetch all teams
	teams := make(map[sportsTeamKey]*SportsTeam)
	for key := range teamKeys {
		team, err := m.SportsTeamByID(ctx, key.ID, key.League)
		if err != nil {
			return err
		}
		teams[key] = team
	}

	// Assign teams to events
	for _, e := range events {
		e.homeTeam = teams[sportsTeamKey{ID: e.HomeTeamID, League: e.League}]
		e.awayTeam = teams[sportsTeamKey{ID: e.AwayTeamID, League: e.League}]
	}

	return nil
}

// SyncGridsFromEvent syncs team names and colors from a sports event to all linked active grids.
// Team names are always synced to match the event's teams. Colors are only set when the grid's
// colors are currently null (unset), so user-customized colors are preserved.
// Returns the total number of rows affected across both queries.
func (m *Model) SyncGridsFromEvent(ctx context.Context, eventID int64) (int64, error) {
	// Update grid team names where they differ or are null
	const namesQuery = `
		UPDATE grids g
		SET home_team_name = st_home.full_name,
		    away_team_name = st_away.full_name,
		    modified = (NOW() AT TIME ZONE 'utc')
		FROM sports_events se
		JOIN sports_teams st_home ON (se.home_team_id = st_home.id AND se.league = st_home.league)
		JOIN sports_teams st_away ON (se.away_team_id = st_away.id AND se.league = st_away.league)
		WHERE g.sports_event_id = se.id
		  AND se.id = $1
		  AND g.state = 'active'
		  AND (g.home_team_name IS DISTINCT FROM st_home.full_name
		       OR g.away_team_name IS DISTINCT FROM st_away.full_name)
	`
	namesResult, err := m.DB.ExecContext(ctx, namesQuery, eventID)
	if err != nil {
		return 0, fmt.Errorf("syncing grid team names: %w", err)
	}
	namesCount, err := namesResult.RowsAffected()
	if err != nil {
		return 0, err
	}

	// Update grid_settings colors only when currently null
	const colorsQuery = `
		UPDATE grid_settings gs
		SET home_team_color_1 = '#' || st_home.color,
		    home_team_color_2 = '#' || st_home.alternate_color,
		    away_team_color_1 = '#' || st_away.color,
		    away_team_color_2 = '#' || st_away.alternate_color,
		    modified = (NOW() AT TIME ZONE 'utc')
		FROM grids g
		JOIN sports_events se ON g.sports_event_id = se.id
		JOIN sports_teams st_home ON (se.home_team_id = st_home.id AND se.league = st_home.league)
		JOIN sports_teams st_away ON (se.away_team_id = st_away.id AND se.league = st_away.league)
		WHERE gs.grid_id = g.id
		  AND se.id = $1
		  AND g.state = 'active'
		  AND gs.home_team_color_1 IS NULL
		  AND gs.away_team_color_1 IS NULL
		  AND st_home.color IS NOT NULL
		  AND st_away.color IS NOT NULL
	`
	colorsResult, err := m.DB.ExecContext(ctx, colorsQuery, eventID)
	if err != nil {
		return 0, fmt.Errorf("syncing grid team colors: %w", err)
	}
	colorsCount, err := colorsResult.RowsAffected()
	if err != nil {
		return 0, err
	}

	return namesCount + colorsCount, nil
}

// PoolTokensByEventID returns pool tokens for all pools that have active grids linked to the given sports event.
func (m *Model) PoolTokensByEventID(ctx context.Context, eventID int64) ([]string, error) {
	const query = `SELECT DISTINCT p.token FROM pools p INNER JOIN grids g ON g.pool_id = p.id WHERE g.sports_event_id = $1 AND g.state = 'active'`
	rows, err := m.DB.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if tokens == nil {
		tokens = []string{}
	}
	return tokens, nil
}

// NotifySportsEventUpdated sends a PostgreSQL NOTIFY on the 'sports_event_updated' channel with the event ID as payload.
func (m *Model) NotifySportsEventUpdated(ctx context.Context, eventID int64) error {
	_, err := m.DB.ExecContext(ctx, `SELECT pg_notify('sports_event_updated', $1)`, fmt.Sprintf("%d", eventID))
	return err
}

// Deprecated aliases for backward compatibility
type BDLEventStatus = SportsEventStatus
type BDLEvent = SportsEvent
type BDLEventJSON = SportsEventJSON

const (
	BDLEventStatusScheduled  = SportsEventStatusScheduled
	BDLEventStatusInProgress = SportsEventStatusInProgress
	BDLEventStatusFinal      = SportsEventStatusFinal
)

func (m *Model) BDLEventByID(ctx context.Context, id int64) (*BDLEvent, error) {
	return m.SportsEventByID(ctx, id)
}

func (m *Model) BDLEventByIDWithTeams(ctx context.Context, id int64) (*BDLEvent, error) {
	return m.SportsEventByIDWithTeams(ctx, id)
}

func (m *Model) BDLEventsByLeague(ctx context.Context, league BDLLeague, status string, limit int) ([]*BDLEvent, error) {
	return m.SportsEventsByLeague(ctx, league, status, limit)
}

func (m *Model) UpcomingBDLEvents(ctx context.Context, league BDLLeague, limit int) ([]*BDLEvent, error) {
	return m.UpcomingSportsEvents(ctx, league, limit)
}

func (m *Model) LinkableBDLEvents(ctx context.Context, league BDLLeague, limit int) ([]*BDLEvent, error) {
	return m.LinkableSportsEvents(ctx, league, limit)
}

func (m *Model) InProgressBDLEvents(ctx context.Context) ([]*BDLEvent, error) {
	return m.InProgressSportsEvents(ctx)
}

func (m *Model) NewBDLEvent() *BDLEvent {
	return m.NewSportsEvent()
}

func (m *Model) UpsertBDLEvent(ctx context.Context, q Queryable, event *BDLEvent) error {
	return m.UpsertSportsEvent(ctx, q, event)
}

func (m *Model) BDLEventCount(ctx context.Context, league BDLLeague) (int, error) {
	return m.SportsEventCount(ctx, league)
}

func (m *Model) LoadTeamsForEvents(ctx context.Context, events []*BDLEvent) error {
	return m.LoadTeamsForSportsEvents(ctx, events)
}
