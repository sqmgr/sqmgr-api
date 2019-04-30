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
	"testing"

	"github.com/onsi/gomega"
)

func TestSessionUser(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	theContext := context.Background()
	theGrid := &Grid{}
	called := false

	joinFn := JoinGrid(func(ctx context.Context, grid *Grid) error {
		g.Expect(ctx).Should(gomega.Equal(theContext))
		g.Expect(grid).Should(gomega.Equal(theGrid))
		called = true

		return nil
	})

	u := NewSessionUser(map[int64]bool{1000: true, 2000: true}, joinFn)
	ok, err := u.IsMemberOf(context.Background(), &Grid{id: 1000})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ok).Should(gomega.BeTrue())

	ok, err = u.IsMemberOf(context.Background(), &Grid{id: 3000})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ok).Should(gomega.BeFalse())

	g.Expect(u.IsAdminOf(context.Background(), &Grid{id: 1000})).Should(gomega.BeFalse())

	g.Expect(u.JoinGrid(theContext, theGrid)).Should(gomega.Succeed())
	g.Expect(called).Should(gomega.BeTrue())
}
