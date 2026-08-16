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

func TestGridSettings(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	s := GridSettings{}

	testDefaultsAreUsed := func(msg string) {
		g.Expect(s.HomeTeamColor1()).Should(gomega.Equal(DefaultHomeTeamColor1), msg)
		g.Expect(s.HomeTeamColor2()).Should(gomega.Equal(DefaultHomeTeamColor2), msg)
		g.Expect(s.AwayTeamColor1()).Should(gomega.Equal(DefaultAwayTeamColor1), msg)
		g.Expect(s.AwayTeamColor2()).Should(gomega.Equal(DefaultAwayTeamColor2), msg)
		g.Expect(s.Notes()).Should(gomega.Equal(""), msg)
		g.Expect(s.BrandingImageURL()).Should(gomega.Equal(""), msg)
		g.Expect(s.BrandingImageAlt()).Should(gomega.Equal(""), msg)
	}

	testDefaultsAreUsed("initial defaults")

	s.SetHomeTeamColor1("B")
	s.SetHomeTeamColor2("C")
	s.SetAwayTeamColor1("F")
	s.SetAwayTeamColor2("G")
	s.SetNotes("I")
	s.SetBrandingImageURL("https://example.com/image.png")
	s.SetBrandingImageAlt("Example Logo")

	g.Expect(s.HomeTeamColor1()).Should(gomega.Equal("B"))
	g.Expect(s.HomeTeamColor2()).Should(gomega.Equal("C"))
	g.Expect(s.AwayTeamColor1()).Should(gomega.Equal("F"))
	g.Expect(s.AwayTeamColor2()).Should(gomega.Equal("G"))
	g.Expect(s.Notes()).Should(gomega.Equal("I"))
	g.Expect(s.BrandingImageURL()).Should(gomega.Equal("https://example.com/image.png"))
	g.Expect(s.BrandingImageAlt()).Should(gomega.Equal("Example Logo"))

	s.SetHomeTeamColor1("")
	s.SetHomeTeamColor2("")
	s.SetAwayTeamColor1("")
	s.SetAwayTeamColor2("")
	s.SetNotes("")
	s.SetBrandingImageURL("")
	s.SetBrandingImageAlt("")

	testDefaultsAreUsed("set back to nil")
}

func TestMaxLength(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	s := &GridSettings{}

	testMaxLength(g, s.Notes, s.SetNotes, NotesMaxLength, "notes")
	testMaxLength(g, s.BrandingImageURL, s.SetBrandingImageURL, BrandingImageURLMaxLength, "brandingImageURL")
	testMaxLength(g, s.BrandingImageAlt, s.SetBrandingImageAlt, BrandingImageAltMaxLength, "brandingImageAlt")
}

func TestGridSettingsIsPristine(t *testing.T) {
	g := gomega.NewWithT(t)

	// a nil receiver means the settings were never loaded. Their contents are
	// unknown, and callers gate destructive work on this check, so it has to
	// fail closed
	var nilSettings *GridSettings
	g.Expect(nilSettings.IsPristine()).Should(gomega.BeFalse())

	// this is what new_grid() inserts: only grid_id, everything else NULL
	g.Expect((&GridSettings{gridID: 1}).IsPristine()).Should(gomega.BeTrue())

	mutations := map[string]func(*GridSettings){
		"home team color 1":  func(s *GridSettings) { s.SetHomeTeamColor1("#E31837") },
		"home team color 2":  func(s *GridSettings) { s.SetHomeTeamColor2("#FFB612") },
		"away team color 1":  func(s *GridSettings) { s.SetAwayTeamColor1("#00338D") },
		"away team color 2":  func(s *GridSettings) { s.SetAwayTeamColor2("#C60C30") },
		"notes":              func(s *GridSettings) { s.SetNotes("bring snacks") },
		"branding image url": func(s *GridSettings) { s.SetBrandingImageURL("https://example.com/logo.png") },
		"branding image alt": func(s *GridSettings) { s.SetBrandingImageAlt("Logo") },
	}

	for name, mutate := range mutations {
		s := &GridSettings{gridID: 1}
		mutate(s)
		g.Expect(s.IsPristine()).Should(gomega.BeFalse(), name)
	}
}
