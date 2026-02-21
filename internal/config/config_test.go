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

package config

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestCORSAllowedOrigins(t *testing.T) {
	g := gomega.NewWithT(t)

	instance = &config{
		corsAllowedOrigins: []string{"https://sqmgr.com", "https://www.sqmgr.com", "http://localhost:8080"},
	}
	defer func() { instance = nil }()

	origins := CORSAllowedOrigins()
	g.Expect(origins).To(gomega.HaveLen(3))
	g.Expect(origins).To(gomega.ContainElement("https://sqmgr.com"))
	g.Expect(origins).To(gomega.ContainElement("https://www.sqmgr.com"))
	g.Expect(origins).To(gomega.ContainElement("http://localhost:8080"))
}

func TestCORSAllowedOrigins_PanicWithoutLoad(t *testing.T) {
	g := gomega.NewWithT(t)

	instance = nil
	g.Expect(func() { CORSAllowedOrigins() }).To(gomega.Panic())
}
