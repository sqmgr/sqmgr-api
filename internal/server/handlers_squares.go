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
	"net/http"

	"github.com/gorilla/mux"
	"github.com/weters/sqmgr/internal/model"
)

func (s *Server) squaresHandler() http.HandlerFunc {
	tpl := s.loadTemplate("squares.html")

	type data struct {
		Squares *model.Squares
	}

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		token := vars["token"]

		squares, err := s.model.SquaresByToken(token)
		if err != nil {
			if err == sql.ErrNoRows {
				s.Error(w, r, http.StatusNotFound)
				return
			}

			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		s.ExecuteTemplate(w, r, tpl, data{squares})
	}
}
