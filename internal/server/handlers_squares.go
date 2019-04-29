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
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/internal/validator"
)

type squaresContextData struct {
	EffectiveUser model.EffectiveUser
	Squares       *model.Squares
	IsMember      bool
	IsAdmin       bool
}

func (s *Server) squaresMemberHandler(mustBeMember, mustBeAdmin bool, nextHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		token := vars["token"]

		squares, err := s.model.SquaresByToken(r.Context(), token)
		if err != nil {
			if err == sql.ErrNoRows {
				s.Error(w, r, http.StatusNotFound)
				return
			}

			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		user, err := s.EffectiveUser(r)
		if err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		isMember, err := user.IsMemberOf(r.Context(), squares)
		if err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		if mustBeMember && !isMember {
			http.Redirect(w, r, fmt.Sprintf("/squares/%s/join", squares.Token), http.StatusSeeOther)
			return
		}

		isAdmin := user.IsAdminOf(r.Context(), squares)
		if mustBeAdmin && !isAdmin {
			s.Error(w, r, http.StatusUnauthorized)
			return
		}

		// add value
		r = r.WithContext(context.WithValue(r.Context(), ctxKeySquares, &squaresContextData{
			EffectiveUser: user,
			Squares:       squares,
			IsMember:      isMember,
			IsAdmin:       isAdmin,
		}))

		nextHandler.ServeHTTP(w, r)
	}
}

func (s *Server) squaresHandler() http.HandlerFunc {
	tpl := s.loadTemplate("squares.html")

	type data struct {
		IsAdmin bool
		Squares *model.Squares
	}

	return func(w http.ResponseWriter, r *http.Request) {
		sqCtxData := r.Context().Value(ctxKeySquares).(*squaresContextData)

		s.ExecuteTemplate(w, r, tpl, data{
			IsAdmin: sqCtxData.IsAdmin,
			Squares: sqCtxData.Squares,
		})
	}
}

func (s *Server) squaresCustomizeHandler() http.HandlerFunc {
	tpl := s.loadTemplate("squares-customize.html", "form-errors.html")

	const didUpdate = "didUpdate"

	type data struct {
		FormValues     map[string]string
		FormErrors     validator.Errors
		Squares        *model.Squares
		DidUpdate      bool
		NotesMaxLength int
		NameMaxLength  int
	}

	str := func(s *string) string {
		if s == nil {
			return ""
		}

		return *s
	}

	return func(w http.ResponseWriter, r *http.Request) {
		sqCtxData := r.Context().Value(ctxKeySquares).(*squaresContextData)
		squares := sqCtxData.Squares

		if err := squares.LoadSettings(); err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		formValues := make(map[string]string)
		tplData := data{
			Squares:        squares,
			FormValues:     formValues,
			NotesMaxLength: model.NotesMaxLength,
			NameMaxLength:  maxNameLen,
		}

		v := validator.New()

		if r.Method == http.MethodPost {
			formValues["Name"] = r.PostFormValue("name")
			formValues["HomeTeamName"] = r.PostFormValue("home-team-name")
			formValues["HomeTeamColor1"] = r.PostFormValue("home-team-color-1")
			formValues["HomeTeamColor2"] = r.PostFormValue("home-team-color-2")
			formValues["HomeTeamColor3"] = r.PostFormValue("home-team-color-3")
			formValues["AwayTeamName"] = r.PostFormValue("away-team-name")
			formValues["AwayTeamColor1"] = r.PostFormValue("away-team-color-1")
			formValues["AwayTeamColor2"] = r.PostFormValue("away-team-color-2")
			formValues["AwayTeamColor3"] = r.PostFormValue("away-team-color-3")
			formValues["Notes"] = r.PostFormValue("notes")

			name := v.Printable("name", r.PostFormValue("name"))
			name = v.MaxLength("name", name, maxNameLen)
			homeTeamName := v.Printable("home-team-name", r.PostFormValue("home-team-name"), true)
			homeTeamName = v.MaxLength("home-team-name", homeTeamName, maxNameLen)
			homeTeamColor1 := v.Color("home-team-color-1", r.PostFormValue("home-team-color-1"), true)
			homeTeamColor2 := v.Color("home-team-color-2", r.PostFormValue("home-team-color-2"), true)
			homeTeamColor3 := v.Color("home-team-color-3", r.PostFormValue("home-team-color-3"), true)
			awayTeamName := v.Printable("away-team-name", r.PostFormValue("away-team-name"), true)
			awayTeamName = v.MaxLength("away-team-name", awayTeamName, maxNameLen)
			awayTeamColor1 := v.Color("away-team-color-1", r.PostFormValue("away-team-color-1"), true)
			awayTeamColor2 := v.Color("away-team-color-2", r.PostFormValue("away-team-color-2"), true)
			awayTeamColor3 := v.Color("away-team-color-3", r.PostFormValue("away-team-color-3"), true)
			notes := v.PrintableWithNewline("notes", r.PostFormValue("notes"), true)
			notes = v.MaxLength("notes", notes, model.NotesMaxLength)

			if v.OK() {
				squares.Name = name
				squares.Settings.HomeTeamName = &homeTeamName
				squares.Settings.HomeTeamColor1 = &homeTeamColor1
				squares.Settings.HomeTeamColor2 = &homeTeamColor2
				squares.Settings.HomeTeamColor3 = &homeTeamColor3
				squares.Settings.AwayTeamName = &awayTeamName
				squares.Settings.AwayTeamColor1 = &awayTeamColor1
				squares.Settings.AwayTeamColor2 = &awayTeamColor2
				squares.Settings.AwayTeamColor3 = &awayTeamColor3
				squares.Settings.SetNotes(notes)

				if err := squares.Save(); err != nil {
					s.Error(w, r, http.StatusInternalServerError, err)
					return
				}

				session := s.Session(r)
				session.AddFlash(true, didUpdate)
				session.Save()

				http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
				return
			}

			tplData.FormErrors = v.Errors
		} else {
			formValues["Name"] = squares.Name
			formValues["HomeTeamName"] = str(squares.Settings.HomeTeamName)
			formValues["HomeTeamColor1"] = str(squares.Settings.HomeTeamColor1)
			formValues["HomeTeamColor2"] = str(squares.Settings.HomeTeamColor2)
			formValues["HomeTeamColor3"] = str(squares.Settings.HomeTeamColor3)
			formValues["AwayTeamName"] = str(squares.Settings.AwayTeamName)
			formValues["AwayTeamColor1"] = str(squares.Settings.AwayTeamColor1)
			formValues["AwayTeamColor2"] = str(squares.Settings.AwayTeamColor2)
			formValues["AwayTeamColor3"] = str(squares.Settings.AwayTeamColor3)
			formValues["Notes"] = squares.Settings.Notes()

			session := s.Session(r)
			tplData.DidUpdate = session.Flashes(didUpdate) != nil
			session.Save()
		}

		s.ExecuteTemplate(w, r, tpl, tplData)
		return
	}
}

func (s *Server) squaresJoinHandler() http.HandlerFunc {
	tpl := s.loadTemplate("squares-join.html")

	type data struct {
		Error string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		sqCtxData := r.Context().Value(ctxKeySquares).(*squaresContextData)
		squares := sqCtxData.Squares
		user := sqCtxData.EffectiveUser

		if sqCtxData.IsMember {
			http.Redirect(w, r, fmt.Sprintf("/squares/%s", squares.Token), http.StatusSeeOther)
			return
		}

		tplData := data{}
		if r.Method == http.MethodPost {
			password := r.PostFormValue("password")
			if squares.PasswordIsValid(password) {

				if err := user.JoinSquares(r.Context(), squares); err != nil {
					s.Error(w, r, http.StatusInternalServerError, err)
					return
				}

				http.Redirect(w, r, fmt.Sprintf("/squares/%s", squares.Token), http.StatusSeeOther)
				return
			}

			tplData.Error = "password is not valid"
		}

		s.ExecuteTemplate(w, r, tpl, tplData)
		return
	}
}
