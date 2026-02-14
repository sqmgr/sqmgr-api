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

// State represents the state of a record
type State string

const (
	// Active means the record is active
	Active State = "active"

	// Pending is specific to a user. They have signed up, but not yet confirmed
	Pending State = "pending"

	// Disabled is when an admin has disabled a record
	Disabled State = "disabled"

	// Deleted is when a user deleted a record
	Deleted State = "deleted"
)
