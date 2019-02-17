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
	"errors"
	"time"

	"github.com/lib/pq"
	"github.com/weters/sqmgr/pkg/tokengen"
	"golang.org/x/crypto/bcrypt"
)

const tokenLen = 6

// Squares represents an individual squares game
type Squares struct {
	db            *sql.DB
	Token         string
	Name          string
	SquaresType   SquaresType
	SquaresUnlock time.Time
	SquaresLock   time.Time
	AdminPassword string
	JoinPassword  string

	// these values will only be set when retrieving from the database
	AdminPasswordHash string
	JoinPasswordHash  string
}

// NewSquares will create a new squares game. This does not persist
// anything to the database. You'll need to set the fields first, and then call Save().
func (m *Model) NewSquares() *Squares {
	return &Squares{db: m.db}
}

// GetSquaresByToken will return the squares based on the token. Will return an error if anything goes wrong.
func (m *Model) GetSquaresByToken(token string) (*Squares, error) {
	row := m.db.QueryRow("SELECT name, square_type, admin_password_hash, join_password_hash, squares_unlock, squares_lock  FROM squares WHERE token = $1", token)
	var r Squares
	var joinPasswordHash sql.NullString
	var squaresLock pq.NullTime
	r.Token = token
	if err := row.Scan(&r.Name, &r.SquaresType, &r.AdminPasswordHash, &joinPasswordHash, &r.SquaresUnlock, &squaresLock); err != nil {
		return nil, err
	}

	if joinPasswordHash.Valid {
		r.JoinPasswordHash = joinPasswordHash.String
	}

	if squaresLock.Valid {
		r.SquaresLock = squaresLock.Time
	}

	return &r, nil
}

// JoinPasswordMatches will return true if the provided joinPassword is valid.
func (s *Squares) JoinPasswordMatches(joinPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(s.JoinPasswordHash), []byte(joinPassword)) == nil
}

// Save will persist the squares to the database. This currently only supports creation. A unique token
// will be generated and updated in the object.
func (s *Squares) Save() error {
	token, err := s.generateUniqueToken()
	if err != nil {
		return err
	}

	var squaresLock *time.Time
	if !s.SquaresLock.IsZero() {
		squaresLock = &s.SquaresLock
	}

	adminPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(s.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var joinPassword *string
	if s.JoinPassword != "" {
		joinPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(s.JoinPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		joinPasswordStr := string(joinPasswordBytes)

		joinPassword = &joinPasswordStr
	}

	_, err = s.db.Exec(`
		INSERT INTO squares (token, name, square_type, admin_password_hash, join_password_hash, squares_unlock, squares_lock)
		VALUES              ($1,    $2,   $3,          $4,                  $5,                 $6,              $7)`,
		token, s.Name, s.SquaresType, string(adminPasswordBytes), joinPassword, s.SquaresUnlock, squaresLock)

	if err != nil {
		return err
	}

	s.Token = token
	return nil
}

func (s *Squares) generateUniqueToken() (string, error) {
	stmt, err := s.db.Prepare("SELECT * FROM new_token($1)")
	if err != nil {
		return "", err
	}

	for i := 0; i < 10; i++ {
		token, err := tokengen.Generate(tokenLen)
		if err != nil {
			return "", err
		}

		row := stmt.QueryRow(token)
		var ok bool
		if err := row.Scan(&ok); err != nil {
			return "", err
		}

		if ok {
			return token, nil
		}
	}

	return "", errors.New("could not generate a unique token")
}
