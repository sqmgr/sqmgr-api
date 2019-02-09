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
	"log"
	"net/http"
	"path/filepath"
)

const baseTemplateName = "base.html"
const templatesDir = "web/templates"

func (s *Server) simpleGetHandler(page string) http.HandlerFunc {
	tpl := s.loadTemplate(page)
	return func(w http.ResponseWriter, r *http.Request) {
		if err := tpl.ExecuteTemplate(w, baseTemplateName, nil); err != nil {
			log.Printf("error: could not render %s: %v", page, err)
		}
	}
}

func (s *Server) createHandler() http.HandlerFunc {
	tpl := s.loadTemplate("create.html")
	return func(w http.ResponseWriter, r *http.Request) {
		if err := tpl.ExecuteTemplate(w, baseTemplateName, nil); err != nil {
			log.Printf("error: could not render index.html: %v", err)
		}
	}
}

func (s *Server) loadTemplate(filename string) *template.Template {
	return template.Must(template.Must(s.baseTemplate.Clone()).ParseFiles(filepath.Join(templatesDir, filename)))
}
