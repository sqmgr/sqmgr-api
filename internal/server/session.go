package server

import (
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/weters/sqmgr/internal/model"
)

type Session struct {
	*sessions.Session
	server *Server
	writer http.ResponseWriter
	req    *http.Request
}

func (s *Server) GetSession(w http.ResponseWriter, r *http.Request) *Session {
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

func (s *Session) Save() {
	if err := s.Session.Save(s.req, s.writer); err != nil {
		log.Printf("error: could not save session: %v", err)
	}
}

func (s *Session) Logout() {
	delete(s.Values, "email")
	delete(s.Values, "password_hash")
}

func (s *Session) Login(u *model.User) {
	s.Values["email"] = u.Email
	s.Values["password_hash"] = u.PasswordHash
}

var ErrNotLoggedIn = errors.New("not logged in")

func (s *Session) LoggedInUser() (*model.User, error) {
	email, _ := s.Values["email"].(string)
	passwordHash, _ := s.Values["password_hash"].(string)
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
