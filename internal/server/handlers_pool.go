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
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/weters/sqmgr/internal/model"
	"net/http"
	"time"
)

type tokenJWTClaim struct {
	jwt.StandardClaims
	EffectiveUserID interface{}
	Token           string
	IsAdmin         bool
}

const responseOK = "OK"
const responseFail = "Fail"

// ErrNotMember is an error when the user does not belong to the pool
var ErrNotMember = errors.New("server: user does not belong to pool")

var jwtTTL = time.Minute * 5

type poolContextData struct {
	EffectiveUser model.EffectiveUser
	Pool          *model.Pool
	IsMember      bool
	IsAdmin       bool
}

func (s *Server) poolMemberHandler(mustBeMember, mustBeAdmin bool, nextHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		token := vars["token"]

		pool, err := s.model.PoolByToken(r.Context(), token)
		if err != nil {
			if err == sql.ErrNoRows {
				s.Error(w, r, http.StatusNotFound)
				return
			}

			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		user, err := s.EffectiveUser(r)
		if err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		isMember, err := user.IsMemberOf(r.Context(), pool)
		if err != nil {
			s.Error(w, r, http.StatusInternalServerError, err)
			return
		}

		if mustBeMember && !isMember {
			http.Redirect(w, r, fmt.Sprintf("/pool/%s/join", pool.Token()), http.StatusSeeOther)
			return
		}

		isAdmin := user.IsAdminOf(r.Context(), pool)
		if mustBeAdmin && !isAdmin {
			s.Error(w, r, http.StatusUnauthorized)
			return
		}

		// add value
		r = r.WithContext(context.WithValue(r.Context(), ctxKeyPool, &poolContextData{
			EffectiveUser: user,
			Pool:          pool,
			IsMember:      isMember,
			IsAdmin:       isAdmin,
		}))

		nextHandler.ServeHTTP(w, r)
	}
}

func (s *Server) poolJWT(ctx context.Context, pcd *poolContextData) (string, error) {
	if !pcd.IsMember {
		return "", ErrNotMember
	}

	return s.jwt.Sign(tokenJWTClaim{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(jwtTTL).Unix(),
		},
		EffectiveUserID: pcd.EffectiveUser.UserID(ctx),
		Token:           pcd.Pool.Token(),
		IsAdmin:         pcd.IsAdmin,
	})
}

func (s *Server) poolJWTHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pcd := r.Context().Value(ctxKeyPool).(*poolContextData)

		jwtStr, err := s.poolJWT(r.Context(), pcd)
		if err != nil {
			if err == ErrNotMember {
				s.ServeJSONError(w, http.StatusUnauthorized, "")
				return
			}

			s.ServeJSONError(w, http.StatusInternalServerError, "", err)
			return
		}

		s.ServeJSON(w, http.StatusOK, jsonResponse{
			Status: responseOK,
			Result: jwtStr,
		})
		return
	}
}

func (s *Server) poolJoinHandler() http.HandlerFunc {
	tpl := s.loadTemplate("pool-join.html")

	type data struct {
		Error string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		sqCtxData := r.Context().Value(ctxKeyPool).(*poolContextData)
		pool := sqCtxData.Pool
		user := sqCtxData.EffectiveUser

		if sqCtxData.IsMember {
			http.Redirect(w, r, fmt.Sprintf("/pool/%s", pool.Token()), http.StatusSeeOther)
			return
		}

		tplData := data{}
		if r.Method == http.MethodPost {
			password := r.PostFormValue("password")
			if pool.PasswordIsValid(password) {

				if err := user.JoinPool(r.Context(), pool); err != nil {
					s.Error(w, r, http.StatusInternalServerError, err)
					return
				}

				http.Redirect(w, r, fmt.Sprintf("/pool/%s", pool.Token()), http.StatusSeeOther)
				return
			}

			tplData.Error = "password is not valid"
		}

		s.ExecuteTemplate(w, r, tpl, tplData)
		return
	}
}
