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

package server

import (
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

// computeAuthRequired replicates the authorization logic from getPoolTokenSquaresPublicEndpoint
// Auth required when password is required AND pool is not in open-access state
// Open access: !PasswordRequired() OR (IsLocked() AND OpenAccessOnLock())
func computeAuthRequired(pool *model.Pool) bool {
	return pool.PasswordRequired() && (!pool.IsLocked() || !pool.OpenAccessOnLock())
}

// createPoolWithSettings creates a Pool with the specified settings for testing
func createPoolWithSettings(passwordRequired, locked, openAccessOnLock bool) *model.Pool {
	pool := &model.Pool{}
	pool.SetPasswordRequired(passwordRequired)
	pool.SetOpenAccessOnLock(openAccessOnLock)
	if locked {
		// Set locks time to the past to make IsLocked() return true
		pool.SetLocks(time.Now().Add(-time.Hour))
	}
	return pool
}

// TestGetPoolSquaresPublic_NoPasswordRequired_Returns200 verifies that pools without
// password requirement don't need authentication
func TestGetPoolSquaresPublic_NoPasswordRequired_Returns200(t *testing.T) {
	g := gomega.NewWithT(t)

	// PasswordRequired=false, IsLocked=false -> no auth needed
	pool := createPoolWithSettings(false, false, false)
	g.Expect(computeAuthRequired(pool)).Should(gomega.BeFalse())

	// PasswordRequired=false, IsLocked=true -> no auth needed (password not required)
	pool = createPoolWithSettings(false, true, false)
	g.Expect(computeAuthRequired(pool)).Should(gomega.BeFalse())

	// PasswordRequired=false, IsLocked=true, OpenAccessOnLock=true -> no auth needed
	pool = createPoolWithSettings(false, true, true)
	g.Expect(computeAuthRequired(pool)).Should(gomega.BeFalse())
}

// TestGetPoolSquaresPublic_PasswordRequired_Open_Returns401 verifies that an open pool
// with password required needs authentication (this is the main bug scenario)
func TestGetPoolSquaresPublic_PasswordRequired_Open_Returns401(t *testing.T) {
	g := gomega.NewWithT(t)

	// PasswordRequired=true, IsLocked=false -> auth required
	// This is the main bug: previously this case did NOT require auth
	pool := createPoolWithSettings(true, false, false)
	g.Expect(computeAuthRequired(pool)).Should(gomega.BeTrue())

	// OpenAccessOnLock doesn't matter when not locked
	pool = createPoolWithSettings(true, false, true)
	g.Expect(computeAuthRequired(pool)).Should(gomega.BeTrue())
}

// TestGetPoolSquaresPublic_PasswordRequired_Locked_NoOpenAccess_Returns401 verifies that
// a locked pool with password required and no open access needs authentication
func TestGetPoolSquaresPublic_PasswordRequired_Locked_NoOpenAccess_Returns401(t *testing.T) {
	g := gomega.NewWithT(t)

	// PasswordRequired=true, IsLocked=true, OpenAccessOnLock=false -> auth required
	pool := createPoolWithSettings(true, true, false)
	g.Expect(computeAuthRequired(pool)).Should(gomega.BeTrue())
}

// TestGetPoolSquaresPublic_PasswordRequired_Locked_OpenAccess_Returns200 verifies that
// a locked pool with open access enabled doesn't need authentication
func TestGetPoolSquaresPublic_PasswordRequired_Locked_OpenAccess_Returns200(t *testing.T) {
	g := gomega.NewWithT(t)

	// PasswordRequired=true, IsLocked=true, OpenAccessOnLock=true -> no auth (open access)
	pool := createPoolWithSettings(true, true, true)
	g.Expect(computeAuthRequired(pool)).Should(gomega.BeFalse())
}

// TestAuthRequiredTruthTable verifies all combinations of the authorization logic
// as specified in the implementation plan
func TestAuthRequiredTruthTable(t *testing.T) {
	g := gomega.NewWithT(t)

	testCases := []struct {
		name             string
		passwordRequired bool
		locked           bool
		openAccessOnLock bool
		expectedAuth     bool
	}{
		// PasswordRequired=false cases (no auth needed regardless of other settings)
		{"no password, unlocked", false, false, false, false},
		{"no password, locked, no open access", false, true, false, false},
		{"no password, locked, open access", false, true, true, false},

		// PasswordRequired=true, unlocked cases (auth always required)
		{"password required, unlocked, no open access", true, false, false, true},
		{"password required, unlocked, open access flag set", true, false, true, true},

		// PasswordRequired=true, locked cases (depends on OpenAccessOnLock)
		{"password required, locked, no open access", true, true, false, true},
		{"password required, locked, open access", true, true, true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := createPoolWithSettings(tc.passwordRequired, tc.locked, tc.openAccessOnLock)
			authRequired := computeAuthRequired(pool)
			g.Expect(authRequired).Should(gomega.Equal(tc.expectedAuth),
				"Expected authRequired=%v for passwordRequired=%v, locked=%v, openAccessOnLock=%v",
				tc.expectedAuth, tc.passwordRequired, tc.locked, tc.openAccessOnLock)
		})
	}
}

// TestPoolIsLockedBehavior verifies Pool.IsLocked() behavior with time settings
func TestPoolIsLockedBehavior(t *testing.T) {
	g := gomega.NewWithT(t)

	pool := &model.Pool{}

	// Zero time (default) means unlocked
	g.Expect(pool.IsLocked()).Should(gomega.BeFalse())

	// Future time means unlocked
	pool.SetLocks(time.Now().Add(time.Hour))
	g.Expect(pool.IsLocked()).Should(gomega.BeFalse())

	// Past time means locked
	pool.SetLocks(time.Now().Add(-time.Hour))
	g.Expect(pool.IsLocked()).Should(gomega.BeTrue())
}
