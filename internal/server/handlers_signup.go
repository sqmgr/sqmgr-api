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
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/internal/validator"
	"net/http"
)

const minPasswordLen = 8

func (s *Server) signupHandler() http.HandlerFunc {
	tpl := s.loadTemplate("signup.html", "form-errors.html")

	type data struct {
		MinPasswordLen int
		FormData       struct {
			Email string
		}
		FormErrors validator.Errors
	}

	return func(w http.ResponseWriter, r *http.Request) {
		tplData := data{MinPasswordLen: minPasswordLen}

		if r.Method == http.MethodPost {
			session := s.Session(r)

			email := r.PostFormValue("email")
			password := r.PostFormValue("password")
			confirmPassword := r.PostFormValue("confirm-password")

			v := validator.New()
			v.Email("email", email)
			v.Password("password", password, confirmPassword, minPasswordLen)
			v.NotPwnedPassword("password", password)

			if v.OK() {
				if user, err := s.model.NewUser(email, password); err != nil {
					if err != model.ErrUserExists {
						s.Error(w, r, http.StatusInternalServerError, "could not call s.model.NewUser(%s, xxx): %v", email, err)
						return
					}

					user, err := s.model.UserByEmail(email, true)
					if err != nil {
						s.Error(w, r, http.StatusInternalServerError, "could not call s.model.UserByEmail(%s): %v", email, err)
						return
					}

					if user.State == model.Active {
						v.AddError("email", "That email address is already registered")
					} else {
						session.AddFlash(user.Email, "email")
						session.Save()
						http.Redirect(w, r, "/signup/complete", http.StatusSeeOther)
						return
					}
				} else {
					session.AddFlash(user.Email, "email")
					session.Save()
					http.Redirect(w, r, "/signup/complete", http.StatusSeeOther)
					return
				}
			}

			tplData.FormData.Email = email
			tplData.FormErrors = v.Errors
		}

		s.ExecuteTemplate(w, r, tpl, tplData)
	}
}

func (s *Server) signupResendGetHandler() http.HandlerFunc {
	tpl := s.loadTemplate("signup-resend.html")

	return func(w http.ResponseWriter, r *http.Request) {
		sess := s.Session(r)
		msgs := sess.Flashes("email")
		sess.Save()

		if len(msgs) > 0 {
			if email, _ := msgs[0].(string); email != "" {
				s.ExecuteTemplate(w, r, tpl, email)
			}
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
}

func (s *Server) signupResendPostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.PostFormValue("email")
		if len(email) == 0 {
			s.Error(w, r, http.StatusBadRequest)
			return
		}

		user, err := s.model.UserByEmail(email, true)
		if err != nil {
			if err != sql.ErrNoRows {
				s.Error(w, r, http.StatusInternalServerError, err)
				return
			}

			logrus.WithField("email", email).Warn("resend called with non-existent user")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if user.State == model.Active {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		} else if user.State != model.Pending {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if err := user.SendVerificationEmail(emailTpl); err != nil {
			logrus.WithError(err).Errorf("could not send verification email to %s", err)
		}

		sess := s.Session(r)
		sess.AddFlash(email, "email")
		sess.Save()

		http.Redirect(w, r, "/signup/resend", http.StatusSeeOther)
		return
	}
}

func (s *Server) signupCompleteHandler() http.HandlerFunc {
	tpl := s.loadTemplate("signup-complete.html")

	type data struct {
		Email string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Session(r)

		flashes := session.Flashes("email")
		if len(flashes) == 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		email, _ := flashes[0].(string)
		if len(email) == 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		session.Save()

		user, err := s.model.UserByEmail(email, true)
		if err != nil {
			if err != sql.ErrNoRows {
				s.Error(w, r, http.StatusInternalServerError, "could not call s.model.NewUser(%s, xxx): %v", email, err)
				return
			}

			logrus.Warnf("email %s found in flash, but could not find account", email)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if user.State == model.Active {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		} else if user.State != model.Pending {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if err := user.SendVerificationEmail(emailTpl); err != nil {
			logrus.WithError(err).Errorf("could not send verification email to %s", err)
		}

		s.ExecuteTemplate(w, r, tpl, data{email})
	}
}

func (s *Server) signupVerifyHandler() http.HandlerFunc {
	tpl := s.loadTemplate("signup-verified.html")

	type data struct {
		User *model.User
	}

	return func(w http.ResponseWriter, r *http.Request) {
		token := mux.Vars(r)["token"]
		user, err := s.model.UserByVerifyToken(token)
		if err != nil {
			if err != sql.ErrNoRows {
				s.Error(w, r, http.StatusInternalServerError, "could not call s.model.UserByVerifyToken(%s): %v", string(token[0:16]), err)
				return
			}

			// user not found
			s.ExecuteTemplate(w, r, tpl, nil)
			return
		}

		if user.State != model.Pending {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user.State = model.Active
		if err := user.Save(); err != nil {
			s.Error(w, r, http.StatusInternalServerError, "could not call user.Save(%d): %v", user.ID, err)
			return
		}

		s.ExecuteTemplate(w, r, tpl, user)
	}
}
