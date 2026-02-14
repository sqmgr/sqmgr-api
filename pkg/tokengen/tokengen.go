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

// Package tokengen is a package that can generate cryptographically secure random tokens
package tokengen

import (
	"crypto/rand"
	"math/big"
)

var max = big.NewInt(62) // 26 lower, 26 upper, 10 numbers

const asciiA = 65
const asciia = 97

// Generate will return a token of length "n".
func Generate(n int) (string, error) {
	token := make([]byte, n)
	for i := range token {
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
