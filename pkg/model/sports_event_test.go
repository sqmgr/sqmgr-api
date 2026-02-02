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
	"os"
	"testing"
	"time"

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
