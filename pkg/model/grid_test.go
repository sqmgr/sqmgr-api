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
	"github.com/onsi/gomega"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGridName(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	grid := &Grid{}

	g.Expect(grid.Name()).Should(gomega.Equal("Away Team vs. Home Team"))
	grid.SetAwayTeamName("Foo")
	g.Expect(grid.Name()).Should(gomega.Equal("Foo vs. Home Team"))
	grid.SetHomeTeamName("Bar")
	g.Expect(grid.Name()).Should(gomega.Equal("Foo vs. Bar"))
	grid.SetAwayTeamName("")
	g.Expect(grid.Name()).Should(gomega.Equal("Away Team vs. Bar"))
	grid.SetHomeTeamName("")
	g.Expect(grid.Name()).Should(gomega.Equal("Away Team vs. Home Team"))

	// Test Label
	grid.SetLabel("Wild Card")
	g.Expect(grid.Name()).Should(gomega.Equal("Wild Card: Away Team vs. Home Team"))

	grid.SetLabel("")
	g.Expect(grid.Name()).Should(gomega.Equal("Away Team vs. Home Team"))

	grid.SetHomeTeamName(strings.Repeat("á", 75) + "é")
	g.Expect(grid.HomeTeamName()).Should(gomega.Equal(strings.Repeat("á", 75)))

	grid.SetAwayTeamName(strings.Repeat("í", 75) + "é")
	g.Expect(grid.AwayTeamName()).Should(gomega.Equal(strings.Repeat("í", 75)))
}

func TestGrid(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	user, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	g.Expect(err).Should(gomega.Succeed())

	pool, err := m.NewPool(context.Background(), user.ID, "My Pool", GridTypeStd25, "my-pass", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(pool.id).Should(gomega.BeNumerically(">", 0))
	g.Expect(pool.token).ShouldNot(gomega.BeEmpty())
	g.Expect(pool.userID).Should(gomega.Equal(user.ID))
	g.Expect(pool.name).Should(gomega.Equal("My Pool"))
	g.Expect(pool.gridType).Should(gomega.Equal(GridTypeStd25))

	newGrid := pool.NewGrid()
	g.Expect(newGrid.Save(context.Background())).Should(gomega.Succeed())

	grids, err := pool.Grids(context.Background(), 0, 1000)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(grids)).Should(gomega.Equal(2))

	grid := grids[0]
	g.Expect(grid.poolID).Should(gomega.Equal(pool.id))
	g.Expect(grid.ord).Should(gomega.Equal(0))
	g.Expect(grid.eventDate.IsZero()).Should(gomega.BeTrue())
	g.Expect(grid.manualDraw).Should(gomega.BeFalse())

	g.Expect(grids[1].id).Should(gomega.Equal(newGrid.id))

	grid.ord = 2
	grid.homeNumbers = []int{1, 2, 3}
	grid.awayNumbers = []int{4, 5, 6}
	now := time.Now()
	grid.eventDate = now
	grid.manualDraw = true
	g.Expect(grid.Save(context.Background())).Should(gomega.Succeed())

	grid, err = pool.GridByID(context.Background(), grid.id)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(grid.ord).Should(gomega.Equal(2))
	g.Expect(grid.homeNumbers).Should(gomega.Equal([]int{1, 2, 3}))
	g.Expect(grid.awayNumbers).Should(gomega.Equal([]int{4, 5, 6}))
	g.Expect(grid.manualDraw).Should(gomega.BeTrue())

	grid.homeNumbers = nil
	grid.awayNumbers = nil
	g.Expect(grid.Save(context.Background())).Should(gomega.Succeed())

	grid, err = pool.GridByID(context.Background(), grid.id)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(grid.homeNumbers).Should(gomega.BeNil())
	g.Expect(grid.awayNumbers).Should(gomega.BeNil())

	g.Expect(grid.SelectRandomNumbers()).Should(gomega.Succeed())
	g.Expect(len(grid.homeNumbers)).Should(gomega.Equal(10))
	g.Expect(len(grid.awayNumbers)).Should(gomega.Equal(10))

	g.Expect(grid.settings).Should(gomega.BeNil())
	g.Expect(grid.LoadSettings(context.Background())).Should(gomega.Succeed())
	g.Expect(grid.settings).ShouldNot(gomega.BeNil())

	grid.settings.SetHomeTeamColor1("red")
	grid.settings.SetHomeTeamColor2("white")
	grid.settings.SetAwayTeamColor1("yellow")
	grid.settings.SetAwayTeamColor2("green")
	grid.settings.SetNotes("my notes")
	g.Expect(grid.Save(context.Background())).Should(gomega.Succeed())

	grid.settings = nil
	g.Expect(grid.LoadSettings(context.Background())).Should(gomega.Succeed())
	g.Expect(grid.settings.HomeTeamColor1()).Should(gomega.Equal("red"))
	g.Expect(grid.settings.HomeTeamColor2()).Should(gomega.Equal("white"))
	g.Expect(grid.settings.AwayTeamColor1()).Should(gomega.Equal("yellow"))
	g.Expect(grid.settings.AwayTeamColor2()).Should(gomega.Equal("green"))
	g.Expect(grid.settings.Notes()).Should(gomega.Equal("my notes"))

	grid.settings.SetHomeTeamColor1("")
	grid.settings.SetHomeTeamColor2("")
	grid.settings.SetAwayTeamColor1("")
	grid.settings.SetAwayTeamColor2("")
	grid.settings.SetNotes("")
	g.Expect(grid.Save(context.Background())).Should(gomega.Succeed())

	grid.settings = nil
	g.Expect(grid.LoadSettings(context.Background())).Should(gomega.Succeed())
	g.Expect(grid.settings.HomeTeamColor1()).Should(gomega.Equal(DefaultHomeTeamColor1))
	g.Expect(grid.settings.HomeTeamColor2()).Should(gomega.Equal(DefaultHomeTeamColor2))
	g.Expect(grid.settings.AwayTeamColor1()).Should(gomega.Equal(DefaultAwayTeamColor1))
	g.Expect(grid.settings.AwayTeamColor2()).Should(gomega.Equal(DefaultAwayTeamColor2))
	g.Expect(grid.settings.Notes()).Should(gomega.Equal(""))
}

// TestRandomNumbers will verify the following:
// 1. slice is length = 10
// 2. slice is random between runs
// 3. each number is returned once
func TestRandomNumbers(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	nums, err := randomNumbers()
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(nums)).Should(gomega.Equal(10))

	found := make(map[int]int)
	for _, num := range nums {
		count := found[num]
		count++
		g.Expect(count).Should(gomega.Equal(1))

		found[num] = count
	}

	// there's a low chance that back-to-back runs _may_ produce the exact same
	// results. So run it up to three times to reduce the chance of this test failing
	diff := false
	for i := 0; i < 3; i++ {
		nums2, err := randomNumbers()
		g.Expect(err).Should(gomega.Succeed())

		if !reflect.DeepEqual(nums2, nums) {
			diff = true
			break
		}
	}

	g.Expect(diff).Should(gomega.BeTrue(), "random numbers generated different order")
}

func TestSelectRandomNumbers(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	grid := &Grid{}
	g.Expect(grid.SelectRandomNumbers()).Should(gomega.Succeed())
	g.Expect(grid.awayNumbers).ShouldNot(gomega.BeNil())
	g.Expect(grid.homeNumbers).ShouldNot(gomega.BeNil())
	g.Expect(grid.SelectRandomNumbers()).Should(gomega.Equal(ErrNumbersAlreadyDrawn))
}

func TestGridDelete(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	pool := getPool(m)
	grid := pool.NewGrid()
	g.Expect(grid.Save(context.Background())).Should(gomega.Succeed())
	grids, err := pool.Grids(context.Background(), 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(grids)).Should(gomega.Equal(2))

	g.Expect(grids[0].Delete(context.Background())).Should(gomega.Succeed())
	grids, err = pool.Grids(context.Background(), 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(grids)).Should(gomega.Equal(1))

	g.Expect(grids[0].Delete(context.Background())).Should(gomega.Equal(ErrLastGrid))
	grids, err = pool.Grids(context.Background(), 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(grids)).Should(gomega.Equal(1))

	count, err := pool.GridsCount(context.Background())
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(1)))

	grids, err = pool.Grids(context.Background(), 0, 10, true)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(grids)).Should(gomega.Equal(2))

	count, err = pool.GridsCount(context.Background(), true)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.Equal(int64(2)))
}

func getPool(m *Model) *Pool {
	user, err := m.GetUser(context.Background(), IssuerSqMGR, randString())
	if err != nil {
		panic(err)
	}

	pool, err := m.NewPool(context.Background(), user.ID, "Test Pool", GridTypeStd25, "my-password", NumberSetConfigStandard)
	if err != nil {
		panic(err)
	}

	return pool
}

func TestGridPayoutConfig(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	grid := &Grid{}

	// Initially nil
	g.Expect(grid.PayoutConfig()).Should(gomega.BeNil())

	// Set a config
	config := NumberSetConfigHF
	grid.SetPayoutConfig(&config)
	g.Expect(grid.PayoutConfig()).ShouldNot(gomega.BeNil())
	g.Expect(*grid.PayoutConfig()).Should(gomega.Equal(NumberSetConfigHF))

	// Clear the config
	grid.SetPayoutConfig(nil)
	g.Expect(grid.PayoutConfig()).Should(gomega.BeNil())
}

func TestGridPayoutConfigJSON(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	grid := &Grid{}

	// JSON without payout config should have nil PayoutConfig
	json := grid.JSON()
	g.Expect(json.PayoutConfig).Should(gomega.BeNil())

	// JSON with payout config should include it
	config := NumberSetConfig123F
	grid.SetPayoutConfig(&config)
	json = grid.JSON()
	g.Expect(json.PayoutConfig).ShouldNot(gomega.BeNil())
	g.Expect(*json.PayoutConfig).Should(gomega.Equal(NumberSetConfig123F))
}

func TestGridPayoutConfigIntegration(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())

	pool := getPool(m)

	grids, err := pool.Grids(context.Background(), 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(grids)).Should(gomega.BeNumerically(">=", 1))

	grid := grids[0]

	// Initially should be nil
	g.Expect(grid.PayoutConfig()).Should(gomega.BeNil())

	// Set payout config and save
	config := NumberSetConfigHF
	grid.SetPayoutConfig(&config)
	g.Expect(grid.Save(context.Background())).Should(gomega.Succeed())

	// Reload and verify
	grid, err = pool.GridByID(context.Background(), grid.id)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(grid.PayoutConfig()).ShouldNot(gomega.BeNil())
	g.Expect(*grid.PayoutConfig()).Should(gomega.Equal(NumberSetConfigHF))

	// Clear payout config and save
	grid.SetPayoutConfig(nil)
	g.Expect(grid.Save(context.Background())).Should(gomega.Succeed())

	// Reload and verify
	grid, err = pool.GridByID(context.Background(), grid.id)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(grid.PayoutConfig()).Should(gomega.BeNil())
}
