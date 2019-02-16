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
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/internal/validator"
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

	type data struct {
		SquaresTypes []model.SquaresType
		FormErrors   validator.Errors
		FormData     struct {
			Name              string
			SquaresType       string
			SquaresUnlockDate string
			SquaresUnlockTime string
			SquaresLockDate   string
			SquaresLockTime   string
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var d data

		d.SquaresTypes = model.SquaresTypes()

		if r.Method == http.MethodPost {
			v := validator.New()

			d.FormData.Name = r.PostFormValue("name")
			d.FormData.SquaresType = r.PostFormValue("squares-type")
			d.FormData.SquaresUnlockDate = r.PostFormValue("squares-unlock-date")
			d.FormData.SquaresUnlockTime = r.PostFormValue("squares-unlock-time")
			d.FormData.SquaresLockDate = r.PostFormValue("squares-lock-date")
			d.FormData.SquaresLockTime = r.PostFormValue("squares-lock-time")

			adminPassword := r.PostFormValue("admin-password")
			confirmAdminPassword := r.PostFormValue("confirm-admin-password")

			joinPassword := r.PostFormValue("join-password")
			confirmJoinPassword := r.PostFormValue("confirm-join-password")

			v.Printable("Squares Name", d.FormData.Name)
			v.Password("Admin Password", adminPassword, confirmAdminPassword, 8)
			if len(joinPassword) > 0 {
				v.Password("Join Password", joinPassword, confirmJoinPassword, 4)
			}

			timezoneOffset := r.PostFormValue("timezone-offset")
			squaresUnlock := v.Datetime("Squares Unlock", d.FormData.SquaresUnlockDate+"T"+d.FormData.SquaresLockTime, timezoneOffset)
			squaresLock := time.Time{}
			if d.FormData.SquaresLockDate != "" && d.FormData.SquaresLockTime != "" {
				squaresLock = v.Datetime("Squares Lock", d.FormData.SquaresLockDate+"T"+d.FormData.SquaresUnlockTime, timezoneOffset)
			}

			squaresType := v.SquaresType("Type", d.FormData.SquaresType)

			if v.OK() {
				sq := s.model.NewSquares()
				sq.Name = d.FormData.Name
				sq.SquaresType = squaresType
				sq.SquaresUnlock = squaresUnlock
				sq.SquaresLock = squaresLock
				sq.AdminPassword = adminPassword
				sq.JoinPassword = joinPassword

				if err := sq.Save(); err != nil {
					s.serveInternalError(w, r, err)
					return
				}

				// handle
				http.Redirect(w, r, "/squares/"+sq.Token, http.StatusSeeOther)
				return
			}

			d.FormErrors = v.Errors
		}

		if err := tpl.ExecuteTemplate(w, baseTemplateName, d); err != nil {
			log.Printf("error: could not render index.html: %v", err)
		}
	}
}

func (s *Server) squaresGetHandler() http.HandlerFunc {
	tpl := s.loadTemplate("squares.html")
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		token := vars["token"]

		squares, err := s.model.GetSquaresByToken(token)
		if err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, r)
				return
			}

			fmt.Printf("error: could not get squares: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if err := tpl.ExecuteTemplate(w, baseTemplateName, squares); err != nil {
			log.Printf("error: could not render squares.html: %v", err)
		}
	}
}

func (s *Server) loadTemplate(filename string) *template.Template {
	return template.Must(template.Must(s.baseTemplate.Clone()).ParseFiles(filepath.Join(templatesDir, filename)))
}

func (s *Server) serveInternalError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("error serving %s %s: %v", r.Method, r.URL.String(), err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	return
}
