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

// Package server represents the SqMGR server application
package server

import (
	"context"
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/pkg/tokengen"
)

// Version is the current version of the server application
var Version = "0.1"

var store *sessions.CookieStore

func init() {
	sessionAuthKey := os.Getenv("SESSION_AUTH_KEY")
	sessionEncKey := os.Getenv("SESSION_ENC_KEY")

	if sessionAuthKey == "" {
		var err error
		sessionAuthKey, err = tokengen.Generate(64)
		if err != nil {
			panic(err)
		}

		log.Printf("WARNING: no SESSION_AUTH_KEY specified, using random value: %s", sessionAuthKey)
	}

	if sessionEncKey == "" {
		var err error
		sessionEncKey, err = tokengen.Generate(32)
		if err != nil {
			panic(err)
		}

		log.Printf("WARNING: no SESSION_ENC_KEY specified, using random value: %s", sessionEncKey)
	}

	store = sessions.NewCookieStore([]byte(sessionAuthKey), []byte(sessionEncKey))
}

// Server represents the server application
type Server struct {
	*mux.Router
	model        *model.Model
	baseTemplate *template.Template
}

// New instantiates a new Server object.
func New(db *sql.DB) *Server {
	funcMap := template.FuncMap{
		"Version": version,
	}

	tpl := template.Must(
		template.New("").Funcs(funcMap).ParseFiles(filepath.Join(templatesDir, "base.html")),
	).Lookup("base.html")

	s := &Server{
		Router:       mux.NewRouter(),
		model:        model.New(db),
		baseTemplate: tpl,
	}

	s.setupRoutes()

	return s
}

// Shutdown will be called when the HTTP server is being shutdown. Any open connections
// should be gracefully closed.
func (s *Server) Shutdown() error {
	return nil
}

type TemplateData struct {
	LoggedInUser *model.User
	Local        interface{}
}

func (s *Server) ExecuteTemplate(w http.ResponseWriter, r *http.Request, t *template.Template, localData interface{}) {
	session := s.Session(r)
	user, err := session.LoggedInUser()
	if err != nil && err != ErrNotLoggedIn {
		log.Printf("error: could not get user: %v", err)
	}

	tplData := TemplateData{
		LoggedInUser: user,
		Local:        localData,
	}

	if err := t.Execute(w, tplData); err != nil {
		log.Printf("error executing template: %v", err)
		return
	}
}

type ctxKey int

const (
	ctxKeySession ctxKey = iota
	ctxKeySession2
)

func version() string {
	ver := Version
	if build := os.Getenv("BUILD_NUMBER"); build != "" {
		ver += "-" + build
	}

	return ver
}

func (s *Server) middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Index(r.URL.Path, "/static/") == 0 {
			h.ServeHTTP(w, r)
			return
		}

		session := s.getSession(w, r)
		ctx := context.WithValue(r.Context(), ctxKeySession, session)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) Session(r *http.Request) *Session {
	session, ok := r.Context().Value(ctxKeySession).(*Session)
	if !ok {
		panic("session not stored in context")
	}
	return session
}
