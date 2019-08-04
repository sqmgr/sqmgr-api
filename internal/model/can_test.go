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
		_, err := m.NewPool(context.Background(), user.ID, "Test", GridTypeStd25, "password")
		g.Expect(err).Should(gomega.Succeed())
	}

	g.Expect(user.Can(context.Background(), ActionCreatePool, user)).Should(gomega.Equal(ActionError("You cannot create more than 3 pools per minute")))

	_, err = m.db.Exec("UPDATE pools SET created = NOW() - INTERVAL '1 hour' WHERE user_id = $1", user.ID)
	g.Expect(err).Should(gomega.Succeed())

	stmt, err := m.db.Prepare("UPDATE pools SET created = (NOW() - INTERVAL '1 minute') WHERE id = $1")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(user.Can(context.Background(), ActionCreatePool, user)).Should(gomega.Succeed())

	for i := 0; i < 7; i++ {
		pool, err := m.NewPool(context.Background(), user.ID, "Test", GridTypeStd25, "password")
		g.Expect(err).Should(gomega.Succeed())

		_, err = stmt.Exec(pool.ID())
		g.Expect(err).Should(gomega.Succeed())
	}

	g.Expect(user.Can(context.Background(), ActionCreatePool, user)).Should(gomega.Equal(ActionError("You cannot create more than 10 pools per day")))

	_, err = m.db.Exec("UPDATE pools SET created = NOW() - INTERVAL '1 day' WHERE user_id = $1", user.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(user.Can(context.Background(), ActionCreatePool, user)).Should(gomega.Succeed())
}
