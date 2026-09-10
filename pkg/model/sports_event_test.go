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
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/onsi/gomega"
)

func TestSportsEventJSON(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	event := &SportsEvent{
		ID:         1,
		ESPNID:     "12345",
		League:     SportsLeagueNBA,
		HomeTeamID: "home-1",
		AwayTeamID: "away-1",
		EventDate:  time.Now(),
		Season:     2024,
		Status:     SportsEventStatusScheduled,
	}

	json := event.JSON()
	g.Expect(json.ID).Should(gomega.Equal(int64(1)))
	g.Expect(json.ESPNID).Should(gomega.Equal("12345"))
	g.Expect(json.League).Should(gomega.Equal(SportsLeagueNBA))
	g.Expect(json.HomeTeamID).Should(gomega.Equal("home-1"))
	g.Expect(json.AwayTeamID).Should(gomega.Equal("away-1"))
	g.Expect(json.Status).Should(gomega.Equal(SportsEventStatusScheduled))
}

func TestSportsEventJSONWithVenue(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	venue := "Madison Square Garden"
	event := &SportsEvent{
		ID:         1,
		ESPNID:     "12345",
		League:     SportsLeagueNBA,
		HomeTeamID: "home-1",
		AwayTeamID: "away-1",
		EventDate:  time.Now(),
		Season:     2024,
		Status:     SportsEventStatusScheduled,
		Venue:      &venue,
	}

	json := event.JSON()
	g.Expect(json.Venue).Should(gomega.Equal("Madison Square Garden"))
}

func TestSportsEventJSONWithName(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	name := "Super Bowl LVIII"
	event := &SportsEvent{
		ID:         1,
		ESPNID:     "12345",
		League:     SportsLeagueNFL,
		HomeTeamID: "home-1",
		AwayTeamID: "away-1",
		EventDate:  time.Now(),
		Season:     2024,
		Status:     SportsEventStatusScheduled,
		Name:       &name,
	}

	json := event.JSON()
	g.Expect(json.Name).Should(gomega.Equal("Super Bowl LVIII"))
}

func TestSportsEventJSONWithNilName(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	event := &SportsEvent{
		ID:         1,
		ESPNID:     "12345",
		League:     SportsLeagueNFL,
		HomeTeamID: "home-1",
		AwayTeamID: "away-1",
		EventDate:  time.Now(),
		Season:     2024,
		Status:     SportsEventStatusScheduled,
		Name:       nil,
	}

	json := event.JSON()
	g.Expect(json.Name).Should(gomega.Equal(""))
}

func TestSportsEventJSONWithTeams(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	homeTeam := &SportsTeam{
		ID:           "home-1",
		League:       SportsLeagueNBA,
		Name:         "Knicks",
		FullName:     "New York Knicks",
		Abbreviation: "NYK",
	}
	awayTeam := &SportsTeam{
		ID:           "away-1",
		League:       SportsLeagueNBA,
		Name:         "Lakers",
		FullName:     "Los Angeles Lakers",
		Abbreviation: "LAL",
	}

	event := &SportsEvent{
		ID:         1,
		ESPNID:     "12345",
		League:     SportsLeagueNBA,
		HomeTeamID: "home-1",
		AwayTeamID: "away-1",
		EventDate:  time.Now(),
		Season:     2024,
		Status:     SportsEventStatusScheduled,
		homeTeam:   homeTeam,
		awayTeam:   awayTeam,
	}

	json := event.JSON()
	g.Expect(json.HomeTeam).ShouldNot(gomega.BeNil())
	g.Expect(json.HomeTeam.Name).Should(gomega.Equal("Knicks"))
	g.Expect(json.AwayTeam).ShouldNot(gomega.BeNil())
	g.Expect(json.AwayTeam.Name).Should(gomega.Equal("Lakers"))
}

func TestSportsEventTeamAccessors(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	event := &SportsEvent{}

	g.Expect(event.HomeTeam()).Should(gomega.BeNil())
	g.Expect(event.AwayTeam()).Should(gomega.BeNil())

	homeTeam := &SportsTeam{ID: "home"}
	awayTeam := &SportsTeam{ID: "away"}

	event.SetHomeTeam(homeTeam)
	event.SetAwayTeam(awayTeam)

	g.Expect(event.HomeTeam()).Should(gomega.Equal(homeTeam))
	g.Expect(event.AwayTeam()).Should(gomega.Equal(awayTeam))
}

func TestSportsEventHalfScores(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	event := &SportsEvent{}

	// With nil quarters
	g.Expect(event.HomeHalfScore()).Should(gomega.BeNil())
	g.Expect(event.AwayHalfScore()).Should(gomega.BeNil())

	// With Q1 only
	q1 := 25
	event.HomeQ1 = &q1
	event.AwayQ1 = &q1
	g.Expect(event.HomeHalfScore()).Should(gomega.BeNil())
	g.Expect(event.AwayHalfScore()).Should(gomega.BeNil())

	// With Q1 and Q2
	q2 := 30
	event.HomeQ2 = &q2
	event.AwayQ2 = &q2
	g.Expect(*event.HomeHalfScore()).Should(gomega.Equal(55))
	g.Expect(*event.AwayHalfScore()).Should(gomega.Equal(55))
}

func TestSportsEventQ3CumulativeScores(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	event := &SportsEvent{}

	// With nil quarters
	g.Expect(event.HomeQ3CumulativeScore()).Should(gomega.BeNil())
	g.Expect(event.AwayQ3CumulativeScore()).Should(gomega.BeNil())

	// With Q1, Q2, Q3
	q1 := 25
	q2 := 30
	q3 := 20
	event.HomeQ1 = &q1
	event.HomeQ2 = &q2
	event.HomeQ3 = &q3
	event.AwayQ1 = &q1
	event.AwayQ2 = &q2
	event.AwayQ3 = &q3

	g.Expect(*event.HomeQ3CumulativeScore()).Should(gomega.Equal(75))
	g.Expect(*event.AwayQ3CumulativeScore()).Should(gomega.Equal(75))
}

func TestSportsEventIsPeriodComplete(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	event := &SportsEvent{
		Status: SportsEventStatusScheduled,
	}

	// Scheduled game, no period complete
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ1)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ2)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ3)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ4)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeFinal)).Should(gomega.BeFalse())

	// In progress, period 2 (Q1 complete, Q2 in progress)
	event.Status = SportsEventStatusInProgress
	period := 2
	event.Period = &period
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ1)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ2)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeFalse())

	// In progress, period 3 (Q2/Half complete)
	period = 3
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ2)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ3)).Should(gomega.BeFalse())

	// Final
	event.Status = SportsEventStatusFinal
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ1)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ2)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ3)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ4)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeFinal)).Should(gomega.BeTrue())

	// End of 1st Quarter: Q1 complete, Q2/Half not yet
	event.Status = SportsEventStatusInProgress
	period = 1
	event.Period = &period
	detail := "End of 1st Quarter"
	event.StatusDetail = &detail
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ1)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ2)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeFalse())

	// Halftime: Q2/Half complete, Q3 not yet
	period = 2
	detail = "Halftime"
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ1)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ2)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ3)).Should(gomega.BeFalse())

	// End of 2nd Quarter: Q2/Half complete
	detail = "End of 2nd Quarter"
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ2)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ3)).Should(gomega.BeFalse())

	// End of 3rd Quarter: Q3 complete, Q4 not yet
	period = 3
	detail = "End of 3rd Quarter"
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ3)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeQ4)).Should(gomega.BeFalse())
	g.Expect(event.IsPeriodComplete(NumberSetTypeFinal)).Should(gomega.BeFalse())
}

func TestSportsEventIsPeriodCompleteNCAAB(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	intPtr := func(i int) *int { return &i }
	strPtr := func(s string) *string { return &s }

	// NCAAB game at halftime (period=1, statusDetail="Halftime")
	event := &SportsEvent{
		League:       SportsLeagueNCAAB,
		Status:       SportsEventStatusInProgress,
		Period:       intPtr(1),
		StatusDetail: strPtr("Halftime"),
	}
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeFinal)).Should(gomega.BeFalse())

	// NCAAB game during 2nd half (period=2) - Half should be complete
	event.Period = intPtr(2)
	event.StatusDetail = nil
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeFinal)).Should(gomega.BeFalse())

	// NCAAB game at end of 1st half (period=1, "End of 1st Half")
	event.Period = intPtr(1)
	event.StatusDetail = strPtr("End of 1st Half")
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeTrue())

	// NCAAB game in 1st half (period=1, no special status) - Half NOT complete
	event.Period = intPtr(1)
	event.StatusDetail = nil
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeFalse())

	// NCAAB game final
	event.Status = SportsEventStatusFinal
	g.Expect(event.IsPeriodComplete(NumberSetTypeHalf)).Should(gomega.BeTrue())
	g.Expect(event.IsPeriodComplete(NumberSetTypeFinal)).Should(gomega.BeTrue())
}

func TestSportsEventScoreForPeriodNCAAB(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// NCAAB: Q1 = 1st half score (35), Q2 = 2nd half score (40), total = 75
	q1, q2, total := 35, 40, 75
	event := &SportsEvent{
		League:    SportsLeagueNCAAB,
		HomeQ1:    &q1,
		HomeQ2:    &q2,
		HomeScore: &total,
		AwayQ1:    &q1,
		AwayQ2:    &q2,
		AwayScore: &total,
	}

	// Half score for NCAAB should be Q1 only (the full 1st-half linescore)
	homeHalf, awayHalf := event.ScoreForPeriod(NumberSetTypeHalf)
	g.Expect(*homeHalf).Should(gomega.Equal(35))
	g.Expect(*awayHalf).Should(gomega.Equal(35))

	// Final score should still be the total
	homeFinal, awayFinal := event.ScoreForPeriod(NumberSetTypeFinal)
	g.Expect(*homeFinal).Should(gomega.Equal(75))
	g.Expect(*awayFinal).Should(gomega.Equal(75))
}

func TestSportsEventScoreForPeriod(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	q1, q2, q3, total := 25, 30, 20, 100
	event := &SportsEvent{
		HomeQ1:    &q1,
		HomeQ2:    &q2,
		HomeQ3:    &q3,
		HomeScore: &total,
		AwayQ1:    &q1,
		AwayQ2:    &q2,
		AwayQ3:    &q3,
		AwayScore: &total,
	}

	homeQ1, awayQ1 := event.ScoreForPeriod(NumberSetTypeQ1)
	g.Expect(*homeQ1).Should(gomega.Equal(25))
	g.Expect(*awayQ1).Should(gomega.Equal(25))

	homeHalf, awayHalf := event.ScoreForPeriod(NumberSetTypeHalf)
	g.Expect(*homeHalf).Should(gomega.Equal(55))
	g.Expect(*awayHalf).Should(gomega.Equal(55))

	homeQ3, awayQ3 := event.ScoreForPeriod(NumberSetTypeQ3)
	g.Expect(*homeQ3).Should(gomega.Equal(75))
	g.Expect(*awayQ3).Should(gomega.Equal(75))

	homeFinal, awayFinal := event.ScoreForPeriod(NumberSetTypeFinal)
	g.Expect(*homeFinal).Should(gomega.Equal(100))
	g.Expect(*awayFinal).Should(gomega.Equal(100))
}

func TestSportsEventNewSportsEvent(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	m := New(nil)
	event := m.NewSportsEvent()

	g.Expect(event).ShouldNot(gomega.BeNil())
	g.Expect(event.Status).Should(gomega.Equal(SportsEventStatusScheduled))
	g.Expect(event.model).Should(gomega.Equal(m))
}

// Integration tests require database connection
func TestSearchSportsEvents(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create test teams
	homeTeam := &SportsTeam{
		ID:           "test-home-" + randString(),
		League:       SportsLeagueNCAAB,
		Name:         "Bulls",
		FullName:     "Buffalo Bulls",
		Abbreviation: "BUFF",
	}
	awayTeam := &SportsTeam{
		ID:           "test-away-" + randString(),
		League:       SportsLeagueNCAAB,
		Name:         "Cardinals",
		FullName:     "Ball State Cardinals",
		Abbreviation: "BALL",
	}

	err := m.UpsertSportsTeam(ctx, nil, homeTeam)
	g.Expect(err).Should(gomega.Succeed())
	err = m.UpsertSportsTeam(ctx, nil, awayTeam)
	g.Expect(err).Should(gomega.Succeed())

	// Create test event
	event := m.NewSportsEvent()
	event.ESPNID = "test-event-" + randString()
	event.League = SportsLeagueNCAAB
	event.HomeTeamID = homeTeam.ID
	event.AwayTeamID = awayTeam.ID
	event.EventDate = time.Now().Add(24 * time.Hour)
	event.Season = 2024
	event.Status = SportsEventStatusScheduled

	err = m.UpsertSportsEvent(ctx, nil, event)
	g.Expect(err).Should(gomega.Succeed())

	// Search by home team name
	events, total, err := m.SearchSportsEvents(ctx, SportsLeagueNCAAB, "", "buffalo", 0, 50)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(total).Should(gomega.BeNumerically(">=", 1))
	g.Expect(len(events)).Should(gomega.BeNumerically(">=", 1))

	// Search by abbreviation
	_, total, err = m.SearchSportsEvents(ctx, SportsLeagueNCAAB, "", "BUFF", 0, 50)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(total).Should(gomega.BeNumerically(">=", 1))

	// Search by away team
	_, total, err = m.SearchSportsEvents(ctx, SportsLeagueNCAAB, "", "ball state", 0, 50)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(total).Should(gomega.BeNumerically(">=", 1))

	// Search with no results
	events, total, err = m.SearchSportsEvents(ctx, SportsLeagueNCAAB, "", "nonexistent-team-xyz123", 0, 50)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(total).Should(gomega.Equal(int64(0)))
	g.Expect(len(events)).Should(gomega.Equal(0))
}

func TestLinkableSportsEventsWithTotal(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create test teams
	homeTeam := &SportsTeam{
		ID:           "test-home-" + randString(),
		League:       SportsLeagueNBA,
		Name:         "Test Home",
		FullName:     "Test Home Team",
		Abbreviation: "TSH",
	}
	awayTeam := &SportsTeam{
		ID:           "test-away-" + randString(),
		League:       SportsLeagueNBA,
		Name:         "Test Away",
		FullName:     "Test Away Team",
		Abbreviation: "TSA",
	}

	err := m.UpsertSportsTeam(ctx, nil, homeTeam)
	g.Expect(err).Should(gomega.Succeed())
	err = m.UpsertSportsTeam(ctx, nil, awayTeam)
	g.Expect(err).Should(gomega.Succeed())

	// Create test event
	event := m.NewSportsEvent()
	event.ESPNID = "test-event-" + randString()
	event.League = SportsLeagueNBA
	event.HomeTeamID = homeTeam.ID
	event.AwayTeamID = awayTeam.ID
	event.EventDate = time.Now().Add(24 * time.Hour)
	event.Season = 2024
	event.Status = SportsEventStatusScheduled

	err = m.UpsertSportsEvent(ctx, nil, event)
	g.Expect(err).Should(gomega.Succeed())

	// Get linkable events with total
	events, total, err := m.LinkableSportsEventsWithTotal(ctx, SportsLeagueNBA, 0, 50)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(total).Should(gomega.BeNumerically(">=", 1))
	g.Expect(len(events)).Should(gomega.BeNumerically(">=", 1))

	// Test pagination
	events, total, err = m.LinkableSportsEventsWithTotal(ctx, SportsLeagueNBA, 0, 1)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(total).Should(gomega.BeNumerically(">=", 1))
	g.Expect(len(events)).Should(gomega.BeNumerically("<=", 1))
}

func TestFinalizeStaleEvents(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// Create test teams
	homeTeam := &SportsTeam{
		ID:           "test-home-" + randString(),
		League:       SportsLeagueNBA,
		Name:         "Test Home",
		FullName:     "Test Home Team",
		Abbreviation: "TSH",
	}
	awayTeam := &SportsTeam{
		ID:           "test-away-" + randString(),
		League:       SportsLeagueNBA,
		Name:         "Test Away",
		FullName:     "Test Away Team",
		Abbreviation: "TSA",
	}

	err := m.UpsertSportsTeam(ctx, nil, homeTeam)
	g.Expect(err).Should(gomega.Succeed())
	err = m.UpsertSportsTeam(ctx, nil, awayTeam)
	g.Expect(err).Should(gomega.Succeed())

	// Create a stale in_progress event (event_date > 1 day ago)
	staleEvent := m.NewSportsEvent()
	staleEvent.ESPNID = "test-stale-" + randString()
	staleEvent.League = SportsLeagueNBA
	staleEvent.HomeTeamID = homeTeam.ID
	staleEvent.AwayTeamID = awayTeam.ID
	staleEvent.EventDate = time.Now().Add(-48 * time.Hour)
	staleEvent.Season = 2024
	staleEvent.Status = SportsEventStatusInProgress
	staleEvent.HomeScore = intPtr(50)
	staleEvent.AwayScore = intPtr(45)
	staleEvent.HomeQ1 = intPtr(10)
	staleEvent.HomeQ2 = intPtr(15)
	staleEvent.AwayQ1 = intPtr(12)
	staleEvent.AwayQ2 = intPtr(11)
	staleEvent.Period = intPtr(2)
	staleEvent.Clock = strPtr("0:00")
	staleEvent.StatusDetail = strPtr("Halftime")

	err = m.UpsertSportsEvent(ctx, nil, staleEvent)
	g.Expect(err).Should(gomega.Succeed())

	// Create a recent in_progress event (event_date within 1 day)
	recentEvent := m.NewSportsEvent()
	recentEvent.ESPNID = "test-recent-" + randString()
	recentEvent.League = SportsLeagueNBA
	recentEvent.HomeTeamID = homeTeam.ID
	recentEvent.AwayTeamID = awayTeam.ID
	recentEvent.EventDate = time.Now().Add(-12 * time.Hour)
	recentEvent.Season = 2024
	recentEvent.Status = SportsEventStatusInProgress
	recentEvent.HomeScore = intPtr(30)
	recentEvent.AwayScore = intPtr(28)
	recentEvent.HomeQ1 = intPtr(8)
	recentEvent.AwayQ1 = intPtr(7)
	recentEvent.Period = intPtr(1)
	recentEvent.Clock = strPtr("5:00")
	recentEvent.StatusDetail = strPtr("End of 1st Quarter")

	err = m.UpsertSportsEvent(ctx, nil, recentEvent)
	g.Expect(err).Should(gomega.Succeed())

	// Finalize stale events
	count, err := m.FinalizeStaleEvents(ctx)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(count).Should(gomega.BeNumerically(">=", 1))

	// Verify the stale event is now final
	staleResult, err := m.SportsEventByESPNID(ctx, staleEvent.ESPNID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(staleResult).ShouldNot(gomega.BeNil())
	g.Expect(staleResult.Status).Should(gomega.Equal(SportsEventStatusFinal))
	g.Expect(staleResult.HomeScore).Should(gomega.BeNil())
	g.Expect(staleResult.AwayScore).Should(gomega.BeNil())
	g.Expect(staleResult.HomeQ1).Should(gomega.BeNil())
	g.Expect(staleResult.HomeQ2).Should(gomega.BeNil())
	g.Expect(staleResult.AwayQ1).Should(gomega.BeNil())
	g.Expect(staleResult.AwayQ2).Should(gomega.BeNil())
	g.Expect(staleResult.Period).Should(gomega.BeNil())
	g.Expect(staleResult.Clock).Should(gomega.BeNil())
	g.Expect(staleResult.StatusDetail).Should(gomega.BeNil())

	// Verify the recent event is still in_progress with scores intact
	recentResult, err := m.SportsEventByESPNID(ctx, recentEvent.ESPNID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(recentResult).ShouldNot(gomega.BeNil())
	g.Expect(recentResult.Status).Should(gomega.Equal(SportsEventStatusInProgress))
	g.Expect(recentResult.HomeScore).Should(gomega.Equal(intPtr(30)))
	g.Expect(recentResult.AwayScore).Should(gomega.Equal(intPtr(28)))
	g.Expect(recentResult.HomeQ1).Should(gomega.Equal(intPtr(8)))
	g.Expect(recentResult.AwayQ1).Should(gomega.Equal(intPtr(7)))
	g.Expect(recentResult.Period).Should(gomega.Equal(intPtr(1)))
}

// createTestGridLinkedToEvent creates teams, an event, a user, a pool, and a grid linked to the event.
// If homeColor/awayColor are non-empty, the team colors are set.
// Returns the event, pool (for reloading grids), and the grid.
func createTestGridLinkedToEvent(t *testing.T, m *Model, ctx context.Context, homeFullName, awayFullName, homeColor, awayColor string) (*SportsEvent, *Pool, *Grid) {
	t.Helper()
	g := gomega.NewWithT(t)

	homeTeam := &SportsTeam{
		ID:             "test-home-" + randString(),
		League:         SportsLeagueNFL,
		Name:           "Home",
		FullName:       homeFullName,
		Abbreviation:   "HOM",
		Color:          strPtr(homeColor),
		AlternateColor: strPtr(homeColor + "aa"),
	}
	awayTeam := &SportsTeam{
		ID:             "test-away-" + randString(),
		League:         SportsLeagueNFL,
		Name:           "Away",
		FullName:       awayFullName,
		Abbreviation:   "AWY",
		Color:          strPtr(awayColor),
		AlternateColor: strPtr(awayColor + "bb"),
	}
	if homeColor == "" {
		homeTeam.Color = nil
		homeTeam.AlternateColor = nil
	}
	if awayColor == "" {
		awayTeam.Color = nil
		awayTeam.AlternateColor = nil
	}

	err := m.UpsertSportsTeam(ctx, nil, homeTeam)
	g.Expect(err).Should(gomega.Succeed())
	err = m.UpsertSportsTeam(ctx, nil, awayTeam)
	g.Expect(err).Should(gomega.Succeed())

	event := m.NewSportsEvent()
	event.ESPNID = "test-event-" + randString()
	event.League = SportsLeagueNFL
	event.HomeTeamID = homeTeam.ID
	event.AwayTeamID = awayTeam.ID
	event.EventDate = time.Now().Add(24 * time.Hour)
	event.Season = 2024
	event.Status = SportsEventStatusScheduled
	err = m.UpsertSportsEvent(ctx, nil, event)
	g.Expect(err).Should(gomega.Succeed())

	user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
	g.Expect(err).Should(gomega.Succeed())

	pool, err := m.NewPool(ctx, user.ID, "Sync Test Pool "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
	g.Expect(err).Should(gomega.Succeed())

	grids, err := pool.Grids(ctx, 0, 10)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(grids)).Should(gomega.BeNumerically(">=", 1))

	grid := grids[0]
	grid.SetBDLEvent(event)
	err = grid.Save(ctx)
	g.Expect(err).Should(gomega.Succeed())

	return event, pool, grid
}

func TestSyncGridsFromEvent(t *testing.T) {
	ensureIntegration(t)
	m := New(getDB())
	ctx := context.Background()

	t.Run("updates team names when they differ", func(t *testing.T) {
		g := gomega.NewWithT(t)

		event, pool, grid := createTestGridLinkedToEvent(t, m, ctx, "Kansas City Chiefs", "San Francisco 49ers", "e31837", "aa0000")

		// Grid starts with nil team names from pool creation
		// SyncGridsFromEvent should update them
		count, err := m.SyncGridsFromEvent(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(count).Should(gomega.BeNumerically(">=", 1))

		// Reload the grid and verify names
		grid, err = pool.GridByID(ctx, grid.ID())
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(grid.HomeTeamName()).Should(gomega.Equal("Kansas City Chiefs"))
		g.Expect(grid.AwayTeamName()).Should(gomega.Equal("San Francisco 49ers"))
	})

	t.Run("sets colors when grid colors are null", func(t *testing.T) {
		g := gomega.NewWithT(t)

		event, _, grid := createTestGridLinkedToEvent(t, m, ctx, "Green Bay Packers", "Chicago Bears", "203731", "0b162a")

		count, err := m.SyncGridsFromEvent(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(count).Should(gomega.BeNumerically(">=", 1))

		// Reload settings and verify colors were set
		err = grid.LoadSettings(ctx)
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(grid.Settings().HomeTeamColor1()).Should(gomega.Equal("#203731"))
		g.Expect(grid.Settings().AwayTeamColor1()).Should(gomega.Equal("#0b162a"))
	})

	t.Run("updates colors when teams change", func(t *testing.T) {
		g := gomega.NewWithT(t)

		event, _, grid := createTestGridLinkedToEvent(t, m, ctx, "Dallas Cowboys", "New York Giants", "003594", "0b2265")

		// First set colors via the sync
		_, err := m.SyncGridsFromEvent(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())

		// Verify initial colors
		err = grid.LoadSettings(ctx)
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(grid.Settings().HomeTeamColor1()).Should(gomega.Equal("#003594"))
		g.Expect(grid.Settings().AwayTeamColor1()).Should(gomega.Equal("#0b2265"))

		// Now change the event's teams to new teams with different colors
		newHomeTeam := &SportsTeam{
			ID:             "test-new-home-" + randString(),
			League:         SportsLeagueNFL,
			Name:           "Eagles",
			FullName:       "Philadelphia Eagles",
			Abbreviation:   "PHI",
			Color:          strPtr("004c54"),
			AlternateColor: strPtr("a5acaf"),
		}
		newAwayTeam := &SportsTeam{
			ID:             "test-new-away-" + randString(),
			League:         SportsLeagueNFL,
			Name:           "Commanders",
			FullName:       "Washington Commanders",
			Abbreviation:   "WSH",
			Color:          strPtr("5a1414"),
			AlternateColor: strPtr("ffb612"),
		}
		err = m.UpsertSportsTeam(ctx, nil, newHomeTeam)
		g.Expect(err).Should(gomega.Succeed())
		err = m.UpsertSportsTeam(ctx, nil, newAwayTeam)
		g.Expect(err).Should(gomega.Succeed())

		// Update event to point to new teams
		event.HomeTeamID = newHomeTeam.ID
		event.AwayTeamID = newAwayTeam.ID
		err = m.UpsertSportsEvent(ctx, nil, event)
		g.Expect(err).Should(gomega.Succeed())

		// Sync again - colors should update to match new teams
		count, err := m.SyncGridsFromEvent(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(count).Should(gomega.BeNumerically(">=", 1))

		// Verify colors updated to new team colors
		err = grid.LoadSettings(ctx)
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(grid.Settings().HomeTeamColor1()).Should(gomega.Equal("#004c54"))
		g.Expect(grid.Settings().AwayTeamColor1()).Should(gomega.Equal("#5a1414"))
	})

	t.Run("no update when names already match", func(t *testing.T) {
		g := gomega.NewWithT(t)

		event, _, _ := createTestGridLinkedToEvent(t, m, ctx, "Miami Dolphins", "Buffalo Bills", "008e97", "00338d")

		// First sync to set names and colors
		_, err := m.SyncGridsFromEvent(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())

		// Second sync should return 0 since nothing changed
		count, err := m.SyncGridsFromEvent(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(count).Should(gomega.Equal(int64(0)))
	})

	t.Run("no update when teams are TBD (null colors)", func(t *testing.T) {
		g := gomega.NewWithT(t)

		// Create event with TBD teams (empty team IDs with no colors)
		event, _, _ := createTestGridLinkedToEvent(t, m, ctx, "TBD", "TBD", "", "")

		// First sync sets names to TBD
		_, err := m.SyncGridsFromEvent(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())

		// Colors should not be set since team colors are null
		// (the colors query requires st_home.color IS NOT NULL)
		// Second sync should only return 0 for names (already matching)
		count, err := m.SyncGridsFromEvent(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(count).Should(gomega.Equal(int64(0)))
	})

	t.Run("updates multiple grids linked to same event", func(t *testing.T) {
		g := gomega.NewWithT(t)

		// Create event with teams
		homeTeam := &SportsTeam{
			ID:             "test-home-multi-" + randString(),
			League:         SportsLeagueNFL,
			Name:           "Eagles",
			FullName:       "Philadelphia Eagles",
			Abbreviation:   "PHI",
			Color:          strPtr("004c54"),
			AlternateColor: strPtr("a5acaf"),
		}
		awayTeam := &SportsTeam{
			ID:             "test-away-multi-" + randString(),
			League:         SportsLeagueNFL,
			Name:           "Cowboys",
			FullName:       "Dallas Cowboys",
			Abbreviation:   "DAL",
			Color:          strPtr("003594"),
			AlternateColor: strPtr("869397"),
		}
		err := m.UpsertSportsTeam(ctx, nil, homeTeam)
		g.Expect(err).Should(gomega.Succeed())
		err = m.UpsertSportsTeam(ctx, nil, awayTeam)
		g.Expect(err).Should(gomega.Succeed())

		event := m.NewSportsEvent()
		event.ESPNID = "test-multi-event-" + randString()
		event.League = SportsLeagueNFL
		event.HomeTeamID = homeTeam.ID
		event.AwayTeamID = awayTeam.ID
		event.EventDate = time.Now().Add(24 * time.Hour)
		event.Season = 2024
		event.Status = SportsEventStatusScheduled
		err = m.UpsertSportsEvent(ctx, nil, event)
		g.Expect(err).Should(gomega.Succeed())

		// Create two grids linked to the same event
		type gridWithPool struct {
			pool *Pool
			grid *Grid
		}
		var items []gridWithPool
		for i := 0; i < 2; i++ {
			user, err := m.GetUser(ctx, IssuerAuth0, "auth0|"+randString())
			g.Expect(err).Should(gomega.Succeed())
			pool, err := m.NewPool(ctx, user.ID, "Multi Grid Pool "+randString(), GridTypeStd100, "password", NumberSetConfigStandard)
			g.Expect(err).Should(gomega.Succeed())
			poolGrids, err := pool.Grids(ctx, 0, 10)
			g.Expect(err).Should(gomega.Succeed())
			grid := poolGrids[0]
			grid.SetBDLEvent(event)
			err = grid.Save(ctx)
			g.Expect(err).Should(gomega.Succeed())
			items = append(items, gridWithPool{pool: pool, grid: grid})
		}

		// Sync should update both grids
		count, err := m.SyncGridsFromEvent(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())
		// At minimum 2 name updates + 2 color updates = 4
		g.Expect(count).Should(gomega.BeNumerically(">=", 4))

		// Verify both grids were updated
		for _, item := range items {
			grid, err := item.pool.GridByID(ctx, item.grid.ID())
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(grid.HomeTeamName()).Should(gomega.Equal("Philadelphia Eagles"))
			g.Expect(grid.AwayTeamName()).Should(gomega.Equal("Dallas Cowboys"))

			err = grid.LoadSettings(ctx)
			g.Expect(err).Should(gomega.Succeed())
			g.Expect(grid.Settings().HomeTeamColor1()).Should(gomega.Equal("#004c54"))
			g.Expect(grid.Settings().AwayTeamColor1()).Should(gomega.Equal("#003594"))
		}
	})
}

func TestPoolTokensByEventID(t *testing.T) {
	ensureIntegration(t)
	m := New(getDB())
	ctx := context.Background()

	t.Run("returns tokens for pools with linked active grids", func(t *testing.T) {
		g := gomega.NewWithT(t)

		event, pool, _ := createTestGridLinkedToEvent(t, m, ctx, "Test Home", "Test Away", "ff0000", "0000ff")

		tokens, err := m.PoolTokensByEventID(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(tokens).Should(gomega.ContainElement(pool.Token()))
	})

	t.Run("returns empty slice when no grids are linked", func(t *testing.T) {
		g := gomega.NewWithT(t)

		// Create an event with no linked grids
		homeTeam := &SportsTeam{
			ID:           "test-home-" + randString(),
			League:       SportsLeagueNFL,
			Name:         "Unlinked Home",
			FullName:     "Unlinked Home Team",
			Abbreviation: "ULH",
		}
		awayTeam := &SportsTeam{
			ID:           "test-away-" + randString(),
			League:       SportsLeagueNFL,
			Name:         "Unlinked Away",
			FullName:     "Unlinked Away Team",
			Abbreviation: "ULA",
		}
		err := m.UpsertSportsTeam(ctx, nil, homeTeam)
		g.Expect(err).Should(gomega.Succeed())
		err = m.UpsertSportsTeam(ctx, nil, awayTeam)
		g.Expect(err).Should(gomega.Succeed())

		event := m.NewSportsEvent()
		event.ESPNID = "test-unlinked-" + randString()
		event.League = SportsLeagueNFL
		event.HomeTeamID = homeTeam.ID
		event.AwayTeamID = awayTeam.ID
		event.EventDate = time.Now().Add(24 * time.Hour)
		event.Season = 2024
		event.Status = SportsEventStatusScheduled
		err = m.UpsertSportsEvent(ctx, nil, event)
		g.Expect(err).Should(gomega.Succeed())

		tokens, err := m.PoolTokensByEventID(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())
		g.Expect(tokens).Should(gomega.HaveLen(0))
	})

	t.Run("returns distinct tokens when multiple grids in same pool", func(t *testing.T) {
		g := gomega.NewWithT(t)

		event, pool, _ := createTestGridLinkedToEvent(t, m, ctx, "Multi Grid Home", "Multi Grid Away", "aabbcc", "ddeeff")

		tokens, err := m.PoolTokensByEventID(ctx, event.ID)
		g.Expect(err).Should(gomega.Succeed())

		// Count occurrences of this pool token
		count := 0
		for _, tok := range tokens {
			if tok == pool.Token() {
				count++
			}
		}
		g.Expect(count).Should(gomega.Equal(1))
	})
}

func TestNotifySportsEventUpdated(t *testing.T) {
	ensureIntegration(t)
	m := New(getDB())
	ctx := context.Background()

	g := gomega.NewWithT(t)

	// Just verify it doesn't error — the notification goes to any listeners
	err := m.NotifySportsEventUpdated(ctx, 12345)
	g.Expect(err).Should(gomega.Succeed())
}

// sportsEventMockRows returns a single-row result set matching sportsEventColumns
func sportsEventMockRows(now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "espn_id", "league", "name", "home_team_id", "away_team_id", "event_date", "season", "week", "postseason", "venue",
		"status", "status_detail", "period", "clock", "home_score", "away_score",
		"home_q1", "home_q2", "home_q3", "home_q4", "home_ot",
		"away_q1", "away_q2", "away_q3", "away_q4", "away_ot",
		"created", "modified", "last_synced",
	}).AddRow(
		int64(1), "espn-1", "nba", nil, "home-1", "away-1", now, 2024, nil, false, nil,
		"scheduled", nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil,
		now, now, now,
	)
}

func TestUpcomingSportsEventsComparesEventDateInUTC(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)

	mock.ExpectQuery(`SELECT .+ FROM sports_events WHERE league = \$1 AND status = 'scheduled' AND event_date >= \(NOW\(\) AT TIME ZONE 'utc'\)`).
		WithArgs("nba", 10).
		WillReturnRows(sportsEventMockRows(time.Now()))

	events, err := m.UpcomingSportsEvents(context.Background(), SportsLeagueNBA, 10)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(events).Should(gomega.HaveLen(1))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestEventsNeedingScoreUpdateComparesEventDateInUTC(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)

	mock.ExpectQuery(`SELECT .+ FROM sports_events ` +
		`WHERE \(status = 'in_progress' AND event_date >= \(NOW\(\) AT TIME ZONE 'utc'\) - INTERVAL '1 day'\) ` +
		`OR \(status = 'scheduled' AND event_date BETWEEN \(NOW\(\) AT TIME ZONE 'utc'\) AND \(NOW\(\) AT TIME ZONE 'utc'\) \+ INTERVAL '2 hours'\) ` +
		`OR \(status != 'final' AND event_date >= \(NOW\(\) AT TIME ZONE 'utc'\) - INTERVAL '1 day' AND event_date < \(NOW\(\) AT TIME ZONE 'utc'\)\)`).
		WillReturnRows(sportsEventMockRows(time.Now()))

	events, err := m.EventsNeedingScoreUpdate(context.Background())
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(events).Should(gomega.HaveLen(1))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestFinalizeStaleEventsComparesEventDateInUTC(t *testing.T) {
	g := gomega.NewWithT(t)

	db, mock, err := sqlmock.New()
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	defer db.Close()

	m := New(db)

	mock.ExpectExec(`UPDATE sports_events SET status = 'final', .+ WHERE status != 'final' AND event_date < \(NOW\(\) AT TIME ZONE 'utc'\) - INTERVAL '1 day'`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	count, err := m.FinalizeStaleEvents(context.Background())
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	g.Expect(count).Should(gomega.Equal(int64(1)))

	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func TestUpcomingSportsEventsForTeam(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	team := &SportsTeam{
		ID:           "test-team-" + randString(),
		League:       SportsLeagueNFL,
		Name:         "Chiefs",
		FullName:     "Kansas City Chiefs",
		Abbreviation: "KC",
	}
	opponent := &SportsTeam{
		ID:           "test-opp-" + randString(),
		League:       SportsLeagueNFL,
		Name:         "Bills",
		FullName:     "Buffalo Bills",
		Abbreviation: "BUF",
	}
	other := &SportsTeam{
		ID:           "test-other-" + randString(),
		League:       SportsLeagueNFL,
		Name:         "Broncos",
		FullName:     "Denver Broncos",
		Abbreviation: "DEN",
	}

	g.Expect(m.UpsertSportsTeam(ctx, nil, team)).Should(gomega.Succeed())
	g.Expect(m.UpsertSportsTeam(ctx, nil, opponent)).Should(gomega.Succeed())
	g.Expect(m.UpsertSportsTeam(ctx, nil, other)).Should(gomega.Succeed())

	newEvent := func(homeID, awayID string, when time.Time, season int, status SportsEventStatus) *SportsEvent {
		e := m.NewSportsEvent()
		e.ESPNID = "test-event-" + randString()
		e.League = SportsLeagueNFL
		e.HomeTeamID = homeID
		e.AwayTeamID = awayID
		e.EventDate = when
		e.Season = season
		e.Status = status
		g.Expect(m.UpsertSportsEvent(ctx, nil, e)).Should(gomega.Succeed())
		return e
	}

	future := time.Now().Add(24 * time.Hour)

	// should be returned (team is home, and away, in the current season)
	newEvent(team.ID, opponent.ID, future, 2025, SportsEventStatusScheduled)
	newEvent(opponent.ID, team.ID, future.Add(7*24*time.Hour), 2025, SportsEventStatusScheduled)

	// should be excluded
	newEvent(team.ID, opponent.ID, time.Now().Add(-24*time.Hour), 2025, SportsEventStatusScheduled) // in the past
	newEvent(team.ID, opponent.ID, future.Add(48*time.Hour), 2025, SportsEventStatusFinal)          // not scheduled
	newEvent(team.ID, opponent.ID, future.Add(365*24*time.Hour), 2026, SportsEventStatusScheduled)  // next season
	newEvent(opponent.ID, other.ID, future.Add(72*time.Hour), 2025, SportsEventStatusScheduled)     // different teams

	// a postponed game still carrying a stale season value must not hijack the
	// selection. The season is pinned to the season of the *next* upcoming
	// game, not to the lowest season that still has an upcoming game.
	newEvent(team.ID, opponent.ID, future.Add(14*24*time.Hour), 2024, SportsEventStatusScheduled)

	events, err := m.UpcomingSportsEventsForTeam(ctx, SportsLeagueNFL, team.ID)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(events).Should(gomega.HaveLen(2))

	for _, e := range events {
		g.Expect(e.Season).Should(gomega.Equal(2025))
		g.Expect(e.Status).Should(gomega.Equal(SportsEventStatusScheduled))
		g.Expect(e.HomeTeamID == team.ID || e.AwayTeamID == team.ID).Should(gomega.BeTrue())
	}

	// results are ordered by event date
	g.Expect(events[0].EventDate.Before(events[1].EventDate)).Should(gomega.BeTrue())

	// a team with no games returns an empty result
	events, err = m.UpcomingSportsEventsForTeam(ctx, SportsLeagueNFL, "test-nonexistent-"+randString())
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(events).Should(gomega.HaveLen(0))
}

func TestUpcomingPostseasonSportsEvents(t *testing.T) {
	if len(os.Getenv("INTEGRATION")) == 0 {
		t.Skip("skipping. to run, use -integration flag")
	}

	g := gomega.NewWithT(t)
	m := New(getDB())
	ctx := context.Background()

	// This query is league-wide rather than team-scoped, so it can't be
	// isolated with random team ids the way the other tests here are. WNBA is
	// used because no other test touches it, and its postseason rows are
	// cleared first so that a re-run against a persistent integration database
	// doesn't see the previous run's events.
	league := SportsLeagueWNBA

	// NCAAF is cleared for the same reason: the empty-league assertion at the
	// end of this test asserts on it, so it must not depend on no other test
	// (or a previous run) having seeded NCAAF postseason rows.
	emptyLeague := SportsLeagueNCAAF

	for _, l := range []SportsLeague{league, emptyLeague} {
		_, err := getDB().ExecContext(ctx, `DELETE FROM sports_events WHERE league = $1 AND postseason = TRUE`, l)
		g.Expect(err).Should(gomega.Succeed())
	}

	// Every unfilled bracket slot is a separate placeholder team as far as
	// ESPN is concerned, which is the whole reason this query ignores teams
	newTeam := func(name string) *SportsTeam {
		team := &SportsTeam{
			ID:           "test-post-" + randString(),
			League:       league,
			Name:         name,
			FullName:     name,
			Abbreviation: "TST",
		}
		g.Expect(m.UpsertSportsTeam(ctx, nil, team)).Should(gomega.Succeed())
		return team
	}

	tbd1 := newTeam("TBD")
	tbd2 := newTeam("TBD")
	tbd3 := newTeam("TBD")
	tbd4 := newTeam("TBD")
	real1 := newTeam("Liberty")

	newEvent := func(homeID, awayID string, when time.Time, season int, postseason bool, status SportsEventStatus) *SportsEvent {
		e := m.NewSportsEvent()
		e.ESPNID = "test-event-" + randString()
		e.League = league
		e.HomeTeamID = homeID
		e.AwayTeamID = awayID
		e.EventDate = when
		e.Season = season
		e.Postseason = postseason
		e.Status = status
		g.Expect(m.UpsertSportsEvent(ctx, nil, e)).Should(gomega.Succeed())
		return e
	}

	future := time.Now().Add(24 * time.Hour)

	// should be returned. Note that no two of these share a team, so an exact
	// team-id match could never have found all three
	first := newEvent(tbd1.ID, tbd2.ID, future, 2025, true, SportsEventStatusScheduled)
	second := newEvent(real1.ID, tbd1.ID, future.Add(72*time.Hour), 2025, true, SportsEventStatusScheduled)
	third := newEvent(tbd3.ID, tbd4.ID, future.Add(7*24*time.Hour), 2025, true, SportsEventStatusScheduled)

	// should be excluded
	newEvent(tbd1.ID, tbd2.ID, future.Add(time.Hour), 2025, false, SportsEventStatusScheduled)        // regular season
	newEvent(tbd1.ID, tbd2.ID, time.Now().Add(-24*time.Hour), 2025, true, SportsEventStatusScheduled) // in the past
	newEvent(tbd1.ID, tbd2.ID, future.Add(48*time.Hour), 2025, true, SportsEventStatusFinal)          // not scheduled
	newEvent(tbd1.ID, tbd2.ID, future.Add(365*24*time.Hour), 2026, true, SportsEventStatusScheduled)  // next season

	events, err := m.UpcomingPostseasonSportsEvents(ctx, league)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(events).Should(gomega.HaveLen(3))

	espnIDs := make([]string, 0, len(events))
	for _, e := range events {
		espnIDs = append(espnIDs, e.ESPNID)

		g.Expect(e.Postseason).Should(gomega.BeTrue())
		g.Expect(e.Season).Should(gomega.Equal(2025))
		g.Expect(e.Status).Should(gomega.Equal(SportsEventStatusScheduled))
	}

	// results are ordered by event date
	g.Expect(espnIDs).Should(gomega.Equal([]string{first.ESPNID, second.ESPNID, third.ESPNID}))

	// a league with no postseason games returns an empty result
	events, err = m.UpcomingPostseasonSportsEvents(ctx, emptyLeague)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(events).Should(gomega.HaveLen(0))
}

func TestSportsEventStatusIsValid(t *testing.T) {
	g := gomega.NewWithT(t)

	for _, status := range []SportsEventStatus{SportsEventStatusScheduled, SportsEventStatusInProgress, SportsEventStatusFinal} {
		g.Expect(status.IsValid()).Should(gomega.BeTrue(), string(status))
	}

	for _, status := range []SportsEventStatus{"", "bogus", "FINAL", "postponed"} {
		g.Expect(status.IsValid()).Should(gomega.BeFalse(), string(status))
	}
}
