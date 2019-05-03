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
	"database/sql"
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"github.com/synacor/argon2id"
)

// NameMaxLength is the maximum length the grid name may be
const NameMaxLength = 50

// Grid is an individual grid board
// This object uses getters and setters to help guard against user input.
type Grid struct {
	model        *Model
	id           int64
	token        string
	userID       int64
	name         string
	gridType     GridType
	passwordHash string
	locks        time.Time
	created      time.Time
	modified     time.Time

	settings GridSettings
	squares  map[int]*GridSquare
}

// GridWithID returns an empty grid object with only the ID set
func GridWithID(id int64) *Grid {
	return &Grid{id: id}
}

type gridJSON struct {
	Token    string       `json:"token"`
	Name     string       `json:"name"`
	GridType GridType     `json:"gridType"`
	Locks    time.Time    `json:"locks"`
	Created  time.Time    `json:"created"`
	Modified time.Time    `json:"modified"`
	Settings GridSettings `json:"settings"`
}

// ID returns the id
func (g *Grid) ID() int64 {
	return g.id
}

// Token is a getter for the token
func (g *Grid) Token() string {
	return g.token
}

// Name is a getter for the name
func (g *Grid) Name() string {
	return g.name
}

// Locks is a getter for the locks date
func (g *Grid) Locks() time.Time {
	return g.locks
}

// Created is a getter for the locks date
func (g *Grid) Created() time.Time {
	return g.created
}

// Modified is a getter for the modified date
func (g *Grid) Modified() time.Time {
	return g.modified
}

// SetName is a setter for the name
func (g *Grid) SetName(name string) {
	if utf8.RuneCountInString(name) > NameMaxLength {
		name = string([]rune(name)[0:NameMaxLength])
	}

	g.name = name
}

// GridType is a getter for the grid type
func (g *Grid) GridType() GridType {
	return g.gridType
}

// SetGridType is a setter for the grid type
func (g *Grid) SetGridType(gridType GridType) {
	g.gridType = gridType
}

// MarshalJSON provides custom JSON marshalling
func (g *Grid) MarshalJSON() ([]byte, error) {
	return json.Marshal(gridJSON{
		Token:    g.token,
		Name:     g.name,
		GridType: g.gridType,
		Locks:    g.locks,
		Created:  g.created,
		Modified: g.modified,
		Settings: g.settings,
	})
}

type executer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type scanFunc func(dest ...interface{}) error

func (m *Model) gridByRow(scan scanFunc, loadSettings bool) (*Grid, error) {
	var locks *time.Time
	s := Grid{model: m}
	if err := scan(&s.id, &s.token, &s.userID, &s.name, &s.gridType, &s.passwordHash, &locks, &s.created, &s.modified); err != nil {
		return nil, err
	}

	// XXX: do we want the ability to let the user choose the time zone?
	s.created = s.created.In(locationNewYork)
	s.modified = s.modified.In(locationNewYork)

	if locks != nil {
		s.locks = *locks
	}

	if loadSettings {
		if err := s.LoadSettings(); err != nil {
			return nil, err
		}
	}

	return &s, nil
}

// GridsJoinedByUser will return a collection of grids that the user joined
func (m *Model) GridsJoinedByUser(ctx context.Context, u *User, offset, limit int) ([]*Grid, error) {
	const query = `
		SELECT grids.*
		FROM grids
		LEFT JOIN grids_users ON grids.id = grids_users.grid_id
		WHERE grids_users.user_id = $1
		ORDER BY grids.id DESC
		OFFSET $2
		LIMIT $3`

	return m.gridsByRows(m.db.QueryContext(ctx, query, u.ID, offset, limit))
}

// GridsJoinedByUserCount will return a how many grids the user joined
func (m *Model) GridsJoinedByUserCount(ctx context.Context, u *User) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM grids
		LEFT JOIN grids_users ON grids.id = grids_users.grid_id
		WHERE grids_users.user_id = $1`

	return m.gridsCount(m.db.QueryRowContext(ctx, query, u.ID))
}

// GridsOwnedByUser will return a collection of grids that were created by the user
func (m *Model) GridsOwnedByUser(ctx context.Context, u *User, offset, limit int) ([]*Grid, error) {
	const query = `
		SELECT *
		FROM grids
		WHERE user_id = $1
		ORDER BY grids.id DESC
		OFFSET $2
		LIMIT $3`

	return m.gridsByRows(m.db.QueryContext(ctx, query, u.ID, offset, limit))
}

// GridsOwnedByUserCount will return how many grids were created by the user
func (m *Model) GridsOwnedByUserCount(ctx context.Context, u *User) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM grids
		WHERE user_id = $1`

	return m.gridsCount(m.db.QueryRowContext(ctx, query, u.ID))
}

func (m *Model) gridsByRows(rows *sql.Rows, err error) ([]*Grid, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	collection := make([]*Grid, 0)
	for rows.Next() {
		grid, err := m.gridByRow(rows.Scan, false)
		if err != nil {
			return nil, err
		}

		collection = append(collection, grid)
	}

	return collection, nil
}

func (m *Model) gridsCount(row *sql.Row) (int64, error) {
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// GridByToken will return the grids with the matching token
func (m *Model) GridByToken(ctx context.Context, token string) (*Grid, error) {
	row := m.db.QueryRowContext(ctx, "SELECT * FROM grids WHERE token = $1", token)
	return m.gridByRow(row.Scan, true)
}

// GridByID will return the grids with the matching ID
func (m *Model) GridByID(id int64) (*Grid, error) {
	row := m.db.QueryRow("SELECT * FROM grids WHERE id = $1", id)
	return m.gridByRow(row.Scan, true)
}

// NewGrid will save new grid into the database
func (m *Model) NewGrid(userID int64, name string, gridType GridType, password string) (*Grid, error) {
	if err := IsValidGridType(string(gridType)); err != nil {
		return nil, err
	}

	token, err := m.NewToken()
	if err != nil {
		return nil, err
	}

	passwordHash, err := argon2id.DefaultHashPassword(password)
	if err != nil {
		return nil, err
	}
	row := m.db.QueryRow("SELECT * FROM new_grid($1, $2, $3, $4, $5, $6)", token, userID, name, gridType, passwordHash, gridType.Squares())

	s, err := m.gridByRow(row.Scan, false)
	if err != nil {
		return nil, err
	}

	s.settings = GridSettings{gridID: s.id}
	return s, nil
}

// SetPassword will set a new password and ensures that it's properly hashed
func (g *Grid) SetPassword(password string) error {
	passwordHash, err := argon2id.DefaultHashPassword(password)
	if err != nil {
		return err
	}

	g.passwordHash = passwordHash
	return nil
}

// LoadSettings will update the settings from the database
func (g *Grid) LoadSettings() error {
	row := g.model.db.QueryRow(`
		SELECT grid_id,
			   home_team_name, home_team_color_1, home_team_color_2,
			   away_team_name, away_team_color_1, away_team_color_2,
			   notes, modified
		FROM grid_settings
		WHERE grid_id = $1
	`, g.id)

	return row.Scan(
		&g.settings.gridID,
		&g.settings.homeTeamName,
		&g.settings.homeTeamColor1,
		&g.settings.homeTeamColor2,
		&g.settings.awayTeamName,
		&g.settings.awayTeamColor1,
		&g.settings.awayTeamColor2,
		&g.settings.notes,
		&g.settings.modified,
	)
}

// Settings returns the grid settings
func (g *Grid) Settings() *GridSettings {
	return &g.settings
}

// Save will save the grid and settings using a transaction
func (g *Grid) Save() error {
	tx, err := g.model.db.Begin()
	if err != nil {
		return err
	}

	var locks *time.Time
	if !g.locks.IsZero() {
		locks = &g.locks
	}

	if _, err := tx.Exec("UPDATE grids SET name = $1, grid_type = $2, password_hash = $3, locks = $4, modified = (NOW() AT TIME ZONE 'utc')  WHERE id = $5",
		g.name, g.gridType, g.passwordHash, locks, g.id); err != nil {
		tx.Rollback()
		return err
	}

	if err := g.settings.Save(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// PasswordIsValid is will return true if the password matches
func (g *Grid) PasswordIsValid(password string) bool {
	if err := argon2id.Compare(g.passwordHash, password); err != nil {
		if err != argon2id.ErrMismatchedHashAndPassword {
			logrus.WithError(err).Error("could not check password")
		}

		return false
	}

	return true
}

// Squares will return the squares that belong to a grid. This method will lazily load the squares
func (g *Grid) Squares() (map[int]*GridSquare, error) {
	if g.squares == nil {
		rows, err := g.model.db.Query("SELECT id, square_id, state, claimant, modified FROM grid_squares WHERE grid_id = $1 ORDER BY square_id", g.id)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		squares := make(map[int]*GridSquare)
		for rows.Next() {
			gs, err := g.squareByRow(rows.Scan)
			if err != nil {
				return nil, err
			}
			squares[gs.SquareID] = gs
		}

		g.squares = squares
	}

	return g.squares, nil
}

// SquareBySquareID will return a single square based on the square ID
func (g *Grid) SquareBySquareID(squareID int) (*GridSquare, error) {
	row := g.model.db.QueryRow("SELECT id, square_id, state, claimant, modified FROM grid_squares WHERE grid_id = $1 AND square_id = $2", g.id, squareID)
	return g.squareByRow(row.Scan)
}

func (g *Grid) squareByRow(scan scanFunc) (*GridSquare, error) {
	gs := GridSquare{
		Model:  g.model,
		GridID: g.id,
	}

	var claimant *string
	if err := scan(&gs.ID, &gs.SquareID, &gs.State, &claimant, &gs.Modified); err != nil {
		return nil, err
	}

	if claimant != nil {
		gs.Claimant = *claimant
	}

	gs.Modified = gs.Modified.In(locationNewYork)

	return &gs, nil
}

// Logs will return all grid square logs for the grid
func (g *Grid) Logs(ctx context.Context, offset, limit int) ([]*GridSquareLog, error) {
	const query = `
		SELECT grid_squares_logs.id, grid_square_id, square_id, user_id, grid_squares_logs.state, grid_squares_logs.claimant, remote_addr, note, grid_squares_logs.created
		FROM grid_squares_logs
		INNER JOIN grid_squares ON grid_squares_logs.grid_square_id = grid_squares.id
		WHERE grid_squares.grid_id = $1
		ORDER BY grid_squares_logs.id DESC
		OFFSET $2
		LIMIT $3`
	rows, err := g.model.db.QueryContext(ctx, query, g.id, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]*GridSquareLog, 0)
	for rows.Next() {
		l, err := gridSquareLogByRow(rows.Scan)
		if err != nil {
			return nil, err
		}

		logs = append(logs, l)
	}

	return logs, nil
}

// LogsCount will return how many logs exist for the given grid
func (g *Grid) LogsCount(ctx context.Context) (int64, error) {
	const query = `
		SELECT COUNT(grid_squares_logs.*)
		FROM grid_squares_logs
		INNER JOIN grid_squares ON grid_squares_logs.grid_square_id = grid_squares.id
		WHERE grid_squares.grid_id = $1`
	row := g.model.db.QueryRowContext(ctx, query, g.id)

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}
