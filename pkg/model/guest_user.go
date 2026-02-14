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
	"context"
	"database/sql"
	"errors"
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

// IsGuestUserExpired checks whether a guest user identified by store/storeID has expired.
// Returns true if the guest user is found and expired, false if not found or not expired.
func (m *Model) IsGuestUserExpired(ctx context.Context, store UserStore, storeID string) (bool, error) {
	const query = `SELECT expires FROM guest_users WHERE store = $1 AND store_id = $2`
	var expires time.Time
	if err := m.DB.QueryRowContext(ctx, query, store, storeID).Scan(&expires); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return time.Now().After(expires), nil
}
