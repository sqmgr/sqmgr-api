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
