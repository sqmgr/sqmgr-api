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
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"net/smtp"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/synacor/argon2id"
	"github.com/weters/sqmgr/internal/config"
	"github.com/weters/sqmgr/pkg/tokengen"
)

// ErrUserExists is an error when the user already exists (when trying to create a new account)
var ErrUserExists = errors.New("model: user already exists")

// ErrUserNotFound is when the user is not found in the database
var ErrUserNotFound = errors.New("model: user not found")

// State represents the state of the user
type State string

const (
	// Active means the user is active
	Active State = "active"

	// Pending is when the account is waiting the user to verify the email
	Pending State = "pending"

	// Disabled is when the user disabled their account
	Disabled State = "disabled"
)

// durationBeforeReauth is a maximum duration allowed before a user is required to resupply their credentials
// before doing some potentially destructive operation
const durationBeforeReauth = time.Minute * 3

// User represents an account
type User struct {
	*Model
	ID           int64
	Email        string
	PasswordHash string
	State        State
	Created      time.Time
	Modified     time.Time
	Metadata     UserMetadata // data not persisted to database
}

// UserMetadata represents data that won't be persisted to the database for a user. May contain information regarding
// logged in states, etc.
type UserMetadata struct {
	LastCredentialCheck time.Time
}

// NewUser will try to save a new user in the database
func (m *Model) NewUser(email, password string) (*User, error) {
	hashedPassword, err := argon2id.DefaultHashPassword(password)
	if err != nil {
		return nil, err
	}

	row := m.db.QueryRow("SELECT * FROM new_user($1, $2)", email, hashedPassword)

	user, err := m.userByRow(row)
	if err != nil {
		return nil, err
	}

	if user.ID == -1 {
		return nil, ErrUserExists
	}

	return user, nil
}

// RequiresReauthentication will return true if the user needs to re-enter their credentials
func (u *User) RequiresReauthentication() bool {
	return time.Now().Sub(u.Metadata.LastCredentialCheck) > durationBeforeReauth
}

// Save will persist any changes to the database
func (u *User) Save() error {
	row := u.db.QueryRow("UPDATE users SET email = $1, password_hash = $2, state = $3, modified = (NOW() AT TIME ZONE 'UTC') WHERE id = $4 RETURNING modified", u.Email, u.PasswordHash, u.State, u.ID)

	return row.Scan(&u.Modified)
}

// UserByVerifyToken will lookup a user by its verification token
func (m *Model) UserByVerifyToken(token string) (*User, error) {
	row := m.db.QueryRow(`
		SELECT users.*
		FROM user_confirmations
		INNER JOIN users ON user_confirmations.user_id = users.id
		WHERE token = $1
	`, token)

	return m.userByRow(row)
}

// UserByEmail will return a user by the email address. optionalAllStates is a varargs that can accept a single bool value.
// If false or not supplied, then only Active users will be returned.
func (m *Model) UserByEmail(email string, optAllowAllStates ...bool) (*User, error) {
	var row *sql.Row

	if len(optAllowAllStates) > 0 && optAllowAllStates[0] {
		// all states
		row = m.db.QueryRow("SELECT * FROM users WHERE email = $1", email)
	} else {
		// only active
		row = m.db.QueryRow("SELECT * FROM users WHERE email = $1 AND state = $2", email, Active)
	}

	return m.userByRow(row)
}

// UserByEmailAndPassword will return a user if the email and password matches. If it doesn't match, ErrUserNotFound is returned.
func (m *Model) UserByEmailAndPassword(email, password string) (*User, error) {
	user, err := m.UserByEmail(email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	if err := argon2id.Compare(user.PasswordHash, password); err != nil {
		if err != argon2id.ErrMismatchedHashAndPassword {
			logrus.WithError(err).Errorf("unexpected error from argon2id")
		}

		return nil, ErrUserNotFound
	}

	return user, nil
}

// PasswordIsValid will check to see if the user can login
func (u *User) PasswordIsValid(password string) bool {
	if err := argon2id.Compare(u.PasswordHash, password); err != nil {
		if err != argon2id.ErrMismatchedHashAndPassword {
			logrus.WithError(err).Errorf("could not validate password")
		}

		return false
	}

	return true
}

// SetPassword will set the new password. This will NOT persist the change to the database
func (u *User) SetPassword(password string) error {
	hash, err := argon2id.DefaultHashPassword(password)
	if err != nil {
		return err
	}

	u.PasswordHash = hash
	return nil
}

// SendVerificationEmail will create a new verification token and send it to the user
func (u *User) SendVerificationEmail(tpl *template.Template) error {
	token, err := tokengen.Generate(64)
	if err != nil {
		return err
	}

	w := bytes.NewBuffer(nil)

	if _, err = u.db.Exec("SELECT * FROM set_user_confirmation($1, $2)", u.ID, token); err != nil {
		return nil
	}

	tpl.Execute(w, map[string]string{
		"VerificationLink": config.GetURL("/signup/verify/" + token),
	})

	body := fmt.Sprintf(`To: %s
From: %s
Subject: %s
Content-Type: text/html; charset=utf-8

%s`, u.Email, config.GetFromAddress(), "SqMGR - Account Verification", w.String())

	if err := smtp.SendMail(config.GetSMTP(), nil, config.GetFromAddress(), []string{u.Email}, []byte(body)); err != nil {
		logrus.WithError(err).WithField("email", u.Email).Errorln("could not send email")
		return err
	}

	return nil
}

// Delete will mark the account as "deleted" by setting random values to the email and password. This will
// keep the account ID in tact in order to not destroy any boards previously created.
func (u *User) Delete() error {
	// first 20 bytes is email, last 20 is password
	randomData := make([]byte, 40)
	if _, err := rand.Read(randomData); err != nil {
		return err
	}

	email := hex.EncodeToString(randomData[0:len(randomData)/2]) + "@deleted.sqmgr.com"

	passwordHash, err := argon2id.DefaultHashPassword(string(randomData[len(randomData)/2:]))
	if err != nil {
		return err
	}

	u.Email = email
	u.PasswordHash = passwordHash
	u.State = Disabled

	return nil
}

func (m *Model) userByRow(row *sql.Row) (*User, error) {
	u := &User{Model: m}

	var email, passwordHash, state *string
	var created *time.Time
	var modified *time.Time

	if err := row.Scan(&u.ID, &email, &passwordHash, &state, &created, &modified); err != nil {
		return nil, err
	}

	if email != nil {
		u.Email = *email
	}

	if passwordHash != nil {
		u.PasswordHash = *passwordHash
	}

	if state != nil {
		u.State = State(*state)
	}

	if created != nil {
		u.Created = *created
	}

	if modified != nil {
		u.Modified = *modified
	}

	return u, nil
}
