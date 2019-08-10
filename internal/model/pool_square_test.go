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
	"github.com/onsi/gomega"
	"strings"
	"testing"
)

func TestPoolSquare_Claimant(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &PoolSquare{}

	okClaimant := strings.Repeat("é", 30)
	s.SetClaimant(okClaimant)
	g.Expect(s.claimant).Should(gomega.Equal(okClaimant))
	g.Expect(s.Claimant()).Should(gomega.Equal(okClaimant))

	tooLongClaimant := strings.Repeat("í", 31)
	s.SetClaimant(tooLongClaimant)
	g.Expect(s.claimant).ShouldNot(gomega.Equal(tooLongClaimant))
	g.Expect(s.Claimant()).ShouldNot(gomega.Equal(tooLongClaimant))
	g.Expect(s.Claimant()).Should(gomega.Equal(string([]rune(tooLongClaimant)[0:30])))
}
