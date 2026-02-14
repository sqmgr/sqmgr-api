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

package tokengen

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestGenerate(t *testing.T) {
	g := gomega.NewWithT(t)

	for i := 0; i < 40; i++ {
		val, err := Generate(i)
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(len(val)).Should(gomega.Equal(i))
	}

	val1, _ := Generate(10)
	val2, _ := Generate(10)
	g.Expect(val1).ShouldNot(gomega.Equal(val2))
}
