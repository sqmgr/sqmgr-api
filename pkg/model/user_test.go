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
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"

	"github.com/onsi/gomega"
)

func TestJoinGrid(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())

	u, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	u2, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	pool, err := m.NewPool(context.Background(), u.ID, "test", GridTypeStd100, "join-password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	g.Expect(u.JoinPool(context.Background(), pool)).Should(gomega.Succeed())
	count, err := u.PoolsJoinedByUserIDCount(context.Background(), u.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(0))) // verify you can't join a pool you own

	ok, err := u.IsMemberOf(context.Background(), pool)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ok).Should(gomega.BeTrue())

	ok, err = u2.IsMemberOf(context.Background(), pool)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ok).Should(gomega.BeFalse())

	g.Expect(u2.JoinPool(context.Background(), pool)).Should(gomega.Succeed())
	g.Expect(u2.JoinPool(context.Background(), pool)).Should(gomega.Succeed(), "test ON CONFLICT")

	ok, err = u2.IsMemberOf(context.Background(), pool)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ok).Should(gomega.BeTrue())

	isManagerOf, err := u.IsManagerOf(context.Background(), pool)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(isManagerOf).Should(gomega.BeTrue())

	isManagerOf, err = u2.IsManagerOf(context.Background(), pool)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(isManagerOf).Should(gomega.BeFalse())

	u2.SetManagerOf(context.Background(), pool, true)
	isManagerOf, err = u2.IsManagerOf(context.Background(), pool)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(isManagerOf).Should(gomega.BeTrue())

	u2.SetManagerOf(context.Background(), pool, false)
	isManagerOf, err = u2.IsManagerOf(context.Background(), pool)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(isManagerOf).Should(gomega.BeFalse())
}

func TestGetUserByID(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())

	u, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(u.ID).Should(gomega.BeNumerically(">", 0))

	u2, err := m.GetUserByID(context.Background(), u.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(u2.ID).Should(gomega.Equal(u.ID))
}

func randString() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return base64.RawURLEncoding.EncodeToString(b)
}

func ensureIntegration(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}
}

func TestUserIsSiteAdmin(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a user
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(user.IsSiteAdmin).Should(gomega.BeFalse())

	// Set user as admin directly in database
	_, err = m.DB.ExecContext(ctx, "UPDATE users SET is_site_admin = true WHERE id = $1", user.ID)
	g.Expect(err).Should(gomega.Succeed())

	// Reload user via GetUserByID
	reloadedUser, err := m.GetUserByID(ctx, user.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(reloadedUser.IsSiteAdmin).Should(gomega.BeTrue())

	// Clean up - reset admin status
	_, err = m.DB.ExecContext(ctx, "UPDATE users SET is_site_admin = false WHERE id = $1", user.ID)
	g.Expect(err).Should(gomega.Succeed())
}

func TestUserEmail(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a user
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(user.Email).Should(gomega.BeNil())

	// Set email
	testEmail := "test@example.com"
	err = user.SetEmail(ctx, testEmail)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(user.Email).ShouldNot(gomega.BeNil())
	g.Expect(*user.Email).Should(gomega.Equal(testEmail))

	// Reload user and verify email persisted
	reloadedUser, err := m.GetUserByID(ctx, user.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(reloadedUser.Email).ShouldNot(gomega.BeNil())
	g.Expect(*reloadedUser.Email).Should(gomega.Equal(testEmail))

	// Update email
	newEmail := "new@example.com"
	err = reloadedUser.SetEmail(ctx, newEmail)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(*reloadedUser.Email).Should(gomega.Equal(newEmail))
}

func TestUserEmailNullable(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a user without email
	user, err := m.GetUser(ctx, IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(user.Email).Should(gomega.BeNil())

	// Reload and verify email is still nil
	reloadedUser, err := m.GetUserByID(ctx, user.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(reloadedUser.Email).Should(gomega.BeNil())
}
