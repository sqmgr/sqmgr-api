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

// Package tokengen is a package that can generate cryptographically secure random tokens
package tokengen

import (
	"crypto/rand"
	"math/big"
)

var max = big.NewInt(62) // 26 lower, 26 upper, 10 numbers

const ascii0 = 48
const asciiA = 65
const asciia = 97

// Generate will return a token of length "n".
func Generate(n int) (string, error) {
	token := make([]byte, n)
	for i, _ := range token {
		bigNum, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}

		num := byte(bigNum.Int64())
		if num < 10 {
			token[i] = 48 + num
		} else if num < 36 {
			token[i] = asciiA + num - 10
		} else {
			token[i] = asciia + num - 36
		}
	}

	return string(token), nil
}
