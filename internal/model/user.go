package model

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"time"

	"github.com/synacor/argon2id"
	"github.com/weters/sqmgr/internal/config"
	"github.com/weters/sqmgr/pkg/tokengen"
)

var ErrUserExists = errors.New("model: user already exists")

type State string

const (
	Active   State = "active"
	Pending  State = "pending"
	Disabled State = "disabled"
)

type User struct {
	*Model
	ID           int64
	Email        string
	PasswordHash string
	State        State
	Created      time.Time
	Modified     time.Time
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

func (u *User) Save() error {
	row := u.db.QueryRow("UPDATE users SET email = $1, password_hash = $2, state = $3, modified = (NOW() AT TIME ZONE 'UTC') WHERE id = $4 RETURNING modified", u.Email, u.PasswordHash, u.State, u.ID)

	return row.Scan(&u.Modified)
}

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

func (m *Model) UserByVerifyToken(token string) (*User, error) {
	row := m.db.QueryRow(`
		SELECT users.*
		FROM user_confirmations
		INNER JOIN users ON user_confirmations.user_id = users.id
		WHERE token = $1
	`, token)

	return m.userByRow(row)
}

func (m *Model) UserByEmail(email string) (*User, error) {
	row := m.db.QueryRow("SELECT * FROM users WHERE email = $1", email)
	return m.userByRow(row)
}

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
		log.Printf("error: could not send email to %s: %v", u.Email, err)
		return err
	}

	return nil
}
