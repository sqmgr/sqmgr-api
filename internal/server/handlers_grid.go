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
