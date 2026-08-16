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

// freshGrid returns a Grid in the state new_grid() leaves it in: a grids row
// with only pool_id and ord supplied, and an all-NULL grid_settings row.
func freshGrid() *Grid {
	return &Grid{
		id:       1,
		poolID:   1,
		ord:      0,
		state:    Active,
		settings: &GridSettings{gridID: 1},
	}
}

func TestGridIsPristine(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(freshGrid().IsPristine()).Should(gomega.BeTrue(), "a freshly created grid is pristine")

	eventID := int64(5001)
	payoutConfig := NumberSetConfigHF

	// each mutation on its own must be enough to make the grid look customized
	mutations := map[string]func(*Grid){
		"label":              func(grid *Grid) { grid.SetLabel("Week 1") },
		"home team name":     func(grid *Grid) { grid.SetHomeTeamName("Kansas City Chiefs") },
		"away team name":     func(grid *Grid) { grid.SetAwayTeamName("Buffalo Bills") },
		"home numbers":       func(grid *Grid) { grid.homeNumbers = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9} },
		"away numbers":       func(grid *Grid) { grid.awayNumbers = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9} },
		"manual draw":        func(grid *Grid) { grid.manualDraw = true },
		"rollover":           func(grid *Grid) { grid.SetRollover(true) },
		"event date":         func(grid *Grid) { grid.SetEventDate(time.Now()) },
		"linked event":       func(grid *Grid) { grid.SetBDLEventID(&eventID) },
		"payout config":      func(grid *Grid) { grid.SetPayoutConfig(&payoutConfig) },
		"state":              func(grid *Grid) { grid.SetState(Deleted) },
		"number sets":        func(grid *Grid) { grid.numberSets = map[NumberSetType]*GridNumberSet{NumberSetTypeAll: {}} },
		"annotations":        func(grid *Grid) { grid.annotations = map[int]*GridAnnotation{1: {}} },
		"notes":              func(grid *Grid) { grid.settings.SetNotes("bring snacks") },
		"home team color 1":  func(grid *Grid) { grid.settings.SetHomeTeamColor1("#E31837") },
		"home team color 2":  func(grid *Grid) { grid.settings.SetHomeTeamColor2("#FFB612") },
		"away team color 1":  func(grid *Grid) { grid.settings.SetAwayTeamColor1("#00338D") },
		"away team color 2":  func(grid *Grid) { grid.settings.SetAwayTeamColor2("#C60C30") },
		"branding image url": func(grid *Grid) { grid.settings.SetBrandingImageURL("https://example.com/logo.png") },
		"branding image alt": func(grid *Grid) { grid.settings.SetBrandingImageAlt("Logo") },
	}

	for name, mutate := range mutations {
		grid := freshGrid()
		mutate(grid)
		g.Expect(grid.IsPristine()).Should(gomega.BeFalse(), name)
	}

	// an empty (but loaded) set of number sets and annotations is what a fresh
	// grid has, so loading them must not make it look customized
	grid := freshGrid()
	grid.numberSets = map[NumberSetType]*GridNumberSet{}
	grid.annotations = map[int]*GridAnnotation{}
	g.Expect(grid.IsPristine()).Should(gomega.BeTrue(), "empty number sets and annotations")

	// settings that were never loaded could hold anything, so the grid can't
	// be claimed to be pristine
	grid = freshGrid()
	grid.settings = nil
	g.Expect(grid.IsPristine()).Should(gomega.BeFalse(), "unloaded settings")

	// clearing a customization returns the grid to pristine
	grid = freshGrid()
	grid.SetLabel("Week 1")
	g.Expect(grid.IsPristine()).Should(gomega.BeFalse())
	grid.SetLabel("")
	g.Expect(grid.IsPristine()).Should(gomega.BeTrue())
}
