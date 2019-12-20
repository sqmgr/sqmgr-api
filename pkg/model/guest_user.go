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
	"time"
)

// GuestUser represents a user in our system who has not registered
type GuestUser struct {
	Store      UserStore
	StoreID    string
	Expires    time.Time
	RemoteAddr string
	Created    time.Time
}

// NewGuestUser will create and return a guest user. Primarily used for fraud prevention
func (m *Model) NewGuestUser(ctx context.Context, store UserStore, storeID string, expiresAt time.Time, remoteAddr string) (*GuestUser, error) {
	const query = `
INSERT INTO guest_users (store, store_id, expires, remote_addr)
VALUES ($1, $2, $3, $4)
RETURNING store, store_id, expires, remote_addr, created`

	row := m.DB.QueryRowContext(ctx, query, store, storeID, expiresAt, ipFromRemoteAddr(remoteAddr))
	var guestUser GuestUser
	if err := row.Scan(&guestUser.Store, &guestUser.StoreID, &guestUser.Expires, &guestUser.RemoteAddr, &guestUser.Created); err != nil {
		return nil, err
	}

	return &guestUser, nil
}
