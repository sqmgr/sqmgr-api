/*
Copyright (C) 2019 Tom Peters

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
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/onsi/gomega"
)

func TestNewPoolInvite(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)

	now := time.Now()
	ttl := time.Hour * 24

	rows := sqlmock.NewRows([]string{"token", "pool_id", "check_id", "expires_at", "created"}).
		AddRow("abcdef1234", int64(1), 0, now.Add(ttl), now)

	mock.ExpectQuery(`INSERT INTO pool_invites`).
		WithArgs(sqlmock.AnyArg(), int64(1), 0, sqlmock.AnyArg()).
		WillReturnRows(rows)

	invite, err := m.NewPoolInvite(context.Background(), 1, 0, ttl)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(invite).ShouldNot(gomega.BeNil())
	g.Expect(invite.Token).Should(gomega.Equal("abcdef1234"))
	g.Expect(invite.PoolID).Should(gomega.Equal(int64(1)))
	g.Expect(invite.CheckID).Should(gomega.Equal(0))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPoolInviteByToken(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)

	now := time.Now()
	expiresAt := now.Add(time.Hour * 24)

	rows := sqlmock.NewRows([]string{"token", "pool_id", "check_id", "expires_at", "created"}).
		AddRow("abcdef1234", int64(1), 0, expiresAt, now)

	mock.ExpectQuery(`SELECT .+ FROM pool_invites WHERE token = \$1`).
		WithArgs("abcdef1234").
		WillReturnRows(rows)

	invite, err := m.PoolInviteByToken(context.Background(), "abcdef1234")
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(invite).ShouldNot(gomega.BeNil())
	g.Expect(invite.Token).Should(gomega.Equal("abcdef1234"))
	g.Expect(invite.PoolID).Should(gomega.Equal(int64(1)))
	g.Expect(invite.CheckID).Should(gomega.Equal(0))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestPoolInviteByToken_NotFound(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)

	mock.ExpectQuery(`SELECT .+ FROM pool_invites WHERE token = \$1`).
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	invite, err := m.PoolInviteByToken(context.Background(), "nonexistent")
	g.Expect(err).Should(gomega.HaveOccurred())
	g.Expect(invite).Should(gomega.BeNil())

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}
