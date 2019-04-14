package model

import (
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

	ok, err := u.IsMemberOf(s)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ok).Should(gomega.BeTrue())

	ok, err = u2.IsMemberOf(s)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ok).Should(gomega.BeFalse())

	g.Expect(u2.JoinSquares(s)).Should(gomega.Succeed())
	g.Expect(u2.JoinSquares(s)).Should(gomega.Succeed(), "test ON CONFLICT")

	ok, err = u2.IsMemberOf(s)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ok).Should(gomega.BeTrue())
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
