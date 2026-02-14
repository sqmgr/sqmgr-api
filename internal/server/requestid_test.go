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
)

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	s := &Server{broker: NewPoolBroker()}
	var capturedID string
	handler := s.requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = requestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if capturedID == "" {
		t.Error("expected request ID to be generated")
	}

	responseID := rr.Header().Get(requestIDHeader)
	if responseID == "" {
		t.Error("expected request ID in response header")
	}

	if capturedID != responseID {
		t.Errorf("context ID %q != response header ID %q", capturedID, responseID)
	}
}

func TestRequestIDMiddleware_ForwardsExisting(t *testing.T) {
	s := &Server{broker: NewPoolBroker()}
	existingID := "upstream-trace-id-123"
	var capturedID string
	handler := s.requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = requestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(requestIDHeader, existingID)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if capturedID != existingID {
		t.Errorf("expected request ID %q, got %q", existingID, capturedID)
	}

	responseID := rr.Header().Get(requestIDHeader)
	if responseID != existingID {
		t.Errorf("expected response header ID %q, got %q", existingID, responseID)
	}
}
