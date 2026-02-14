/*
Copyright (C) 2024 Tom Peters

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
	"testing"

	"github.com/onsi/gomega"
)

func TestGridNumberSetHasNumbers(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	gns := &GridNumberSet{}
	g.Expect(gns.HasNumbers()).Should(gomega.BeFalse())

	gns.homeNumbers = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	g.Expect(gns.HasNumbers()).Should(gomega.BeFalse())

	gns.awayNumbers = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	g.Expect(gns.HasNumbers()).Should(gomega.BeTrue())
}

func TestGridNumberSetSetNumbers(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	gns := &GridNumberSet{}

	// Invalid numbers (not 0-9)
	err := gns.SetNumbers([]int{1, 2, 3}, []int{4, 5, 6})
	g.Expect(err).Should(gomega.Equal(ErrNumbersAreInvalid))

	// Valid numbers
	homeNums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	awayNums := []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	err = gns.SetNumbers(homeNums, awayNums)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(gns.homeNumbers).Should(gomega.Equal(homeNums))
	g.Expect(gns.awayNumbers).Should(gomega.Equal(awayNums))
	g.Expect(gns.manualDraw).Should(gomega.BeTrue())

	// Cannot set numbers twice
	err = gns.SetNumbers(homeNums, awayNums)
	g.Expect(err).Should(gomega.Equal(ErrNumbersAlreadyDrawn))
}

func TestGridNumberSetSelectRandomNumbers(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	gns := &GridNumberSet{}

	err := gns.SelectRandomNumbers()
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(gns.HasNumbers()).Should(gomega.BeTrue())
	g.Expect(gns.manualDraw).Should(gomega.BeFalse())

	// Verify numbers are 0-9 in some order
	g.Expect(len(gns.homeNumbers)).Should(gomega.Equal(10))
	g.Expect(len(gns.awayNumbers)).Should(gomega.Equal(10))

	homeSum := 0
	awaySum := 0
	for i := 0; i < 10; i++ {
		homeSum += gns.homeNumbers[i]
		awaySum += gns.awayNumbers[i]
	}
	// Sum of 0-9 is 45
	g.Expect(homeSum).Should(gomega.Equal(45))
	g.Expect(awaySum).Should(gomega.Equal(45))

	// Cannot draw twice
	err = gns.SelectRandomNumbers()
	g.Expect(err).Should(gomega.Equal(ErrNumbersAlreadyDrawn))
}

func TestGridNumberSetJSON(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	gns := &GridNumberSet{
		id:          123,
		setType:     NumberSetTypeQ1,
		homeNumbers: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		awayNumbers: []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
		manualDraw:  true,
	}

	json := gns.JSON()
	g.Expect(json.ID).Should(gomega.Equal(int64(123)))
	g.Expect(json.SetType).Should(gomega.Equal(NumberSetTypeQ1))
	g.Expect(json.HomeNumbers).Should(gomega.Equal([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))
	g.Expect(json.AwayNumbers).Should(gomega.Equal([]int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}))
	g.Expect(json.ManualDraw).Should(gomega.BeTrue())
}
