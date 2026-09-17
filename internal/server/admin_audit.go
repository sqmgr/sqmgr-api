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
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

// analyticsStatementTimeout bounds how long any single admin analytics query
// may run.
const analyticsStatementTimeout = 30 * time.Second

// auditWriteTimeout bounds the detached audit log write.
const auditWriteTimeout = 5 * time.Second

// runAnalytics runs fn against a read-only transaction.
func (s *Server) runAnalytics(ctx context.Context, fn func(a *model.Analytics) error) error {
	return model.RunReadOnly(ctx, s.model.DB, analyticsStatementTimeout, fn)
}

// queryAnalytics runs a single analytics query inside a read-only transaction
// and returns its result.
func queryAnalytics[T any](s *Server, ctx context.Context, fn func(a *model.Analytics) (T, error)) (T, error) {
	var out T
	err := s.runAnalytics(ctx, func(a *model.Analytics) error {
		var err error
		out, err = fn(a)
		return err
	})
	return out, err
}

// parsePagination reads the offset and limit query parameters, applying a
// default and maximum to the limit and clamping a negative offset to zero.
func parsePagination(r *http.Request, defaultLimit, maxLimit int) (offset int64, limit int) {
	offset, _ = strconv.ParseInt(r.FormValue("offset"), 10, 64)
	if offset < 0 {
		offset = 0
	}

	limit, _ = strconv.Atoi(r.FormValue("limit"))
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	return offset, limit
}

// recordAudit writes an admin audit entry. Failures are logged rather than
// returned because the action itself has already succeeded.
func (s *Server) recordAudit(ctx context.Context, admin *model.User, rec model.AdminAuditRecord) {
	// The action has already committed, so the audit row must not be lost
	// just because the client disconnected before we got here.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), auditWriteTimeout)
	defer cancel()

	rec.AdminUserID = admin.ID
	if _, err := s.model.RecordAdminAction(ctx, rec); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"action":     rec.Action,
			"targetType": rec.TargetType,
			"targetID":   rec.TargetID,
			"adminID":    admin.ID,
		}).Error("could not record admin audit entry")
	}
}

// getAdminAuditEndpoint lists admin audit entries, newest first
func (s *Server) getAdminAuditEndpoint() http.HandlerFunc {
	type response struct {
		Entries []*model.AdminAuditEntry `json:"entries"`
		Total   int64                    `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var filter model.AdminAuditFilter

		if action := r.FormValue("action"); action != "" {
			if !model.AdminAuditAction(action).IsValid() {
				s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid action %q", action))
				return
			}
			filter.Action = model.AdminAuditAction(action)
		}

		if targetType := r.FormValue("targetType"); targetType != "" {
			if !model.AdminAuditTargetType(targetType).IsValid() {
				s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid target type %q", targetType))
				return
			}
			filter.TargetType = model.AdminAuditTargetType(targetType)
		}

		filter.TargetID = strings.TrimSpace(r.FormValue("targetId"))
		offset, limit := parsePagination(r, defaultAdminPoolsLimit, maxAdminPoolsLimit)

		entries, total, err := s.model.AdminAuditLog(r.Context(), filter, offset, limit)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, response{Entries: entries, Total: total})
	}
}
