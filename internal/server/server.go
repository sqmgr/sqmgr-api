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
	"html/template"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

var Version = "0.1"

type Server struct {
	*mux.Router
	Version      string
	baseTemplate *template.Template
}

func New() *Server {
	funcMap := template.FuncMap{
		"Version": version,
	}

	tpl := template.Must(
		template.New("").Funcs(funcMap).ParseFiles(filepath.Join(templatesDir, baseTemplateName)),
	)

	s := &Server{
		Router:       mux.NewRouter(),
		Version:      Version,
		baseTemplate: tpl,
	}

	s.setupRoutes()

	return s
}

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
