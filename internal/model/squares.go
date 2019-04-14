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

func (m *Model) squaresByRow(row *sql.Row, loadSettings bool) (*Squares, error) {
	var locks *time.Time
	s := Squares{model: m}
	if err := row.Scan(&s.ID, &s.Token, &s.UserID, &s.Name, &s.SquaresType, &s.passwordHash, &locks, &s.Created, &s.Modified); err != nil {
		return nil, err
	}

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

func (m *Model) SquaresByToken(token string) (*Squares, error) {
	row := m.db.QueryRow("SELECT * FROM squares WHERE token = $1", token)
	return m.squaresByRow(row, true)
}

func (m *Model) SquaresByID(id int64) (*Squares, error) {
	row := m.db.QueryRow("SELECT * FROM squares WHERE id = $1", id)
	return m.squaresByRow(row, true)
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

	s, err := m.squaresByRow(row, false)
	if err != nil {
		return nil, err
	}

	s.Settings = SquaresSettings{SquaresID: s.ID}
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
		SELECT home_team_name, home_team_color_1, home_team_color_2, home_team_color_3,
			   away_team_name, away_team_color_1, away_team_color_2, away_team_color_3,
			   modified
		FROM squares_settings
		WHERE squares_id = $1
	`, s.ID)

	return row.Scan(
		&s.Settings.HomeTeamName,
		&s.Settings.HomeTeamColor1,
		&s.Settings.HomeTeamColor2,
		&s.Settings.HomeTeamColor3,
		&s.Settings.AwayTeamName,
		&s.Settings.AwayTeamColor1,
		&s.Settings.AwayTeamColor2,
		&s.Settings.AwayTeamColor3,
		&s.Settings.Modified,
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
