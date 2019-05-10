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
	"crypto/rsa"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"os"
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

// Options are options passed to the New() constructor
type Options struct {
	PrivateKeyFile string
	PublicKeyFile  string
}

// New will instantiate a new SMJWT object. This method accepts an optional Options object. If you do not
// pass in this object, you must specify JWT_PRIVATE_KEY and/or JWT_PUBLIC_KEY as an environment variable if you plan to
// use Sign() and Validate() respectively.
func New(opts ...Options) (*SMJWT, error) {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.PublicKeyFile == "" {
		opt.PublicKeyFile = os.Getenv("JWT_PUBLIC_KEY")
	}

	if opt.PrivateKeyFile == "" {
		opt.PrivateKeyFile = os.Getenv("JWT_PRIVATE_KEY")
	}

	j := SMJWT{}

	if opt.PrivateKeyFile != "" {
		file, err := ioutil.ReadFile(opt.PrivateKeyFile)
		if err != nil {
			return nil, err
		}

		key, err := jwt.ParseRSAPrivateKeyFromPEM(file)
		if err != nil {
			return nil, err
		}

		j.privateKey = key
	}

	if opt.PublicKeyFile != "" {
		file, err := ioutil.ReadFile(opt.PublicKeyFile)
		if err != nil {
			return nil, err
		}

		key, err := jwt.ParseRSAPublicKeyFromPEM(file)
		if err != nil {
			return nil, err
		}

		j.publicKey = key
	}

	return &j, nil
}

// Sign will sign the JWT claims.
func (j *SMJWT) Sign(claims jwt.Claims) (string, error) {
	if j.privateKey == nil {
		return "", ErrNoPrivateKeySpecified
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(j.privateKey)
}

// Validate will return the token if there were no errors and everything is fully valid. This method takes
// an optional second argument of a jwt.Claims object that can be used to specify the claims type. If this is left out,
// it will default to jwt.StandardClaims.
func (j *SMJWT) Validate(tokenStr string, customClaims ...jwt.Claims) (*jwt.Token, error) {
	if j.publicKey == nil {
		return nil, ErrNoPublicKeySpecified
	}

	var claimsType jwt.Claims
	if len(customClaims) > 0 {
		claimsType = customClaims[0]
	} else {
		claimsType = &jwt.StandardClaims{}
	}

	token, err := jwt.ParseWithClaims(tokenStr, claimsType, func(token *jwt.Token) (i interface{}, e error) {
		return j.publicKey, nil
	})

	// a-ok!
	if err == nil && token.Valid {
		return token, nil
	}

	if ve, ok := err.(*jwt.ValidationError); ok && ve.Errors&jwt.ValidationErrorExpired > 0 {
		return nil, ErrExpired
	}

	return nil, err
}
