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
