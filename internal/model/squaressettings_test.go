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
	"testing"

	"github.com/onsi/gomega"
)

func TestSquaresSettings(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	s := SquaresSettings{}

	testDefaultsAreUsed := func(msg string) {
		g.Expect(s.HomeTeamName()).Should(gomega.Equal(defaultHomeTeamName), msg)
		g.Expect(s.HomeTeamColor1()).Should(gomega.Equal(defaultHomeTeamColor1), msg)
		g.Expect(s.HomeTeamColor2()).Should(gomega.Equal(defaultHomeTeamColor2), msg)
		g.Expect(s.HomeTeamColor3()).Should(gomega.Equal(defaultHomeTeamColor3), msg)
		g.Expect(s.AwayTeamName()).Should(gomega.Equal(defaultAwayTeamName), msg)
		g.Expect(s.AwayTeamColor1()).Should(gomega.Equal(defaultAwayTeamColor1), msg)
		g.Expect(s.AwayTeamColor2()).Should(gomega.Equal(defaultAwayTeamColor2), msg)
		g.Expect(s.AwayTeamColor3()).Should(gomega.Equal(defaultAwayTeamColor3), msg)
	}

	testDefaultsAreUsed("initial defaults")

	s.SetHomeTeamName("A")
	s.SetHomeTeamColor1("B")
	s.SetHomeTeamColor2("C")
	s.SetHomeTeamColor3("D")
	s.SetAwayTeamName("E")
	s.SetAwayTeamColor1("F")
	s.SetAwayTeamColor2("G")
	s.SetAwayTeamColor3("H")

	g.Expect(s.HomeTeamName()).Should(gomega.Equal("A"))
	g.Expect(s.HomeTeamColor1()).Should(gomega.Equal("B"))
	g.Expect(s.HomeTeamColor2()).Should(gomega.Equal("C"))
	g.Expect(s.HomeTeamColor3()).Should(gomega.Equal("D"))
	g.Expect(s.AwayTeamName()).Should(gomega.Equal("E"))
	g.Expect(s.AwayTeamColor1()).Should(gomega.Equal("F"))
	g.Expect(s.AwayTeamColor2()).Should(gomega.Equal("G"))
	g.Expect(s.AwayTeamColor3()).Should(gomega.Equal("H"))

	s.SetHomeTeamName("")
	s.SetHomeTeamColor1("")
	s.SetHomeTeamColor2("")
	s.SetHomeTeamColor3("")
	s.SetAwayTeamName("")
	s.SetAwayTeamColor1("")
	s.SetAwayTeamColor2("")
	s.SetAwayTeamColor3("")

	testDefaultsAreUsed("set back to nil")
}
