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

package validator

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestURL(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	t.Run("valid https URL", func(t *testing.T) {
		v := New()
		result := v.URL("url", "https://example.com/image.png")
		g.Expect(v.OK()).Should(gomega.BeTrue())
		g.Expect(result).Should(gomega.Equal("https://example.com/image.png"))
	})

	t.Run("valid http URL", func(t *testing.T) {
		v := New()
		result := v.URL("url", "http://example.com/image.png")
		g.Expect(v.OK()).Should(gomega.BeTrue())
		g.Expect(result).Should(gomega.Equal("http://example.com/image.png"))
	})

	t.Run("invalid protocol ftp", func(t *testing.T) {
		v := New()
		result := v.URL("url", "ftp://example.com/file.txt")
		g.Expect(v.OK()).Should(gomega.BeFalse())
		g.Expect(result).Should(gomega.Equal(""))
		g.Expect(v.Errors["url"]).Should(gomega.ContainElement("must be an http or https URL"))
	})

	t.Run("invalid protocol javascript", func(t *testing.T) {
		v := New()
		result := v.URL("url", "javascript:alert(1)")
		g.Expect(v.OK()).Should(gomega.BeFalse())
		g.Expect(result).Should(gomega.Equal(""))
	})

	t.Run("empty required URL", func(t *testing.T) {
		v := New()
		result := v.URL("url", "")
		g.Expect(v.OK()).Should(gomega.BeFalse())
		g.Expect(result).Should(gomega.Equal(""))
		g.Expect(v.Errors["url"]).Should(gomega.ContainElement("must be a valid URL"))
	})

	t.Run("empty optional URL", func(t *testing.T) {
		v := New()
		result := v.URL("url", "", true)
		g.Expect(v.OK()).Should(gomega.BeTrue())
		g.Expect(result).Should(gomega.Equal(""))
	})

	t.Run("URL with no host", func(t *testing.T) {
		v := New()
		result := v.URL("url", "https:///path")
		g.Expect(v.OK()).Should(gomega.BeFalse())
		g.Expect(result).Should(gomega.Equal(""))
	})

	t.Run("URL with query params", func(t *testing.T) {
		v := New()
		result := v.URL("url", "https://example.com/image.png?size=large&format=webp")
		g.Expect(v.OK()).Should(gomega.BeTrue())
		g.Expect(result).Should(gomega.Equal("https://example.com/image.png?size=large&format=webp"))
	})
}
