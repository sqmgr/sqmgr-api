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
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/onsi/gomega"
)

func TestAdminAuditValidation(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(AdminAuditPoolArchive.IsValid()).Should(gomega.BeTrue())
	g.Expect(AdminAuditSportsSync.IsValid()).Should(gomega.BeTrue())
	g.Expect(AdminAuditAction("pool.delete").IsValid()).Should(gomega.BeFalse())
	g.Expect(AdminAuditAction("").IsValid()).Should(gomega.BeFalse())

	g.Expect(AdminAuditTargetPool.IsValid()).Should(gomega.BeTrue())
	g.Expect(AdminAuditTargetEvent.IsValid()).Should(gomega.BeTrue())
	g.Expect(AdminAuditTargetSportsSync.IsValid()).Should(gomega.BeTrue())
	g.Expect(AdminAuditTargetType("user").IsValid()).Should(gomega.BeFalse())
}

func TestRecordAdminAction(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)
	ctx := context.Background()

	mock.ExpectQuery(`INSERT INTO admin_audit_log`).
		WithArgs(int64(7), "pool.archive", "pool", "abc123", "My Pool", `{"previous":false}`, "user asked").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	id, err := m.RecordAdminAction(ctx, AdminAuditRecord{
		AdminUserID: 7,
		Action:      AdminAuditPoolArchive,
		TargetType:  AdminAuditTargetPool,
		TargetID:    "abc123",
		TargetLabel: "My Pool",
		Details:     map[string]interface{}{"previous": false},
		Reason:      "  user asked ",
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(id).Should(gomega.Equal(int64(42)))

	// Empty label, details, and reason are stored as NULL
	mock.ExpectQuery(`INSERT INTO admin_audit_log`).
		WithArgs(int64(7), "pool.join", "pool", "abc123", nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(43)))

	id, err = m.RecordAdminAction(ctx, AdminAuditRecord{
		AdminUserID: 7,
		Action:      AdminAuditPoolJoin,
		TargetType:  AdminAuditTargetPool,
		TargetID:    "abc123",
		Reason:      "   ",
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(id).Should(gomega.Equal(int64(43)))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestRecordAdminAction_Rejected(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)
	ctx := context.Background()

	_, err = m.RecordAdminAction(ctx, AdminAuditRecord{AdminUserID: 1, Action: "bogus", TargetType: AdminAuditTargetPool, TargetID: "x"})
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("unknown action")))

	_, err = m.RecordAdminAction(ctx, AdminAuditRecord{AdminUserID: 1, Action: AdminAuditPoolJoin, TargetType: "user", TargetID: "x"})
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("unknown target type")))

	// Nothing reached the database
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestAdminAuditFilterConditions(t *testing.T) {
	g := gomega.NewWithT(t)

	var args queryArgs
	g.Expect(AdminAuditFilter{}.conditions(&args)).Should(gomega.BeEmpty())
	g.Expect(args).Should(gomega.BeEmpty())

	args = nil
	conds := AdminAuditFilter{Action: AdminAuditPoolLock, TargetType: AdminAuditTargetPool, TargetID: "tok"}.conditions(&args)
	g.Expect(conds).Should(gomega.Equal([]string{"a.action = $1", "a.target_type = $2", "a.target_id = $3"}))
	g.Expect(args).Should(gomega.Equal(queryArgs{"pool.lock", "pool", "tok"}))
}

func TestAdminAuditLog(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM admin_audit_log a WHERE a.target_type = \$1 AND a.target_id = \$2`).
		WithArgs("pool", "tok").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))

	mock.ExpectQuery(`SELECT a.id, a.admin_user_id, u.email, a.action, .+ FROM admin_audit_log a LEFT JOIN users u ON u.id = a.admin_user_id WHERE a.target_type = \$1 AND a.target_id = \$2 ORDER BY a.id DESC OFFSET \$3 LIMIT \$4`).
		WithArgs("pool", "tok", int64(0), 25).
		WillReturnRows(sqlmock.NewRows([]string{"id", "admin_user_id", "email", "action", "target_type", "target_id", "target_label", "details", "reason", "created"}).
			AddRow(int64(2), int64(7), "admin@example.com", "pool.archive", "pool", "tok", "My Pool", []byte(`{"previous":false}`), "cleanup", now).
			AddRow(int64(1), int64(7), nil, "pool.join", "pool", "tok", nil, nil, nil, now))

	entries, total, err := m.AdminAuditLog(ctx, AdminAuditFilter{TargetType: AdminAuditTargetPool, TargetID: "tok"}, -5, 0)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(total).Should(gomega.Equal(int64(2)))
	g.Expect(entries).Should(gomega.HaveLen(2))

	g.Expect(entries[0].Action).Should(gomega.Equal(AdminAuditPoolArchive))
	g.Expect(*entries[0].AdminEmail).Should(gomega.Equal("admin@example.com"))
	g.Expect(entries[0].Details).Should(gomega.Equal(map[string]interface{}{"previous": false}))
	g.Expect(*entries[0].Reason).Should(gomega.Equal("cleanup"))

	g.Expect(entries[1].Action).Should(gomega.Equal(AdminAuditPoolJoin))
	g.Expect(entries[1].AdminEmail).Should(gomega.BeNil())
	g.Expect(entries[1].Details).Should(gomega.BeNil())
	g.Expect(entries[1].Reason).Should(gomega.BeNil())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestAdminAuditLogIntegration(t *testing.T) {
	ensureIntegration(t)
	g := gomega.NewWithT(t)

	m := New(getDB())
	ctx := context.Background()

	admin, err := m.GetUser(ctx, IssuerAuth0, "audit-admin-"+randString())
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	target := "audit-target-" + randString()
	id, err := m.RecordAdminAction(ctx, AdminAuditRecord{
		AdminUserID: admin.ID,
		Action:      AdminAuditPoolTransferOwnership,
		TargetType:  AdminAuditTargetPool,
		TargetID:    target,
		TargetLabel: "Integration Pool",
		Details:     map[string]interface{}{"previousOwnerId": float64(1), "newOwnerId": float64(2)},
		Reason:      "integration test",
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(id).Should(gomega.BeNumerically(">", 0))

	entries, total, err := m.AdminAuditLog(ctx, AdminAuditFilter{TargetID: target}, 0, 10)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(total).Should(gomega.Equal(int64(1)))
	g.Expect(entries).Should(gomega.HaveLen(1))
	g.Expect(entries[0].ID).Should(gomega.Equal(id))
	g.Expect(entries[0].AdminUserID).Should(gomega.Equal(admin.ID))
	g.Expect(entries[0].Action).Should(gomega.Equal(AdminAuditPoolTransferOwnership))
	g.Expect(*entries[0].TargetLabel).Should(gomega.Equal("Integration Pool"))
	g.Expect(entries[0].Details).Should(gomega.Equal(map[string]interface{}{"previousOwnerId": float64(1), "newOwnerId": float64(2)}))
	g.Expect(*entries[0].Reason).Should(gomega.Equal("integration test"))

	// Filtering by a different action excludes it
	_, total, err = m.AdminAuditLog(ctx, AdminAuditFilter{TargetID: target, Action: AdminAuditPoolJoin}, 0, 10)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(total).Should(gomega.Equal(int64(0)))
}
