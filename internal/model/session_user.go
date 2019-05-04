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
	"context"
	"github.com/google/uuid"
)

// SessionUser represents a user who has not logged in
type SessionUser struct {
	userID   string
	gridIDs  map[int64]bool
	joinFunc JoinGrid
}

// NewSessionUser returns a new session user
func NewSessionUser(userID string, ids map[int64]bool, joinFunc JoinGrid) *SessionUser {
	if userID == "" {
		userID = uuid.New().String()
	}
	return &SessionUser{userID, ids, joinFunc}
}

// IsAdminOf will always return false for a session-based user
func (u *SessionUser) IsAdminOf(ctx context.Context, s *Grid) bool {
	return false
}

// IsMemberOf will return true if the user is a member of the grid
func (u *SessionUser) IsMemberOf(ctx context.Context, s *Grid) (bool, error) {
	_, found := u.gridIDs[s.id]
	return found, nil
}

// JoinGrid will attempt to join the grid
func (u *SessionUser) JoinGrid(ctx context.Context, s *Grid) error {
	return u.joinFunc(ctx, s)
}

// UserID will always return 0
func (u *SessionUser) UserID(ctx context.Context) interface{} {
	return u.userID
}

// OpaqueUserID returns an opaque user ID
func (u *SessionUser) OpaqueUserID(ctx context.Context) (string, error) {
	return opaqueID(u.userID)
}

// JoinGrid is a function which can be called to join grids
type JoinGrid func(ctx context.Context, s *Grid) error
