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

// Package model handles various models
package model

import "database/sql"

// Model is an object that can be used to interact with a database
type Model struct {
	DB *sql.DB
}

// New returns a new model. The DB can be any database, but it's most likely a postgres handle.
func New(db *sql.DB) *Model {
	return &Model{db}
}
