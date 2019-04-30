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

type gridContextData struct {
	EffectiveUser model.EffectiveUser
	Grid          *model.Grid
	IsMember      bool
	IsAdmin       bool
}

func (s *Server) squaresMemberHandler(mustBeMember, mustBeAdmin bool, nextHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		token := vars["token"]

		grid, err := s.model.GridByToken(r.Context(), token)
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

		isMember, err := user.IsMemberOf(r.Context(), grid)
		if err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		if mustBeMember && !isMember {
			http.Redirect(w, r, fmt.Sprintf("/grid/%s/join", grid.Token()), http.StatusSeeOther)
			return
		}

		isAdmin := user.IsAdminOf(r.Context(), grid)
		if mustBeAdmin && !isAdmin {
			s.Error(w, r, http.StatusUnauthorized)
			return
		}

		// add value
		r = r.WithContext(context.WithValue(r.Context(), ctxKeyGrid, &gridContextData{
			EffectiveUser: user,
			Grid:          grid,
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
		Grid *model.Grid
	}

	return func(w http.ResponseWriter, r *http.Request) {
		gridCtxData := r.Context().Value(ctxKeyGrid).(*gridContextData)

		s.ExecuteTemplate(w, r, tpl, data{
			IsAdmin: gridCtxData.IsAdmin,
			Grid: gridCtxData.Grid,
		})
	}
}

func (s *Server) squaresCustomizeHandler() http.HandlerFunc {
	tpl := s.loadTemplate("squares-customize.html", "form-errors.html")

	const didUpdate = "didUpdate"

	type data struct {
		FormValues        map[string]string
		FormErrors        validator.Errors
		Grid              *model.Grid
		DidUpdate         bool
		NotesMaxLength    int
		NameMaxLength     int
		TeamNameMaxLength int
	}

	return func(w http.ResponseWriter, r *http.Request) {
		gridCtxData := r.Context().Value(ctxKeyGrid).(*gridContextData)
		grid := gridCtxData.Grid

		if err := grid.LoadSettings(); err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		formValues := make(map[string]string)
		tplData := data{
			Grid:              grid,
			FormValues:        formValues,
			NotesMaxLength:    model.NotesMaxLength,
			NameMaxLength:     maxNameLen,
			TeamNameMaxLength: model.TeamNameMaxLength,
		}

		v := validator.New()

		if r.Method == http.MethodPost {
			formValues["Name"] = r.PostFormValue("name")
			formValues["HomeTeamName"] = r.PostFormValue("home-team-name")
			formValues["HomeTeamColor1"] = r.PostFormValue("home-team-color-1")
			formValues["HomeTeamColor2"] = r.PostFormValue("home-team-color-2")
			formValues["AwayTeamName"] = r.PostFormValue("away-team-name")
			formValues["AwayTeamColor1"] = r.PostFormValue("away-team-color-1")
			formValues["AwayTeamColor2"] = r.PostFormValue("away-team-color-2")
			formValues["Notes"] = r.PostFormValue("notes")

			name := v.Printable("name", r.PostFormValue("name"))
			name = v.MaxLength("name", name, maxNameLen)
			homeTeamName := v.Printable("home-team-name", r.PostFormValue("home-team-name"), true)
			homeTeamName = v.MaxLength("home-team-name", homeTeamName, model.TeamNameMaxLength)
			homeTeamColor1 := v.Color("home-team-color-1", r.PostFormValue("home-team-color-1"), true)
			homeTeamColor2 := v.Color("home-team-color-2", r.PostFormValue("home-team-color-2"), true)
			awayTeamName := v.Printable("away-team-name", r.PostFormValue("away-team-name"), true)
			awayTeamName = v.MaxLength("away-team-name", awayTeamName, model.TeamNameMaxLength)
			awayTeamColor1 := v.Color("away-team-color-1", r.PostFormValue("away-team-color-1"), true)
			awayTeamColor2 := v.Color("away-team-color-2", r.PostFormValue("away-team-color-2"), true)
			notes := v.PrintableWithNewline("notes", r.PostFormValue("notes"), true)
			notes = v.MaxLength("notes", notes, model.NotesMaxLength)

			if v.OK() {
				grid.SetName(name)
				settings := grid.Settings()
				settings.SetHomeTeamName(homeTeamName)
				settings.SetHomeTeamColor1(homeTeamColor1)
				settings.SetHomeTeamColor2(homeTeamColor2)
				settings.SetAwayTeamName(awayTeamName)
				settings.SetAwayTeamColor1(awayTeamColor1)
				settings.SetAwayTeamColor2(awayTeamColor2)
				settings.SetNotes(notes)

				if err := grid.Save(); err != nil {
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
			settings := grid.Settings()
			formValues["Name"] = grid.Name()
			formValues["HomeTeamName"] = settings.HomeTeamName()
			formValues["HomeTeamColor1"] = settings.HomeTeamColor1()
			formValues["HomeTeamColor2"] = settings.HomeTeamColor2()
			formValues["AwayTeamName"] = settings.AwayTeamName()
			formValues["AwayTeamColor1"] = settings.AwayTeamColor1()
			formValues["AwayTeamColor2"] = settings.AwayTeamColor2()
			formValues["Notes"] = settings.Notes()

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
		sqCtxData := r.Context().Value(ctxKeyGrid).(*gridContextData)
		grid := sqCtxData.Grid
		user := sqCtxData.EffectiveUser

		if sqCtxData.IsMember {
			http.Redirect(w, r, fmt.Sprintf("/grid/%s", grid.Token()), http.StatusSeeOther)
			return
		}

		tplData := data{}
		if r.Method == http.MethodPost {
			password := r.PostFormValue("password")
			if grid.PasswordIsValid(password) {

				if err := user.JoinGrid(r.Context(), grid); err != nil {
					s.Error(w, r, http.StatusInternalServerError, err)
					return
				}

				http.Redirect(w, r, fmt.Sprintf("/grid/%s", grid.Token()), http.StatusSeeOther)
				return
			}

			tplData.Error = "password is not valid"
		}

		s.ExecuteTemplate(w, r, tpl, tplData)
		return
	}
}
