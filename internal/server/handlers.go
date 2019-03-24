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
	"github.com/gorilla/sessions"
	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/internal/validator"
)

const sessionName = "squares"
const templatesDir = "web/templates"

func (s *Server) simpleGetHandler(page string) http.HandlerFunc {
	tpl := s.loadTemplate(page)
	return func(w http.ResponseWriter, r *http.Request) {
		if err := tpl.Execute(w, nil); err != nil {
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

		if err := tpl.Execute(w, d); err != nil {
			log.Printf("error: could not render index.html: %v", err)
		}
	}
}

func (s *Server) squaresGetHandler() http.HandlerFunc {
	tpl := s.loadTemplate("squares.html")

	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, sessionName)
		if err != nil {
			log.Printf("error: could not decode session: %v", err)
		}

		squares, ok := s.getSquares(w, r)
		if !ok {
			return
		}

		if !s.canViewSquares(session, squares) {
			http.Redirect(w, r, fmt.Sprintf("/squares/%s/login", squares.Token), http.StatusSeeOther)
			return
		}

		session.Values["test"] = true
		session.Save(r, w)
		if err := tpl.Execute(w, squares); err != nil {
			log.Printf("error: could not render squares.html: %v", err)
		}
	}
}

func (s *Server) squaresLoginHandler() http.HandlerFunc {
	tpl := s.loadTemplate("squares-login.html")
	type templateData struct {
		Error   string
		Squares *model.Squares
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var td templateData

		session, err := store.Get(r, sessionName)
		if err != nil {
			log.Printf("error: could not decode session: %v", err)
		}

		squares, ok := s.getSquares(w, r)
		if !ok {
			return
		}
		td.Squares = squares

		if s.canViewSquares(session, squares) {
			http.Redirect(w, r, fmt.Sprintf("/squares/%s", squares.Token), http.StatusSeeOther)
			return
		}

		if r.Method == http.MethodPost {
			password := r.PostFormValue("password")
			if squares.JoinPasswordMatches(password) {
				session.Values[squares.Token] = true
				if err := session.Save(r, w); err != nil {
					log.Printf("error: could not save session: %v", err)
				}

				http.Redirect(w, r, fmt.Sprintf("/squares/%s", squares.Token), http.StatusSeeOther)
				return
			}

			td.Error = "password does not match"
		}

		if err := tpl.Execute(w, td); err != nil {
			log.Printf("error: could not render squares-template.html: %v", err)
		}
	}
}

// getSquares will attempt to load squares based on the token value. If the squares can not be loaded
// for any reason, the correct headers will be set and the calling method can just check for the "ok" state.
// If it's false, no additional processing should be done.
func (s *Server) getSquares(w http.ResponseWriter, r *http.Request) (*model.Squares, bool) {
	vars := mux.Vars(r)
	token := vars["token"]

	squares, err := s.model.GetSquaresByToken(token)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return nil, false
		}

		fmt.Printf("error: could not get squares: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil, false
	}

	return squares, true
}

func (s *Server) canViewSquares(session *sessions.Session, squares *model.Squares) bool {
	if squares.JoinPasswordHash == "" {
		return true
	}

	val, _ := session.Values[squares.Token].(bool)
	return val
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
