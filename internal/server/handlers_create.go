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
	"net/http"

	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/internal/validator"
)

const minJoinPasswordLen = 5

func (s *Server) createHandler() http.HandlerFunc {
	tpl := s.loadTemplate("create.html")

	type data struct {
		MinJoinPasswordLen int
		FormData           map[string]string
		FormErrors         validator.Errors
		SquaresTypes       []model.SquaresType
	}

	return func(w http.ResponseWriter, r *http.Request) {
		tplData := data{
			MinJoinPasswordLen: minJoinPasswordLen,
			SquaresTypes:       model.SquaresTypes(),
		}

		user := s.AuthUser(r)

		if r.Method == http.MethodPost {
			v := validator.New()

			squaresName := r.PostFormValue("squares-name")
			squaresType := r.PostFormValue("squares-type")
			password := r.PostFormValue("password")
			confirmPassword := r.PostFormValue("confirm-password")

			v.Printable("Squares Name", squaresName)
			v.Password("Join Password", password, confirmPassword, minJoinPasswordLen)
			if err := model.IsValidSquaresType(squaresType); err != nil {
				v.AddError("Squares Configuration", "you must select a valid configuration option")
			}

			if v.OK() {
				squares, err := s.model.NewSquares(user.ID, squaresName, model.SquaresType(squaresType), password)
				if err != nil {
					s.Error(w, r, http.StatusInternalServerError, err)
					return
				}

				http.Redirect(w, r, "/squares/"+squares.Token, http.StatusSeeOther)
				return
			}

			tplData.FormErrors = v.Errors
		}

		s.ExecuteTemplate(w, r, tpl, tplData)
	}
}
