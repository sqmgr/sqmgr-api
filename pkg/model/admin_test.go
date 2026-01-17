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
	"testing"

	"github.com/onsi/gomega"
)

func TestGetAdminStats(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create test users - auth0 user (counted as TotalUsers)
	auth0User, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(auth0User.ID).Should(gomega.BeNumerically(">", 0))

	// Create test user - sqmgr user (counted as GuestUsers)
	sqmgrUser, err := m.GetUser(ctx, IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(sqmgrUser.ID).Should(gomega.BeNumerically(">", 0))

	// Get initial stats
	initialStats, err := m.GetAdminStats(ctx)
	g.Expect(err).Should(gomega.Succeed())

	// Create an active pool
	activePool, err := m.NewPool(ctx, auth0User.ID, "Active Test Pool", GridTypeStd100, "password")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(activePool).ShouldNot(gomega.BeNil())

	// Create an archived pool
	archivedPool, err := m.NewPool(ctx, auth0User.ID, "Archived Test Pool", GridTypeStd25, "password")
	g.Expect(err).Should(gomega.Succeed())
	archivedPool.SetArchived(true)
	g.Expect(archivedPool.Save(ctx)).Should(gomega.Succeed())

	// Get updated stats
	stats, err := m.GetAdminStats(ctx)
	g.Expect(err).Should(gomega.Succeed())

	// Verify counts increased correctly
	g.Expect(stats.TotalPools).Should(gomega.Equal(initialStats.TotalPools + 2))
	g.Expect(stats.ActivePools).Should(gomega.Equal(initialStats.ActivePools + 1))
	g.Expect(stats.ArchivedPools).Should(gomega.Equal(initialStats.ArchivedPools + 1))
}

func TestGetAllPools(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a user for the pools
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	// Create multiple pools with distinct names
	poolNames := []string{
		"GetAllPools Test A " + randString(),
		"GetAllPools Test B " + randString(),
		"GetAllPools Test C " + randString(),
	}

	for _, name := range poolNames {
		_, err := m.NewPool(ctx, user.ID, name, GridTypeStd100, "password")
		g.Expect(err).Should(gomega.Succeed())
	}

	// Test pagination - get first 2 pools (ordered by id DESC)
	pools, err := m.GetAllPools(ctx, "", 0, 2)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(pools)).Should(gomega.Equal(2))

	// Verify returned fields are populated
	g.Expect(pools[0].Token).ShouldNot(gomega.BeEmpty())
	g.Expect(pools[0].Name).ShouldNot(gomega.BeEmpty())
	g.Expect(pools[0].GridType).ShouldNot(gomega.BeEmpty())
	g.Expect(pools[0].OwnerID).Should(gomega.BeNumerically(">", 0))
	g.Expect(pools[0].Created).ShouldNot(gomega.BeEmpty())

	// Test offset - skip first pool
	poolsWithOffset, err := m.GetAllPools(ctx, "", 1, 2)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(poolsWithOffset)).Should(gomega.Equal(2))

	// The first pool in offset results should be different from first pool without offset
	g.Expect(poolsWithOffset[0].Token).ShouldNot(gomega.Equal(pools[0].Token))
}

func TestGetAllPoolsWithSearch(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a user for the pools
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	// Create pools with distinct, searchable names
	uniquePrefix := "SearchTest" + randString()[:4]
	_, err = m.NewPool(ctx, user.ID, uniquePrefix+" Alpha Pool", GridTypeStd100, "password")
	g.Expect(err).Should(gomega.Succeed())

	_, err = m.NewPool(ctx, user.ID, uniquePrefix+" Beta Pool", GridTypeStd25, "password")
	g.Expect(err).Should(gomega.Succeed())

	_, err = m.NewPool(ctx, user.ID, "Different Name Pool", GridTypeStd50, "password")
	g.Expect(err).Should(gomega.Succeed())

	// Search for pools with unique prefix (case-insensitive)
	pools, err := m.GetAllPools(ctx, uniquePrefix, 0, 100)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(pools)).Should(gomega.Equal(2))

	// Verify all returned pools contain the search term
	for _, pool := range pools {
		g.Expect(pool.Name).Should(gomega.ContainSubstring(uniquePrefix))
	}

	// Search for partial match (case-insensitive ILIKE)
	poolsLower, err := m.GetAllPools(ctx, "alpha", 0, 100)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(poolsLower)).Should(gomega.BeNumerically(">=", 1))
}

func TestGetAllPoolsCount(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Get initial count
	initialCount, err := m.GetAllPoolsCount(ctx, "")
	g.Expect(err).Should(gomega.Succeed())

	// Create a user and some pools
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	uniquePrefix := "CountTest" + randString()[:4]
	_, err = m.NewPool(ctx, user.ID, uniquePrefix+" Pool 1", GridTypeStd100, "password")
	g.Expect(err).Should(gomega.Succeed())

	_, err = m.NewPool(ctx, user.ID, uniquePrefix+" Pool 2", GridTypeStd25, "password")
	g.Expect(err).Should(gomega.Succeed())

	_, err = m.NewPool(ctx, user.ID, "Other Pool", GridTypeStd50, "password")
	g.Expect(err).Should(gomega.Succeed())

	// Test count without search
	totalCount, err := m.GetAllPoolsCount(ctx, "")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(totalCount).Should(gomega.Equal(initialCount + 3))

	// Test count with search filter
	filteredCount, err := m.GetAllPoolsCount(ctx, uniquePrefix)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(filteredCount).Should(gomega.Equal(int64(2)))
}
