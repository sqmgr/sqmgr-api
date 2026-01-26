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
