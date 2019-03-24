package server

import (
	"net/http"

	"github.com/weters/sqmgr/internal/model"
)

func (s *Server) loginHandler() http.HandlerFunc {
	type data struct {
		FormData struct {
			Email string
		}
	}

	tpl := s.loadTemplate("login.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tplData := data{}

		session := s.Session(r)
		session.Logout()

		if r.Method == http.MethodPost {
			email := r.PostFormValue("email")
			password := r.PostFormValue("password")

			if user, err := s.model.UserByEmailAndPassword(email, password); err != nil {
				if err == model.ErrUserNotFound {
					tplData.FormData.Email = email
				} else {
					session.Save()
					s.Error(w, r, http.StatusInternalServerError, "could not call s.model.UserByEmailAndPassword(%s, xxx): %v", email, err)
					return
				}
			} else {
				session.Login(user)
				session.Save()
				http.Redirect(w, r, "/info", http.StatusSeeOther)
				return
			}
		}

		session.Save()
		s.ExecuteTemplate(w, r, tpl, tplData)
	}
}
