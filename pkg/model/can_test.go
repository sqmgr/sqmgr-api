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
	"github.com/onsi/gomega"
	"testing"
)

func TestCanCreatePool(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())

	user, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	g.Expect(user.Can(context.Background(), ActionCreatePool, user)).Should(gomega.Succeed())
	for i := 0; i < 3; i++ {
		_, err := m.NewPool(context.Background(), user.ID, "Test", GridTypeStd25, "password", NumberSetConfigStandard)
		g.Expect(err).Should(gomega.Succeed())
	}

	g.Expect(user.Can(context.Background(), ActionCreatePool, user)).Should(gomega.Equal(ActionError("You cannot create more than 3 pools per minute")))

	_, err = m.DB.Exec("UPDATE pools SET created = NOW() - INTERVAL '1 hour' WHERE user_id = $1", user.ID)
	g.Expect(err).Should(gomega.Succeed())

	stmt, err := m.DB.Prepare("UPDATE pools SET created = (NOW() - INTERVAL '1 minute') WHERE id = $1")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(user.Can(context.Background(), ActionCreatePool, user)).Should(gomega.Succeed())

	for i := 0; i < 7; i++ {
		pool, err := m.NewPool(context.Background(), user.ID, "Test", GridTypeStd25, "password", NumberSetConfigStandard)
		g.Expect(err).Should(gomega.Succeed())

		_, err = stmt.Exec(pool.ID())
		g.Expect(err).Should(gomega.Succeed())
	}

	g.Expect(user.Can(context.Background(), ActionCreatePool, user)).Should(gomega.Equal(ActionError("You cannot create more than 10 pools per day")))

	_, err = m.DB.Exec("UPDATE pools SET created = NOW() - INTERVAL '1 day' WHERE user_id = $1", user.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(user.Can(context.Background(), ActionCreatePool, user)).Should(gomega.Succeed())
}
