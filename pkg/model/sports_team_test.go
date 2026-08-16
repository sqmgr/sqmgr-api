/*
Copyright (C) 2019 Tom Peters

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package model

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestSportsTeamIsPlaceholder(t *testing.T) {
	tests := []struct {
		desc     string
		team     *SportsTeam
		expected bool
	}{
		{
			desc:     "full name is the placeholder",
			team:     &SportsTeam{ID: "-1", FullName: "TBD"},
			expected: true,
		},
		{
			desc:     "no full name, and the short name is the placeholder",
			team:     &SportsTeam{ID: "-2", Name: "TBD"},
			expected: true,
		},
		{
			desc:     "only the abbreviation is populated, and it's the placeholder",
			team:     &SportsTeam{ID: "-3", Abbreviation: "TBD"},
			expected: true,
		},
		{
			desc:     "every name field is the placeholder",
			team:     &SportsTeam{ID: "-4", Name: "TBD", FullName: "TBD", Abbreviation: "TBD"},
			expected: true,
		},
		{
			desc:     "lower case",
			team:     &SportsTeam{ID: "-5", FullName: "tbd"},
			expected: true,
		},
		{
			desc:     "mixed case",
			team:     &SportsTeam{ID: "-6", Name: "TbD"},
			expected: true,
		},
		{
			desc:     "surrounded by whitespace",
			team:     &SportsTeam{ID: "-7", FullName: "  TBD  "},
			expected: true,
		},
		{
			desc:     "surrounded by tabs and newlines",
			team:     &SportsTeam{ID: "-8", Abbreviation: "\tTBD\n"},
			expected: true,
		},
		{
			desc: "a real team",
			team: &SportsTeam{
				ID:           "12",
				Name:         "Chiefs",
				FullName:     "Kansas City Chiefs",
				Abbreviation: "KC",
			},
			expected: false,
		},
		{
			// only the displayed name decides. This team shows up as
			// "Tampa Bay Buccaneers" everywhere, so it's a real team no matter
			// what the fields further down the fallback chain say
			desc: "a real full name with a placeholder short name",
			team: &SportsTeam{
				ID:           "27",
				Name:         "TBD",
				FullName:     "Tampa Bay Buccaneers",
				Abbreviation: "TB",
			},
			expected: false,
		},
		{
			desc: "a real full name with a placeholder abbreviation",
			team: &SportsTeam{
				ID:           "28",
				Name:         "Buccaneers",
				FullName:     "Tampa Bay Buccaneers",
				Abbreviation: "TBD",
			},
			expected: false,
		},
		{
			// the display name falls through a blank full name to the short
			// name, so the placeholder is still caught
			desc:     "a blank full name in front of a placeholder short name",
			team:     &SportsTeam{ID: "-9", FullName: "   ", Name: "TBD"},
			expected: true,
		},
		{
			desc: "a real team whose name only contains the marker",
			team: &SportsTeam{
				ID:           "13",
				Name:         "Warriors",
				FullName:     "TBD State Warriors",
				Abbreviation: "TBDS",
			},
			expected: false,
		},
		{
			desc: "a real team with a similar looking name",
			team: &SportsTeam{
				ID:           "14",
				Name:         "Death",
				FullName:     "Tampa Bay Death",
				Abbreviation: "TB",
			},
			expected: false,
		},
		{
			desc:     "no names at all",
			team:     &SportsTeam{ID: "15"},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(test.team.IsPlaceholder()).Should(gomega.Equal(test.expected))
		})
	}
}

func TestSportsTeamDisplayName(t *testing.T) {
	tests := []struct {
		desc     string
		team     *SportsTeam
		expected string
	}{
		{
			desc: "the full name wins",
			team: &SportsTeam{
				ID:           "12",
				Name:         "Chiefs",
				FullName:     "Kansas City Chiefs",
				Abbreviation: "KC",
			},
			expected: "Kansas City Chiefs",
		},
		{
			desc:     "falls back to the short name",
			team:     &SportsTeam{ID: "12", Name: "Chiefs", Abbreviation: "KC"},
			expected: "Chiefs",
		},
		{
			desc:     "falls back to the abbreviation",
			team:     &SportsTeam{ID: "12", Abbreviation: "KC"},
			expected: "KC",
		},
		{
			desc:     "falls back to the id",
			team:     &SportsTeam{ID: "12"},
			expected: "12",
		},
		{
			desc:     "blank names are skipped",
			team:     &SportsTeam{ID: "12", FullName: "  ", Name: "\t\n", Abbreviation: "KC"},
			expected: "KC",
		},
		{
			desc:     "surrounding whitespace is trimmed",
			team:     &SportsTeam{ID: "12", FullName: "  Kansas City Chiefs  "},
			expected: "Kansas City Chiefs",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(test.team.DisplayName()).Should(gomega.Equal(test.expected))
		})
	}
}
