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

var ErrNoPrivateKeySpecified = errors.New("jwt: no private key specified")
var ErrNoPublicKeySpecified = errors.New("jwt: no public key specified")
var ErrExpired = errors.New("jwt: token has expired")

type JWT struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

type JWTOptions struct {
	PrivateKeyFile string
	PublicKeyFile  string
}

func New(opts ...JWTOptions) (*JWT, error) {
	var opt JWTOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.PublicKeyFile == "" {
		opt.PublicKeyFile = os.Getenv("JWT_PUBLIC_KEY")
	}

	if opt.PrivateKeyFile == "" {
		opt.PrivateKeyFile = os.Getenv("JWT_PRIVATE_KEY")
	}

	j := JWT{}

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

func (j *JWT) Sign(claims jwt.Claims) (string, error) {
	if j.privateKey == nil {
		return "", ErrNoPrivateKeySpecified
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(j.privateKey)
}

func (j *JWT) Validate(tokenStr string) (*jwt.Token, error) {
	if j.publicKey == nil {
		return nil, ErrNoPublicKeySpecified
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (i interface{}, e error) {
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
