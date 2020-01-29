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
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/onsi/gomega"
	"github.com/synacor/argon2id"
)

var db *sql.DB

func getDB() *sql.DB {
	if db != nil {
		return db
	}

	dsn := "sslmode=disable user=postgres database=integration"
	if env := os.Getenv("SQMGR_CONF_DSN"); env != "" {
		dsn = env
	}

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}
	if err := db.Ping(); err != nil {
		panic(err)
	}

	return db
}

func TestNewToken(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	token1, err := m.NewToken()
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(token1).ShouldNot(gomega.Equal(""))

	token2, err := m.NewToken()
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(token2).ShouldNot(gomega.Equal(token1))
}

func TestPool(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	user, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	pool, err := m.NewPool(context.Background(), user.ID, "My Pool", GridTypeStd100, "my-other-unique-password")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(pool).ShouldNot(gomega.BeNil())

	grid, err := pool.DefaultGrid(context.Background())
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(grid.Name()).Should(gomega.Equal("Away Team vs. Home Team"))

	g.Expect(pool.CheckID()).Should(gomega.Equal(0))
	g.Expect(pool.Archived()).Should(gomega.BeFalse())
	g.Expect(pool.id).Should(gomega.BeNumerically(">", 0))
	g.Expect(pool.userID).Should(gomega.Equal(user.ID))
	g.Expect(pool.token).Should(gomega.MatchRegexp(`^[A-Za-z0-9_-]{8}\z`))
	g.Expect(pool.name).Should(gomega.Equal("My Pool"))
	g.Expect(pool.passwordHash).ShouldNot(gomega.Equal("my-other-unique-password"))
	g.Expect(argon2id.Compare(pool.passwordHash, "my-other-unique-password")).Should(gomega.Succeed())

	originalPasswordHash := pool.passwordHash
	g.Expect(pool.SetPassword("my-other-unique-password")).Should(gomega.Succeed())
	g.Expect(pool.passwordHash).ShouldNot(gomega.Equal(originalPasswordHash))

	pool.name = "Different Name"
	pool.gridType = GridTypeStd25

	pool.IncrementCheckID()
	pool.SetArchived(true)

	err = pool.Save(context.Background())
	g.Expect(err).Should(gomega.Succeed())

	pool2, err := m.PoolByID(pool.id)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(pool2).ShouldNot(gomega.BeNil())

	g.Expect(pool2.name).Should(gomega.Equal("Different Name"))
	g.Expect(pool2.gridType).Should(gomega.Equal(GridTypeStd25))
	g.Expect(pool2.CheckID()).Should(gomega.Equal(1))
	g.Expect(pool2.Archived()).Should(gomega.BeTrue())

	pool3, err := m.PoolByToken(context.Background(), pool2.token)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(pool3).ShouldNot(gomega.BeNil())
	g.Expect(pool3).Should(gomega.Equal(pool2))
}

func TestNewGridInvalidGridType(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	m := New(nil)
	s, err := m.NewPool(context.Background(), 1, "my name", GridType("invalid"), "my password")
	g.Expect(s).Should(gomega.BeNil())
	g.Expect(err).Should(gomega.MatchError(ErrInvalidGridType))
}

func TestGridCollections(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	user, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	pool, err := m.NewPool(context.Background(), user.ID, "Test for Collection", GridTypeStd100, "my-other-unique-password")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(pool).ShouldNot(gomega.BeNil())

	user2, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	collection, err := m.PoolsJoinedByUserID(context.Background(), user.ID, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(0))

	collection, err = m.PoolsJoinedByUserID(context.Background(), user2.ID, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(0))

	g.Expect(user2.JoinPool(context.Background(), pool)).Should(gomega.Succeed())
	collection, err = m.PoolsJoinedByUserID(context.Background(), user2.ID, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(1))

	collection, err = m.PoolsOwnedByUserID(context.Background(), user.ID, false, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(1))

	collection, err = m.PoolsOwnedByUserID(context.Background(), user2.ID, false, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(0))
}

func TestGridCollectionPagination(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	user1, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	user2, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	for i := 0; i < 30; i++ {
		pool, err := m.NewPool(context.Background(), user1.ID, randString(), GridTypeStd100, "my-other-unique-password")
		g.Expect(err).Should(gomega.Succeed())

		if i < 20 {
			g.Expect(user2.JoinPool(context.Background(), pool)).Should(gomega.Succeed())
		}
	}

	count, err := m.PoolsOwnedByUserIDCount(context.Background(), user1.ID, false)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(30)))

	count, err = m.PoolsOwnedByUserIDCount(context.Background(), user2.ID, false)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(0)))

	count, err = m.PoolsJoinedByUserIDCount(context.Background(), user1.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(0)))

	count, err = m.PoolsJoinedByUserIDCount(context.Background(), user2.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(20)))

}

func TestAccessors(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	created := time.Now()
	modified := time.Now()

	s := &Pool{
		created:  created,
		modified: modified,
	}

	testMaxLength(g, s.Name, s.SetName, NameMaxLength, "name")

	s.id = 12345
	g.Expect(s.ID()).Should(gomega.Equal(int64(12345)))

	s.token = "my-token"
	g.Expect(s.Token()).Should(gomega.Equal("my-token"))

	s.SetGridType(GridTypeStd25)
	g.Expect(s.GridType()).Should(gomega.Equal(GridTypeStd25))
	g.Expect(s.Created()).Should(gomega.Equal(created))
	g.Expect(s.Modified()).Should(gomega.Equal(modified))

	var err error
	s.passwordHash, err = argon2id.DefaultHashPassword("test")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(s.PasswordIsValid("test")).Should(gomega.BeTrue())
	g.Expect(s.PasswordIsValid("no-match")).Should(gomega.BeFalse())
}

func TestGridSquares(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	user, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	pool, err := m.NewPool(context.Background(), user.ID, "Test Pool", GridTypeStd25, "a password")
	g.Expect(err).Should(gomega.Succeed())

	squares, err := pool.Squares()
	g.Expect(err).Should(gomega.Succeed())

	g.Expect(len(squares)).Should(gomega.Equal(25))

	square := squares[15]
	g.Expect(square.SquareID).Should(gomega.Equal(15))
	g.Expect(square.claimant).Should(gomega.Equal(""))

	square.claimant = "Test User"
	square.State = PoolSquareStateClaimed
	square.SetUserID(user.ID)
	err = square.Save(context.Background(), m.DB, true, PoolSquareLog{
		Note:       "Test Note",
		RemoteAddr: "127.0.0.1",
	})
	g.Expect(err).Should(gomega.Succeed())

	pool.squares = nil // force a fresh fetch
	squares, err = pool.Squares()
	g.Expect(err).Should(gomega.Succeed())

	square = squares[15]
	g.Expect(square.claimant).Should(gomega.Equal("Test User"))

	err = square.Save(context.Background(), m.DB, true, PoolSquareLog{
		Note: "A new note",
	})
	g.Expect(err).Should(gomega.Succeed())

	squares2, err := pool.SquareBySquareID(15)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(squares2.ID).Should(gomega.Equal(square.ID))

	g.Expect(square.LoadLogs(context.Background())).Should(gomega.Succeed())

	g.Expect(len(square.Logs)).Should(gomega.Equal(2))

	g.Expect(square.Logs[0].SquareID()).Should(gomega.Equal(15))
	g.Expect(square.Logs[0].Note).Should(gomega.Equal("A new note"))
	g.Expect(square.Logs[0].RemoteAddr).Should(gomega.Equal(""))
	g.Expect(square.Logs[0].userID).Should(gomega.Equal(user.ID))
	g.Expect(square.Logs[0].Claimant()).Should(gomega.Equal("Test User"))

	g.Expect(square.Logs[1].Note).Should(gomega.Equal("Test Note"))
	g.Expect(square.Logs[1].RemoteAddr).Should(gomega.Equal("127.0.0.1"))
	g.Expect(square.Logs[1].userID).Should(gomega.Equal(user.ID))
	g.Expect(square.Logs[1].Claimant()).Should(gomega.Equal("Test User"))

	logs, err := pool.Logs(context.Background(), 0, 1000)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(logs)).Should(gomega.BeNumerically(">", 0))

	count, err := pool.LogsCount(context.Background())
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(len(logs))))

	square.claimant = "New User"
	err = square.Save(context.Background(), m.DB, false, PoolSquareLog{
		Note: "",
	})
	g.Expect(err).Should(gomega.Equal(ErrSquareAlreadyClaimed))
}

func TestLocks(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	p := &Pool{}
	g.Expect(p.IsLocked()).Should(gomega.BeFalse())
	g.Expect(p.Locks()).Should(gomega.Equal(time.Time{}))
	then := time.Now().Add(time.Minute)
	p.SetLocks(then)
	g.Expect(p.Locks()).Should(gomega.Equal(then))
	g.Expect(p.IsLocked()).Should(gomega.BeFalse())

	p.SetLocks(time.Now().Add(time.Second * -1))
	g.Expect(p.IsLocked()).Should(gomega.BeTrue())
}

func TestArchiving(t *testing.T) {
	ensureIntegration(t)

	g := gomega.NewWithT(t)
	m := New(getDB())

	user, err := m.GetUser(context.Background(), IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	for i := 0; i < 3; i++ {
		pool, err := m.NewPool(context.Background(), user.ID, "Test", GridTypeStd25, "a-password")
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(pool).ShouldNot(gomega.BeNil())
	}

	pools, err := user.PoolsOwnedByUserID(context.Background(), user.ID, false, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(pools)).Should(gomega.Equal(3))

	count, err := user.PoolsOwnedByUserIDCount(context.Background(), user.ID, false)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(3)))

	// set one of the pools as archived

	pools[0].SetArchived(true)
	g.Expect(pools[0].Save(context.Background())).Should(gomega.Succeed())

	//

	pools, err = user.PoolsOwnedByUserID(context.Background(), user.ID, false, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(pools)).Should(gomega.Equal(2))

	count, err = user.PoolsOwnedByUserIDCount(context.Background(), user.ID, false)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(2)))

	//

	pools, err = user.PoolsOwnedByUserID(context.Background(), user.ID, true, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(pools)).Should(gomega.Equal(3))

	count, err = user.PoolsOwnedByUserIDCount(context.Background(), user.ID, true)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(3)))
}

func TestNumberOfSquares(t *testing.T) {
	g := gomega.NewWithT(t)

	p := Pool{gridType: GridTypeStd25}
	g.Expect(p.NumberOfSquares()).Should(gomega.Equal(25))

	p.gridType = GridTypeStd100
	g.Expect(p.NumberOfSquares()).Should(gomega.Equal(100))

	p.gridType = GridTypeRoll100
	g.Expect(p.NumberOfSquares()).Should(gomega.Equal(100))
}
