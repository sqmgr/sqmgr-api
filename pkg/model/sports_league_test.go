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

func TestSportsLeagueIsValid(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	g.Expect(SportsLeagueNFL.IsValid()).Should(gomega.BeTrue())
	g.Expect(SportsLeagueNBA.IsValid()).Should(gomega.BeTrue())
	g.Expect(SportsLeagueWNBA.IsValid()).Should(gomega.BeTrue())
	g.Expect(SportsLeagueNCAAB.IsValid()).Should(gomega.BeTrue())
	g.Expect(SportsLeagueNCAAF.IsValid()).Should(gomega.BeTrue())
	g.Expect(SportsLeague("invalid").IsValid()).Should(gomega.BeFalse())
}

func TestIsValidSportsLeague(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	g.Expect(IsValidSportsLeague("nfl")).Should(gomega.BeTrue())
	g.Expect(IsValidSportsLeague("nba")).Should(gomega.BeTrue())
	g.Expect(IsValidSportsLeague("wnba")).Should(gomega.BeTrue())
	g.Expect(IsValidSportsLeague("ncaab")).Should(gomega.BeTrue())
	g.Expect(IsValidSportsLeague("ncaaf")).Should(gomega.BeTrue())
	g.Expect(IsValidSportsLeague("invalid")).Should(gomega.BeFalse())
	g.Expect(IsValidSportsLeague("")).Should(gomega.BeFalse())
}

func TestValidSportsLeagues(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	leagues := ValidSportsLeagues()
	g.Expect(len(leagues)).Should(gomega.Equal(5))

	// Check that all leagues are present
	keys := make(map[SportsLeague]bool)
	for _, l := range leagues {
		keys[l.Key] = true
	}

	g.Expect(keys[SportsLeagueNFL]).Should(gomega.BeTrue())
	g.Expect(keys[SportsLeagueNBA]).Should(gomega.BeTrue())
	g.Expect(keys[SportsLeagueWNBA]).Should(gomega.BeTrue())
	g.Expect(keys[SportsLeagueNCAAB]).Should(gomega.BeTrue())
	g.Expect(keys[SportsLeagueNCAAF]).Should(gomega.BeTrue())
}

func TestSportsLeagueScan(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	var l SportsLeague

	// Test with string
	err := l.Scan("nfl")
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(l).Should(gomega.Equal(SportsLeagueNFL))

	// Test with bytes
	err = l.Scan([]byte("nba"))
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(l).Should(gomega.Equal(SportsLeagueNBA))

	// Test with nil
	err = l.Scan(nil)
	g.Expect(err).Should(gomega.HaveOccurred())

	// Test with unsupported type
	err = l.Scan(123)
	g.Expect(err).Should(gomega.HaveOccurred())
}

func TestSportsLeagueValue(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	l := SportsLeagueNFL
	val, err := l.Value()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(val).Should(gomega.Equal("nfl"))
}

// Test backward compatibility aliases
func TestBDLLeagueAliases(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// BDLLeague should be an alias for SportsLeague
	var l BDLLeague = BDLLeagueNFL
	g.Expect(l).Should(gomega.Equal(SportsLeagueNFL))

	// Constants should match
	g.Expect(BDLLeagueNFL).Should(gomega.Equal(SportsLeagueNFL))
	g.Expect(BDLLeagueNBA).Should(gomega.Equal(SportsLeagueNBA))
	g.Expect(BDLLeagueWNBA).Should(gomega.Equal(SportsLeagueWNBA))
	g.Expect(BDLLeagueNCAAB).Should(gomega.Equal(SportsLeagueNCAAB))
	g.Expect(BDLLeagueNCAAF).Should(gomega.Equal(SportsLeagueNCAAF))

	// Backward compatibility functions
	g.Expect(IsValidBDLLeague("nfl")).Should(gomega.BeTrue())
	g.Expect(len(ValidBDLLeagues())).Should(gomega.Equal(5))
}
