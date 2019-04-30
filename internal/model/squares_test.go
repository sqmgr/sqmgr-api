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

func TestSquares(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	user, err := m.NewUser(randString()+"@sqmgr.com", "my-unique-password")
	g.Expect(err).Should(gomega.Succeed())

	squares, err := m.NewSquares(user.ID, "My Squares", SquaresTypeStd100, "my-other-unique-password")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(squares).ShouldNot(gomega.BeNil())

	g.Expect(squares.ID).Should(gomega.BeNumerically(">", 0))
	g.Expect(squares.UserID).Should(gomega.Equal(user.ID))
	g.Expect(squares.Token).Should(gomega.MatchRegexp(`^[A-Za-z0-9_-]{8}\z`))
	g.Expect(squares.Name).Should(gomega.Equal("My Squares"))
	g.Expect(squares.passwordHash).ShouldNot(gomega.Equal("my-other-unique-password"))
	g.Expect(argon2id.Compare(squares.passwordHash, "my-other-unique-password")).Should(gomega.Succeed())

	originalPasswordHash := squares.passwordHash
	squares.SetPassword("my-other-unique-password")
	g.Expect(squares.passwordHash).ShouldNot(gomega.Equal(originalPasswordHash))

	g.Expect(squares.Settings).Should(gomega.Equal(SquaresSettings{
		squaresID:      squares.ID,
		homeTeamName:   nil,
		homeTeamColor1: nil,
		homeTeamColor2: nil,
		awayTeamName:   nil,
		awayTeamColor1: nil,
		awayTeamColor2: nil,
		notes:          nil,
	}))

	future := time.Now().UTC().Add(time.Hour)
	squares.Name = "Different Name"
	squares.Locks = future
	squares.SquaresType = SquaresTypeStd25

	awayTeamName := "Different Away Team"
	squares.Settings.SetAwayTeamName(awayTeamName)

	err = squares.Save()
	g.Expect(err).Should(gomega.Succeed())

	squares2, err := m.SquaresByID(squares.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(squares2).ShouldNot(gomega.BeNil())

	g.Expect(squares2.Name).Should(gomega.Equal("Different Name"))
	g.Expect(squares2.Locks.Unix()).Should(gomega.Equal(future.Unix()))
	g.Expect(squares2.SquaresType).Should(gomega.Equal(SquaresTypeStd25))
	g.Expect(squares2.Settings.HomeTeamName()).Should(gomega.Equal(DefaultHomeTeamName))
	g.Expect(squares2.Settings.AwayTeamName()).Should(gomega.Equal("Different Away Team"))

	squares3, err := m.SquaresByToken(context.Background(), squares2.Token)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(squares3).ShouldNot(gomega.BeNil())
	g.Expect(squares3).Should(gomega.Equal(squares2))

	loadedSquares, err := m.SquaresByID(squares.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(loadedSquares.LoadSettings()).Should(gomega.Succeed())
	g.Expect(loadedSquares.Settings.squaresID).Should(gomega.Equal(squares.ID))
}

func TestNewSquaresInvalidSquaresType(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	m := New(nil)
	s, err := m.NewSquares(1, "my name", SquaresType("invalid"), "my password")
	g.Expect(s).Should(gomega.BeNil())
	g.Expect(err).Should(gomega.MatchError(ErrInvalidSquaresType))
}

func TestSquaresCollection(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	user, err := m.NewUser(randString()+"@sqmgr.com", "my-unique-password")
	g.Expect(err).Should(gomega.Succeed())

	squares, err := m.NewSquares(user.ID, "Test for Collection", SquaresTypeStd100, "my-other-unique-password")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(squares).ShouldNot(gomega.BeNil())

	user2, err := m.NewUser(randString()+"@sqmgr.com", "my-unique-password-2")
	g.Expect(err).Should(gomega.Succeed())

	collection, err := m.SquaresCollectionJoinedByUser(context.Background(), user, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(0))

	collection, err = m.SquaresCollectionJoinedByUser(context.Background(), user2, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(0))

	g.Expect(user2.JoinSquares(context.Background(), squares)).Should(gomega.Succeed())
	collection, err = m.SquaresCollectionJoinedByUser(context.Background(), user2, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(1))

	collection, err = m.SquaresCollectionOwnedByUser(context.Background(), user, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(1))

	collection, err = m.SquaresCollectionOwnedByUser(context.Background(), user2, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(collection)).Should(gomega.Equal(0))
}

func TestSquaresCollectionPagination(t *testing.T) {
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
		squares, err := m.NewSquares(user1.ID, randString(), SquaresTypeStd100, "my-other-unique-password")
		g.Expect(err).Should(gomega.Succeed())

		if i < 20 {
			g.Expect(user2.JoinSquares(context.Background(), squares)).Should(gomega.Succeed())
		}
	}

	count, err := m.SquaresCollectionOwnedByUserCount(context.Background(), user1)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(30)))

	count, err = m.SquaresCollectionOwnedByUserCount(context.Background(), user2)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(0)))

	count, err = m.SquaresCollectionJoinedByUserCount(context.Background(), user1)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(0)))

	count, err = m.SquaresCollectionJoinedByUserCount(context.Background(), user2)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(20)))

}
