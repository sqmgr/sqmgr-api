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
