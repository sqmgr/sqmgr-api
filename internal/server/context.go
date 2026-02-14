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

	"github.com/sqmgr/sqmgr-api/pkg/model"
)

type sqmgrContext int

const (
	ctxUserKey sqmgrContext = iota
	ctxUserIDKey
	ctxPoolKey
	ctxGridKey
	ctxSquareIDKey
	ctxRequestIDKey
)

func userFromContext(ctx context.Context) (*model.User, bool) {
	u, ok := ctx.Value(ctxUserKey).(*model.User)
	return u, ok
}

func userIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(ctxUserIDKey).(int64)
	return id, ok
}

func poolFromContext(ctx context.Context) (*model.Pool, bool) {
	p, ok := ctx.Value(ctxPoolKey).(*model.Pool)
	return p, ok
}

func gridFromContext(ctx context.Context) (*model.Grid, bool) {
	g, ok := ctx.Value(ctxGridKey).(*model.Grid)
	return g, ok
}

func squareIDFromContext(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(ctxSquareIDKey).(int)
	return id, ok
}

func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxRequestIDKey).(string)
	return id
}
