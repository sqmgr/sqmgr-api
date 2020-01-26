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
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sqmgr/sqmgr-api/pkg/model"
	"net/http"
	"strconv"
	"time"
)

// User has 7 days on this token
const guestJWTExpiresDuration = time.Hour * 24 * 7

// userHandler will ensure the authenticated user has the permission to access the user resource
func (s *Server) userHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID, _ := strconv.ParseInt(vars["id"], 10, 64)

		user := r.Context().Value(ctxUserKey).(*model.User)
		if user.ID != userID {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) getUserSelfEndpoint() http.HandlerFunc {
	type response struct {
		UserID  int64           `json:"id"`
		StoreID string          `json:"store_id"`
		Store   model.UserStore `json:"store"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(ctxUserKey).(*model.User)
		s.writeJSONResponse(w, http.StatusOK, response{
			UserID:  user.ID,
			StoreID: user.StoreID,
			Store:   user.Store,
		})
	}
}

func (s *Server) getUserIDPoolMembershipEndpoint() http.HandlerFunc {
	const defaultPerPage = 10
	const maxPerPage = 50

	type resp struct {
		Pools []*model.PoolJSON `json:"pools"`
		Total int64             `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(ctxUserIDKey).(int64)
		membership := mux.Vars(r)["membership"]

		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit <= 0 {
			limit = defaultPerPage
		}

		if limit > maxPerPage {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("limit cannot exceed %d", maxPerPage))
			return
		}

		includeArchived := r.FormValue("includeArchived") == "true"

		var getPools func(context.Context, int64, bool, int64, int) ([]*model.Pool, error)
		var getPoolsCount func(context.Context, int64, bool) (int64, error)
		if membership == "own" {
			getPools = s.model.PoolsOwnedByUserID
			getPoolsCount = s.model.PoolsOwnedByUserIDCount
		} else {
			getPools = func(ctx context.Context, userID int64, includeArchived bool, offset int64, limit int) ([]*model.Pool, error) {
				return s.model.PoolsJoinedByUserID(ctx, userID, offset, limit)
			}

			getPoolsCount = func(ctx context.Context, userID int64, includeArchived bool) (int64, error) {
				return s.model.PoolsJoinedByUserIDCount(ctx, userID)
			}
		}

		pools, err := getPools(r.Context(), userID, includeArchived, offset, limit)
		if err != nil {
			s.writeJSONResponse(w, http.StatusInternalServerError, err)
			return
		}

		total, err := getPoolsCount(r.Context(), userID, includeArchived)
		if err != nil {
			s.writeJSONResponse(w, http.StatusInternalServerError, err)
			return
		}

		poolsJSON := make([]*model.PoolJSON, len(pools))
		for i, p := range pools {
			poolsJSON[i] = p.JSON()
		}

		respObj := resp{
			Pools: poolsJSON,
			Total: total,
		}

		s.writeJSONResponse(w, http.StatusOK, respObj)
	}
}

func (s *Server) deleteUserIDPoolTokenEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(ctxUserIDKey).(int64)
		user, err := s.model.GetUserByID(r.Context(), userID)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, errors.New("user not found"))
				return
			}
		}

		poolToken := mux.Vars(r)["token"]
		pool, err := s.model.PoolByToken(r.Context(), poolToken)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, errors.New("pool not found"))
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if err := user.LeavePool(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) postUserIDGuestJWT() http.HandlerFunc {
	type payload struct {
		// The SqMGR JWT
		JWT string `json:"jwt"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(ctxUserIDKey).(int64)
		user, err := s.model.GetUserByID(r.Context(), userID)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, errors.New("user not found"))
				return
			}
		}

		var thePayload payload
		if ok := s.parseJSONPayload(w, r, &thePayload); !ok {
			return
		}

		token, err := s.smjwt.Validate(thePayload.JWT)
		if err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("cannot parse guest JWT"))
			return
		}

		claims := token.Claims.(*jwt.StandardClaims)
		if !claims.VerifyIssuer(model.IssuerSqMGR, true) ||
			!claims.VerifyAudience(audienceSqMGR, true) {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("invalid guest JWT"))
			return
		}

		guestUser, err := s.model.GetUser(r.Context(), model.IssuerSqMGR, claims.Subject)
		if err != nil {
			// not checking for ErrNoRows since that shouldn't happen. If it does, treat it like a 500
			s.writeJSONResponse(w, http.StatusInternalServerError, err)
			return
		}

		pools, err := s.model.PoolsJoinedByUserID(r.Context(), guestUser.ID, 0, 25)
		if err != nil {
			s.writeJSONResponse(w, http.StatusInternalServerError, err)
			return
		}

		for _, pool := range pools {
			if err := user.JoinPool(r.Context(), pool); err != nil {
				s.writeJSONResponse(w, http.StatusInternalServerError, err)
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) postUserGuestEndpoint() http.HandlerFunc {
	type response struct {
		JWT       string `json:"jwt"`
		ExpiresAt int64  `json:"expiresAt"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		u, err := uuid.NewRandom()
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		expiresAt := time.Now().Add(guestJWTExpiresDuration)

		uid := fmt.Sprintf("sqmgr|%s", u.String())
		claims := jwt.StandardClaims{
			Audience:  audienceSqMGR,
			ExpiresAt: expiresAt.Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    model.IssuerSqMGR,
			Subject:   uid,
		}

		sign, err := s.smjwt.Sign(claims)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if _, err := s.model.NewGuestUser(r.Context(), model.UserStoreSqMGR, uid, expiresAt, r.RemoteAddr); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusCreated, response{
			JWT:       sign,
			ExpiresAt: expiresAt.Unix(),
		})
	}
}
