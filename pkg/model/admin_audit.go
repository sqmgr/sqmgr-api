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

package model

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// AdminAuditAction identifies what a site admin did.
type AdminAuditAction string

// Actions recorded in the admin audit log.
const (
	AdminAuditPoolJoin              AdminAuditAction = "pool.join"
	AdminAuditPoolArchive           AdminAuditAction = "pool.archive"
	AdminAuditPoolUnarchive         AdminAuditAction = "pool.unarchive"
	AdminAuditPoolLock              AdminAuditAction = "pool.lock"
	AdminAuditPoolUnlock            AdminAuditAction = "pool.unlock"
	AdminAuditPoolResetPassword     AdminAuditAction = "pool.resetPassword"
	AdminAuditPoolTransferOwnership AdminAuditAction = "pool.transferOwnership"
	AdminAuditPoolRevokeInvites     AdminAuditAction = "pool.revokeInvites"
	AdminAuditPoolAddManager        AdminAuditAction = "pool.addManager"
	AdminAuditPoolRemoveManager     AdminAuditAction = "pool.removeManager"
	AdminAuditEventRefresh          AdminAuditAction = "event.refresh"
	AdminAuditEventOverride         AdminAuditAction = "event.override"
	AdminAuditEventClearOverride    AdminAuditAction = "event.clearOverride"
	AdminAuditSportsSync            AdminAuditAction = "sports.sync"
)

// AdminAuditTargetType identifies what kind of record an action applied to.
type AdminAuditTargetType string

// Target types recorded in the admin audit log.
const (
	AdminAuditTargetPool       AdminAuditTargetType = "pool"
	AdminAuditTargetEvent      AdminAuditTargetType = "event"
	AdminAuditTargetSportsSync AdminAuditTargetType = "sports_sync"
)

// adminAuditActions is the set of known actions, used to validate filters.
var adminAuditActions = map[AdminAuditAction]bool{
	AdminAuditPoolJoin:              true,
	AdminAuditPoolArchive:           true,
	AdminAuditPoolUnarchive:         true,
	AdminAuditPoolLock:              true,
	AdminAuditPoolUnlock:            true,
	AdminAuditPoolResetPassword:     true,
	AdminAuditPoolTransferOwnership: true,
	AdminAuditPoolRevokeInvites:     true,
	AdminAuditPoolAddManager:        true,
	AdminAuditPoolRemoveManager:     true,
	AdminAuditEventRefresh:          true,
	AdminAuditEventOverride:         true,
	AdminAuditEventClearOverride:    true,
	AdminAuditSportsSync:            true,
}

// IsValid reports whether the action is one the audit log knows about.
func (a AdminAuditAction) IsValid() bool {
	return adminAuditActions[a]
}

// IsValid reports whether the target type is one the audit log knows about.
func (t AdminAuditTargetType) IsValid() bool {
	switch t {
	case AdminAuditTargetPool, AdminAuditTargetEvent, AdminAuditTargetSportsSync:
		return true
	}
	return false
}

// AdminAuditEntry is one recorded admin action.
type AdminAuditEntry struct {
	ID          int64                  `json:"id"`
	AdminUserID int64                  `json:"adminUserId"`
	AdminEmail  *string                `json:"adminEmail"`
	Action      AdminAuditAction       `json:"action"`
	TargetType  AdminAuditTargetType   `json:"targetType"`
	TargetID    string                 `json:"targetId"`
	TargetLabel *string                `json:"targetLabel"`
	Details     map[string]interface{} `json:"details"`
	Reason      *string                `json:"reason"`
	Created     time.Time              `json:"created"`
}

// AdminAuditRecord describes an action to record.
type AdminAuditRecord struct {
	AdminUserID int64
	Action      AdminAuditAction
	TargetType  AdminAuditTargetType
	TargetID    string
	TargetLabel string
	Details     map[string]interface{}
	Reason      string
}

// RecordAdminAction appends an entry to the admin audit log and returns its ID.
func (m *Model) RecordAdminAction(ctx context.Context, rec AdminAuditRecord) (int64, error) {
	if !rec.Action.IsValid() {
		return 0, fmt.Errorf("recording admin action: unknown action %q", rec.Action)
	}
	if !rec.TargetType.IsValid() {
		return 0, fmt.Errorf("recording admin action: unknown target type %q", rec.TargetType)
	}

	var details interface{}
	if len(rec.Details) > 0 {
		b, err := json.Marshal(rec.Details)
		if err != nil {
			return 0, fmt.Errorf("encoding admin action details: %w", err)
		}
		details = string(b)
	}

	var label, reason *string
	if rec.TargetLabel != "" {
		label = &rec.TargetLabel
	}
	if r := strings.TrimSpace(rec.Reason); r != "" {
		reason = &r
	}

	const query = `
		INSERT INTO admin_audit_log (admin_user_id, action, target_type, target_id, target_label, details, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id int64
	if err := m.DB.QueryRowContext(ctx, query,
		rec.AdminUserID, string(rec.Action), string(rec.TargetType), rec.TargetID, label, details, reason,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("recording admin action: %w", err)
	}

	return id, nil
}

// AdminAuditFilter restricts which audit entries are listed. Empty fields do
// not restrict the results.
type AdminAuditFilter struct {
	Action     AdminAuditAction
	TargetType AdminAuditTargetType
	TargetID   string
}

// conditions renders the filter as SQL conditions against the audit log
// aliased as "a", binding values into args.
func (f AdminAuditFilter) conditions(args *queryArgs) []string {
	var conds []string
	if f.Action != "" {
		conds = append(conds, "a.action = "+args.add(string(f.Action)))
	}
	if f.TargetType != "" {
		conds = append(conds, "a.target_type = "+args.add(string(f.TargetType)))
	}
	if f.TargetID != "" {
		conds = append(conds, "a.target_id = "+args.add(f.TargetID))
	}
	return conds
}

// AdminAuditLog returns a page of audit entries, newest first, along with the
// total number matching the filter.
func (m *Model) AdminAuditLog(ctx context.Context, f AdminAuditFilter, offset int64, limit int) ([]*AdminAuditEntry, int64, error) {
	var args queryArgs
	where := whereClause(f.conditions(&args))

	var total int64
	if err := m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_audit_log a"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting admin audit entries: %w", err)
	}

	if offset < 0 {
		offset = 0
	}
	limit = clampLimit(limit, DefaultAnalyticsLimit, MaxAnalyticsLimit)

	query := `
		SELECT a.id, a.admin_user_id, u.email, a.action, a.target_type, a.target_id, a.target_label, a.details, a.reason, a.created
		FROM admin_audit_log a
		LEFT JOIN users u ON u.id = a.admin_user_id` + where + `
		ORDER BY a.id DESC
		OFFSET ` + args.add(offset) + ` LIMIT ` + args.add(limit)

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying admin audit entries: %w", err)
	}
	defer rows.Close()

	entries := make([]*AdminAuditEntry, 0)
	for rows.Next() {
		e := &AdminAuditEntry{}
		var details []byte
		if err := rows.Scan(&e.ID, &e.AdminUserID, &e.AdminEmail, &e.Action, &e.TargetType, &e.TargetID, &e.TargetLabel, &details, &e.Reason, &e.Created); err != nil {
			return nil, 0, fmt.Errorf("scanning admin audit row: %w", err)
		}
		if len(details) > 0 {
			if err := json.Unmarshal(details, &e.Details); err != nil {
				return nil, 0, fmt.Errorf("decoding admin audit details: %w", err)
			}
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("reading admin audit rows: %w", err)
	}

	return entries, total, nil
}
