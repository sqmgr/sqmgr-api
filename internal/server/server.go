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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/pkg/tokengen"
)

// Version is the current version of the server application
var Version = "0.1"

type ctxKey int

// TemplateData represents data that will be sent to an HTML template
type TemplateData struct {
	LoggedInUser *model.User
	Local        interface{}
}

const (
	ctxKeySession ctxKey = iota
	ctxKeyUser
	ctxKeyGrid
)

var store *sessions.CookieStore

var funcMap = template.FuncMap{
	"Version": version,
}

func init() {
	sessionAuthKey := os.Getenv("SESSION_AUTH_KEY")
	sessionEncKey := os.Getenv("SESSION_ENC_KEY")

	if sessionAuthKey == "" {
		var err error
		sessionAuthKey, err = tokengen.Generate(64)
		if err != nil {
			panic(err)
		}

		logrus.WithField("SESSION_AUTH_KEY", sessionAuthKey).Warnln("no SESSION_AUTH_KEY specified, using random value")
	}

	if sessionEncKey == "" {
		var err error
		sessionEncKey, err = tokengen.Generate(32)
		if err != nil {
			panic(err)
		}

		logrus.WithField("SESSION_ENC_KEY", sessionEncKey).Warnln("no SESSION_ENC_KEY specified, using random value")
	}

	store = sessions.NewCookieStore([]byte(sessionAuthKey), []byte(sessionEncKey))
}

// Server represents the server application
type Server struct {
	*mux.Router
	Reload        bool
	model         *model.Model
	baseTemplate  *template.Template
	errorTemplate *template.Template
}

// New instantiates a new Server object.
func New(db *sql.DB) *Server {

	tpl := template.Must(
		template.New("").Funcs(funcMap).ParseFiles(filepath.Join(templatesDir, "base.html")),
	).Lookup("base.html")

	s := &Server{
		Router:       mux.NewRouter(),
		model:        model.New(db),
		baseTemplate: tpl,
	}

	s.errorTemplate = s.loadTemplate("error.html")

	s.setupRoutes()

	return s
}

// Shutdown will be called when the HTTP server is being shutdown. Any open connections
// should be gracefully closed.
func (s *Server) Shutdown() error {
	return nil
}

// ExecuteTemplateFragment executes a fragment in a template. This differs from ExecuteTemplate in that it doesn't use base.html
func (s *Server) ExecuteTemplateFragment(w http.ResponseWriter, r *http.Request, t *template.Template, name string, tplData interface{}) {
	// if Reload is enabled, will attempt to read the templates from disk again.
	if s.Reload {
		t = reload(t, name)
	}

	if err := t.ExecuteTemplate(w, name, tplData); err != nil {
		logrus.WithError(err).Errorln("could not execute template")
		return
	}
}

// ExecuteTemplate will execute the template. Template values can be passed in as localData and will be accessible in
// the template at {{.Local.*}}
func (s *Server) ExecuteTemplate(w http.ResponseWriter, r *http.Request, t *template.Template, localData interface{}) {
	session := s.Session(r)
	user, err := session.LoggedInUser()
	if err != nil && err != ErrNotLoggedIn {
		logrus.WithError(err).Errorln("could not request user")
	}

	tplData := TemplateData{
		LoggedInUser: user,
		Local:        localData,
	}

	// if Reload is enabled, will attempt to read the templates from disk again.
	if s.Reload {
		t = reload(t, "base.html")
	}

	if err := t.Execute(w, tplData); err != nil {
		logrus.WithError(err).Errorln("could not execute template")
		return
	}
}

func reload(t *template.Template, name string) *template.Template {
	newTemplate := template.New("").Funcs(funcMap)
	whatReloaded := make([]string, 0)
	for _, tpl := range t.Templates() {
		if strings.HasSuffix(tpl.Name(), ".html") {
			whatReloaded = append(whatReloaded, tpl.Name())
			var err error
			newTemplate, err = newTemplate.ParseFiles(filepath.Join(templatesDir, tpl.Name()))
			if err != nil {
				panic(err)
			}
		}
	}

	logrus.Debugf("reloaded templates %s", strings.Join(whatReloaded, ", "))
	return newTemplate.Lookup("base.html")
}

// Error will serve a custom error page. err is a varargs and if supplied, will log the error.
func (s *Server) Error(w http.ResponseWriter, r *http.Request, statusCode int, errInfo ...interface{}) {
	if len(errInfo) > 0 {
		strVal, ok := errInfo[0].(string)
		if ok && len(errInfo) > 1 {
			// similar to fmt.Sprintf(...)
			logrus.Errorf(strVal, errInfo[1:]...)
		} else {
			err, ok := errInfo[0].(error)
			if ok {
				// we have an error object
				stack := debug.Stack()
				logrus.WithError(err).Errorf(string(stack))
			} else {
				// not an error... probably a string
				logrus.Errorln(errInfo[0])
			}
		}
	}

	w.WriteHeader(statusCode)
	s.ExecuteTemplate(w, r, s.errorTemplate, map[string]interface{}{
		"StatusCode": statusCode,
		"Status":     http.StatusText(statusCode),
	})
}

// Session will return the current cookie session for the user. It will grab it from the context. It should've
// been set in the middleware.
func (s *Server) Session(r *http.Request) *Session {
	session, ok := r.Context().Value(ctxKeySession).(*Session)
	if !ok {
		panic("session not stored in context")
	}

	return session
}

func (s *Server) authHandler(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := s.Session(r)
		user, err := session.LoggedInUser()
		if err != nil {
			if err != ErrNotLoggedIn {
				s.Error(w, r, http.StatusInternalServerError, err)
				return
			}

			u, _ := url.Parse("/login")
			query := url.Values{}
			query.Set("bounce-to", bounceToURL(r.URL))
			u.RawQuery = query.Encode()

			http.Redirect(w, r, u.String(), http.StatusSeeOther)
			return
		}

		newR := r.WithContext(context.WithValue(r.Context(), ctxKeyUser, user))
		nextHandler.ServeHTTP(w, newR)
	})
}

// EffectiveUser will return a model.EffectiveUser which is either a logged in *model.User or a *model.SessionUser
func (s *Server) EffectiveUser(r *http.Request) (model.EffectiveUser, error) {
	sess := s.Session(r)

	if user, err := sess.LoggedInUser(); err != nil {
		if err != ErrNotLoggedIn {
			return nil, err
		}
	} else {
		logrus.Trace("effective user is logged in")
		return user, nil
	}

	ids, ok := sess.Values[gridIDKey].(map[int64]bool)
	if !ok {
		ids = make(map[int64]bool)
	}

	userID, _ := sess.Values[userIDKey].(string)
	logrus.WithField("userID", userID).Trace("effective user is session user")
	eu := model.NewSessionUser(userID, ids, model.JoinGrid(func(ctx context.Context, grid *model.Grid) error {
		ids, ok := sess.Values[gridIDKey].(map[int64]bool)
		if !ok {
			ids = make(map[int64]bool)
		}

		ids[grid.ID()] = true

		sess.Values[gridIDKey] = ids
		sess.Values[userIDKey] = userID
		sess.Save()

		return nil
	}))

	userID = eu.UserID(r.Context()).(string)

	return eu, nil
}

// AuthUser will return the currently authenticated user. This MUST only be called the calling handler has been
// wrapped by authHandler
func (s *Server) AuthUser(r *http.Request) *model.User {
	user, ok := r.Context().Value(ctxKeyUser).(*model.User)
	if !ok {
		panic("user not stored in context")
	}

	return user
}

func (s *Server) requestWithSession(w http.ResponseWriter, r *http.Request) *http.Request {
	session := newSession(w, r, s)
	ctx := context.WithValue(r.Context(), ctxKeySession, session)
	return r.WithContext(ctx)
}

func (s *Server) noDirListing(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			s.Error(w, s.requestWithSession(w, r), http.StatusNotFound)
			return
		}

		h.ServeHTTP(w, r)
	}
}

func (s *Server) middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") || strings.HasSuffix(r.URL.Path, ".txt") {
			h.ServeHTTP(w, r)
			return
		}

		h.ServeHTTP(w, s.requestWithSession(w, r))
	})
}

func version() string {
	ver := Version
	if build := os.Getenv("BUILD_NUMBER"); build != "" {
		ver += "-" + build
	}

	return ver
}
