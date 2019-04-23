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

	p := DefaultPagination(3, 7)
	g.Expect(p).Should(gomega.Equal(Pagination{1, 2, 3, 4, 5, 6, 7}))

	p = DefaultPagination(1, 10)
	g.Expect(p).Should(gomega.Equal(Pagination{1, 2, 3, 4, 5, 0, 10}))

	p = DefaultPagination(2, 10)
	g.Expect(p).Should(gomega.Equal(Pagination{1, 2, 3, 4, 5, 0, 10}))

	p = DefaultPagination(3, 10)
	g.Expect(p).Should(gomega.Equal(Pagination{1, 2, 3, 4, 5, 0, 10}))

	p = DefaultPagination(4, 10)
	g.Expect(p).Should(gomega.Equal(Pagination{1, 2, 3, 4, 5, 6, 0, 10}))

	p = DefaultPagination(5, 10)
	g.Expect(p).Should(gomega.Equal(Pagination{1, 0, 3, 4, 5, 6, 7, 0, 10}))

	p = DefaultPagination(6, 10)
	g.Expect(p).Should(gomega.Equal(Pagination{1, 0, 4, 5, 6, 7, 8, 0, 10}))

	p = DefaultPagination(7, 10)
	g.Expect(p).Should(gomega.Equal(Pagination{1, 0, 5, 6, 7, 8, 9, 10}))

	p = DefaultPagination(8, 10)
	g.Expect(p).Should(gomega.Equal(Pagination{1, 0, 6, 7, 8, 9, 10}))

	p = DefaultPagination(9, 10)
	g.Expect(p).Should(gomega.Equal(Pagination{1, 0, 6, 7, 8, 9, 10}))

	p = DefaultPagination(10, 10)
	g.Expect(p).Should(gomega.Equal(Pagination{1, 0, 6, 7, 8, 9, 10}))
}

func TestPaginationBuilder(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	b := NewPaginationBuilder(5, 10)
	b.CapBuffer = 2
	b.WindowBuffer = 1
	g.Expect(b.Build()).Should(gomega.Equal(Pagination{1, 2, 0, 5, 0, 9, 10}))

	b.WindowBuffer = 2
	g.Expect(b.Build()).Should(gomega.Equal(Pagination{1, 2, 0, 5, 6, 0, 9, 10}))

	b.WindowBuffer = 3
	g.Expect(b.Build()).Should(gomega.Equal(Pagination{1, 2, 0, 4, 5, 6, 0, 9, 10}))

	b.WindowBuffer = 4
	g.Expect(b.Build()).Should(gomega.Equal(Pagination{1, 2, 0, 4, 5, 6, 7, 0, 9, 10}))
}
