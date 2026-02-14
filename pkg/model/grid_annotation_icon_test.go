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

import (
	"github.com/onsi/gomega"
	"testing"
)

func TestGridAnnotationIcon(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(len(AnnotationIcons) > 0).Should(gomega.BeTrue())
	g.Expect(AnnotationIcons.IsValidIcon(-1)).Should(gomega.BeFalse())
	g.Expect(AnnotationIcons.IsValidIcon(int16(len(AnnotationIcons)))).Should(gomega.BeFalse())
	g.Expect(AnnotationIcons.IsValidIcon(int16(len(AnnotationIcons)) - 1)).Should(gomega.BeTrue())
}
