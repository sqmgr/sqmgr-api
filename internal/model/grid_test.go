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

	var err error
	db, err = sql.Open("postgres", "sslmode=disable user=postgres database=integration")
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

func TestGrid(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	user, err := m.NewUser(randString()+"@sqmgr.com", "my-unique-password")
	g.Expect(err).Should(gomega.Succeed())

	grid, err := m.NewGrid(user.ID, "My Grid", GridTypeStd100, "my-other-unique-password")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(grid).ShouldNot(gomega.BeNil())

	g.Expect(grid.id).Should(gomega.BeNumerically(">", 0))
	g.Expect(grid.userID).Should(gomega.Equal(user.ID))
	g.Expect(grid.token).Should(gomega.MatchRegexp(`^[A-Za-z0-9_-]{8}\z`))
	g.Expect(grid.name).Should(gomega.Equal("My Grid"))
	g.Expect(grid.passwordHash).ShouldNot(gomega.Equal("my-other-unique-password"))
	g.Expect(argon2id.Compare(grid.passwordHash, "my-other-unique-password")).Should(gomega.Succeed())

	originalPasswordHash := grid.passwordHash
	grid.SetPassword("my-other-unique-password")
	g.Expect(grid.passwordHash).ShouldNot(gomega.Equal(originalPasswordHash))

	g.Expect(grid.settings).Should(gomega.Equal(GridSettings{
		gridID:         grid.id,
		homeTeamName:   nil,
		homeTeamColor1: nil,
		homeTeamColor2: nil,
		awayTeamName:   nil,
		awayTeamColor1: nil,
		awayTeamColor2: nil,
		notes:          nil,
	}))

	future := time.Now().UTC().Add(time.Hour)
	grid.name = "Different Name"
	grid.locks = future
	grid.gridType = GridTypeStd25

	awayTeamName := "Different Away Team"
	grid.settings.SetAwayTeamName(awayTeamName)

	err = grid.Save()
	g.Expect(err).Should(gomega.Succeed())

	grid2, err := m.GridByID(grid.id)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(grid2).ShouldNot(gomega.BeNil())

	g.Expect(grid2.name).Should(gomega.Equal("Different Name"))
	g.Expect(grid2.locks.Unix()).Should(gomega.Equal(future.Unix()))
	g.Expect(grid2.gridType).Should(gomega.Equal(GridTypeStd25))
	g.Expect(grid2.settings.HomeTeamName()).Should(gomega.Equal(DefaultHomeTeamName))
	g.Expect(grid2.settings.AwayTeamName()).Should(gomega.Equal("Different Away Team"))

	grid3, err := m.GridByToken(context.Background(), grid2.token)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(grid3).ShouldNot(gomega.BeNil())
	g.Expect(grid3).Should(gomega.Equal(grid2))

	loadedGrid, err := m.GridByID(grid.id)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(loadedGrid.LoadSettings()).Should(gomega.Succeed())
	g.Expect(loadedGrid.settings.gridID).Should(gomega.Equal(grid.id))
}

func TestNewGridInvalidGridType(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	m := New(nil)
	s, err := m.NewGrid(1, "my name", GridType("invalid"), "my password")
	g.Expect(s).Should(gomega.BeNil())
	g.Expect(err).Should(gomega.MatchError(ErrInvalidGridType))
}

func TestGridCollections(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	user, err := m.NewUser(randString()+"@sqmgr.com", "my-unique-password")
	g.Expect(err).Should(gomega.Succeed())

	grid, err := m.NewGrid(user.ID, "Test for Collection", GridTypeStd100, "my-other-unique-password")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(grid).ShouldNot(gomega.BeNil())

	user2, err := m.NewUser(randString()+"@sqmgr.com", "my-unique-password-2")
	g.Expect(err).Should(gomega.Succeed())

	collection, err := m.GridsJoinedByUser(context.Background(), user, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(0))

	collection, err = m.GridsJoinedByUser(context.Background(), user2, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(0))

	g.Expect(user2.JoinGrid(context.Background(), grid)).Should(gomega.Succeed())
	collection, err = m.GridsJoinedByUser(context.Background(), user2, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(1))

	collection, err = m.GridsOwnedByUser(context.Background(), user, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(1))

	collection, err = m.GridsOwnedByUser(context.Background(), user2, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(0))
}

func TestGridCollectionPagination(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	user1, err := m.NewUser(randString()+"@sqmgr.com", "my-unique-password")
	g.Expect(err).Should(gomega.Succeed())

	user2, err := m.NewUser(randString()+"@sqmgr.com", "my-unique-password")
	g.Expect(err).Should(gomega.Succeed())

	for i := 0; i < 30; i++ {
		grid, err := m.NewGrid(user1.ID, randString(), GridTypeStd100, "my-other-unique-password")
		g.Expect(err).Should(gomega.Succeed())

		if i < 20 {
			g.Expect(user2.JoinGrid(context.Background(), grid)).Should(gomega.Succeed())
		}
	}

	count, err := m.GridsOwnedByUserCount(context.Background(), user1)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(30)))

	count, err = m.GridsOwnedByUserCount(context.Background(), user2)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(0)))

	count, err = m.GridsJoinedByUserCount(context.Background(), user1)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(0)))

	count, err = m.GridsJoinedByUserCount(context.Background(), user2)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(20)))

}

func TestAccessors(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	locks := time.Now()
	created := time.Now()
	modified := time.Now()

	s := &Grid{
		locks:    locks,
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
	g.Expect(s.Locks()).Should(gomega.Equal(locks))
	g.Expect(s.Created()).Should(gomega.Equal(created))
	g.Expect(s.Modified()).Should(gomega.Equal(modified))

	g.Expect(s.Settings()).Should(gomega.Equal(&s.settings))

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

	user, err := m.NewUser(randString()+"@sqmgr.com", "password")
	g.Expect(err).Should(gomega.Succeed())

	grid, err := m.NewGrid(user.ID, "Test Grid", GridTypeStd25, "a password")
	g.Expect(err).Should(gomega.Succeed())

	squares, err := grid.Squares()
	g.Expect(err).Should(gomega.Succeed())

	g.Expect(len(squares)).Should(gomega.Equal(25))

	square := squares[15]
	g.Expect(square.SquareID).Should(gomega.Equal(15))
	g.Expect(square.Claimant).Should(gomega.Equal(""))

	square.Claimant = "Test User"
	square.State = GridSquareStateClaimed
	square.SetUserIdentifier(user.ID)
	err = square.Save(context.Background(), true, GridSquareLog{
		Note:       "Test Note",
		RemoteAddr: "127.0.0.1",
	})
	g.Expect(err).Should(gomega.Succeed())

	grid.squares = nil // force a fresh fetch
	squares, err = grid.Squares()
	g.Expect(err).Should(gomega.Succeed())

	square = squares[15]
	g.Expect(square.Claimant).Should(gomega.Equal("Test User"))

	err = square.Save(context.Background(), true, GridSquareLog{
		Note: "A new note",
	})
	g.Expect(err).Should(gomega.Succeed())

	squares2, err := grid.SquareBySquareID(15)
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

	logs, err := grid.Logs(context.Background(), 0, 1000)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(logs)).Should(gomega.BeNumerically(">", 0))

	count, err := grid.LogsCount(context.Background())
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(len(logs))))

	square.Claimant = "New User"
	err = square.Save(context.Background(), false, GridSquareLog{
		Note: "",
	})
	g.Expect(err).Should(gomega.Equal(ErrSquareAlreadyClaimed))
}
