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
	"time"

	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/internal/validator"
)

const sessionName = "squares"
const templatesDir = "web/templates"

func (s *Server) simpleGetHandler(page string) http.HandlerFunc {
	tpl := s.loadTemplate(page)
	return func(w http.ResponseWriter, r *http.Request) {
		s.ExecuteTemplate(w, r, tpl, nil)
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
			v.NotPwnedPassword("Admin Password", adminPassword)
			if len(joinPassword) > 0 {
				v.Password("Join Password", joinPassword, confirmJoinPassword, 4)
			}

			timezoneOffset := r.PostFormValue("timezone-offset")
			squaresUnlock := v.Datetime("Squares Unlock", d.FormData.SquaresUnlockDate+"T"+d.FormData.SquaresUnlockTime, timezoneOffset)
			squaresLock := time.Time{}
			if d.FormData.SquaresLockDate != "" && d.FormData.SquaresLockTime != "" {
				squaresLock = v.Datetime("Squares Lock", d.FormData.SquaresLockDate+"T"+d.FormData.SquaresLockTime, timezoneOffset)
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

		s.ExecuteTemplate(w, r, tpl, d)
	}
}

func (s *Server) loadTemplate(filenames ...string) *template.Template {
	fullFilenames := make([]string, len(filenames))
	for i, filename := range filenames {
		fullFilenames[i] = filepath.Join(templatesDir, filename)
	}

	return template.Must(template.Must(s.baseTemplate.Clone()).ParseFiles(fullFilenames...))
}

func (s *Server) serveInternalError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("error serving %s %s: %v", r.Method, r.URL.String(), err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	return
}
