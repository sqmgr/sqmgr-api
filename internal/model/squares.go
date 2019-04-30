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
	"time"

	"github.com/sirupsen/logrus"
	"github.com/synacor/argon2id"
)

// Squares is an individual squares board
type Squares struct {
	model        *Model
	ID           int64       `json:"-"`
	Token        string      `json:"token"`
	UserID       int64       `json:"-"`
	Name         string      `json:"name"`
	SquaresType  SquaresType `json:"squaresType"`
	passwordHash string
	Locks        time.Time `json:"locks"`
	Created      time.Time `json:"created"`
	Modified     time.Time `json:"modified"`

	Settings SquaresSettings `json:"settings"`
}

type executer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type scanFunc func(dest ...interface{}) error

func (m *Model) squaresByRow(scan scanFunc, loadSettings bool) (*Squares, error) {
	var locks *time.Time
	s := Squares{model: m}
	if err := scan(&s.ID, &s.Token, &s.UserID, &s.Name, &s.SquaresType, &s.passwordHash, &locks, &s.Created, &s.Modified); err != nil {
		return nil, err
	}

	// XXX: do we want the ability to let the user choose the time zone?
	s.Created = s.Created.In(locationNewYork)
	s.Modified = s.Modified.In(locationNewYork)

	if locks != nil {
		s.Locks = *locks
	}

	if loadSettings {
		if err := s.LoadSettings(); err != nil {
			return nil, err
		}
	}

	return &s, nil
}

// SquaresCollectionJoinedByUser will return a collection of squares that the user joined
func (m *Model) SquaresCollectionJoinedByUser(ctx context.Context, u *User, offset, limit int) ([]*Squares, error) {
	const query = `
		SELECT squares.*
		FROM squares
		LEFT JOIN squares_users ON squares.id = squares_users.squares_id
		WHERE squares_users.user_id = $1
		ORDER BY squares.id DESC
		OFFSET $2
		LIMIT $3`

	return m.squaresCollectionByRows(m.db.QueryContext(ctx, query, u.ID, offset, limit))
}

// SquaresCollectionJoinedByUserCount will return a how many squares the user joined
func (m *Model) SquaresCollectionJoinedByUserCount(ctx context.Context, u *User) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM squares
		LEFT JOIN squares_users ON squares.id = squares_users.squares_id
		WHERE squares_users.user_id = $1`

	return m.squaresCollectionCount(m.db.QueryRowContext(ctx, query, u.ID))
}

// SquaresCollectionOwnedByUser will return a collection of squares that were created by the user
func (m *Model) SquaresCollectionOwnedByUser(ctx context.Context, u *User, offset, limit int) ([]*Squares, error) {
	const query = `
		SELECT *
		FROM squares
		WHERE user_id = $1
		ORDER BY squares.id DESC
		OFFSET $2
		LIMIT $3`

	return m.squaresCollectionByRows(m.db.QueryContext(ctx, query, u.ID, offset, limit))
}

// SquaresCollectionOwnedByUserCount will return how many squares were created by the user
func (m *Model) SquaresCollectionOwnedByUserCount(ctx context.Context, u *User) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM squares
		WHERE user_id = $1`

	return m.squaresCollectionCount(m.db.QueryRowContext(ctx, query, u.ID))
}

func (m *Model) squaresCollectionByRows(rows *sql.Rows, err error) ([]*Squares, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	squaresCollection := make([]*Squares, 0)
	for rows.Next() {
		squares, err := m.squaresByRow(rows.Scan, false)
		if err != nil {
			return nil, err
		}

		squaresCollection = append(squaresCollection, squares)
	}

	return squaresCollection, nil
}

func (m *Model) squaresCollectionCount(row *sql.Row) (int64, error) {
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// SquaresByToken will return the squares with the matching token
func (m *Model) SquaresByToken(ctx context.Context, token string) (*Squares, error) {
	row := m.db.QueryRowContext(ctx, "SELECT * FROM squares WHERE token = $1", token)
	return m.squaresByRow(row.Scan, true)
}

// SquaresByID will return the squares with the matching ID
func (m *Model) SquaresByID(id int64) (*Squares, error) {
	row := m.db.QueryRow("SELECT * FROM squares WHERE id = $1", id)
	return m.squaresByRow(row.Scan, true)
}

// NewSquares will save new squares into the database
func (m *Model) NewSquares(userID int64, name string, squaresType SquaresType, password string) (*Squares, error) {
	if err := IsValidSquaresType(string(squaresType)); err != nil {
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
	row := m.db.QueryRow("SELECT * FROM new_squares($1, $2, $3, $4, $5)", token, userID, name, squaresType, passwordHash)

	s, err := m.squaresByRow(row.Scan, false)
	if err != nil {
		return nil, err
	}

	s.Settings = SquaresSettings{squaresID: s.ID}
	return s, nil
}

// SetPassword will set a new password and ensures that it's properly hashed
func (s *Squares) SetPassword(password string) error {
	passwordHash, err := argon2id.DefaultHashPassword(password)
	if err != nil {
		return err
	}

	s.passwordHash = passwordHash
	return nil
}

// LoadSettings will update the settings from the database
func (s *Squares) LoadSettings() error {
	row := s.model.db.QueryRow(`
		SELECT squares_id,
			   home_team_name, home_team_color_1, home_team_color_2,
			   away_team_name, away_team_color_1, away_team_color_2,
			   notes, modified
		FROM squares_settings
		WHERE squares_id = $1
	`, s.ID)

	return row.Scan(
		&s.Settings.squaresID,
		&s.Settings.homeTeamName,
		&s.Settings.homeTeamColor1,
		&s.Settings.homeTeamColor2,
		&s.Settings.awayTeamName,
		&s.Settings.awayTeamColor1,
		&s.Settings.awayTeamColor2,
		&s.Settings.notes,
		&s.Settings.modified,
	)
}

// Save will save the squares and settings using a transaction
func (s *Squares) Save() error {
	tx, err := s.model.db.Begin()
	if err != nil {
		return err
	}

	var locks *time.Time
	if !s.Locks.IsZero() {
		locks = &s.Locks
	}

	if _, err := tx.Exec("UPDATE squares SET name = $1, squares_type = $2, password_hash = $3, locks = $4, modified = (NOW() AT TIME ZONE 'utc')  WHERE id = $5",
		s.Name, s.SquaresType, s.passwordHash, locks, s.ID); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.Settings.Save(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// PasswordIsValid is will return true if the password matches
func (s *Squares) PasswordIsValid(password string) bool {
	if err := argon2id.Compare(s.passwordHash, password); err != nil {
		if err != argon2id.ErrMismatchedHashAndPassword {
			logrus.WithError(err).Error("could not check password")
		}

		return false
	}

	return true
}
