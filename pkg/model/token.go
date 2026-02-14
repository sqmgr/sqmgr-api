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
	defer stmt.Close()

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
