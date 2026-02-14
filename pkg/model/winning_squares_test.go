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
	"testing"

	"github.com/onsi/gomega"
)

func TestCalculateWinningSquare(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Standard number arrangement: position i has number i
	homeNumbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	awayNumbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	// Score 0-0: home digit 0 at position 0, away digit 0 at position 0
	// Square = (0 * 10) + 0 + 1 = 1
	result := CalculateWinningSquare(0, 0, homeNumbers, awayNumbers, GridTypeStd100)
	g.Expect(result).Should(gomega.Equal(1))

	// Score 10-7: home digit 0 at position 0, away digit 7 at position 7
	// Square = (7 * 10) + 0 + 1 = 71
	result = CalculateWinningSquare(10, 7, homeNumbers, awayNumbers, GridTypeStd100)
	g.Expect(result).Should(gomega.Equal(71))

	// Score 24-17: home digit 4 at position 4, away digit 7 at position 7
	// Square = (7 * 10) + 4 + 1 = 75
	result = CalculateWinningSquare(24, 17, homeNumbers, awayNumbers, GridTypeStd100)
	g.Expect(result).Should(gomega.Equal(75))

	// Score 33-28: home digit 3 at position 3, away digit 8 at position 8
	// Square = (8 * 10) + 3 + 1 = 84
	result = CalculateWinningSquare(33, 28, homeNumbers, awayNumbers, GridTypeStd100)
	g.Expect(result).Should(gomega.Equal(84))
}

func TestCalculateWinningSquareShuffled(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Shuffled numbers: position i has a different number
	homeNumbers := []int{3, 7, 1, 9, 0, 5, 2, 8, 4, 6}
	awayNumbers := []int{8, 2, 5, 0, 6, 1, 9, 4, 7, 3}

	// Score 24-17:
	// Home digit 4: find position where homeNumbers[pos] == 4 -> position 8
	// Away digit 7: find position where awayNumbers[pos] == 7 -> position 8
	// Square = (8 * 10) + 8 + 1 = 89
	result := CalculateWinningSquare(24, 17, homeNumbers, awayNumbers, GridTypeStd100)
	g.Expect(result).Should(gomega.Equal(89))

	// Score 10-20:
	// Home digit 0: find position where homeNumbers[pos] == 0 -> position 4
	// Away digit 0: find position where awayNumbers[pos] == 0 -> position 3
	// Square = (3 * 10) + 4 + 1 = 35
	result = CalculateWinningSquare(10, 20, homeNumbers, awayNumbers, GridTypeStd100)
	g.Expect(result).Should(gomega.Equal(35))
}

func TestCalculateWinningSquareInvalidNumbers(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Invalid array lengths
	result := CalculateWinningSquare(10, 7, nil, nil, GridTypeStd100)
	g.Expect(result).Should(gomega.Equal(0))

	result = CalculateWinningSquare(10, 7, []int{0, 1, 2}, []int{0, 1, 2}, GridTypeStd100)
	g.Expect(result).Should(gomega.Equal(0))

	result = CalculateWinningSquare(10, 7, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, []int{0, 1, 2}, GridTypeStd100)
	g.Expect(result).Should(gomega.Equal(0))
}

func TestGetWinningSquares(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Create a mock event with scores
	homeQ1 := 7
	awayQ1 := 3
	homeQ2 := 7 // Half = 14
	awayQ2 := 7 // Half = 10
	homeQ3 := 7 // Q3 cumulative = 21
	awayQ3 := 7 // Q3 cumulative = 17
	homeScore := 28
	awayScore := 24

	event := &BDLEvent{
		Status:    BDLEventStatusFinal,
		HomeQ1:    &homeQ1,
		AwayQ1:    &awayQ1,
		HomeQ2:    &homeQ2,
		AwayQ2:    &awayQ2,
		HomeQ3:    &homeQ3,
		AwayQ3:    &awayQ3,
		HomeScore: &homeScore,
		AwayScore: &awayScore,
	}

	homeNumbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	awayNumbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	// Test with standard config (uses final score)
	result := GetWinningSquares(event, NumberSetConfigStandard, GridTypeStd100, homeNumbers, awayNumbers, nil)
	g.Expect(result.Squares).Should(gomega.HaveLen(1))
	// Final: 28-24, home digit 8, away digit 4
	// Square = (4 * 10) + 8 + 1 = 49
	g.Expect(result.Squares[NumberSetTypeAll]).Should(gomega.Equal(49))
}

func TestGetWinningSquaresHFConfig(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	homeQ1 := 7
	awayQ1 := 3
	homeQ2 := 7 // Half = 14
	awayQ2 := 7 // Half = 10
	homeScore := 28
	awayScore := 24

	event := &BDLEvent{
		Status:    BDLEventStatusFinal,
		HomeQ1:    &homeQ1,
		AwayQ1:    &awayQ1,
		HomeQ2:    &homeQ2,
		AwayQ2:    &awayQ2,
		HomeScore: &homeScore,
		AwayScore: &awayScore,
	}

	// For multi-set config, we need numberSets
	homeNums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	awayNums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	halfSet := &GridNumberSet{
		homeNumbers: homeNums,
		awayNumbers: awayNums,
	}
	finalSet := &GridNumberSet{
		homeNumbers: homeNums,
		awayNumbers: awayNums,
	}

	numberSets := map[NumberSetType]*GridNumberSet{
		NumberSetTypeHalf:  halfSet,
		NumberSetTypeFinal: finalSet,
	}

	result := GetWinningSquares(event, NumberSetConfigHF, GridTypeStd100, nil, nil, numberSets)
	g.Expect(result.Squares).Should(gomega.HaveLen(2))

	// Half: 14-10, home digit 4, away digit 0
	// Square = (0 * 10) + 4 + 1 = 5
	g.Expect(result.Squares[NumberSetTypeHalf]).Should(gomega.Equal(5))

	// Final: 28-24, home digit 8, away digit 4
	// Square = (4 * 10) + 8 + 1 = 49
	g.Expect(result.Squares[NumberSetTypeFinal]).Should(gomega.Equal(49))
}

func TestBDLEventScoreForPeriod(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	homeQ1 := 7
	awayQ1 := 3
	homeQ2 := 7
	awayQ2 := 7
	homeQ3 := 7
	awayQ3 := 7
	homeScore := 28
	awayScore := 24

	event := &BDLEvent{
		HomeQ1:    &homeQ1,
		AwayQ1:    &awayQ1,
		HomeQ2:    &homeQ2,
		AwayQ2:    &awayQ2,
		HomeQ3:    &homeQ3,
		AwayQ3:    &awayQ3,
		HomeScore: &homeScore,
		AwayScore: &awayScore,
	}

	// Q1 scores
	home, away := event.ScoreForPeriod(NumberSetTypeQ1)
	g.Expect(*home).Should(gomega.Equal(7))
	g.Expect(*away).Should(gomega.Equal(3))

	// Q2/Half scores (cumulative through Q2)
	home, away = event.ScoreForPeriod(NumberSetTypeQ2)
	g.Expect(*home).Should(gomega.Equal(14))
	g.Expect(*away).Should(gomega.Equal(10))

	home, away = event.ScoreForPeriod(NumberSetTypeHalf)
	g.Expect(*home).Should(gomega.Equal(14))
	g.Expect(*away).Should(gomega.Equal(10))

	// Q3 cumulative
	home, away = event.ScoreForPeriod(NumberSetTypeQ3)
	g.Expect(*home).Should(gomega.Equal(21))
	g.Expect(*away).Should(gomega.Equal(17))

	// Final/Q4/All
	home, away = event.ScoreForPeriod(NumberSetTypeFinal)
	g.Expect(*home).Should(gomega.Equal(28))
	g.Expect(*away).Should(gomega.Equal(24))

	home, away = event.ScoreForPeriod(NumberSetTypeAll)
	g.Expect(*home).Should(gomega.Equal(28))
	g.Expect(*away).Should(gomega.Equal(24))
}

func TestBDLEventScoreForPeriodNilScores(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	event := &BDLEvent{
		Status: BDLEventStatusScheduled,
	}

	// All scores should be nil when game hasn't started
	home, away := event.ScoreForPeriod(NumberSetTypeQ1)
	g.Expect(home).Should(gomega.BeNil())
	g.Expect(away).Should(gomega.BeNil())

	home, away = event.ScoreForPeriod(NumberSetTypeHalf)
	g.Expect(home).Should(gomega.BeNil())
	g.Expect(away).Should(gomega.BeNil())

	home, away = event.ScoreForPeriod(NumberSetTypeFinal)
	g.Expect(home).Should(gomega.BeNil())
	g.Expect(away).Should(gomega.BeNil())
}

func TestCalculateWinningSquareStd25(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Standard number arrangement: position i has number i
	homeNumbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	awayNumbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	// For std25 (5x5), each square covers 2x2 positions
	// homePos5 = homePos / 2, awayPos5 = awayPos / 2
	// squareID = (awayPos5 * 5) + homePos5 + 1

	// Score 0-0: home digit 0 at pos 0, away digit 0 at pos 0
	// homePos5 = 0, awayPos5 = 0, square = (0 * 5) + 0 + 1 = 1
	result := CalculateWinningSquare(0, 0, homeNumbers, awayNumbers, GridTypeStd25)
	g.Expect(result).Should(gomega.Equal(1))

	// Score 1-1: home digit 1 at pos 1, away digit 1 at pos 1
	// homePos5 = 0, awayPos5 = 0, square = 1 (same as 0-0)
	result = CalculateWinningSquare(1, 1, homeNumbers, awayNumbers, GridTypeStd25)
	g.Expect(result).Should(gomega.Equal(1))

	// Score 2-2: home digit 2 at pos 2, away digit 2 at pos 2
	// homePos5 = 1, awayPos5 = 1, square = (1 * 5) + 1 + 1 = 7
	result = CalculateWinningSquare(2, 2, homeNumbers, awayNumbers, GridTypeStd25)
	g.Expect(result).Should(gomega.Equal(7))

	// Score 24-17: home digit 4 at pos 4, away digit 7 at pos 7
	// homePos5 = 2, awayPos5 = 3, square = (3 * 5) + 2 + 1 = 18
	result = CalculateWinningSquare(24, 17, homeNumbers, awayNumbers, GridTypeStd25)
	g.Expect(result).Should(gomega.Equal(18))

	// Score 28-24: home digit 8 at pos 8, away digit 4 at pos 4
	// homePos5 = 4, awayPos5 = 2, square = (2 * 5) + 4 + 1 = 15
	result = CalculateWinningSquare(28, 24, homeNumbers, awayNumbers, GridTypeStd25)
	g.Expect(result).Should(gomega.Equal(15))

	// Score 99-99: home digit 9 at pos 9, away digit 9 at pos 9
	// homePos5 = 4, awayPos5 = 4, square = (4 * 5) + 4 + 1 = 25
	result = CalculateWinningSquare(99, 99, homeNumbers, awayNumbers, GridTypeStd25)
	g.Expect(result).Should(gomega.Equal(25))
}

func TestCalculateWinningSquareStd50(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Standard number arrangement: position i has number i
	homeNumbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	awayNumbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	// For std50 (5x10), each square covers 2 away positions, 1 home position
	// awayPos5 = awayPos / 2, homePos stays the same
	// squareID = (awayPos5 * 10) + homePos + 1

	// Score 0-0: home digit 0 at pos 0, away digit 0 at pos 0
	// awayPos5 = 0, square = (0 * 10) + 0 + 1 = 1
	result := CalculateWinningSquare(0, 0, homeNumbers, awayNumbers, GridTypeStd50)
	g.Expect(result).Should(gomega.Equal(1))

	// Score 0-1: home digit 0 at pos 0, away digit 1 at pos 1
	// awayPos5 = 0, square = (0 * 10) + 0 + 1 = 1 (same row)
	result = CalculateWinningSquare(0, 1, homeNumbers, awayNumbers, GridTypeStd50)
	g.Expect(result).Should(gomega.Equal(1))

	// Score 0-2: home digit 0 at pos 0, away digit 2 at pos 2
	// awayPos5 = 1, square = (1 * 10) + 0 + 1 = 11
	result = CalculateWinningSquare(0, 2, homeNumbers, awayNumbers, GridTypeStd50)
	g.Expect(result).Should(gomega.Equal(11))

	// Score 24-17: home digit 4 at pos 4, away digit 7 at pos 7
	// awayPos5 = 3, square = (3 * 10) + 4 + 1 = 35
	result = CalculateWinningSquare(24, 17, homeNumbers, awayNumbers, GridTypeStd50)
	g.Expect(result).Should(gomega.Equal(35))

	// Score 28-24: home digit 8 at pos 8, away digit 4 at pos 4
	// awayPos5 = 2, square = (2 * 10) + 8 + 1 = 29
	result = CalculateWinningSquare(28, 24, homeNumbers, awayNumbers, GridTypeStd50)
	g.Expect(result).Should(gomega.Equal(29))

	// Score 99-99: home digit 9 at pos 9, away digit 9 at pos 9
	// awayPos5 = 4, square = (4 * 10) + 9 + 1 = 50
	result = CalculateWinningSquare(99, 99, homeNumbers, awayNumbers, GridTypeStd50)
	g.Expect(result).Should(gomega.Equal(50))
}

func TestIsPeriodComplete(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Helper to create pointer to int
	intPtr := func(i int) *int { return &i }

	// Test: Game in Q1 (period=1) - no quarters complete
	event := &BDLEvent{
		Status: BDLEventStatusInProgress,
		Period: intPtr(1),
	}
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ1)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ3)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeFinal)).Should(gomega.BeFalse())

	// Test: Game in Q2 (period=2) - Q1 complete
	event.Period = intPtr(2)
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ1)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ3)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeFinal)).Should(gomega.BeFalse())

	// Test: Game in Q3 (period=3) - Q1, Q2, Half complete
	event.Period = intPtr(3)
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ1)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ2)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ3)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeFinal)).Should(gomega.BeFalse())

	// Test: Game in Q4 (period=4) - Q1, Q2, Half, Q3 complete
	event.Period = intPtr(4)
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ1)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ2)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ3)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeFinal)).Should(gomega.BeFalse())

	// Test: Game final - all complete
	event.Status = BDLEventStatusFinal
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ1)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ2)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ3)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeFinal)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeAll)).Should(gomega.BeTrue())
}

func TestGetWinningSquaresHFConfigFallbackToLegacy(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	homeQ1 := 7
	awayQ1 := 3
	homeQ2 := 7 // Half = 14
	awayQ2 := 7 // Half = 10
	homeScore := 28
	awayScore := 24

	event := &BDLEvent{
		Status:    BDLEventStatusFinal,
		HomeQ1:    &homeQ1,
		AwayQ1:    &awayQ1,
		HomeQ2:    &homeQ2,
		AwayQ2:    &awayQ2,
		HomeScore: &homeScore,
		AwayScore: &awayScore,
	}

	// Legacy numbers (no numberSets)
	homeNumbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	awayNumbers := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	// Test HF config with nil numberSets - should fall back to legacy numbers
	result := GetWinningSquares(event, NumberSetConfigHF, GridTypeStd100, homeNumbers, awayNumbers, nil)
	g.Expect(result.Squares).Should(gomega.HaveLen(2))

	// Half: 14-10, home digit 4, away digit 0
	// Square = (0 * 10) + 4 + 1 = 5
	g.Expect(result.Squares[NumberSetTypeHalf]).Should(gomega.Equal(5))

	// Final: 28-24, home digit 8, away digit 4
	// Square = (4 * 10) + 8 + 1 = 49
	g.Expect(result.Squares[NumberSetTypeFinal]).Should(gomega.Equal(49))
}

func TestGetWinningSquaresInProgress(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	intPtr := func(i int) *int { return &i }

	// Create a mock event in Q3 (Q1, Q2/Half complete, Q3 in progress)
	event := &BDLEvent{
		Status:    BDLEventStatusInProgress,
		Period:    intPtr(3),
		HomeQ1:    intPtr(7),
		AwayQ1:    intPtr(3),
		HomeQ2:    intPtr(7), // Half = 14
		AwayQ2:    intPtr(7), // Half = 10
		HomeQ3:    intPtr(7), // Q3 cumulative = 21 (in progress, shouldn't count)
		AwayQ3:    intPtr(7), // Q3 cumulative = 17
		HomeScore: intPtr(21),
		AwayScore: intPtr(17),
	}

	homeNums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	awayNums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	q1Set := &GridNumberSet{homeNumbers: homeNums, awayNumbers: awayNums}
	q2Set := &GridNumberSet{homeNumbers: homeNums, awayNumbers: awayNums}
	q3Set := &GridNumberSet{homeNumbers: homeNums, awayNumbers: awayNums}
	finalSet := &GridNumberSet{homeNumbers: homeNums, awayNumbers: awayNums}

	numberSets := map[NumberSetType]*GridNumberSet{
		NumberSetTypeQ1:    q1Set,
		NumberSetTypeQ2:    q2Set,
		NumberSetTypeQ3:    q3Set,
		NumberSetTypeFinal: finalSet,
	}

	result := GetWinningSquares(event, NumberSetConfig123F, GridTypeStd100, nil, nil, numberSets)

	// Should only have Q1 and Q2 winning squares (Q3 and Final not complete)
	g.Expect(result.Squares).Should(gomega.HaveLen(2))
	g.Expect(result.Squares).Should(gomega.HaveKey(NumberSetTypeQ1))
	g.Expect(result.Squares).Should(gomega.HaveKey(NumberSetTypeQ2))
	g.Expect(result.Squares).ShouldNot(gomega.HaveKey(NumberSetTypeQ3))
	g.Expect(result.Squares).ShouldNot(gomega.HaveKey(NumberSetTypeFinal))
}

func TestGetWinningSquaresNCAABHFConfig(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	intPtr := func(i int) *int { return &i }

	// NCAAB game in 2nd half (period=2): Q1=35 (1st half score), Q2=40 (2nd half score)
	event := &BDLEvent{
		League:    SportsLeagueNCAAB,
		Status:    BDLEventStatusInProgress,
		Period:    intPtr(2),
		HomeQ1:    intPtr(35),
		AwayQ1:    intPtr(28),
		HomeQ2:    intPtr(20), // 2nd half in progress
		AwayQ2:    intPtr(15),
		HomeScore: intPtr(55),
		AwayScore: intPtr(43),
	}

	homeNums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	awayNums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	halfSet := &GridNumberSet{homeNumbers: homeNums, awayNumbers: awayNums}
	finalSet := &GridNumberSet{homeNumbers: homeNums, awayNumbers: awayNums}

	numberSets := map[NumberSetType]*GridNumberSet{
		NumberSetTypeHalf:  halfSet,
		NumberSetTypeFinal: finalSet,
	}

	result := GetWinningSquares(event, NumberSetConfigHF, GridTypeStd100, nil, nil, numberSets)

	// Half should be complete (period >= 2 for NCAAB) so halftime square should exist
	g.Expect(result.Squares).Should(gomega.HaveKey(NumberSetTypeHalf))

	// Halftime score for NCAAB = Q1 only: home=35, away=28
	// home digit 5 at pos 5, away digit 8 at pos 8
	// Square = (8 * 10) + 5 + 1 = 86
	g.Expect(result.Squares[NumberSetTypeHalf]).Should(gomega.Equal(86))

	// Final should NOT be complete (still in progress)
	g.Expect(result.Squares).ShouldNot(gomega.HaveKey(NumberSetTypeFinal))
}

func TestGetWinningPeriodsForSquare(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	intPtr := func(i int) *int { return &i }

	// Create a mock event with final scores
	event := &BDLEvent{
		Status:    BDLEventStatusFinal,
		HomeQ1:    intPtr(7), // Q1: 7-3
		AwayQ1:    intPtr(3),
		HomeQ2:    intPtr(7), // Half: 14-10
		AwayQ2:    intPtr(7),
		HomeQ3:    intPtr(7), // Q3: 21-17
		AwayQ3:    intPtr(7),
		HomeScore: intPtr(28), // Final: 28-24
		AwayScore: intPtr(24),
	}

	// Standard numbers: position i has number i
	homeNums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	awayNums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	halfSet := &GridNumberSet{homeNumbers: homeNums, awayNumbers: awayNums}
	finalSet := &GridNumberSet{homeNumbers: homeNums, awayNumbers: awayNums}

	numberSets := map[NumberSetType]*GridNumberSet{
		NumberSetTypeHalf:  halfSet,
		NumberSetTypeFinal: finalSet,
	}

	// Get winning squares for HF config
	winningSquares := GetWinningSquares(event, NumberSetConfigHF, GridTypeStd100, nil, nil, numberSets)

	// Half: 14-10, home digit 4, away digit 0 -> Square = (0 * 10) + 4 + 1 = 5
	// Final: 28-24, home digit 8, away digit 4 -> Square = (4 * 10) + 8 + 1 = 49

	// Test square 5 (wins Half)
	results := GetWinningPeriodsForSquare(5, winningSquares, event, "Chiefs", "Eagles")
	g.Expect(results).Should(gomega.HaveLen(1))
	g.Expect(results[0].Period).Should(gomega.Equal(NumberSetTypeHalf))
	g.Expect(results[0].Label).Should(gomega.Equal("Halftime"))
	g.Expect(results[0].HomeScore).Should(gomega.Equal(14))
	g.Expect(results[0].AwayScore).Should(gomega.Equal(10))
	g.Expect(results[0].HomeTeamName).Should(gomega.Equal("Chiefs"))
	g.Expect(results[0].AwayTeamName).Should(gomega.Equal("Eagles"))

	// Test square 49 (wins Final)
	results = GetWinningPeriodsForSquare(49, winningSquares, event, "Chiefs", "Eagles")
	g.Expect(results).Should(gomega.HaveLen(1))
	g.Expect(results[0].Period).Should(gomega.Equal(NumberSetTypeFinal))
	g.Expect(results[0].Label).Should(gomega.Equal("Final"))
	g.Expect(results[0].HomeScore).Should(gomega.Equal(28))
	g.Expect(results[0].AwayScore).Should(gomega.Equal(24))

	// Test square 1 (doesn't win anything)
	results = GetWinningPeriodsForSquare(1, winningSquares, event, "Chiefs", "Eagles")
	g.Expect(results).Should(gomega.BeEmpty())
}

func TestGetWinningPeriodsForSquareMultipleWins(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	intPtr := func(i int) *int { return &i }

	// Create an event where one square wins multiple periods
	// Q1: 10-0 (digit 0, digit 0) -> Square 1
	// Half: 20-0 (digit 0, digit 0) -> Square 1
	// Final: 30-0 (digit 0, digit 0) -> Square 1
	event := &BDLEvent{
		Status:    BDLEventStatusFinal,
		HomeQ1:    intPtr(10),
		AwayQ1:    intPtr(0),
		HomeQ2:    intPtr(10),
		AwayQ2:    intPtr(0),
		HomeQ3:    intPtr(10),
		AwayQ3:    intPtr(0),
		HomeScore: intPtr(30),
		AwayScore: intPtr(0),
	}

	homeNums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	awayNums := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	halfSet := &GridNumberSet{homeNumbers: homeNums, awayNumbers: awayNums}
	finalSet := &GridNumberSet{homeNumbers: homeNums, awayNumbers: awayNums}

	numberSets := map[NumberSetType]*GridNumberSet{
		NumberSetTypeHalf:  halfSet,
		NumberSetTypeFinal: finalSet,
	}

	winningSquares := GetWinningSquares(event, NumberSetConfigHF, GridTypeStd100, nil, nil, numberSets)

	// Square 1 should win both Half and Final
	results := GetWinningPeriodsForSquare(1, winningSquares, event, "HOU", "IND")
	g.Expect(results).Should(gomega.HaveLen(2))

	// Results should be sorted: Half before Final
	g.Expect(results[0].Period).Should(gomega.Equal(NumberSetTypeHalf))
	g.Expect(results[0].Label).Should(gomega.Equal("Halftime"))
	g.Expect(results[0].HomeScore).Should(gomega.Equal(20))
	g.Expect(results[0].AwayScore).Should(gomega.Equal(0))
	g.Expect(results[0].HomeTeamName).Should(gomega.Equal("HOU"))
	g.Expect(results[0].AwayTeamName).Should(gomega.Equal("IND"))

	g.Expect(results[1].Period).Should(gomega.Equal(NumberSetTypeFinal))
	g.Expect(results[1].Label).Should(gomega.Equal("Final"))
	g.Expect(results[1].HomeScore).Should(gomega.Equal(30))
	g.Expect(results[1].AwayScore).Should(gomega.Equal(0))
}

func TestGetWinningPeriodsForSquareNilInputs(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	intPtr := func(i int) *int { return &i }

	event := &BDLEvent{
		Status:    BDLEventStatusFinal,
		HomeScore: intPtr(10),
		AwayScore: intPtr(7),
	}

	winningSquares := &WinningSquaresResult{
		Squares: map[NumberSetType]int{NumberSetTypeAll: 1},
	}

	// Test with invalid squareID
	results := GetWinningPeriodsForSquare(0, winningSquares, event, "HOU", "IND")
	g.Expect(results).Should(gomega.BeNil())

	results = GetWinningPeriodsForSquare(-1, winningSquares, event, "HOU", "IND")
	g.Expect(results).Should(gomega.BeNil())

	// Test with nil winningSquares
	results = GetWinningPeriodsForSquare(1, nil, event, "HOU", "IND")
	g.Expect(results).Should(gomega.BeNil())

	// Test with nil event
	results = GetWinningPeriodsForSquare(1, winningSquares, nil, "HOU", "IND")
	g.Expect(results).Should(gomega.BeNil())
}
