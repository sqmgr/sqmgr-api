/*
Copyright (C) 2020 Tom Peters

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

// GridAnnotationIconMapping is a mapping of an internal int to a font-awesome icon
type GridAnnotationIconMapping map[int16]GridAnnotationIcon

// IsValidIcon will validate that the icon is a valid icon
func (g GridAnnotationIconMapping) IsValidIcon(icon int16) bool {
	_, ok := g[icon]
	return ok
}

// GridAnnotationIcon is a font-awesome icon
type GridAnnotationIcon struct {
	Name string `json:"name"`
}

// AnnotationIcons maps "icon" values to a GridAnnotationIcon object
var AnnotationIcons = GridAnnotationIconMapping{
	0: {
		Name: "trophy",
	},
	1: {
		Name: "dollar-sign",
	},
	2: {
		Name: "money-bill",
	},
	3: {
		Name: "exclamation-circle",
	},
	4: {
		Name: "dice",
	},
	5: {
		Name: "arrow-alt-circle-right",
	},
	6: {
		Name: "football-ball",
	},
	7: {
		Name: "bookmark",
	},
	8: {
		Name: "award",
	},
	9: {
		Name: "bomb",
	},
}
