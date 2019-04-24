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
	"database/sql"
	"encoding/base64"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/weters/sqmgr/internal/model"
	"github.com/weters/sqmgr/internal/validator"
)

const rowsPerTable = 10

type accountSquaresHandlerType int

const (
	joined accountSquaresHandlerType = iota
	owned
)

type squaresTemplateData struct {
	Pagination *Pagination
	Squares    []*model.Squares
}

type collectionFn func(ctx context.Context, u *model.User, offset int, limit int) ([]*model.Squares, error)
type countFn func(ctx context.Context, u *model.User) (int64, error)

// page is index-1 based.
func getSquaresData(ctx context.Context, user *model.User, link string, collFn collectionFn, cntFn countFn, page int) (*squaresTemplateData, error) {
	offset := (page - 1) * rowsPerTable // remember to adjust for page being index-1 based, not index 0
	squares, err := collFn(ctx, user, offset, rowsPerTable)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	count, err := cntFn(ctx, user)
	if err != nil {
		return nil, err
	}

	total := int64(math.Ceil(float64(count) / float64(rowsPerTable)))

	p := NewPagination(int(total), page)
	p.SetBaseURL(link)

	return &squaresTemplateData{
		Pagination: p,
		Squares:    squares,
	}, nil
}

func (s *Server) accountHandler() http.HandlerFunc {
	type data struct {
		User          *model.User
		OwnedSquares  *squaresTemplateData
		JoinedSquares *squaresTemplateData
	}

	tpl := s.loadTemplate("account.html", "account-squares-table.html", "pagination.html")
	return func(w http.ResponseWriter, r *http.Request) {
		user := s.AuthUser(r)
		ctx := r.Context()

		joinedData, err := getSquaresData(ctx, user, "/account/joined", s.model.SquaresCollectionJoinedByUser, s.model.SquaresCollectionJoinedByUserCount, 1)
		if err != nil && err != sql.ErrNoRows {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		ownedData, err := getSquaresData(ctx, user, "/account/owned", s.model.SquaresCollectionOwnedByUser, s.model.SquaresCollectionOwnedByUserCount, 1)
		if err != nil && err != sql.ErrNoRows {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		s.ExecuteTemplate(w, r, tpl, data{
			User:          user,
			OwnedSquares:  ownedData,
			JoinedSquares: joinedData,
		})
	}
}

func (s *Server) accountSquaresHandler(asType accountSquaresHandlerType) http.HandlerFunc {
	tpl := s.loadTemplate("account-squares-table.html", "pagination.html")

	var clFn collectionFn
	var cnFn countFn

	switch asType {
	case owned:
		clFn = s.model.SquaresCollectionOwnedByUser
		cnFn = s.model.SquaresCollectionOwnedByUserCount
	default:
		clFn = s.model.SquaresCollectionJoinedByUser
		cnFn = s.model.SquaresCollectionJoinedByUserCount
	}

	return func(w http.ResponseWriter, r *http.Request) {
		page := r.FormValue("page")
		pageInt, _ := strconv.Atoi(page)
		if pageInt <= 0 {
			s.Error(w, r, http.StatusBadRequest)
			return
		}

		user := s.AuthUser(r)
		ctx := r.Context()
		data, err := getSquaresData(ctx, user, r.URL.Path, clFn, cnFn, pageInt)
		if err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		s.ExecuteTemplateFragment(w, r, tpl, "squares", data)
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
		user := s.AuthUser(r)

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

			if user.PasswordIsValid(newPassword) {
				v.AddError("New Password", "Your new password cannot be the same as your current password")
			}

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

func (s *Server) accountDeletedHandler() http.HandlerFunc {
	tpl := s.loadTemplate("account-deleted.html")

	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Session(r)
		didDelete := len(session.Flashes("account-deleted")) > 0
		session.Save()

		if !didDelete {
			s.Error(w, r, http.StatusNotFound)
			return
		}

		s.ExecuteTemplate(w, r, tpl, nil)
	}
}

func (s *Server) accountDeleteHandler() http.HandlerFunc {
	tpl := s.loadTemplate("account-delete.html")

	return func(w http.ResponseWriter, r *http.Request) {
		user := s.AuthUser(r)

		if user.RequiresReauthentication() {
			http.Redirect(w, r, "/account/verify?b="+base64.RawURLEncoding.EncodeToString([]byte("/account/delete")), http.StatusSeeOther)
			return
		}

		if r.Method == http.MethodPost {
			email := r.PostFormValue("email")
			if email == user.Email {
				if err := user.Delete(); err != nil {
					s.Error(w, r, http.StatusInternalServerError, err)
					return
				}

				if err := user.Save(); err != nil {
					s.Error(w, r, http.StatusInternalServerError, err)
					return
				}

				session := s.Session(r)
				session.AddFlash(true, "account-deleted")
				session.Save()

				http.Redirect(w, r, "/account/deleted", http.StatusSeeOther)
				return
			}
		}

		s.ExecuteTemplate(w, r, tpl, user)
	}
}

func (s *Server) accountVerifyHandler() http.HandlerFunc {
	tpl := s.loadTemplate("account-verify.html")

	type data struct {
		User          *model.User
		WrongPassword bool
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user := s.AuthUser(r)

		tplData := data{
			User: user,
		}

		if r.Method == http.MethodPost {
			password := r.PostFormValue("password")

			if user.PasswordIsValid(password) {
				bounceTo := "/account"
				if b := r.FormValue("b"); len(b) > 0 {
					bounceToBytes, _ := base64.RawURLEncoding.DecodeString(b)
					if len(bounceToBytes) > 0 && bounceToBytes[0] == '/' && !strings.HasPrefix(string(bounceToBytes), "//") {
						bounceTo = string(bounceToBytes)
					}
				}

				session := s.Session(r)
				session.Login(user)
				session.Save()

				http.Redirect(w, r, bounceTo, http.StatusSeeOther)
				return
			}

			tplData.WrongPassword = true
		}

		s.ExecuteTemplate(w, r, tpl, tplData)
	}
}
