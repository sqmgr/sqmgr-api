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

package smjwt

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/onsi/gomega"
	"os"
	"testing"
	"time"
)

func TestWithConstructor(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	j := New()
	g.Expect(j.LoadPrivateKey("testdata/private.pem")).Should(gomega.Succeed())
	g.Expect(j.LoadPublicKey("testdata/public.pem")).Should(gomega.Succeed())

	s, err := j.Sign(jwt.StandardClaims{
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		Id:        "my-id",
	})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(s).Should(gomega.MatchRegexp(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\z`))
	token, err := j.Validate(s)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(token).ShouldNot(gomega.BeNil())
	g.Expect(token.Claims.(*jwt.StandardClaims).Id).Should(gomega.Equal("my-id"))

	sExp, err := j.Sign(jwt.StandardClaims{
		ExpiresAt: time.Now().Add(time.Minute * -1).Unix(),
		Id:        "my-id",
	})
	g.Expect(err).Should(gomega.Succeed())
	_, err = j.Validate(sExp)
	g.Expect(err).Should(gomega.Equal(ErrExpired))

	_, err = j.Validate("bad-token")
	g.Expect(err).ShouldNot(gomega.Succeed())

	// load the incorrect public key
	g.Expect(j.LoadPublicKey("testdata/bad-public.pem")).Should(gomega.Succeed())
	_, err = j.Validate(s)
	jwtErr := err.(*jwt.ValidationError)
	g.Expect(jwtErr.Errors & jwt.ValidationErrorSignatureInvalid).Should(gomega.BeNumerically(">", 0))
}

func TestWithMissingKeys(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	_ = os.Setenv("JWT_PRIVATE_KEY", "")
	_ = os.Setenv("JWT_PUBLIC_KEY", "")

	j := New()
	_, err := j.Sign(jwt.StandardClaims{})
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
		jwt.StandardClaims
		Name string
	}

	s, err := j.Sign(customClaims{
		StandardClaims: jwt.StandardClaims{
			Id: "my-id",
		},
		Name: "my-name",
	})
	g.Expect(err).Should(gomega.Succeed())
	token, err := j.Validate(s, &customClaims{})
	g.Expect(err).Should(gomega.Succeed())

	myToken, ok := token.Claims.(*customClaims)
	g.Expect(ok).Should(gomega.BeTrue())
	g.Expect(myToken.Id).Should(gomega.Equal("my-id"))
	g.Expect(myToken.Name).Should(gomega.Equal("my-name"))
}
