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
)

func (s *Server) loginHandler() http.HandlerFunc {
	type data struct {
		FormData struct {
			Email      string
			RememberMe string
		}
	}

	tpl := s.loadTemplate("login.html")
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Session(r)
		session.Logout()

		tplData := data{}
		if rememberMe, _ := session.Values[rememberMeKey].(bool); rememberMe {
			tplData.FormData.RememberMe = "yes"
		}

		if r.Method == http.MethodPost {
			email := r.PostFormValue("email")
			password := r.PostFormValue("password")
			rememberMe := r.PostFormValue("remember-me")

			if user, err := s.model.UserByEmailAndPassword(email, password); err != nil {
				if err == model.ErrUserNotFound {
					tplData.FormData.Email = email
					tplData.FormData.RememberMe = rememberMe
				} else {
					session.Save()
					s.Error(w, r, http.StatusInternalServerError, "could not call s.model.UserByEmailAndPassword(%s, xxx): %v", email, err)
					return
				}
			} else {
				session.Login(user, len(rememberMe) > 0)
				session.Save()
				http.Redirect(w, r, "/account", http.StatusSeeOther)
				return
			}
		}

		session.Save()
		s.ExecuteTemplate(w, r, tpl, tplData)
	}
}
