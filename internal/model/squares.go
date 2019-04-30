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

// NameMaxLength is the maximum length the squares name may be
const NameMaxLength = 50

// Squares is an individual squares board
// This object uses getters and setters to help guard against user input.
type Squares struct {
	model        *Model
	id           int64
	token        string
	userID       int64
	name         string
	squaresType  SquaresType
	passwordHash string
	locks        time.Time
	created      time.Time
	modified     time.Time

	settings SquaresSettings
}

// SquaresWithID returns an empty squares object with only the ID set
func SquaresWithID(id int64) *Squares {
	return &Squares{id: id}
}

type squaresJSON struct {
	Token       string          `json:"token"`
	Name        string          `json:"name"`
	SquaresType SquaresType     `json:"squaresType"`
	Locks       time.Time       `json:"locks"`
	Created     time.Time       `json:"created"`
	Modified    time.Time       `json:"modified"`
	Settings    SquaresSettings `json:"settings"`
}

// ID returns the id
func (s *Squares) ID() int64 {
	return s.id
}

// Token is a getter for the token
func (s *Squares) Token() string {
	return s.token
}

// Name is a getter for the name
func (s *Squares) Name() string {
	return s.name
}

// Locks is a getter for the locks date
func (s *Squares) Locks() time.Time {
	return s.locks
}

// Created is a getter for the locks date
func (s *Squares) Created() time.Time {
	return s.created
}

// Modified is a getter for the modified date
func (s *Squares) Modified() time.Time {
	return s.modified
}

// SetName is a setter for the name
func (s *Squares) SetName(name string) {
	if utf8.RuneCountInString(name) > NameMaxLength {
		name = string([]rune(name)[0:NameMaxLength])
	}

	s.name = name
}

// SquaresType is a getter for the squares type
func (s *Squares) SquaresType() SquaresType {
	return s.squaresType
}

// SetSquaresType is a setter for the squares type
func (s *Squares) SetSquaresType(squaresType SquaresType) {
	s.squaresType = squaresType
}

// MarshalJSON provides custom JSON marshalling
func (s *Squares) MarshalJSON() ([]byte, error) {
	return json.Marshal(squaresJSON{
		Token:       s.token,
		Name:        s.name,
		SquaresType: s.squaresType,
		Locks:       s.locks,
		Created:     s.created,
		Modified:    s.modified,
		Settings:    s.settings,
	})
}

type executer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type scanFunc func(dest ...interface{}) error

func (m *Model) squaresByRow(scan scanFunc, loadSettings bool) (*Squares, error) {
	var locks *time.Time
	s := Squares{model: m}
	if err := scan(&s.id, &s.token, &s.userID, &s.name, &s.squaresType, &s.passwordHash, &locks, &s.created, &s.modified); err != nil {
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

	s.settings = SquaresSettings{squaresID: s.id}
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
	`, s.id)

	return row.Scan(
		&s.settings.squaresID,
		&s.settings.homeTeamName,
		&s.settings.homeTeamColor1,
		&s.settings.homeTeamColor2,
		&s.settings.awayTeamName,
		&s.settings.awayTeamColor1,
		&s.settings.awayTeamColor2,
		&s.settings.notes,
		&s.settings.modified,
	)
}

// Settings returns the square settings
func (s *Squares) Settings() *SquaresSettings {
	return &s.settings
}

// Save will save the squares and settings using a transaction
func (s *Squares) Save() error {
	tx, err := s.model.db.Begin()
	if err != nil {
		return err
	}

	var locks *time.Time
	if !s.locks.IsZero() {
		locks = &s.locks
	}

	if _, err := tx.Exec("UPDATE squares SET name = $1, squares_type = $2, password_hash = $3, locks = $4, modified = (NOW() AT TIME ZONE 'utc')  WHERE id = $5",
		s.name, s.squaresType, s.passwordHash, locks, s.id); err != nil {
		tx.Rollback()
		return err
	}

	if err := s.settings.Save(tx); err != nil {
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
