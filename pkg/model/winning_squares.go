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

// CalculateWinningSquare finds the square ID that matches the score
// homeNumbers/awayNumbers are the 10-element arrays (index = position, value = number)
// Returns 0 if no match (numbers not drawn or score doesn't match any square)
func CalculateWinningSquare(homeScore, awayScore int, homeNumbers, awayNumbers []int, gridType GridType) int {
	if len(homeNumbers) != 10 || len(awayNumbers) != 10 {
		return 0
	}

	homeDigit := homeScore % 10
	awayDigit := awayScore % 10

	// Find positions where these digits appear
	homePos := -1
	awayPos := -1
	for i, n := range homeNumbers {
		if n == homeDigit {
			homePos = i
			break
		}
	}
	for i, n := range awayNumbers {
		if n == awayDigit {
			awayPos = i
			break
		}
	}

	if homePos == -1 || awayPos == -1 {
		return 0
	}

	// Convert position to square ID based on grid type
	switch gridType {
	case GridTypeStd25:
		// 5x5 grid: each square covers 2x2 positions
		homePos5 := homePos / 2
		awayPos5 := awayPos / 2
		return (awayPos5 * 5) + homePos5 + 1
	case GridTypeStd50:
		// 5x10 grid: each square covers 2 away positions, 1 home position
		awayPos5 := awayPos / 2
		return (awayPos5 * 10) + homePos + 1
	default:
		// 10x10 grid (std100, roll100)
		return (awayPos * 10) + homePos + 1
	}
}

// WinningSquaresResult contains the winning squares for each applicable period
type WinningSquaresResult struct {
	Squares map[NumberSetType]int `json:"squares"`
}

// WinningPeriodInfo contains information about a winning period for a square
type WinningPeriodInfo struct {
	Period       NumberSetType `json:"period"`
	Label        string        `json:"label"`
	HomeScore    int           `json:"homeScore"`
	AwayScore    int           `json:"awayScore"`
	HomeTeamName string        `json:"homeTeamName"`
	AwayTeamName string        `json:"awayTeamName"`
}

// GetWinningSquares returns winning squares for each applicable period
// based on the grid's number configuration and the event's scores
// Only returns winning squares for periods that are complete
func GetWinningSquares(event *BDLEvent, config NumberSetConfig, gridType GridType, homeNumbers, awayNumbers []int, numberSets map[NumberSetType]*GridNumberSet) *WinningSquaresResult {
	result := &WinningSquaresResult{
		Squares: make(map[NumberSetType]int),
	}

	setTypes := GetSetTypes(config)
	if setTypes == nil {
		return result
	}

	for _, setType := range setTypes {
		// Only include winning squares for completed periods
		if !event.IsPeriodComplete(setType) {
			continue
		}

		homeScore, awayScore := event.ScoreForPeriod(setType)
		if homeScore == nil || awayScore == nil {
			continue // Score not available yet
		}

		var homeNums, awayNums []int

		// For standard config, use the legacy homeNumbers/awayNumbers
		if config == NumberSetConfigStandard {
			homeNums = homeNumbers
			awayNums = awayNumbers
		} else {
			// For multi-set configs, try to use the appropriate number set
			ns, ok := numberSets[setType]
			if ok && ns.HasNumbers() {
				homeNums = ns.HomeNumbers()
				awayNums = ns.AwayNumbers()
			} else {
				// Fall back to legacy numbers if number sets don't exist
				// This supports grids that had numbers drawn before setting a payout config
				homeNums = homeNumbers
				awayNums = awayNumbers
			}
		}

		if len(homeNums) != 10 || len(awayNums) != 10 {
			continue
		}

		squareID := CalculateWinningSquare(*homeScore, *awayScore, homeNums, awayNums, gridType)
		if squareID > 0 {
			result.Squares[setType] = squareID
		}
	}

	return result
}

// GetGridWinningSquares is a convenience method that calculates winning squares for a grid
func (g *Grid) GetGridWinningSquares(event *BDLEvent, config NumberSetConfig, gridType GridType) *WinningSquaresResult {
	return GetWinningSquares(event, config, gridType, g.HomeNumbers(), g.AwayNumbers(), g.NumberSets())
}

// GetWinningPeriodsForSquare returns the winning period information for a specific square.
// It returns the periods that the square won along with their scores.
func GetWinningPeriodsForSquare(squareID int, winningSquares *WinningSquaresResult, event *BDLEvent, homeTeamName, awayTeamName string) []WinningPeriodInfo {
	if squareID <= 0 || winningSquares == nil || event == nil {
		return nil
	}

	// Define the order for sorting periods
	periodOrder := map[NumberSetType]int{
		NumberSetTypeQ1:    1,
		NumberSetTypeHalf:  2,
		NumberSetTypeQ2:    3,
		NumberSetTypeQ3:    4,
		NumberSetTypeFinal: 5,
		NumberSetTypeAll:   6,
		NumberSetTypeQ4:    7,
	}

	var results []WinningPeriodInfo

	for period, winnerSquareID := range winningSquares.Squares {
		if winnerSquareID == squareID {
			homeScore, awayScore := event.ScoreForPeriod(period)
			if homeScore != nil && awayScore != nil {
				results = append(results, WinningPeriodInfo{
					Period:       period,
					Label:        period.LongLabel(),
					HomeScore:    *homeScore,
					AwayScore:    *awayScore,
					HomeTeamName: homeTeamName,
					AwayTeamName: awayTeamName,
				})
			}
		}
	}

	// Sort results by period order
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if periodOrder[results[i].Period] > periodOrder[results[j].Period] {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}
