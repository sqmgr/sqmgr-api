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

	panic(fmt.Sprintf("unsupported action %d", action))
}
