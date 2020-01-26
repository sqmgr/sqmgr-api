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
	"errors"
	"github.com/sqmgr/sqmgr-api/pkg/tokengen"
)

// ErrRetryLimitExceeded is an error when too many attempts were made
var ErrRetryLimitExceeded = errors.New("internal/model: maximum number of retries attempted")

const maxRetries = 3
const tokenLen = 8

// NewToken will attempt to generate a random unique token up to X times.
func (m *Model) NewToken() (string, error) {
	stmt, err := m.DB.Prepare("SELECT new_token FROM new_token($1)")
	if err != nil {
		return "", err
	}

	for i := 0; i <= maxRetries; i++ {
		token, err := tokengen.Generate(tokenLen)
		if err != nil {
			return "", err
		}

		row := stmt.QueryRow(token)

		var ok bool
		if err := row.Scan(&ok); err != nil {
			return "", err
		}

		if ok {
			return token, nil
		}
	}

	return "", ErrRetryLimitExceeded
}
