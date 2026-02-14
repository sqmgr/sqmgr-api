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
	"context"
	"encoding/json"
	"time"
	"unicode/utf8"
)

const (
	// NotesMaxLength is the maximum number of characters the notes can be
	NotesMaxLength = 500
	// BrandingImageURLMaxLength is the maximum number of characters for the branding image URL
	BrandingImageURLMaxLength = 2048
	// BrandingImageAltMaxLength is the maximum number of characters for the branding image alt text
	BrandingImageAltMaxLength = 255
)

// constants for default colors
const (
	DefaultHomeTeamColor1 = "#555555"
	DefaultHomeTeamColor2 = "#999999"
	DefaultAwayTeamColor1 = "#666666"
	DefaultAwayTeamColor2 = "#333333"
)

// GridSettings will contain various user-defined settings
// This object uses getters and setters to help guard against user input.
type GridSettings struct {
	gridID           int64
	homeTeamColor1   *string
	homeTeamColor2   *string
	awayTeamColor1   *string
	awayTeamColor2   *string
	notes            *string
	brandingImageURL *string
	brandingImageAlt *string
	modified         *time.Time
}

// gridSettingsJSON is used for custom serialization
type gridSettingsJSON struct {
	HomeTeamColor1   string `json:"homeTeamColor1"`
	HomeTeamColor2   string `json:"homeTeamColor2"`
	AwayTeamColor1   string `json:"awayTeamColor1"`
	AwayTeamColor2   string `json:"awayTeamColor2"`
	Notes            string `json:"notes"`
	BrandingImageURL string `json:"brandingImageUrl,omitempty"`
	BrandingImageAlt string `json:"brandingImageAlt,omitempty"`
}

// MarshalJSON adds custom JSON marshalling support
func (g GridSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(gridSettingsJSON{
		HomeTeamColor1:   g.HomeTeamColor1(),
		HomeTeamColor2:   g.HomeTeamColor2(),
		AwayTeamColor1:   g.AwayTeamColor1(),
		AwayTeamColor2:   g.AwayTeamColor2(),
		Notes:            g.Notes(),
		BrandingImageURL: g.BrandingImageURL(),
		BrandingImageAlt: g.BrandingImageAlt(),
	})
}

// Save will save the settings
func (g *GridSettings) Save(ctx context.Context, q Queryable) error {
	_, err := q.ExecContext(ctx, `
		UPDATE grid_settings SET
			home_team_color_1 = $1,
			home_team_color_2 = $2,
			away_team_color_1 = $3,
			away_team_color_2 = $4,
			notes = $5,
			branding_image_url = $6,
			branding_image_alt = $7,
			modified = (NOW() AT TIME ZONE 'utc')
		WHERE grid_id = $8
	`,
		g.homeTeamColor1,
		g.homeTeamColor2,
		g.awayTeamColor1,
		g.awayTeamColor2,
		g.notes,
		g.brandingImageURL,
		g.brandingImageAlt,
		g.gridID,
	)

	return err
}

// SetNotes will set the notes of the grid
func (g *GridSettings) SetNotes(str string) {
	if len(str) == 0 {
		g.notes = nil
		return
	}

	nRunes := utf8.RuneCountInString(str)
	if nRunes > NotesMaxLength {
		strChars := []rune(str)
		str = string(strChars[0:NotesMaxLength])
	}

	g.notes = &str
}

// Notes returns the notes
func (g *GridSettings) Notes() string {
	if g.notes == nil {
		return ""
	}

	return *g.notes
}

// SetHomeTeamColor1 is a setter for the home team primary color
func (g *GridSettings) SetHomeTeamColor1(color string) {
	if color == "" {
		g.homeTeamColor1 = nil
		return
	}

	g.homeTeamColor1 = &color
}

// SetHomeTeamColor2 is a setter for the home team secondary color
func (g *GridSettings) SetHomeTeamColor2(color string) {
	if color == "" {
		g.homeTeamColor2 = nil
		return
	}

	g.homeTeamColor2 = &color
}

// SetAwayTeamColor1 is a setter for the away team primary color
func (g *GridSettings) SetAwayTeamColor1(color string) {
	if color == "" {
		g.awayTeamColor1 = nil
		return
	}

	g.awayTeamColor1 = &color
}

// SetAwayTeamColor2 is a setter for the away team secondary color
func (g *GridSettings) SetAwayTeamColor2(color string) {
	if color == "" {
		g.awayTeamColor2 = nil
		return
	}

	g.awayTeamColor2 = &color
}

// HomeTeamColor1 is a getter for the home team primary color
func (g *GridSettings) HomeTeamColor1() string {
	if g.homeTeamColor1 == nil {
		return DefaultHomeTeamColor1
	}

	return *g.homeTeamColor1
}

// HomeTeamColor2 is a getter for the home team secondary color
func (g *GridSettings) HomeTeamColor2() string {
	if g.homeTeamColor2 == nil {
		return DefaultHomeTeamColor2
	}

	return *g.homeTeamColor2
}

// AwayTeamColor1 is a getter for the away team primary color
func (g *GridSettings) AwayTeamColor1() string {
	if g.awayTeamColor1 == nil {
		return DefaultAwayTeamColor1
	}

	return *g.awayTeamColor1
}

// AwayTeamColor2 is a getter for the away team secondary color
func (g *GridSettings) AwayTeamColor2() string {
	if g.awayTeamColor2 == nil {
		return DefaultAwayTeamColor2
	}

	return *g.awayTeamColor2
}

// SetBrandingImageURL is a setter for the branding image URL
func (g *GridSettings) SetBrandingImageURL(url string) {
	if len(url) == 0 {
		g.brandingImageURL = nil
		return
	}

	nRunes := utf8.RuneCountInString(url)
	if nRunes > BrandingImageURLMaxLength {
		urlChars := []rune(url)
		url = string(urlChars[0:BrandingImageURLMaxLength])
	}

	g.brandingImageURL = &url
}

// BrandingImageURL returns the branding image URL
func (g *GridSettings) BrandingImageURL() string {
	if g.brandingImageURL == nil {
		return ""
	}

	return *g.brandingImageURL
}

// SetBrandingImageAlt is a setter for the branding image alt text
func (g *GridSettings) SetBrandingImageAlt(alt string) {
	if len(alt) == 0 {
		g.brandingImageAlt = nil
		return
	}

	nRunes := utf8.RuneCountInString(alt)
	if nRunes > BrandingImageAltMaxLength {
		altChars := []rune(alt)
		alt = string(altChars[0:BrandingImageAltMaxLength])
	}

	g.brandingImageAlt = &alt
}

// BrandingImageAlt returns the branding image alt text
func (g *GridSettings) BrandingImageAlt() string {
	if g.brandingImageAlt == nil {
		return ""
	}

	return *g.brandingImageAlt
}
