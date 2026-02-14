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
	"time"
)

// SportsTeam represents a cached team from the sports API
type SportsTeam struct {
	model *Model

	ID             string
	League         SportsLeague
	Name           string
	FullName       string
	Abbreviation   string
	Conference     *string
	Division       *string
	Location       *string
	Color          *string
	AlternateColor *string
	Created        time.Time
	Modified       time.Time
}

// SportsTeamJSON represents team data for JSON serialization
type SportsTeamJSON struct {
	ID             string       `json:"id"`
	League         SportsLeague `json:"league"`
	Name           string       `json:"name"`
	FullName       string       `json:"fullName"`
	Abbreviation   string       `json:"abbreviation"`
	Conference     string       `json:"conference,omitempty"`
	Division       string       `json:"division,omitempty"`
	Location       string       `json:"location,omitempty"`
	Color          string       `json:"color,omitempty"`
	AlternateColor string       `json:"alternateColor,omitempty"`
}

// JSON returns the JSON representation of the team
func (t *SportsTeam) JSON() *SportsTeamJSON {
	json := &SportsTeamJSON{
		ID:           t.ID,
		League:       t.League,
		Name:         t.Name,
		FullName:     t.FullName,
		Abbreviation: t.Abbreviation,
	}
	if t.Conference != nil {
		json.Conference = *t.Conference
	}
	if t.Division != nil {
		json.Division = *t.Division
	}
	if t.Location != nil {
		json.Location = *t.Location
	}
	if t.Color != nil {
		json.Color = *t.Color
	}
	if t.AlternateColor != nil {
		json.AlternateColor = *t.AlternateColor
	}
	return json
}

const sportsTeamColumns = `id, league, name, full_name, abbreviation, conference, division, location, color, alternate_color, created, modified`

func (m *Model) sportsTeamByRow(scan scanFunc) (*SportsTeam, error) {
	team := &SportsTeam{model: m}
	if err := scan(
		&team.ID,
		&team.League,
		&team.Name,
		&team.FullName,
		&team.Abbreviation,
		&team.Conference,
		&team.Division,
		&team.Location,
		&team.Color,
		&team.AlternateColor,
		&team.Created,
		&team.Modified,
	); err != nil {
		return nil, err
	}
	return team, nil
}

// SportsTeamByID returns a team by its ID and league
func (m *Model) SportsTeamByID(ctx context.Context, id string, league SportsLeague) (*SportsTeam, error) {
	const query = `SELECT ` + sportsTeamColumns + ` FROM sports_teams WHERE id = $1 AND league = $2`
	row := m.DB.QueryRowContext(ctx, query, id, league)
	return m.sportsTeamByRow(row.Scan)
}

// SportsTeamsByLeague returns all teams for a given league
func (m *Model) SportsTeamsByLeague(ctx context.Context, league SportsLeague) ([]*SportsTeam, error) {
	const query = `SELECT ` + sportsTeamColumns + ` FROM sports_teams WHERE league = $1 ORDER BY name`
	rows, err := m.DB.QueryContext(ctx, query, league)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*SportsTeam
	for rows.Next() {
		team, err := m.sportsTeamByRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return teams, nil
}

// UpsertSportsTeam inserts or updates a sports team
func (m *Model) UpsertSportsTeam(ctx context.Context, q Queryable, team *SportsTeam) error {
	if q == nil {
		q = m.DB
	}

	const query = `
		INSERT INTO sports_teams (id, league, name, full_name, abbreviation, conference, division, location, color, alternate_color, created, modified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, (NOW() AT TIME ZONE 'utc'), (NOW() AT TIME ZONE 'utc'))
		ON CONFLICT (id, league) DO UPDATE SET
			name = EXCLUDED.name,
			full_name = EXCLUDED.full_name,
			abbreviation = EXCLUDED.abbreviation,
			conference = COALESCE(EXCLUDED.conference, sports_teams.conference),
			division = COALESCE(EXCLUDED.division, sports_teams.division),
			location = COALESCE(EXCLUDED.location, sports_teams.location),
			color = COALESCE(EXCLUDED.color, sports_teams.color),
			alternate_color = COALESCE(EXCLUDED.alternate_color, sports_teams.alternate_color),
			modified = (NOW() AT TIME ZONE 'utc')
	`

	_, err := q.ExecContext(ctx, query,
		team.ID,
		team.League,
		team.Name,
		team.FullName,
		team.Abbreviation,
		team.Conference,
		team.Division,
		team.Location,
		team.Color,
		team.AlternateColor,
	)
	return err
}

// NewSportsTeam creates a new SportsTeam instance
func (m *Model) NewSportsTeam() *SportsTeam {
	return &SportsTeam{model: m}
}

// SportsTeamCount returns the count of teams for a league
func (m *Model) SportsTeamCount(ctx context.Context, league SportsLeague) (int, error) {
	var count int
	var err error

	if league == "" {
		err = m.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sports_teams`).Scan(&count)
	} else {
		err = m.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sports_teams WHERE league = $1`, league).Scan(&count)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

// Deprecated aliases for backward compatibility
type BDLTeam = SportsTeam
type BDLTeamJSON = SportsTeamJSON

func (m *Model) BDLTeamByID(ctx context.Context, id int64, league BDLLeague) (*BDLTeam, error) {
	// Convert int64 to string for the new schema
	return m.SportsTeamByID(ctx, idToString(id), league)
}

func (m *Model) BDLTeamsByLeague(ctx context.Context, league BDLLeague) ([]*BDLTeam, error) {
	return m.SportsTeamsByLeague(ctx, league)
}

func (m *Model) UpsertBDLTeam(ctx context.Context, q Queryable, team *BDLTeam) error {
	return m.UpsertSportsTeam(ctx, q, team)
}

func (m *Model) NewBDLTeam() *BDLTeam {
	return m.NewSportsTeam()
}

func (m *Model) BDLTeamCount(ctx context.Context, league BDLLeague) (int, error) {
	return m.SportsTeamCount(ctx, league)
}

// Helper to convert int64 ID to string
func idToString(id int64) string {
	return fmt.Sprintf("%d", id)
}
