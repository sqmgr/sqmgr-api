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
	"fmt"
	"time"
)

const maxPoolsPerMinute = 3
const maxPoolsPerDay = 10

// Action is a type of action that a user wants to take
type Action int

// Action constants
const (
	ActionCreatePool Action = iota
)

// ActionError provides a reason why a user cannot perform an action
type ActionError string

// Error returns the human-friendly error value
func (a ActionError) Error() string {
	return string(a)
}

// Can will return nil if the user can perform an action. If the user cannot, this
// method will return the error, which will be the reason why. If the error returned is
// an ActionError, it can be safely shown to the end user. Any other error should be interpreted
// as a 500.
func (m *Model) Can(ctx context.Context, action Action, u *User) error {
	switch action {
	// NOTE: we will probably want to do something a little more efficient in the future,
	// like having a true "leaky bucket" rate limiting or something similar.
	case ActionCreatePool:
		count, err := u.PoolsCreatedWithin(ctx, time.Minute)
		if err != nil {
			return err
		}

		if count >= maxPoolsPerMinute {
			return ActionError(fmt.Sprintf("You cannot create more than %d pools per minute", maxPoolsPerMinute))
		}

		count, err = u.PoolsCreatedWithin(ctx, time.Hour*24)
		if err != nil {
			return err
		}

		if count >= maxPoolsPerDay {
			return ActionError(fmt.Sprintf("You cannot create more than %d pools per day", maxPoolsPerDay))
		}

		return nil
	}

	return fmt.Errorf("unsupported action %d", action)
}
