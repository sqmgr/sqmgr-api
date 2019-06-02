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
	"github.com/gorilla/mux"
	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/internal/validator"
	"net/http"
	"strconv"
)

func (s *Server) poolHandler() http.HandlerFunc {
	type data struct {
		Pool    *model.Pool
		Grids   []*model.Grid
		IsAdmin bool
		User    model.EffectiveUser
	}

	tpl := s.loadTemplate("pool.html")

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		pcd := ctx.Value(ctxKeyPool).(*poolContextData)

		grids, err := pcd.Pool.Grids(ctx, 0, 100) // TODO
		if err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		s.ExecuteTemplate(w, r, tpl, data{
			Pool:    pcd.Pool,
			Grids:   grids,
			IsAdmin: pcd.IsAdmin,
			User:    pcd.EffectiveUser,
		})
	}
}

func (s *Server) poolGridHandler() http.HandlerFunc {
	tpl := s.loadTemplate("grid.html")

	type data struct {
		IsAdmin          bool
		Pool             *model.Pool
		Grid             *model.Grid
		GridSquareStates []model.PoolSquareState
		OpaqueUserID     string
		JWT              string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		pcd := ctx.Value(ctxKeyPool).(*poolContextData)

		pool := pcd.Pool
		gridID, _ := strconv.ParseInt(mux.Vars(r)["grid"], 10, 64)
		grid, err := pool.GridByID(ctx, gridID)
		if err != nil {
			s.NoRowsOrError(w, r, err)
			return
		}

		if err := grid.LoadSettings(context.Background()); err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		signedJWT, err := s.poolJWT(ctx, pcd)
		if err != nil {
			// don't need to worry about ErrNotMember since we already ensured they are
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		oid, err := pcd.EffectiveUser.OpaqueUserID(ctx)
		if err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		s.ExecuteTemplate(w, r, tpl, data{
			IsAdmin:          pcd.IsAdmin,
			Pool:             pcd.Pool,
			Grid:             grid,
			GridSquareStates: model.PoolSquareStates,
			OpaqueUserID:     oid,
			JWT:              signedJWT,
		})
	}
}

func (s *Server) poolGridCustomizeHandler() http.HandlerFunc {
	tpl := s.loadTemplate("grid-customize.html", "form-errors.html")

	const didUpdate = "didUpdate"

	type data struct {
		FormValues        map[string]string
		FormErrors        validator.Errors
		Grid              *model.Grid
		Pool              *model.Pool
		DidUpdate         bool
		NotesMaxLength    int
		TeamNameMaxLength int
	}

	return func(w http.ResponseWriter, r *http.Request) {
		poolCtxData := r.Context().Value(ctxKeyPool).(*poolContextData)

		pool := poolCtxData.Pool
		gridID, _ := strconv.ParseInt(mux.Vars(r)["grid"], 10, 64)
		grid, err := pool.GridByID(r.Context(), gridID)
		if err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
		}

		if err := grid.LoadSettings(r.Context()); err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		formValues := make(map[string]string)
		tplData := data{
			Grid:              grid,
			Pool:              pool,
			FormValues:        formValues,
			NotesMaxLength:    model.NotesMaxLength,
			TeamNameMaxLength: model.TeamNameMaxLength,
		}

		v := validator.New()

		if r.Method == http.MethodPost {
			formValues["EventDate"] = r.PostFormValue("event-date")
			formValues["HomeTeamName"] = r.PostFormValue("home-team-name")
			formValues["HomeTeamColor1"] = r.PostFormValue("home-team-color-1")
			formValues["HomeTeamColor2"] = r.PostFormValue("home-team-color-2")
			formValues["AwayTeamName"] = r.PostFormValue("away-team-name")
			formValues["AwayTeamColor1"] = r.PostFormValue("away-team-color-1")
			formValues["AwayTeamColor2"] = r.PostFormValue("away-team-color-2")
			formValues["Notes"] = r.PostFormValue("notes")

			eventDate := v.Datetime("Event Date", r.PostFormValue("event-date"), "00:00", "0", true)
			homeTeamName := v.Printable("Home Team Name", r.PostFormValue("home-team-name"), true)
			homeTeamName = v.MaxLength("Home Team Name", homeTeamName, model.TeamNameMaxLength)
			homeTeamColor1 := v.Color("Home Team Colors", r.PostFormValue("home-team-color-1"), true)
			homeTeamColor2 := v.Color("Home Team Colors", r.PostFormValue("home-team-color-2"), true)
			awayTeamName := v.Printable("Away Team Name", r.PostFormValue("away-team-name"), true)
			awayTeamName = v.MaxLength("Away Team Name", awayTeamName, model.TeamNameMaxLength)
			awayTeamColor1 := v.Color("Away Team Colors", r.PostFormValue("away-team-color-1"), true)
			awayTeamColor2 := v.Color("Away Team Colors", r.PostFormValue("away-team-color-2"), true)
			notes := v.PrintableWithNewline("Notes", r.PostFormValue("notes"), true)
			notes = v.MaxLength("Notes", notes, model.NotesMaxLength)

			if v.OK() {
				grid.SetEventDate(eventDate)
				grid.SetHomeTeamName(homeTeamName)
				grid.SetAwayTeamName(awayTeamName)
				settings := grid.Settings()
				settings.SetHomeTeamColor1(homeTeamColor1)
				settings.SetHomeTeamColor2(homeTeamColor2)
				settings.SetAwayTeamColor1(awayTeamColor1)
				settings.SetAwayTeamColor2(awayTeamColor2)
				settings.SetNotes(notes)

				if err := grid.Save(r.Context()); err != nil {
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
			if !grid.EventDate().IsZero() {
				formValues["EventDate"] = grid.EventDate().Format("2006-01-02")
			}

			formValues["HomeTeamName"] = grid.HomeTeamName()
			formValues["AwayTeamName"] = grid.AwayTeamName()
			formValues["HomeTeamColor1"] = settings.HomeTeamColor1()
			formValues["HomeTeamColor2"] = settings.HomeTeamColor2()
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
