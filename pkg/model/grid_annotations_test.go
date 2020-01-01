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
	"context"
	"github.com/onsi/gomega"
	"testing"
)

func TestAnnotationBySquareID(t *testing.T) {
	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	user, err := m.GetUser(ctx, IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	pool, err := m.NewPool(ctx, user.ID, "My Pool", GridTypeStd25, "my-pass")
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
