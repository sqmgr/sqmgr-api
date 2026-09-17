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
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/onsi/gomega"
)

func TestPoolTransferOwnership(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)
	pool := &Pool{model: m, id: 10, userID: 1}
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE pools SET user_id = \$1, modified = \(NOW\(\) AT TIME ZONE 'utc'\) WHERE id = \$2`).
		WithArgs(int64(2), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO pools_users \(pool_id, user_id\) VALUES \(\$1, \$2\) ON CONFLICT DO NOTHING`).
		WithArgs(int64(10), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	g.Expect(pool.TransferOwnership(ctx, 2)).Should(gomega.Succeed())
	g.Expect(pool.UserID()).Should(gomega.Equal(int64(2)))

	// Transferring to the current owner is a no-op
	g.Expect(pool.TransferOwnership(ctx, 2)).Should(gomega.Succeed())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPoolTransferOwnership_RollsBackOnFailure(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)
	pool := &Pool{model: m, id: 10, userID: 1}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE pools SET user_id`).
		WithArgs(int64(2), int64(10)).
		WillReturnError(errors.New("connection lost"))
	mock.ExpectRollback()

	g.Expect(pool.TransferOwnership(context.Background(), 2)).Should(gomega.MatchError(gomega.ContainSubstring("updating pool owner")))
	g.Expect(pool.UserID()).Should(gomega.Equal(int64(1)))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPoolRevokeInvites(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)
	pool := &Pool{model: m, id: 10}

	mock.ExpectExec(`UPDATE pool_invites SET expires_at = \(NOW\(\) AT TIME ZONE 'utc'\) WHERE pool_id = \$1 AND expires_at > \(NOW\(\) AT TIME ZONE 'utc'\)`).
		WithArgs(int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 3))

	revoked, err := pool.RevokeInvites(context.Background())
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(revoked).Should(gomega.Equal(int64(3)))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPoolAdminActionsIntegration(t *testing.T) {
	ensureIntegration(t)
	g := gomega.NewWithT(t)

	m := New(getDB())
	ctx := context.Background()

	owner, err := m.GetUser(ctx, IssuerAuth0, "transfer-owner-"+randString())
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	newOwner, err := m.GetUser(ctx, IssuerAuth0, "transfer-new-owner-"+randString())
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	pool, err := m.NewPool(ctx, owner.ID, "Transfer Test "+randString(), GridTypeStd25, "password123", NumberSetConfigStandard)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	// Two active invites plus one already expired
	_, err = m.NewPoolInvite(ctx, pool.ID(), pool.CheckID(), time.Hour)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	_, err = m.NewPoolInvite(ctx, pool.ID(), pool.CheckID(), time.Hour)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	_, err = m.NewPoolInvite(ctx, pool.ID(), pool.CheckID(), -time.Hour)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())

	revoked, err := pool.RevokeInvites(ctx)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(revoked).Should(gomega.Equal(int64(2)))

	invite, err := pool.ActiveInvite(ctx)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(invite).Should(gomega.BeNil())

	g.Expect(pool.TransferOwnership(ctx, newOwner.ID)).Should(gomega.Succeed())

	reloaded, err := m.PoolByToken(ctx, pool.Token())
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(reloaded.UserID()).Should(gomega.Equal(newOwner.ID))

	isManager, err := newOwner.IsManagerOf(ctx, reloaded)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(isManager).Should(gomega.BeTrue())

	// The previous owner is still a member but no longer a manager
	isMember, err := owner.IsMemberOf(ctx, reloaded)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(isMember).Should(gomega.BeTrue())
	isManager, err = owner.IsManagerOf(ctx, reloaded)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(isManager).Should(gomega.BeFalse())
}
