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

import "errors"

// GridType represents a board type
type GridType string

// Allowed grid types
const (
	GridTypeStd100 GridType = "std100"
	GridTypeStd25  GridType = "std25"
	GridTypeRoll100 GridType = "roll100"
)

var validGridTypes = map[GridType]bool{
	GridTypeStd100: true,
	GridTypeStd25:  true,
	GridTypeRoll100:  true,
}

// ErrInvalidGridType is an error when a string has been typecast to a grid type that does not exist
var ErrInvalidGridType = errors.New("internal/model: invalid grid type")

// Description returns a human friendly notes of the grid type
func (g GridType) Description() string {
	switch g {
	case GridTypeStd100:
		return "Standard, 100 squares"
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
	return []GridType{GridTypeStd100, GridTypeStd25, GridTypeRoll100}
}
