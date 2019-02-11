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

type SquaresType string

const (
	SquaresTypeStd100 SquaresType = "std100"
	SquaresTypeStd25  SquaresType = "std25"
)

var validSquaresTypes = map[SquaresType]bool{
	SquaresTypeStd100: true,
	SquaresTypeStd25:  true,
}

func (s SquaresType) Description() string {
	switch s {
	case SquaresTypeStd100:
		return "Standard, 100 squares"
	case SquaresTypeStd25:
		return "Standard, 25 squares"
	}

	return string(s)
}

func IsValidSquaresType(val string) bool {
	_, ok := validSquaresTypes[SquaresType(val)]
	return ok
}

func SquaresTypes() []SquaresType {
	return []SquaresType{SquaresTypeStd100, SquaresTypeStd25}
}
