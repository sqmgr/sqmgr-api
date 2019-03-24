package server

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/internal/validator"
)

func (s *Server) signupHandler() http.HandlerFunc {
	tpl := s.loadTemplate("signup.html", "form-errors.html")

	type data struct {
		FormData struct {
			Email string
		}
		FormErrors validator.Errors
	}

	return func(w http.ResponseWriter, r *http.Request) {
		tplData := data{}

		if r.Method == http.MethodPost {
			session, err := store.Get(r, sessionName)
			if err != nil {
				log.Printf("error could not decode session %s: %v", sessionName, err)
			}

			email := r.PostFormValue("email")
			password := r.PostFormValue("password")
			confirmPassword := r.PostFormValue("confirm-password")

			v := validator.New()
			v.Email("email", email)
			v.Password("password", password, confirmPassword, 8)
			v.NotPwnedPassword("password", password)

			if v.OK() {
				if user, err := s.model.NewUser(email, password); err != nil {
					if err != model.ErrUserExists {
						log.Printf("error: could not call s.model.NewUser(%s, xxx): %v", email, err)
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}

					user, err := s.model.UserByEmail(email)
					if err != nil {
						log.Printf("error: could not call s.model.UserByEmail(%s): %v", email, err)
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}

					if user.State == model.Active {
						v.AddError("email", "That email address is already registered")
					} else {
						session.AddFlash(user.Email, "email")
						if err := session.Save(r, w); err != nil {
							log.Printf("error: could not save session: %v", err)
						}

						http.Redirect(w, r, "/signup/complete", http.StatusSeeOther)
						return
					}
				} else {
					session.AddFlash(user.Email, "email")
					if err := session.Save(r, w); err != nil {
						log.Printf("error: could not save session: %v", err)
					}

					http.Redirect(w, r, "/signup/complete", http.StatusSeeOther)
					return
				}
			}

			tplData.FormData.Email = email
			tplData.FormErrors = v.Errors
		}

		if err := tpl.ExecuteTemplate(w, baseTemplateName, tplData); err != nil {
			log.Printf("error: could not render signup.html: %v", err)
			return
		}
	}
}

func (s *Server) signupCompleteHandler() http.HandlerFunc {
	tpl := s.loadTemplate("signup-complete.html")
	emailTpl := template.Must(template.ParseFiles(filepath.Join(templatesDir, "email", "verification.html")))

	type data struct {
		Email string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, sessionName)
		if err != nil {
			log.Printf("error: could not decode session: %s: %v", sessionName, err)
		}

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

		if err := session.Save(r, w); err != nil {
			log.Printf("error: could not save session: %v", err)
		}

		user, err := s.model.UserByEmail(email)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Printf("error: could not call s.model.NewUser(%s, xxx): %v", email, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			log.Printf("warning: email %s found in flash, but could not find account", email)
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
			log.Printf("error: could not send verification email to %s: %v", email, err)
		}

		if err := tpl.ExecuteTemplate(w, baseTemplateName, data{email}); err != nil {
			log.Printf("error: could not render template signup-complete.html: %v", err)
			return
		}
	}
}

func (s *Server) signupVerifyHandler() http.HandlerFunc {
	tpl := s.loadTemplate("signup-verified.html")

	return func(w http.ResponseWriter, r *http.Request) {
		token := mux.Vars(r)["token"]
		user, err := s.model.UserByVerifyToken(token)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Printf("error: could not call s.model.UserByVerifyToken(%s): %v", string(token[0:16]), err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			// user not found
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if user.State != model.Pending {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user.State = model.Active
		if err := user.Save(); err != nil {
			log.Printf("error: could not call user.Save(%d): %v", user.ID, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if err := tpl.ExecuteTemplate(w, baseTemplateName, user); err != nil {
			log.Printf("error: could not render template signup-verified.html: %v", err)
			return
		}
	}
}
