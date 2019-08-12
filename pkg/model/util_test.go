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