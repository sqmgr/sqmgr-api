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

package server

import (
	"encoding/gob"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr/internal/model"
)

// ErrNotLoggedIn is an error when the user is not logged in
var ErrNotLoggedIn = errors.New("not logged in")

// Session is a wrapper around gorilla sessions that makes it easier to request and save sessions.
type Session struct {
	*sessions.Session
	server *Server
	writer http.ResponseWriter
	req    *http.Request
}

type loginSession struct {
	Version      uint
	Email        string
	PasswordHash string
	Created      time.Time
}

const (
	sessionNameLT = "sqmgr-lt" // long term storage
	sessionNameST = "sqmgr-st" // short term (i.e. browser session) storage
)

const (
	loginKey          = "login"
	loginVersion uint = 1
)

const (
	rememberMeKey = "rememberMe"
	gridIDKey     = "gridIDs"
	userIDKey     = "userID"
)

func init() {
	gob.Register(loginSession{})
	gob.Register(map[int64]bool{})
}

func newSession(w http.ResponseWriter, r *http.Request, s *Server) *Session {
	session, err := store.Get(r, sessionNameLT)
	if err != nil {
		logrus.WithError(err).Errorln("could not request session")
	}

	if rememberMe, _ := session.Values[rememberMeKey].(bool); !rememberMe {
		session.Options.MaxAge = 0
	}

	return &Session{
		Session: session,
		server:  s,
		writer:  w,
		req:     r,
	}
}

// Save will save the session.
func (s *Session) Save() {
	if err := s.Session.Save(s.req, s.writer); err != nil {
		logrus.WithError(err).Errorln("could not save session")
	}
}

// Logout will log the user out
func (s *Session) Logout() {
	delete(s.Values, loginKey)
}

// Login will log the user in
func (s *Session) Login(u *model.User, optionalRememberMe ...bool) {
	s.Values[loginKey] = loginSession{
		Version:      loginVersion,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Created:      time.Now(),
	}

	if len(optionalRememberMe) > 0 {
		if optionalRememberMe[0] {
			s.Values[rememberMeKey] = true
			s.Options.MaxAge = store.Options.MaxAge
		} else {
			delete(s.Values, rememberMeKey)
			s.Options.MaxAge = 0
		}
	}
}

// LoggedInUser will grab the currently logged in user
func (s *Session) LoggedInUser() (*model.User, error) {
	login, ok := s.Values[loginKey].(loginSession)
	if !ok {
		return nil, ErrNotLoggedIn
	}

	if len(login.Email) == 0 || login.Version != loginVersion {
		return nil, ErrNotLoggedIn
	}

	user, err := s.server.model.UserByEmail(login.Email)
	if err != nil {
		return nil, err
	}

	if user.PasswordHash != login.PasswordHash {
		s.Logout()
		return nil, ErrNotLoggedIn
	}

	user.Metadata.LastCredentialCheck = login.Created

	return user, nil
}
