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
	"context"
	"testing"
)

func TestUserFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	_, ok := userFromContext(ctx)
	if ok {
		t.Error("expected ok to be false for missing user")
	}
}

func TestUserFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxUserKey, "not a user")
	_, ok := userFromContext(ctx)
	if ok {
		t.Error("expected ok to be false for wrong type")
	}
}

func TestUserIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	_, ok := userIDFromContext(ctx)
	if ok {
		t.Error("expected ok to be false for missing user ID")
	}
}

func TestPoolFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	_, ok := poolFromContext(ctx)
	if ok {
		t.Error("expected ok to be false for missing pool")
	}
}

func TestGridFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	_, ok := gridFromContext(ctx)
	if ok {
		t.Error("expected ok to be false for missing grid")
	}
}

func TestSquareIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	_, ok := squareIDFromContext(ctx)
	if ok {
		t.Error("expected ok to be false for missing square ID")
	}
}

func TestRequestIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	id := requestIDFromContext(ctx)
	if id != "" {
		t.Errorf("expected empty string for missing request ID, got %q", id)
	}
}

func TestRequestIDFromContext_Present(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxRequestIDKey, "test-id-123")
	id := requestIDFromContext(ctx)
	if id != "test-id-123" {
		t.Errorf("expected %q, got %q", "test-id-123", id)
	}
}
