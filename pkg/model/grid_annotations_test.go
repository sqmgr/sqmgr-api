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
	"context"
	"github.com/onsi/gomega"
	"os"
	"testing"
)

func TestAnnotationBySquareID(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	user, err := m.GetUser(ctx, IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	pool, err := m.NewPool(ctx, user.ID, "My Pool", GridTypeStd25, "my-pass", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	grid, err := pool.DefaultGrid(ctx)
	g.Expect(err).Should(gomega.Succeed())

	annotation, err := grid.AnnotationBySquareID(ctx, 1)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(annotation.Annotation).Should(gomega.Equal(""))
	g.Expect(annotation.Icon).Should(gomega.Equal(int16(0)))
	g.Expect(annotation.Created.IsZero()).Should(gomega.BeTrue(), "no created date")
	g.Expect(annotation.Modified.IsZero()).Should(gomega.BeTrue(), "no modified date")

	annotation.Annotation = "My Test"
	annotation.Icon = 1
	g.Expect(annotation.Save(ctx)).Should(gomega.Succeed())
	g.Expect(annotation.Created.IsZero()).Should(gomega.BeFalse(), "has created date")
	g.Expect(annotation.Modified.IsZero()).Should(gomega.BeFalse(), "has modified date")

	annotation, err = grid.AnnotationBySquareID(ctx, 1)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(annotation.Annotation).Should(gomega.Equal("My Test"))
	g.Expect(annotation.Icon).Should(gomega.Equal(int16(1)))
	g.Expect(annotation.Created.IsZero()).Should(gomega.BeFalse(), "has created date")
	g.Expect(annotation.Modified.IsZero()).Should(gomega.BeFalse(), "has modified date")
	g.Expect(annotation.Created).Should(gomega.Equal(annotation.Modified))

	annotation.Annotation = "My Test-Updated"
	annotation.Icon = 2
	g.Expect(annotation.Save(ctx)).Should(gomega.Succeed())

	annotation, err = grid.AnnotationBySquareID(ctx, 1)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(annotation.Modified.After(annotation.Created)).Should(gomega.BeTrue())

	annotation, err = grid.AnnotationBySquareID(ctx, 2)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(annotation.Annotation).Should(gomega.Equal(""))
	annotation.Annotation = "Second Test"
	g.Expect(annotation.Save(ctx)).Should(gomega.Succeed())

	annotations, err := grid.Annotations(ctx)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(annotations)).Should(gomega.Equal(2))
	g.Expect(annotations[1].Annotation).Should(gomega.Equal("My Test-Updated"))
	g.Expect(annotations[1].Icon).Should(gomega.Equal(int16(2)))
	g.Expect(annotations[2].Annotation).Should(gomega.Equal("Second Test"))
}
