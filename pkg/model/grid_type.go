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

import "errors"

// GridType represents a board type
type GridType string

// Allowed grid types
const (
	GridTypeStd100  GridType = "std100"
	GridTypeStd50   GridType = "std50"
	GridTypeStd25   GridType = "std25"
	GridTypeRoll100 GridType = "roll100"
)

var validGridTypes = map[GridType]bool{
	GridTypeStd100:  true,
	GridTypeStd50:   true,
	GridTypeStd25:   true,
	GridTypeRoll100: true,
}

// ErrInvalidGridType is an error when a string has been typecast to a grid type that does not exist
var ErrInvalidGridType = errors.New("internal/model: invalid grid type")

// Description returns a human friendly notes of the grid type
func (g GridType) Description() string {
	switch g {
	case GridTypeStd100:
		return "Standard, 100 squares"
	case GridTypeStd50:
		return "Standard, 50 squares"
	case GridTypeStd25:
		return "Standard, 25 squares"
	case GridTypeRoll100:
		return "Rollover, 100 squares"
	}

	return string(g)
}

// Squares will return the number of squares in a grid
func (g GridType) Squares() int {
	switch g {
	case GridTypeStd25:
		return 25
	case GridTypeStd50:
		return 50
	default:
		return 100
	}
}

// IsValidGridType will check to see if the string is a valid grid type. If it's valid, nil is returned.
func IsValidGridType(val string) error {
	if _, ok := validGridTypes[GridType(val)]; !ok {
		return ErrInvalidGridType
	}

	return nil
}

// GridTypes returns a list of allowed grid types
func GridTypes() []GridType {
	return []GridType{GridTypeStd100, GridTypeStd50, GridTypeStd25, GridTypeRoll100}
}
