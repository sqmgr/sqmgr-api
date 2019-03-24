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
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/weters/sqmgr/internal/model"
)

// ErrNotLoggedIn is an error when the user is not logged in
var ErrNotLoggedIn = errors.New("not logged in")

// Session is a wrapper around gorilla sessions that makes it easier to get and save sessions.
type Session struct {
	*sessions.Session
	server *Server
	writer http.ResponseWriter
	req    *http.Request
}

const (
	loginEmail        = "le"
	loginPasswordHash = "lph"
)

func newSession(w http.ResponseWriter, r *http.Request, s *Server) *Session {
	session, err := store.Get(r, sessionName)
	if err != nil {
		log.Printf("error: could not get session: %v", err)
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
		log.Printf("error: could not save session: %v", err)
	}
}

// Logout will log the user out
func (s *Session) Logout() {
	delete(s.Values, loginEmail)
	delete(s.Values, loginPasswordHash)
}

// Login will log the user in
func (s *Session) Login(u *model.User) {
	s.Values[loginEmail] = u.Email
	s.Values[loginPasswordHash] = u.PasswordHash
}

// LoggedInUser will grab the currently logged in user
func (s *Session) LoggedInUser() (*model.User, error) {
	email, _ := s.Values[loginEmail].(string)
	passwordHash, _ := s.Values[loginPasswordHash].(string)
	if len(email) == 0 || len(passwordHash) == 0 {
		return nil, ErrNotLoggedIn
	}

	user, err := s.server.model.UserByEmail(email)
	if err != nil {
		return nil, err
	}

	if user.PasswordHash != passwordHash {
		s.Logout()
		return nil, ErrNotLoggedIn
	}

	return user, nil
}
