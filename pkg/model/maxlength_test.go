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
	"strings"

	"github.com/onsi/gomega"
)

func testMaxLength(g *gomega.GomegaWithT, getter func() string, setter func(string), maxLength int, msg string) {
	str := strings.Repeat("á", maxLength)

	// no truncation
	setter(str)
	g.Expect(getter()).Should(gomega.Equal(str), msg)
	g.Expect(len(getter())).Should(gomega.Equal(maxLength*2), msg) // á is two-bytes

	// truncation
	setter(str + "é")
	g.Expect(getter()).Should(gomega.Equal(str), msg)
	g.Expect(len([]rune(getter()))).Should(gomega.Equal(maxLength), msg)
}
