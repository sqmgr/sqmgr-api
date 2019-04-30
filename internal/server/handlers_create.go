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

const maxNameLen = 50
const minJoinPasswordLen = 5

func (s *Server) createHandler() http.HandlerFunc {
	tpl := s.loadTemplate("create.html", "form-errors.html")

	type data struct {
		MinJoinPasswordLen int
		FormData           map[string]string
		FormErrors         validator.Errors
		GridTypes          []model.GridType
		NameMaxLength      int
	}

	return func(w http.ResponseWriter, r *http.Request) {
		tplData := data{
			MinJoinPasswordLen: minJoinPasswordLen,
			GridTypes:          model.GridTypes(),
			// need to explicitly include Type for "eq" operator in template
			FormData:      map[string]string{"Type": ""},
			NameMaxLength: maxNameLen,
		}

		user := s.AuthUser(r)

		if r.Method == http.MethodPost {
			v := validator.New()

			gridName := r.PostFormValue("grid-name")
			gridType := r.PostFormValue("grid-type")
			password := r.PostFormValue("password")
			confirmPassword := r.PostFormValue("confirm-password")

			tplData.FormData["Name"] = gridName
			tplData.FormData["Type"] = gridType

			v.Printable("Grid Name", gridName)
			v.MaxLength("Grid Name", gridName, maxNameLen)
			v.Password("Join Password", password, confirmPassword, minJoinPasswordLen)
			if err := model.IsValidGridType(gridType); err != nil {
				v.AddError("Grid Configuration", "you must select a valid configuration option")
			}

			if v.OK() {
				grid, err := s.model.NewGrid(user.ID, gridName, model.GridType(gridType), password)
				if err != nil {
					s.Error(w, r, http.StatusInternalServerError, err)
					return
				}

				http.Redirect(w, r, "/squares/"+grid.Token(), http.StatusSeeOther)
				return
			}

			tplData.FormErrors = v.Errors
		}

		s.ExecuteTemplate(w, r, tpl, tplData)
	}
}
