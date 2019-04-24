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
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"

	"github.com/onsi/gomega"
)

func TestJoinSquares(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())

	u, err := m.NewUser(randString()+"@sqmgr.com", "test1234")
	g.Expect(err).Should(gomega.Succeed())

	u2, err := m.NewUser(randString()+"@sqmgr.com", "test1234")
	g.Expect(err).Should(gomega.Succeed())

	s, err := m.NewSquares(u.ID, "test", SquaresTypeStd100, "join-password")
	g.Expect(err).Should(gomega.Succeed())

	ok, err := u.IsMemberOf(context.Background(), s)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ok).Should(gomega.BeTrue())

	ok, err = u2.IsMemberOf(context.Background(), s)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ok).Should(gomega.BeFalse())

	g.Expect(u2.JoinSquares(context.Background(), s)).Should(gomega.Succeed())
	g.Expect(u2.JoinSquares(context.Background(), s)).Should(gomega.Succeed(), "test ON CONFLICT")

	ok, err = u2.IsMemberOf(context.Background(), s)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ok).Should(gomega.BeTrue())

	g.Expect(u.IsAdminOf(context.Background(), s)).Should(gomega.BeTrue())
	g.Expect(u2.IsAdminOf(context.Background(), s)).Should(gomega.BeFalse())
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
