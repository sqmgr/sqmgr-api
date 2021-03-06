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

func TestGridSettings(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	s := GridSettings{}

	testDefaultsAreUsed := func(msg string) {
		g.Expect(s.HomeTeamColor1()).Should(gomega.Equal(DefaultHomeTeamColor1), msg)
		g.Expect(s.HomeTeamColor2()).Should(gomega.Equal(DefaultHomeTeamColor2), msg)
		g.Expect(s.AwayTeamColor1()).Should(gomega.Equal(DefaultAwayTeamColor1), msg)
		g.Expect(s.AwayTeamColor2()).Should(gomega.Equal(DefaultAwayTeamColor2), msg)
		g.Expect(s.Notes()).Should(gomega.Equal(""), msg)
	}

	testDefaultsAreUsed("initial defaults")

	s.SetHomeTeamColor1("B")
	s.SetHomeTeamColor2("C")
	s.SetAwayTeamColor1("F")
	s.SetAwayTeamColor2("G")
	s.SetNotes("I")

	g.Expect(s.HomeTeamColor1()).Should(gomega.Equal("B"))
	g.Expect(s.HomeTeamColor2()).Should(gomega.Equal("C"))
	g.Expect(s.AwayTeamColor1()).Should(gomega.Equal("F"))
	g.Expect(s.AwayTeamColor2()).Should(gomega.Equal("G"))
	g.Expect(s.Notes()).Should(gomega.Equal("I"))

	s.SetHomeTeamColor1("")
	s.SetHomeTeamColor2("")
	s.SetAwayTeamColor1("")
	s.SetAwayTeamColor2("")
	s.SetNotes("")

	testDefaultsAreUsed("set back to nil")
}

func TestMaxLength(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	s := &GridSettings{}

	testMaxLength(g, s.Notes, s.SetNotes, NotesMaxLength, "notes")
}
