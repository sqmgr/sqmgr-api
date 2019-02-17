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
	"database/sql"
	"html/template"
	"log"
	"os"
	"path/filepath"

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
		template.New("").Funcs(funcMap).ParseFiles(filepath.Join(templatesDir, baseTemplateName)),
	)

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

func version() string {
	ver := Version
	if build := os.Getenv("BUILD_NUMBER"); build != "" {
		ver += "-" + build
	}

	return ver
}
