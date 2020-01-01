/*
Copyright 2020 Tom Peters

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

// GridAnnotationIconMapping is a mapping of an internal int to a font-awesome icon
type GridAnnotationIconMapping map[int16]GridAnnotationIcon

// IsValidIcon will validate that the icon is a valid icon
func (g GridAnnotationIconMapping) IsValidIcon(icon int16) bool {
	_, ok := g[icon]
	return ok
}

// GridAnnotationIcon is a font-awesome icon
type GridAnnotationIcon struct {
	Name    string `json:"name"`
	Unicode string `json:"unicode"`
}

// AnnotationIcons maps "icon" values to a GridAnnotationIcon object
var AnnotationIcons = GridAnnotationIconMapping{
	0: {
		Name:    "trophy",
		Unicode: "f091",
	},
	1: {
		Name:    "dollar-sign",
		Unicode: "f155",
	},
	2: {
		Name:    "money-bill",
		Unicode: "f0d6",
	},
	3: {
		Name:    "exclamation-circle",
		Unicode: "f06a",
	},
	4: {
		Name:    "dice",
		Unicode: "f522",
	},
	5: {
		Name: "arrow-alt-circle-right",
		Unicode: "f35a",
	},
}
