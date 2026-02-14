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

package smjwt

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/onsi/gomega"
)

func TestWithConstructor(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	j := New()
	g.Expect(j.LoadPrivateKey("testdata/private.pem")).Should(gomega.Succeed())
	g.Expect(j.LoadPublicKey("testdata/public.pem")).Should(gomega.Succeed())

	s, err := j.Sign(jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		ID:        "my-id",
	})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(s).Should(gomega.MatchRegexp(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\z`))
	token, err := j.Validate(s)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(token).ShouldNot(gomega.BeNil())
	g.Expect(token.Claims.(*jwt.RegisteredClaims).ID).Should(gomega.Equal("my-id"))

	sExp, err := j.Sign(jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * -1)),
		ID:        "my-id",
	})
	g.Expect(err).Should(gomega.Succeed())
	_, err = j.Validate(sExp)
	g.Expect(err).Should(gomega.Equal(ErrExpired))

	_, err = j.Validate("bad-token")
	g.Expect(err).ShouldNot(gomega.Succeed())

	// load the incorrect public key
	g.Expect(j.LoadPublicKey("testdata/bad-public.pem")).Should(gomega.Succeed())
	_, err = j.Validate(s)
	g.Expect(errors.Is(err, jwt.ErrTokenSignatureInvalid)).Should(gomega.BeTrue())
}

func TestWithMissingKeys(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	_ = os.Setenv("JWT_PRIVATE_KEY", "")
	_ = os.Setenv("JWT_PUBLIC_KEY", "")

	j := New()
	_, err := j.Sign(jwt.RegisteredClaims{})
	g.Expect(err).Should(gomega.Equal(ErrNoPrivateKeySpecified))

	_, err = j.Validate("fake-token")
	g.Expect(err).Should(gomega.Equal(ErrNoPublicKeySpecified))

}

func TestWithCustomClaims(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	j := New()
	g.Expect(j.LoadPrivateKey("testdata/private.pem")).Should(gomega.Succeed())
	g.Expect(j.LoadPublicKey("testdata/public.pem")).Should(gomega.Succeed())

	type customClaims struct {
		jwt.RegisteredClaims
		Name string
	}

	s, err := j.Sign(customClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID: "my-id",
		},
		Name: "my-name",
	})
	g.Expect(err).Should(gomega.Succeed())
	token, err := j.Validate(s, &customClaims{})
	g.Expect(err).Should(gomega.Succeed())

	myToken, ok := token.Claims.(*customClaims)
	g.Expect(ok).Should(gomega.BeTrue())
	g.Expect(myToken.ID).Should(gomega.Equal("my-id"))
	g.Expect(myToken.Name).Should(gomega.Equal("my-name"))
}
