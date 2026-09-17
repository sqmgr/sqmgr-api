/*
Copyright (C) 2026 Tom Peters

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sqmgr/sqmgr-api/internal/validator"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

// Admin pool actions accepted by postAdminPoolActionEndpoint.
const (
	adminPoolActionArchive           = "archive"
	adminPoolActionUnarchive         = "unarchive"
	adminPoolActionLock              = "lock"
	adminPoolActionUnlock            = "unlock"
	adminPoolActionResetPassword     = "resetPassword"
	adminPoolActionTransferOwnership = "transferOwnership"
	adminPoolActionRevokeInvites     = "revokeInvites"
	adminPoolActionAddManager        = "addManager"
	adminPoolActionRemoveManager     = "removeManager"
)

// adminPoolAnalytics runs fn against the analytics layer for the pool named in
// the route, translating an unknown token into a 404.
func (s *Server) adminPoolAnalytics(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, a *model.Analytics, token string) error) bool {
	token := mux.Vars(r)["token"]

	err := s.runAnalytics(r.Context(), func(a *model.Analytics) error {
		return fn(r.Context(), a, token)
	})
	if err != nil {
		if errors.Is(err, model.ErrPoolNotFound) {
			s.writeErrorResponse(w, http.StatusNotFound, nil)
			return false
		}
		s.writeErrorResponse(w, http.StatusInternalServerError, err)
		return false
	}

	return true
}

// getAdminPoolEndpoint returns full details for one pool
func (s *Server) getAdminPoolEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var details *model.PoolDetails
		ok := s.adminPoolAnalytics(w, r, func(ctx context.Context, a *model.Analytics, token string) error {
			var err error
			details, err = a.PoolDetails(ctx, token)
			return err
		})
		if !ok {
			return
		}

		s.writeJSONResponse(w, http.StatusOK, details)
	}
}

// getAdminPoolMembersEndpoint returns everyone with access to a pool
func (s *Server) getAdminPoolMembersEndpoint() http.HandlerFunc {
	type response struct {
		Members []*model.PoolMember `json:"members"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var members []*model.PoolMember
		ok := s.adminPoolAnalytics(w, r, func(ctx context.Context, a *model.Analytics, token string) error {
			var err error
			members, err = a.PoolMembers(ctx, token)
			return err
		})
		if !ok {
			return
		}
		if members == nil {
			members = []*model.PoolMember{}
		}

		s.writeJSONResponse(w, http.StatusOK, response{Members: members})
	}
}

// getAdminPoolActivityEndpoint returns a page of the pool's square change log
func (s *Server) getAdminPoolActivityEndpoint() http.HandlerFunc {
	type response struct {
		Activity []*model.PoolActivityEntry `json:"activity"`
		Total    int64                      `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit := parsePagination(r, defaultAdminPoolsLimit, maxAdminPoolsLimit)

		var activity []*model.PoolActivityEntry
		var total int64
		ok := s.adminPoolAnalytics(w, r, func(ctx context.Context, a *model.Analytics, token string) error {
			var err error
			if activity, err = a.PoolActivity(ctx, token, offset, limit); err != nil {
				return err
			}
			total, err = a.PoolActivityCount(ctx, token)
			return err
		})
		if !ok {
			return
		}

		s.writeJSONResponse(w, http.StatusOK, response{Activity: activity, Total: total})
	}
}

// getAdminPoolSquaresEndpoint returns the pool's squares and who claimed them
func (s *Server) getAdminPoolSquaresEndpoint() http.HandlerFunc {
	type response struct {
		Squares []*model.AnalyticsPoolSquare `json:"squares"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		includeUnclaimed := r.FormValue("includeUnclaimed") == "true"

		var squares []*model.AnalyticsPoolSquare
		ok := s.adminPoolAnalytics(w, r, func(ctx context.Context, a *model.Analytics, token string) error {
			var err error
			squares, err = a.PoolSquares(ctx, token, includeUnclaimed)
			return err
		})
		if !ok {
			return
		}
		if squares == nil {
			squares = []*model.AnalyticsPoolSquare{}
		}

		s.writeJSONResponse(w, http.StatusOK, response{Squares: squares})
	}
}

// adminPoolActionUser loads the user a pool action applies to, writing the
// error response itself when the ID is missing or unknown.
func (s *Server) adminPoolActionUser(w http.ResponseWriter, r *http.Request, userID int64) (*model.User, bool) {
	if userID <= 0 {
		s.writeErrorResponse(w, http.StatusBadRequest, errors.New("userId is required"))
		return nil, false
	}

	user, err := s.model.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeErrorResponse(w, http.StatusNotFound, fmt.Errorf("user %d not found", userID))
			return nil, false
		}
		s.writeErrorResponse(w, http.StatusInternalServerError, err)
		return nil, false
	}

	return user, true
}

// postAdminPoolActionEndpoint performs an administrative change to a pool and
// records it in the audit log
func (s *Server) postAdminPoolActionEndpoint() http.HandlerFunc {
	type payload struct {
		Action   string `json:"action"`
		Reason   string `json:"reason"`
		Password string `json:"password"`
		UserID   int64  `json:"userId"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := userFromContext(r.Context())
		if !ok {
			s.writeErrorResponse(w, http.StatusInternalServerError, nil)
			return
		}

		var req payload
		if ok := s.parseJSONPayload(w, r, &req); !ok {
			return
		}

		pool, err := s.model.PoolByToken(r.Context(), mux.Vars(r)["token"])
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		rec := model.AdminAuditRecord{
			TargetType:  model.AdminAuditTargetPool,
			TargetID:    pool.Token(),
			TargetLabel: pool.Name(),
			Reason:      req.Reason,
			Details:     map[string]interface{}{},
		}

		switch req.Action {
		case adminPoolActionArchive, adminPoolActionUnarchive:
			rec.Action = model.AdminAuditPoolArchive
			if req.Action == adminPoolActionUnarchive {
				rec.Action = model.AdminAuditPoolUnarchive
			}
			rec.Details["previous"] = pool.Archived()
			pool.SetArchived(req.Action == adminPoolActionArchive)
			err = pool.Save(r.Context())

		case adminPoolActionLock, adminPoolActionUnlock:
			rec.Action = model.AdminAuditPoolLock
			rec.Details["previouslyLocked"] = pool.IsLocked()
			if req.Action == adminPoolActionLock {
				pool.SetLocks(time.Now())
			} else {
				rec.Action = model.AdminAuditPoolUnlock
				pool.SetLocks(time.Time{})
			}
			err = pool.Save(r.Context())

		case adminPoolActionResetPassword:
			rec.Action = model.AdminAuditPoolResetPassword
			v := validator.New()
			password := v.Password("Password", req.Password, minJoinPasswordLength)
			if !v.OK() {
				s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
					Status:           statusError,
					Error:            validationErrorMessage,
					ValidationErrors: v.Errors,
				})
				return
			}
			if err := pool.SetPassword(password); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
			pool.IncrementCheckID()
			err = pool.Save(r.Context())

		case adminPoolActionTransferOwnership:
			rec.Action = model.AdminAuditPoolTransferOwnership
			newOwner, ok := s.adminPoolActionUser(w, r, req.UserID)
			if !ok {
				return
			}
			if newOwner.ID == pool.UserID() {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("that user already owns the pool"))
				return
			}
			rec.Details["previousOwnerId"] = pool.UserID()
			rec.Details["newOwnerId"] = newOwner.ID
			err = pool.TransferOwnership(r.Context(), newOwner.ID)

		case adminPoolActionAddManager, adminPoolActionRemoveManager:
			rec.Action = model.AdminAuditPoolAddManager
			if req.Action == adminPoolActionRemoveManager {
				rec.Action = model.AdminAuditPoolRemoveManager
			}
			manager, ok := s.adminPoolActionUser(w, r, req.UserID)
			if !ok {
				return
			}
			if manager.ID == pool.UserID() {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("the pool owner is always a manager"))
				return
			}
			// Guest accounts are transient, so they cannot be given a lasting role
			if req.Action == adminPoolActionAddManager && manager.Store != model.UserStoreAuth0 {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("only registered users can be pool managers"))
				return
			}
			rec.Details["userId"] = manager.ID
			rec.Details["userEmail"] = manager.Email

			var changed bool
			if req.Action == adminPoolActionAddManager {
				changed, err = manager.AddManagerOf(r.Context(), pool)
			} else {
				changed, err = manager.RemoveManagerOf(r.Context(), pool)
			}
			if err == nil && !changed {
				msg := "that user is already a manager of the pool"
				if req.Action == adminPoolActionRemoveManager {
					msg = "that user is not a manager of the pool"
				}
				s.writeErrorResponse(w, http.StatusConflict, errors.New(msg))
				return
			}

		case adminPoolActionRevokeInvites:
			rec.Action = model.AdminAuditPoolRevokeInvites
			var revoked int64
			revoked, err = pool.RevokeInvites(r.Context())
			rec.Details["revoked"] = revoked

		default:
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid action %q", req.Action))
			return
		}

		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		// Open grids re-fetch the pool so locks, archiving, and ownership
		// changes show up without a reload, as they do for manager edits.
		s.broker.Publish(pool.Token(), PoolEvent{Type: EventPoolUpdated})

		s.recordAudit(r.Context(), admin, rec)
		w.WriteHeader(http.StatusNoContent)
	}
}
