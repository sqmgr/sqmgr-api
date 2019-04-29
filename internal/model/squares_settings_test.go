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
	"strings"
	"testing"

	"github.com/onsi/gomega"
)

func TestSquaresSettings(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	s := SquaresSettings{}

	testDefaultsAreUsed := func(msg string) {
		g.Expect(s.HomeTeamName).Should(gomega.BeNil(), msg)
		g.Expect(s.HomeTeamColor1).Should(gomega.BeNil(), msg)
		g.Expect(s.HomeTeamColor2).Should(gomega.BeNil(), msg)
		g.Expect(s.HomeTeamColor3).Should(gomega.BeNil(), msg)
		g.Expect(s.AwayTeamName).Should(gomega.BeNil(), msg)
		g.Expect(s.AwayTeamColor1).Should(gomega.BeNil(), msg)
		g.Expect(s.AwayTeamColor2).Should(gomega.BeNil(), msg)
		g.Expect(s.AwayTeamColor3).Should(gomega.BeNil(), msg)
		g.Expect(s.notes).Should(gomega.BeNil(), msg)
	}

	testDefaultsAreUsed("initial defaults")

	ptr := func(s string) *string { return &s }
	s.HomeTeamName = ptr("A")
	s.HomeTeamColor1 = ptr("B")
	s.HomeTeamColor2 = ptr("C")
	s.HomeTeamColor3 = ptr("D")
	s.AwayTeamName = ptr("E")
	s.AwayTeamColor1 = ptr("F")
	s.AwayTeamColor2 = ptr("G")
	s.AwayTeamColor3 = ptr("H")
	s.SetNotes("I")

	g.Expect(*s.HomeTeamName).Should(gomega.Equal("A"))
	g.Expect(*s.HomeTeamColor1).Should(gomega.Equal("B"))
	g.Expect(*s.HomeTeamColor2).Should(gomega.Equal("C"))
	g.Expect(*s.HomeTeamColor3).Should(gomega.Equal("D"))
	g.Expect(*s.AwayTeamName).Should(gomega.Equal("E"))
	g.Expect(*s.AwayTeamColor1).Should(gomega.Equal("F"))
	g.Expect(*s.AwayTeamColor2).Should(gomega.Equal("G"))
	g.Expect(*s.AwayTeamColor3).Should(gomega.Equal("H"))
	g.Expect(s.Notes()).Should(gomega.Equal("I"))

	s.HomeTeamName = nil
	s.HomeTeamColor1 = nil
	s.HomeTeamColor2 = nil
	s.HomeTeamColor3 = nil
	s.AwayTeamName = nil
	s.AwayTeamColor1 = nil
	s.AwayTeamColor2 = nil
	s.AwayTeamColor3 = nil
	s.SetNotes("")

	testDefaultsAreUsed("set back to nil")
}

func TestNotesLength(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	s := &SquaresSettings{}

	str := strings.Repeat("é", NotesMaxLength)
	s.SetNotes(str)

	g.Expect(s.Notes()).Should(gomega.Equal(str))
	g.Expect(len(s.Notes())).Should(gomega.Equal(NotesMaxLength * 2)) // é is two bytes

	truncStr := strings.Repeat("á", NotesMaxLength)
	longerStr := truncStr + "á"
	s.SetNotes(longerStr)
	g.Expect(s.Notes()).Should(gomega.Equal(truncStr))
	g.Expect(len(s.Notes())).Should(gomega.Equal(NotesMaxLength * 2)) // é is two bytes
	g.Expect(len([]rune(s.Notes()))).Should(gomega.Equal(NotesMaxLength))
}
