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

	"github.com/weters/sqmgr/internal/validator"
)

func (s *Server) accountHandler() http.HandlerFunc {
	tpl := s.loadTemplate("account.html")
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.LoggedInUserOrRedirect(w, r)
		if !ok {
			return
		}

		s.ExecuteTemplate(w, r, tpl, user)
	}
}

func (s *Server) accountChangePasswordHandler() http.HandlerFunc {
	tpl := s.loadTemplate("account-change-password.html", "form-errors.html")

	const passwordChanged = "pwc"

	type data struct {
		PasswordChanged bool
		FormErrors      validator.Errors
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.LoggedInUserOrRedirect(w, r)
		if !ok {
			return
		}

		tplData := data{}
		if r.Method == http.MethodPost {
			v := validator.New()
			password := r.PostFormValue("password")
			newPassword := r.PostFormValue("new-password")
			confirmNewPassword := r.PostFormValue("confirm-new-password")

			if !user.PasswordIsValid(password) {
				v.AddError("Current Password", "Current password does not match")
			}

			v.Password("New Password", newPassword, confirmNewPassword, 8)
			v.NotPwnedPassword("New Password", newPassword)

			if v.OK() {
				if err := user.SetPassword(newPassword); err != nil {
					s.Error(w, r, http.StatusInternalServerError, err)
					return
				}

				if err := user.Save(); err != nil {
					s.Error(w, r, http.StatusInternalServerError, err)
					return
				}

				session := s.Session(r)
				session.AddFlash(true, passwordChanged)
				session.Login(user)
				session.Save()
				http.Redirect(w, r, "/account/change-password", http.StatusSeeOther)
				return
			}

			tplData.FormErrors = v.Errors
		} else {
			session := s.Session(r)
			tplData.PasswordChanged = session.Flashes(passwordChanged) != nil
			session.Save()
		}

		s.ExecuteTemplate(w, r, tpl, tplData)
	}
}
