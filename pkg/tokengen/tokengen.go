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
	"encoding/base64"
	"math"
)

// Generate will return a token of length "n".
func Generate(n int) (string, error) {
	byteSize := int(math.Ceil(float64(n) * 0.75))
	b := make([]byte, byteSize)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return string(base64.RawURLEncoding.EncodeToString(b[:])[0:n]), nil
}
