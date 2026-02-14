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
	"github.com/onsi/gomega"
	"testing"
)

func TestIPFromRemoteAddr(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	g.Expect(ipFromRemoteAddr("127.0.0.1")).Should(gomega.Equal("127.0.0.1"))
	g.Expect(ipFromRemoteAddr("127.0.0.1:8000")).Should(gomega.Equal("127.0.0.1"))
	g.Expect(ipFromRemoteAddr("[::1]")).Should(gomega.Equal("[::1]"))
	g.Expect(ipFromRemoteAddr("[::1]:8000")).Should(gomega.Equal("[::1]"))
	g.Expect(ipFromRemoteAddr("1:2:3:4")).Should(gomega.Equal("1:2:3:4"))
}
