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
	"crypto/rsa"
	"errors"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// ErrNoPrivateKeySpecified is an error when a private key hasn't been specified
var ErrNoPrivateKeySpecified = errors.New("jwt: no private key specified")

// ErrNoPublicKeySpecified is an error when a public key hasn't been specified
var ErrNoPublicKeySpecified = errors.New("jwt: no public key specified")

// ErrExpired is an error when the JWT token has expired
var ErrExpired = errors.New("jwt: token has expired")

// SMJWT is a help library for signing a JWT claim
type SMJWT struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

// PublicKey returns the public key
func (s *SMJWT) PublicKey() *rsa.PublicKey {
	return s.publicKey
}

// New will return a new SMJWT object. Before you can use either Sign() or Validate(), you'll need to call
// LoadPrivateKey() and/or LoadPublicKey() respectively
func New() *SMJWT {
	return &SMJWT{}
}

// LoadPublicKey will load the public key from the specified filename.
func (s *SMJWT) LoadPublicKey(filename string) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM(file)
	if err != nil {
		return err
	}

	s.publicKey = key
	return nil
}

// LoadPrivateKey will load the private key from the specified filename.
func (s *SMJWT) LoadPrivateKey(filename string) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(file)
	if err != nil {
		return err
	}

	s.privateKey = key

	return nil
}

// Sign will sign the JWT claims. You MUST call LoadPrivateKey() before you can use this method.
func (s *SMJWT) Sign(claims jwt.Claims) (string, error) {
	if s.privateKey == nil {
		return "", ErrNoPrivateKeySpecified
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

// Validate will return the token if there were no errors and everything is fully valid. This method takes
// an optional second argument of a jwt.Claims object that can be used to specify the claims type. If this is left out,
// it will default to jwt.RegisteredClaims.
// You MUST call LoadPublicKey before you can use this method.
func (s *SMJWT) Validate(tokenStr string, customClaims ...jwt.Claims) (*jwt.Token, error) {
	if s.publicKey == nil {
		return nil, ErrNoPublicKeySpecified
	}

	var claimsType jwt.Claims
	if len(customClaims) > 0 {
		claimsType = customClaims[0]
	} else {
		claimsType = &jwt.RegisteredClaims{}
	}

	token, err := jwt.ParseWithClaims(tokenStr, claimsType, func(token *jwt.Token) (interface{}, error) {
		return s.publicKey, nil
	})

	// a-ok!
	if err == nil && token.Valid {
		return token, nil
	}

	if errors.Is(err, jwt.ErrTokenExpired) {
		return nil, ErrExpired
	}

	return nil, err
}
