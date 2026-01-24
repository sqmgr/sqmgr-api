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
	initialStats, err := m.GetAdminStats(ctx, "all")
	g.Expect(err).Should(gomega.Succeed())

	// Create an active pool
	activePool, err := m.NewPool(ctx, auth0User.ID, "Active Test Pool", GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(activePool).ShouldNot(gomega.BeNil())

	// Create an archived pool
	archivedPool, err := m.NewPool(ctx, auth0User.ID, "Archived Test Pool", GridTypeStd25, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	archivedPool.SetArchived(true)
	g.Expect(archivedPool.Save(ctx)).Should(gomega.Succeed())

	// Get updated stats
	stats, err := m.GetAdminStats(ctx, "all")
	g.Expect(err).Should(gomega.Succeed())

	// Verify counts increased correctly
	g.Expect(stats.TotalPools).Should(gomega.Equal(initialStats.TotalPools + 2))
	g.Expect(stats.ActivePools).Should(gomega.Equal(initialStats.ActivePools + 1))
	g.Expect(stats.ArchivedPools).Should(gomega.Equal(initialStats.ArchivedPools + 1))
}

func TestGetAdminStatsWithPeriod(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a test user
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	// Create a test pool
	pool, err := m.NewPool(ctx, user.ID, "Period Test Pool "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(pool).ShouldNot(gomega.BeNil())

	// Test "all" period - should include the new pool
	allStats, err := m.GetAdminStats(ctx, "all")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(allStats.TotalPools).Should(gomega.BeNumerically(">", 0))

	// Test "24h" period - should include the just-created pool
	dayStats, err := m.GetAdminStats(ctx, "24h")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(dayStats.TotalPools).Should(gomega.BeNumerically(">", 0))

	// Test invalid period - should default to "all" behavior (no time filter)
	invalidStats, err := m.GetAdminStats(ctx, "invalid")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(invalidStats.TotalPools).Should(gomega.Equal(allStats.TotalPools))

	// Test empty period - should default to "all" behavior
	emptyStats, err := m.GetAdminStats(ctx, "")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(emptyStats.TotalPools).Should(gomega.Equal(allStats.TotalPools))
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
		_, err := m.NewPool(ctx, user.ID, name, GridTypeStd100, "password", NumberSetConfigStandard)
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
	// ClaimedCount should be 0 for new pools with no claimed squares
	g.Expect(pools[0].ClaimedCount).Should(gomega.BeNumerically(">=", 0))

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
	_, err = m.NewPool(ctx, user.ID, uniquePrefix+" Alpha Pool", GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	_, err = m.NewPool(ctx, user.ID, uniquePrefix+" Beta Pool", GridTypeStd25, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	_, err = m.NewPool(ctx, user.ID, "Different Name Pool", GridTypeStd50, "password", NumberSetConfigStandard)
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
	_, err = m.NewPool(ctx, user.ID, uniquePrefix+" Pool 1", GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	_, err = m.NewPool(ctx, user.ID, uniquePrefix+" Pool 2", GridTypeStd25, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	_, err = m.NewPool(ctx, user.ID, "Other Pool", GridTypeStd50, "password", NumberSetConfigStandard)
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

func TestGetAllPoolsClaimedCount(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a user and pool
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	uniqueName := "ClaimedCountTest " + randString()
	pool, err := m.NewPool(ctx, user.ID, uniqueName, GridTypeStd25, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	// Get the pool via admin query - should have 0 claimed squares initially
	pools, err := m.GetAllPools(ctx, uniqueName, 0, 1)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(pools)).Should(gomega.Equal(1))
	g.Expect(pools[0].ClaimedCount).Should(gomega.Equal(int64(0)))

	// Claim a square
	squares, err := pool.Squares()
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(squares)).Should(gomega.Equal(25))

	// Get first available square from map
	var square *PoolSquare
	for _, sq := range squares {
		square = sq
		break
	}
	g.Expect(square).ShouldNot(gomega.BeNil())

	square.claimant = "Test Claimant"
	square.State = PoolSquareStateClaimed
	square.SetUserID(user.ID)
	err = square.Save(ctx, m.DB, true, PoolSquareLog{
		Note:       "Test claim",
		RemoteAddr: "127.0.0.1",
	})
	g.Expect(err).Should(gomega.Succeed())

	// Verify claimed count increased
	pools, err = m.GetAllPools(ctx, uniqueName, 0, 1)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(pools)).Should(gomega.Equal(1))
	g.Expect(pools[0].ClaimedCount).Should(gomega.Equal(int64(1)))
}

func TestGetAllPoolsOwnerInfo(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create an auth0 user with email
	auth0User, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	testEmail := "test-" + randString()[:8] + "@example.com"
	err = auth0User.SetEmail(ctx, testEmail)
	g.Expect(err).Should(gomega.Succeed())

	// Create a guest user (no email)
	guestUser, err := m.GetUser(ctx, IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	// Create pools for each user
	uniquePrefix := "OwnerInfo" + randString()[:4]
	auth0Pool, err := m.NewPool(ctx, auth0User.ID, uniquePrefix+" Auth0 Pool", GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(auth0Pool).ShouldNot(gomega.BeNil())

	guestPool, err := m.NewPool(ctx, guestUser.ID, uniquePrefix+" Guest Pool", GridTypeStd25, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(guestPool).ShouldNot(gomega.BeNil())

	// Get pools with owner info
	pools, err := m.GetAllPools(ctx, uniquePrefix, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(pools)).Should(gomega.Equal(2))

	// Find each pool and verify owner info
	var foundAuth0Pool, foundGuestPool *AdminPool
	for _, pool := range pools {
		if pool.OwnerID == auth0User.ID {
			foundAuth0Pool = pool
		}
		if pool.OwnerID == guestUser.ID {
			foundGuestPool = pool
		}
	}

	// Verify auth0 user's pool has email and correct store
	g.Expect(foundAuth0Pool).ShouldNot(gomega.BeNil())
	g.Expect(foundAuth0Pool.OwnerStore).Should(gomega.Equal("auth0"))
	g.Expect(foundAuth0Pool.OwnerEmail).ShouldNot(gomega.BeNil())
	g.Expect(*foundAuth0Pool.OwnerEmail).Should(gomega.Equal(testEmail))

	// Verify guest user's pool has nil email and correct store
	g.Expect(foundGuestPool).ShouldNot(gomega.BeNil())
	g.Expect(foundGuestPool.OwnerStore).Should(gomega.Equal("sqmgr"))
	g.Expect(foundGuestPool.OwnerEmail).Should(gomega.BeNil())
}

func TestGetUserStats(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a user
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	// Create another user to join pools
	otherUser, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	// Get initial stats (should be all zeros for new user)
	initialStats, err := m.GetUserStats(ctx, user.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(initialStats.PoolsCreated).Should(gomega.Equal(int64(0)))
	g.Expect(initialStats.PoolsJoined).Should(gomega.Equal(int64(0)))
	g.Expect(initialStats.ActivePools).Should(gomega.Equal(int64(0)))
	g.Expect(initialStats.ArchivedPools).Should(gomega.Equal(int64(0)))

	// Create an active pool
	activePool, err := m.NewPool(ctx, user.ID, "User Stats Active Pool "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(activePool).ShouldNot(gomega.BeNil())

	// Create an archived pool
	archivedPool, err := m.NewPool(ctx, user.ID, "User Stats Archived Pool "+randString(), GridTypeStd25, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	archivedPool.SetArchived(true)
	g.Expect(archivedPool.Save(ctx)).Should(gomega.Succeed())

	// Create a pool owned by another user and join it
	otherPool, err := m.NewPool(ctx, otherUser.ID, "Other User Pool "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	err = user.JoinPool(ctx, otherPool)
	g.Expect(err).Should(gomega.Succeed())

	// Get updated stats
	stats, err := m.GetUserStats(ctx, user.ID)
	g.Expect(err).Should(gomega.Succeed())

	g.Expect(stats.PoolsCreated).Should(gomega.Equal(int64(2)))
	g.Expect(stats.PoolsJoined).Should(gomega.Equal(int64(1)))
	g.Expect(stats.ActivePools).Should(gomega.Equal(int64(1)))
	g.Expect(stats.ArchivedPools).Should(gomega.Equal(int64(1)))
}

func TestGetPoolsByUserID(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a user
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	// Create some pools
	pool1, err := m.NewPool(ctx, user.ID, "User Pools Test A "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(pool1).ShouldNot(gomega.BeNil())

	pool2, err := m.NewPool(ctx, user.ID, "User Pools Test B "+randString(), GridTypeStd25, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(pool2).ShouldNot(gomega.BeNil())

	archivedPool, err := m.NewPool(ctx, user.ID, "User Pools Test Archived "+randString(), GridTypeStd50, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	archivedPool.SetArchived(true)
	g.Expect(archivedPool.Save(ctx)).Should(gomega.Succeed())

	// Get active pools only (default)
	activePools, err := m.GetPoolsByUserID(ctx, user.ID, false, 0, 100)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(activePools)).Should(gomega.Equal(2))

	// Verify all returned pools are not archived
	for _, pool := range activePools {
		g.Expect(pool.Archived).Should(gomega.BeFalse())
		g.Expect(pool.OwnerID).Should(gomega.Equal(user.ID))
	}

	// Get all pools including archived
	allPools, err := m.GetPoolsByUserID(ctx, user.ID, true, 0, 100)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(allPools)).Should(gomega.Equal(3))

	// Verify owner info is populated
	g.Expect(allPools[0].OwnerStore).Should(gomega.Equal("auth0"))
}

func TestGetPoolsByUserID_Pagination(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a user
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	// Create 3 pools
	for i := 0; i < 3; i++ {
		_, err := m.NewPool(ctx, user.ID, "Pagination Test Pool "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
		g.Expect(err).Should(gomega.Succeed())
	}

	// Get first 2 pools
	firstPage, err := m.GetPoolsByUserID(ctx, user.ID, true, 0, 2)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(firstPage)).Should(gomega.Equal(2))

	// Get next page with offset
	secondPage, err := m.GetPoolsByUserID(ctx, user.ID, true, 2, 2)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(secondPage)).Should(gomega.Equal(1))

	// Verify no overlap
	g.Expect(secondPage[0].Token).ShouldNot(gomega.Equal(firstPage[0].Token))
	g.Expect(secondPage[0].Token).ShouldNot(gomega.Equal(firstPage[1].Token))
}

func TestGetPoolsByUserIDCount(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a user
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	// Initial count should be 0
	initialCount, err := m.GetPoolsByUserIDCount(ctx, user.ID, true)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(initialCount).Should(gomega.Equal(int64(0)))

	// Create an active pool
	_, err = m.NewPool(ctx, user.ID, "Count Test Active "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	// Create an archived pool
	archivedPool, err := m.NewPool(ctx, user.ID, "Count Test Archived "+randString(), GridTypeStd25, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	archivedPool.SetArchived(true)
	g.Expect(archivedPool.Save(ctx)).Should(gomega.Succeed())

	// Count active only
	activeCount, err := m.GetPoolsByUserIDCount(ctx, user.ID, false)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(activeCount).Should(gomega.Equal(int64(1)))

	// Count all including archived
	totalCount, err := m.GetPoolsByUserIDCount(ctx, user.ID, true)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(totalCount).Should(gomega.Equal(int64(2)))
}

func TestGetAllUsers(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create test users with emails
	email1 := "testuser1-" + randString()[:8] + "@example.com"
	user1, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = user1.SetEmail(ctx, email1)
	g.Expect(err).Should(gomega.Succeed())

	email2 := "testuser2-" + randString()[:8] + "@example.com"
	user2, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = user2.SetEmail(ctx, email2)
	g.Expect(err).Should(gomega.Succeed())

	// Create a pool for user1 to verify pools_owned count
	_, err = m.NewPool(ctx, user1.ID, "User1 Pool "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	// Test pagination
	users, err := m.GetAllUsers(ctx, "", 0, 2, "", "")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(users)).Should(gomega.Equal(2))

	// Verify returned fields are populated
	g.Expect(users[0].ID).Should(gomega.BeNumerically(">", 0))
	g.Expect(users[0].Store).ShouldNot(gomega.BeEmpty())
	g.Expect(users[0].Created).ShouldNot(gomega.BeEmpty())
}

func TestGetAllUsersWithSearch(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create users with searchable emails
	uniquePrefix := "searchable" + randString()[:4]
	email1 := uniquePrefix + "-alpha@example.com"
	user1, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = user1.SetEmail(ctx, email1)
	g.Expect(err).Should(gomega.Succeed())

	email2 := uniquePrefix + "-beta@example.com"
	user2, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = user2.SetEmail(ctx, email2)
	g.Expect(err).Should(gomega.Succeed())

	// Search for users with unique prefix
	users, err := m.GetAllUsers(ctx, uniquePrefix, 0, 100, "", "")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(users)).Should(gomega.Equal(2))

	// Verify all returned users contain the search term in email
	for _, user := range users {
		g.Expect(*user.Email).Should(gomega.ContainSubstring(uniquePrefix))
	}
}

func TestGetAllUsersCount(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Get initial count
	initialCount, err := m.GetAllUsersCount(ctx, "")
	g.Expect(err).Should(gomega.Succeed())

	// Create new users
	uniquePrefix := "counttest" + randString()[:4]
	email1 := uniquePrefix + "-one@example.com"
	user1, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = user1.SetEmail(ctx, email1)
	g.Expect(err).Should(gomega.Succeed())

	email2 := uniquePrefix + "-two@example.com"
	user2, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = user2.SetEmail(ctx, email2)
	g.Expect(err).Should(gomega.Succeed())

	// Test count without search
	totalCount, err := m.GetAllUsersCount(ctx, "")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(totalCount).Should(gomega.Equal(initialCount + 2))

	// Test count with search filter
	filteredCount, err := m.GetAllUsersCount(ctx, uniquePrefix)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(filteredCount).Should(gomega.Equal(int64(2)))
}

func TestGetAllUsersPoolCounts(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create a user with a searchable email
	uniquePrefix := "poolcount" + randString()[:4]
	email := uniquePrefix + "@example.com"
	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = user.SetEmail(ctx, email)
	g.Expect(err).Should(gomega.Succeed())

	// Create another user whose pool we'll join
	otherUser, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	// Create pools owned by the user
	_, err = m.NewPool(ctx, user.ID, "Owned Pool 1 "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	_, err = m.NewPool(ctx, user.ID, "Owned Pool 2 "+randString(), GridTypeStd25, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	// Create a pool owned by another user and have the test user join it
	otherPool, err := m.NewPool(ctx, otherUser.ID, "Other Pool "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	err = user.JoinPool(ctx, otherPool)
	g.Expect(err).Should(gomega.Succeed())

	// Fetch the user via GetAllUsers
	users, err := m.GetAllUsers(ctx, uniquePrefix, 0, 1, "", "")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(users)).Should(gomega.Equal(1))

	// Verify pool counts
	g.Expect(users[0].PoolsOwned).Should(gomega.Equal(int64(2)))
	g.Expect(users[0].PoolsJoined).Should(gomega.Equal(int64(1)))
}

func TestGetAllUsersSorting(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create users with different pool counts
	uniquePrefix := "sorttest" + randString()[:4]

	// User 1: 2 pools owned
	email1 := uniquePrefix + "-user1@example.com"
	user1, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = user1.SetEmail(ctx, email1)
	g.Expect(err).Should(gomega.Succeed())
	_, err = m.NewPool(ctx, user1.ID, "Sort Pool 1A "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	_, err = m.NewPool(ctx, user1.ID, "Sort Pool 1B "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	// User 2: 1 pool owned
	email2 := uniquePrefix + "-user2@example.com"
	user2, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = user2.SetEmail(ctx, email2)
	g.Expect(err).Should(gomega.Succeed())
	_, err = m.NewPool(ctx, user2.ID, "Sort Pool 2A "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	// User 3: 3 pools owned
	email3 := uniquePrefix + "-user3@example.com"
	user3, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = user3.SetEmail(ctx, email3)
	g.Expect(err).Should(gomega.Succeed())
	_, err = m.NewPool(ctx, user3.ID, "Sort Pool 3A "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	_, err = m.NewPool(ctx, user3.ID, "Sort Pool 3B "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	_, err = m.NewPool(ctx, user3.ID, "Sort Pool 3C "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	// Test sorting by poolsOwned descending
	usersDesc, err := m.GetAllUsers(ctx, uniquePrefix, 0, 10, "poolsOwned", "desc")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(usersDesc)).Should(gomega.Equal(3))
	g.Expect(usersDesc[0].PoolsOwned).Should(gomega.Equal(int64(3)))
	g.Expect(usersDesc[1].PoolsOwned).Should(gomega.Equal(int64(2)))
	g.Expect(usersDesc[2].PoolsOwned).Should(gomega.Equal(int64(1)))

	// Test sorting by poolsOwned ascending
	usersAsc, err := m.GetAllUsers(ctx, uniquePrefix, 0, 10, "poolsOwned", "asc")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(usersAsc)).Should(gomega.Equal(3))
	g.Expect(usersAsc[0].PoolsOwned).Should(gomega.Equal(int64(1)))
	g.Expect(usersAsc[1].PoolsOwned).Should(gomega.Equal(int64(2)))
	g.Expect(usersAsc[2].PoolsOwned).Should(gomega.Equal(int64(3)))

	// Test sorting by created descending (most recent first - user3 was created last)
	usersCreatedDesc, err := m.GetAllUsers(ctx, uniquePrefix, 0, 10, "created", "desc")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(usersCreatedDesc)).Should(gomega.Equal(3))
	g.Expect(usersCreatedDesc[0].ID).Should(gomega.Equal(user3.ID))
	g.Expect(usersCreatedDesc[2].ID).Should(gomega.Equal(user1.ID))

	// Test invalid sort column defaults to id
	usersInvalidSort, err := m.GetAllUsers(ctx, uniquePrefix, 0, 10, "invalid", "desc")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(usersInvalidSort)).Should(gomega.Equal(3))
}

func TestGetAllUsersExcludesGuestUsers(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	uniquePrefix := "guestexclude" + randString()[:4]

	// Create a registered user (auth0)
	registeredEmail := uniquePrefix + "-registered@example.com"
	registeredUser, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = registeredUser.SetEmail(ctx, registeredEmail)
	g.Expect(err).Should(gomega.Succeed())

	// Create a guest user (sqmgr) - this should NOT appear in the list
	_, err = m.GetUser(ctx, IssuerSqMGR, uniquePrefix+"-guest")
	g.Expect(err).Should(gomega.Succeed())

	// Get initial count of registered users
	initialCount, err := m.GetAllUsersCount(ctx, "")
	g.Expect(err).Should(gomega.Succeed())

	// The count should only include registered users
	// Search for users with our unique prefix - should only return the registered user
	users, err := m.GetAllUsers(ctx, uniquePrefix, 0, 100, "", "")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(users)).Should(gomega.Equal(1))
	g.Expect(*users[0].Email).Should(gomega.Equal(registeredEmail))
	g.Expect(users[0].Store).Should(gomega.Equal(UserStoreAuth0))

	// Verify count with search only includes registered user
	filteredCount, err := m.GetAllUsersCount(ctx, uniquePrefix)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(filteredCount).Should(gomega.Equal(int64(1)))

	// Create another registered user to verify count increases
	registeredEmail2 := uniquePrefix + "-registered2@example.com"
	registeredUser2, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())
	err = registeredUser2.SetEmail(ctx, registeredEmail2)
	g.Expect(err).Should(gomega.Succeed())

	// Count should increase by 1
	newCount, err := m.GetAllUsersCount(ctx, "")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(newCount).Should(gomega.Equal(initialCount + 1))
}
