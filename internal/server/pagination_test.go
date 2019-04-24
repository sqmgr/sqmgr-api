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

package server

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestDefaultPagination(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	p := NewPagination(7, 3)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 2, 3, 4, 5, 6, 7}))
	g.Expect(p.PrevPage()).Should(gomega.Equal(2))
	g.Expect(p.NextPage()).Should(gomega.Equal(4))
	g.Expect(p.Total()).Should(gomega.Equal(7))
	g.Expect(p.CurrentPage()).Should(gomega.Equal(3))

	p = NewPagination(10, 1)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 2, 3, 4, 5, 0, 10}))
	g.Expect(p.PrevPage()).Should(gomega.Equal(1))
	g.Expect(p.NextPage()).Should(gomega.Equal(2))

	p = NewPagination(10, 2)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 2, 3, 4, 5, 0, 10}))

	p = NewPagination(10, 3)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 2, 3, 4, 5, 0, 10}))

	p = NewPagination(10, 4)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 2, 3, 4, 5, 6, 0, 10}))

	p = NewPagination(10, 5)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 0, 3, 4, 5, 6, 7, 0, 10}))

	p = NewPagination(10, 6)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 0, 4, 5, 6, 7, 8, 0, 10}))

	p = NewPagination(10, 7)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 0, 5, 6, 7, 8, 9, 10}))

	p = NewPagination(10, 8)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 0, 6, 7, 8, 9, 10}))

	p = NewPagination(10, 9)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 0, 6, 7, 8, 9, 10}))

	p = NewPagination(10, 10)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 0, 6, 7, 8, 9, 10}))
	g.Expect(p.PrevPage()).Should(gomega.Equal(9))
	g.Expect(p.NextPage()).Should(gomega.Equal(10))
}

func TestPaginationBuilder(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	p := NewPagination(10, 5)
	p.SetCapBuffer(2)
	p.SetWindowBuffer(1)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 2, 0, 5, 0, 9, 10}))

	p = NewPagination(10, 5)
	p.SetCapBuffer(2)
	p.SetWindowBuffer(2)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 2, 0, 5, 6, 0, 9, 10}))

	p = NewPagination(10, 5)
	p.SetCapBuffer(2)
	p.SetWindowBuffer(3)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 2, 0, 4, 5, 6, 0, 9, 10}))

	p = NewPagination(10, 5)
	p.SetCapBuffer(2)
	p.SetWindowBuffer(4)
	g.Expect(p.Pages()).Should(gomega.Equal([]int{1, 2, 0, 4, 5, 6, 7, 0, 9, 10}))
}
