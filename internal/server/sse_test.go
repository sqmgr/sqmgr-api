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

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
)

func TestExtractBearerToken_Valid(t *testing.T) {
	g := gomega.NewWithT(t)

	token, ok := extractBearerToken("Bearer abc123")
	g.Expect(ok).Should(gomega.BeTrue())
	g.Expect(token).Should(gomega.Equal("abc123"))
}

func TestExtractBearerToken_CaseInsensitive(t *testing.T) {
	g := gomega.NewWithT(t)

	token, ok := extractBearerToken("bearer abc123")
	g.Expect(ok).Should(gomega.BeTrue())
	g.Expect(token).Should(gomega.Equal("abc123"))
}

func TestExtractBearerToken_Invalid(t *testing.T) {
	g := gomega.NewWithT(t)

	_, ok := extractBearerToken("Basic abc123")
	g.Expect(ok).Should(gomega.BeFalse())

	_, ok = extractBearerToken("abc123")
	g.Expect(ok).Should(gomega.BeFalse())

	_, ok = extractBearerToken("")
	g.Expect(ok).Should(gomega.BeFalse())
}

func TestSSEEndpoint_MissingAccessToken(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token:[A-Za-z0-9_-]+}/events").Methods(http.MethodGet).Handler(s.getPoolTokenEventsEndpoint())

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token/events", nil)
	rec := httptest.NewRecorder()

	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnauthorized))
}

func TestSSEEndpoint_InvalidAccessToken(t *testing.T) {
	g := gomega.NewWithT(t)

	s := &Server{
		Router: mux.NewRouter(),
		broker: NewPoolBroker(),
	}

	s.Router.Path("/pool/{token:[A-Za-z0-9_-]+}/events").Methods(http.MethodGet).Handler(s.getPoolTokenEventsEndpoint())

	req := httptest.NewRequest(http.MethodGet, "/pool/test-token/events?access_token=invalid-jwt", nil)
	rec := httptest.NewRecorder()

	s.Router.ServeHTTP(rec, req)

	g.Expect(rec.Code).Should(gomega.Equal(http.StatusUnauthorized))
}
