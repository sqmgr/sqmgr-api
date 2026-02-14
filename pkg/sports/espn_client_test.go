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

package sports

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onsi/gomega"
)

func TestNewClient(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	client := NewClient(Config{})

	g.Expect(client).ShouldNot(gomega.BeNil())
	g.Expect(client.baseURL).Should(gomega.Equal(defaultESPNBaseURL))
}

func TestNewClientCustomURL(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	client := NewClient(Config{
		BaseURL: "https://custom.api.com",
	})

	g.Expect(client.baseURL).Should(gomega.Equal("https://custom.api.com"))
}

func TestLeagueIsValid(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	g.Expect(LeagueNFL.IsValid()).Should(gomega.BeTrue())
	g.Expect(LeagueNBA.IsValid()).Should(gomega.BeTrue())
	g.Expect(LeagueWNBA.IsValid()).Should(gomega.BeTrue())
	g.Expect(LeagueNCAAB.IsValid()).Should(gomega.BeTrue())
	g.Expect(LeagueNCAAF.IsValid()).Should(gomega.BeTrue())
	g.Expect(League("invalid").IsValid()).Should(gomega.BeFalse())
}

func TestLeagueESPNPath(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	g.Expect(LeagueNFL.ESPNPath()).Should(gomega.Equal("football/nfl"))
	g.Expect(LeagueNBA.ESPNPath()).Should(gomega.Equal("basketball/nba"))
	g.Expect(LeagueWNBA.ESPNPath()).Should(gomega.Equal("basketball/wnba"))
	g.Expect(LeagueNCAAB.ESPNPath()).Should(gomega.Equal("basketball/mens-college-basketball"))
	g.Expect(LeagueNCAAF.ESPNPath()).Should(gomega.Equal("football/college-football"))
}

func TestAllLeagues(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	leagues := AllLeagues()
	g.Expect(len(leagues)).Should(gomega.Equal(5))
	g.Expect(leagues).Should(gomega.ContainElement(LeagueNFL))
	g.Expect(leagues).Should(gomega.ContainElement(LeagueNBA))
}

func TestGetTeams(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path and limit parameter
		g.Expect(r.URL.Path).Should(gomega.Equal("/football/nfl/teams"))
		g.Expect(r.URL.Query().Get("limit")).Should(gomega.Equal("1000"))

		// Return mock response
		response := espnTeamsResponse{
			Sports: []struct {
				Leagues []struct {
					Teams []struct {
						Team espnTeam `json:"team"`
					} `json:"teams"`
				} `json:"leagues"`
			}{
				{
					Leagues: []struct {
						Teams []struct {
							Team espnTeam `json:"team"`
						} `json:"teams"`
					}{
						{
							Teams: []struct {
								Team espnTeam `json:"team"`
							}{
								{Team: espnTeam{ID: "1", Name: "Patriots", DisplayName: "New England Patriots", Abbreviation: "NE", Location: "New England", Color: "002244", AlternateColor: "c60c30"}},
								{Team: espnTeam{ID: "2", Name: "Cowboys", DisplayName: "Dallas Cowboys", Abbreviation: "DAL", Location: "Dallas", Color: "003594", AlternateColor: "869397"}},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100, // High rate limit for tests
	})

	teams, err := client.GetTeams(context.Background(), LeagueNFL)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(teams)).Should(gomega.Equal(2))
	g.Expect(teams[0].Name).Should(gomega.Equal("Patriots"))
	g.Expect(teams[0].DisplayName).Should(gomega.Equal("New England Patriots"))
	g.Expect(teams[0].Abbreviation).Should(gomega.Equal("NE"))
	g.Expect(teams[0].Color).Should(gomega.Equal("002244"))
	g.Expect(teams[0].AlternateColor).Should(gomega.Equal("c60c30"))
}

func TestGetTeamsInvalidLeague(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	client := NewClient(Config{})

	_, err := client.GetTeams(context.Background(), League("invalid"))
	g.Expect(err).ShouldNot(gomega.Succeed())
}

func TestGetScoreboard(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.URL.Path).Should(gomega.Equal("/football/nfl/scoreboard"))

		response := espnScoreboardResponse{
			Events: []espnEvent{
				{
					ID:     "401547417",
					Date:   "2024-02-11T23:30:00Z",
					Name:   "Kansas City Chiefs at San Francisco 49ers",
					Season: espnSeason{Year: 2024, Type: 2},
					Week:   &espnWeek{Number: 22},
					Status: espnStatus{
						Period: 4,
						Type: espnStatusType{
							Name:      "STATUS_FINAL",
							State:     "post",
							Completed: true,
						},
					},
					Competitions: []espnCompetition{
						{
							ID:    "401547417",
							Date:  "2024-02-11T23:30:00Z",
							Venue: &espnVenue{FullName: "Allegiant Stadium"},
							Notes: []espnNote{{Type: "event", Headline: "Super Bowl LVIII"}},
							Competitors: []espnCompetitor{
								{
									ID:       "12",
									HomeAway: "home",
									Team:     espnTeam{ID: "12", Name: "Chiefs", DisplayName: "Kansas City Chiefs", Abbreviation: "KC", Location: "Kansas City"},
									Score:    "25",
									Linescores: []espnLinescore{
										{Value: 0},
										{Value: 10},
										{Value: 0},
										{Value: 15},
									},
								},
								{
									ID:       "25",
									HomeAway: "away",
									Team:     espnTeam{ID: "25", Name: "49ers", DisplayName: "San Francisco 49ers", Abbreviation: "SF", Location: "San Francisco"},
									Score:    "22",
									Linescores: []espnLinescore{
										{Value: 0},
										{Value: 10},
										{Value: 3},
										{Value: 9},
									},
								},
							},
							Status: espnStatus{
								Period: 4,
								Type: espnStatusType{
									Name:      "STATUS_FINAL",
									State:     "post",
									Completed: true,
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	events, err := client.GetScoreboard(context.Background(), LeagueNFL, ScoreboardOptions{})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(events)).Should(gomega.Equal(1))

	event := events[0]
	g.Expect(event.ID).Should(gomega.Equal("401547417"))
	g.Expect(event.Name).Should(gomega.Equal("Super Bowl LVIII"))
	g.Expect(event.Status).Should(gomega.Equal(EventStatusFinal))
	g.Expect(event.HomeTeam.Abbreviation).Should(gomega.Equal("KC"))
	g.Expect(event.AwayTeam.Abbreviation).Should(gomega.Equal("SF"))
	g.Expect(*event.HomeTeamScore).Should(gomega.Equal(25))
	g.Expect(*event.AwayTeamScore).Should(gomega.Equal(22))
	g.Expect(event.Venue).Should(gomega.Equal("Allegiant Stadium"))
	g.Expect(*event.Week).Should(gomega.Equal(22))

	// Verify quarter scores
	g.Expect(*event.HomeQ1).Should(gomega.Equal(0))
	g.Expect(*event.HomeQ2).Should(gomega.Equal(10))
	g.Expect(*event.HomeQ3).Should(gomega.Equal(0))
	g.Expect(*event.HomeQ4).Should(gomega.Equal(15))
	g.Expect(*event.AwayQ1).Should(gomega.Equal(0))
	g.Expect(*event.AwayQ2).Should(gomega.Equal(10))
	g.Expect(*event.AwayQ3).Should(gomega.Equal(3))
	g.Expect(*event.AwayQ4).Should(gomega.Equal(9))
}

func TestGetScoreboardScheduledGame(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := espnScoreboardResponse{
			Events: []espnEvent{
				{
					ID:     "401547500",
					Date:   "2024-09-05T20:20:00Z",
					Name:   "Baltimore Ravens at Kansas City Chiefs",
					Season: espnSeason{Year: 2024, Type: 2},
					Week:   &espnWeek{Number: 1},
					Status: espnStatus{
						Period: 0,
						Type: espnStatusType{
							Name:      "STATUS_SCHEDULED",
							State:     "pre",
							Completed: false,
						},
					},
					Competitions: []espnCompetition{
						{
							ID:   "401547500",
							Date: "2024-09-05T20:20:00Z",
							// No Notes - regular season game uses event.Name
							Competitors: []espnCompetitor{
								{
									HomeAway: "home",
									Team:     espnTeam{ID: "12", Name: "Chiefs", Abbreviation: "KC"},
									Score:    "",
								},
								{
									HomeAway: "away",
									Team:     espnTeam{ID: "30", Name: "Ravens", Abbreviation: "BAL"},
									Score:    "",
								},
							},
							Status: espnStatus{
								Period: 0,
								Type: espnStatusType{
									Name:  "STATUS_SCHEDULED",
									State: "pre",
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	events, err := client.GetScoreboard(context.Background(), LeagueNFL, ScoreboardOptions{})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(events)).Should(gomega.Equal(1))

	event := events[0]
	// Regular season game without notes has no name (empty)
	g.Expect(event.Name).Should(gomega.Equal(""))
	g.Expect(event.Status).Should(gomega.Equal(EventStatusScheduled))
	g.Expect(event.HomeTeamScore).Should(gomega.BeNil())
	g.Expect(event.AwayTeamScore).Should(gomega.BeNil())
}

func TestGetScoreboardInProgressGame(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := espnScoreboardResponse{
			Events: []espnEvent{
				{
					ID:     "401547501",
					Date:   "2024-09-05T20:20:00Z",
					Season: espnSeason{Year: 2024, Type: 2},
					Status: espnStatus{
						Period: 2,
						Type: espnStatusType{
							Name:      "STATUS_IN_PROGRESS",
							State:     "in",
							Completed: false,
						},
					},
					Competitions: []espnCompetition{
						{
							Competitors: []espnCompetitor{
								{
									HomeAway: "home",
									Team:     espnTeam{ID: "12", Name: "Chiefs", Abbreviation: "KC"},
									Score:    "14",
									Linescores: []espnLinescore{
										{Value: 7},
										{Value: 7},
									},
								},
								{
									HomeAway: "away",
									Team:     espnTeam{ID: "30", Name: "Ravens", Abbreviation: "BAL"},
									Score:    "10",
									Linescores: []espnLinescore{
										{Value: 3},
										{Value: 7},
									},
								},
							},
							Status: espnStatus{
								Period: 2,
								Type: espnStatusType{
									Name:  "STATUS_IN_PROGRESS",
									State: "in",
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	events, err := client.GetScoreboard(context.Background(), LeagueNFL, ScoreboardOptions{})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(events)).Should(gomega.Equal(1))

	event := events[0]
	g.Expect(event.Status).Should(gomega.Equal(EventStatusInProgress))
	g.Expect(event.Period).Should(gomega.Equal(2))
	g.Expect(*event.HomeTeamScore).Should(gomega.Equal(14))
	g.Expect(*event.AwayTeamScore).Should(gomega.Equal(10))
}

func TestGetScoreboardWithDateFilter(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify date query parameter
		g.Expect(r.URL.Query().Get("dates")).Should(gomega.Equal("20240911"))

		response := espnScoreboardResponse{Events: []espnEvent{}}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	_, err := client.GetScoreboard(context.Background(), LeagueNFL, ScoreboardOptions{
		Date: "20240911",
	})
	g.Expect(err).Should(gomega.Succeed())
}

func TestGetScoreboardWithWeekFilter(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify week query parameter
		g.Expect(r.URL.Query().Get("week")).Should(gomega.Equal("5"))
		g.Expect(r.URL.Query().Get("seasonYear")).Should(gomega.Equal("2024"))

		response := espnScoreboardResponse{Events: []espnEvent{}}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	_, err := client.GetScoreboard(context.Background(), LeagueNFL, ScoreboardOptions{
		Week:   5,
		Season: 2024,
	})
	g.Expect(err).Should(gomega.Succeed())
}

func TestGetScoreboardInvalidLeague(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	client := NewClient(Config{})

	_, err := client.GetScoreboard(context.Background(), League("invalid"), ScoreboardOptions{})
	g.Expect(err).ShouldNot(gomega.Succeed())
}

func TestGetNFLSchedule(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.URL.Query().Get("week")).Should(gomega.Equal("10"))
		g.Expect(r.URL.Query().Get("seasonYear")).Should(gomega.Equal("2024"))

		response := espnScoreboardResponse{Events: []espnEvent{}}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	_, err := client.GetNFLSchedule(context.Background(), 2024, 10, SeasonTypeRegular)
	g.Expect(err).Should(gomega.Succeed())
}

func TestParseEventWithOT(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := espnScoreboardResponse{
			Events: []espnEvent{
				{
					ID:     "401547600",
					Date:   "2024-02-11T23:30:00Z",
					Season: espnSeason{Year: 2024, Type: 3},
					Status: espnStatus{
						Period: 5,
						Type: espnStatusType{
							Name:      "STATUS_FINAL_OT",
							State:     "post",
							Completed: true,
						},
					},
					Competitions: []espnCompetition{
						{
							Competitors: []espnCompetitor{
								{
									HomeAway: "home",
									Team:     espnTeam{ID: "12", Name: "Chiefs", Abbreviation: "KC"},
									Score:    "25",
									Linescores: []espnLinescore{
										{Value: 0},
										{Value: 10},
										{Value: 0},
										{Value: 12},
										{Value: 3},
									},
								},
								{
									HomeAway: "away",
									Team:     espnTeam{ID: "25", Name: "49ers", Abbreviation: "SF"},
									Score:    "22",
									Linescores: []espnLinescore{
										{Value: 0},
										{Value: 10},
										{Value: 3},
										{Value: 9},
										{Value: 0},
									},
								},
							},
							Status: espnStatus{
								Period: 5,
								Type: espnStatusType{
									Name:      "STATUS_FINAL_OT",
									Completed: true,
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	events, err := client.GetScoreboard(context.Background(), LeagueNFL, ScoreboardOptions{})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(events)).Should(gomega.Equal(1))

	event := events[0]
	g.Expect(event.Status).Should(gomega.Equal(EventStatusFinal))
	g.Expect(*event.HomeOT).Should(gomega.Equal(3))
	g.Expect(*event.AwayOT).Should(gomega.Equal(0))
}

func TestGetSeasonInfo(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.URL.Path).Should(gomega.Equal("/basketball/wnba/scoreboard"))

		response := map[string]interface{}{
			"leagues": []map[string]interface{}{
				{
					"season": map[string]interface{}{
						"year":      2026,
						"startDate": "2026-05-01T07:00:00Z",
						"endDate":   "2026-10-15T06:59:00Z",
						"type": map[string]interface{}{
							"id":   "2",
							"type": 2,
							"name": "Regular Season",
						},
					},
					"calendar": []string{
						"2026-05-08T07:00Z",
						"2026-05-09T07:00Z",
					},
				},
			},
			"events": []interface{}{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	seasonInfo, err := client.GetSeasonInfo(context.Background(), LeagueWNBA)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(seasonInfo).ShouldNot(gomega.BeNil())
	g.Expect(seasonInfo.Year).Should(gomega.Equal(2026))
	g.Expect(seasonInfo.Type).Should(gomega.Equal("Regular Season"))
	g.Expect(seasonInfo.StartDate.Year()).Should(gomega.Equal(2026))
	g.Expect(seasonInfo.StartDate.Month()).Should(gomega.Equal(time.May))
	g.Expect(seasonInfo.EndDate.Year()).Should(gomega.Equal(2026))
	g.Expect(seasonInfo.EndDate.Month()).Should(gomega.Equal(time.October))
}

func TestGetSeasonInfoInvalidLeague(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	client := NewClient(Config{})

	_, err := client.GetSeasonInfo(context.Background(), League("invalid"))
	g.Expect(err).ShouldNot(gomega.Succeed())
}

func TestGetSeasonInfoNoLeagues(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"leagues": []interface{}{},
			"events":  []interface{}{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	_, err := client.GetSeasonInfo(context.Background(), LeagueNBA)
	g.Expect(err).ShouldNot(gomega.Succeed())
	g.Expect(err.Error()).Should(gomega.ContainSubstring("no league info"))
}

func TestGetTeamSchedule(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.URL.Path).Should(gomega.Equal("/basketball/mens-college-basketball/teams/2084/schedule"))

		response := espnTeamScheduleResponse{
			Team: espnTeam{
				ID:           "2084",
				Name:         "Buffalo",
				DisplayName:  "Buffalo Bulls",
				Abbreviation: "BUF",
				Location:     "Buffalo",
			},
			Events: []espnScheduleEvent{
				{
					ID:   "401823425",
					Date: "2025-11-03T23:30Z",
					Name: "Southern Miss Golden Eagles at Buffalo Bulls",
					Season: espnSeason{
						Year: 2026,
						Type: 2,
					},
					Week: &espnWeek{Number: 1},
					Competitions: []espnScheduleCompetition{
						{
							ID:    "401823425",
							Date:  "2025-11-03T23:30Z",
							Venue: &espnVenue{FullName: "Alumni Arena"},
							Competitors: []espnScheduleCompetitor{
								{
									ID:       "2084",
									HomeAway: "home",
									Team: espnTeam{
										ID:           "2084",
										Name:         "Buffalo",
										DisplayName:  "Buffalo Bulls",
										Abbreviation: "BUF",
									},
									Score: &espnScore{Value: 85, DisplayValue: "85"},
								},
								{
									ID:       "2572",
									HomeAway: "away",
									Team: espnTeam{
										ID:           "2572",
										Name:         "Southern Miss",
										DisplayName:  "Southern Miss Golden Eagles",
										Abbreviation: "USM",
									},
									Score: &espnScore{Value: 79, DisplayValue: "79"},
								},
							},
							Status: espnStatus{
								Period: 2,
								Type: espnStatusType{
									Name:      "STATUS_FINAL",
									State:     "post",
									Completed: true,
								},
							},
						},
					},
				},
				{
					ID:   "401823500",
					Date: "2025-11-10T19:00Z",
					Name: "Buffalo Bulls at Ohio Bobcats",
					Season: espnSeason{
						Year: 2026,
						Type: 2,
					},
					Competitions: []espnScheduleCompetition{
						{
							ID:   "401823500",
							Date: "2025-11-10T19:00Z",
							Competitors: []espnScheduleCompetitor{
								{
									ID:       "195",
									HomeAway: "home",
									Team: espnTeam{
										ID:           "195",
										Name:         "Ohio",
										DisplayName:  "Ohio Bobcats",
										Abbreviation: "OHIO",
									},
								},
								{
									ID:       "2084",
									HomeAway: "away",
									Team: espnTeam{
										ID:           "2084",
										Name:         "Buffalo",
										DisplayName:  "Buffalo Bulls",
										Abbreviation: "BUF",
									},
								},
							},
							Status: espnStatus{
								Period: 0,
								Type: espnStatusType{
									Name:  "STATUS_SCHEDULED",
									State: "pre",
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	events, err := client.GetTeamSchedule(context.Background(), LeagueNCAAB, "2084")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(len(events)).Should(gomega.Equal(2))

	// First event - completed game (no notes, so no name)
	event1 := events[0]
	g.Expect(event1.ID).Should(gomega.Equal("401823425"))
	g.Expect(event1.Name).Should(gomega.Equal(""))
	g.Expect(event1.Status).Should(gomega.Equal(EventStatusFinal))
	g.Expect(event1.HomeTeam.Abbreviation).Should(gomega.Equal("BUF"))
	g.Expect(event1.AwayTeam.Abbreviation).Should(gomega.Equal("USM"))
	g.Expect(*event1.HomeTeamScore).Should(gomega.Equal(85))
	g.Expect(*event1.AwayTeamScore).Should(gomega.Equal(79))
	g.Expect(event1.Venue).Should(gomega.Equal("Alumni Arena"))
	g.Expect(*event1.Week).Should(gomega.Equal(1))

	// Second event - scheduled game (no notes, so no name)
	event2 := events[1]
	g.Expect(event2.ID).Should(gomega.Equal("401823500"))
	g.Expect(event2.Name).Should(gomega.Equal(""))
	g.Expect(event2.Status).Should(gomega.Equal(EventStatusScheduled))
	g.Expect(event2.HomeTeam.Abbreviation).Should(gomega.Equal("OHIO"))
	g.Expect(event2.AwayTeam.Abbreviation).Should(gomega.Equal("BUF"))
	g.Expect(event2.HomeTeamScore).Should(gomega.BeNil())
	g.Expect(event2.AwayTeamScore).Should(gomega.BeNil())
}

func TestGetTeamScheduleInvalidLeague(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	client := NewClient(Config{})

	_, err := client.GetTeamSchedule(context.Background(), League("invalid"), "123")
	g.Expect(err).ShouldNot(gomega.Succeed())
}

func TestParseESPNDate(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	tests := []struct {
		name     string
		input    string
		wantYear int
		wantErr  bool
	}{
		{
			name:     "ESPN format without seconds",
			input:    "2026-05-01T07:00Z",
			wantYear: 2026,
			wantErr:  false,
		},
		{
			name:     "RFC3339 format",
			input:    "2026-05-01T07:00:00Z",
			wantYear: 2026,
			wantErr:  false,
		},
		{
			name:     "RFC3339 with offset",
			input:    "2026-05-01T07:00:00-04:00",
			wantYear: 2026,
			wantErr:  false,
		},
		{
			name:    "invalid format",
			input:   "2026/05/01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseESPNDate(tt.input)
			if tt.wantErr {
				g.Expect(err).ShouldNot(gomega.Succeed())
			} else {
				g.Expect(err).Should(gomega.Succeed())
				g.Expect(result.Year()).Should(gomega.Equal(tt.wantYear))
			}
		})
	}
}

func TestGetEventSummary(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.URL.Path).Should(gomega.Equal("/basketball/mens-college-basketball/summary"))
		g.Expect(r.URL.Query().Get("event")).Should(gomega.Equal("401809352"))

		response := espnSummaryResponse{
			Header: espnSummaryHeader{
				ID: "401809352",
				Season: espnSeason{
					Year: 2026,
					Type: 2,
				},
				Competitions: []espnSummaryCompetition{
					{
						ID:    "401809352",
						Date:  "2026-02-02T00:00Z",
						Venue: &espnVenue{FullName: "Jerry Richardson Indoor Stadium"},
						Competitors: []espnSummaryCompetitor{
							{
								ID:       "2747",
								HomeAway: "home",
								Winner:   false,
								Team: espnSummaryTeam{
									ID:           "2747",
									Name:         "Terriers",
									DisplayName:  "Wofford Terriers",
									Abbreviation: "WOF",
									Location:     "Wofford",
									Color:        "897048",
								},
								Score: "72",
								Linescores: []espnSummaryLinescore{
									{DisplayValue: "39"},
									{DisplayValue: "33"},
								},
							},
							{
								ID:       "2193",
								HomeAway: "away",
								Winner:   true,
								Team: espnSummaryTeam{
									ID:           "2193",
									Name:         "Buccaneers",
									DisplayName:  "East Tennessee State Buccaneers",
									Abbreviation: "ETSU",
									Location:     "East Tennessee State",
									Color:        "041e42",
								},
								Score: "86",
								Linescores: []espnSummaryLinescore{
									{DisplayValue: "47"},
									{DisplayValue: "39"},
								},
							},
						},
						Status: espnStatus{
							Period:       2,
							DisplayClock: "0:00",
							Type: espnStatusType{
								Name:        "STATUS_FINAL",
								State:       "post",
								Completed:   true,
								Description: "Final",
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	event, err := client.GetEventSummary(context.Background(), LeagueNCAAB, "401809352")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(event).ShouldNot(gomega.BeNil())

	g.Expect(event.ID).Should(gomega.Equal("401809352"))
	g.Expect(event.Season).Should(gomega.Equal(2026))
	g.Expect(event.Status).Should(gomega.Equal(EventStatusFinal))
	g.Expect(event.Period).Should(gomega.Equal(2))
	g.Expect(event.Clock).Should(gomega.Equal("0:00"))
	g.Expect(event.StatusDetail).Should(gomega.Equal("Final"))
	g.Expect(event.Venue).Should(gomega.Equal("Jerry Richardson Indoor Stadium"))

	// Home team
	g.Expect(event.HomeTeam.ID).Should(gomega.Equal("2747"))
	g.Expect(event.HomeTeam.Abbreviation).Should(gomega.Equal("WOF"))
	g.Expect(*event.HomeTeamScore).Should(gomega.Equal(72))
	g.Expect(*event.HomeQ1).Should(gomega.Equal(39))
	g.Expect(*event.HomeQ2).Should(gomega.Equal(33))

	// Away team
	g.Expect(event.AwayTeam.ID).Should(gomega.Equal("2193"))
	g.Expect(event.AwayTeam.Abbreviation).Should(gomega.Equal("ETSU"))
	g.Expect(*event.AwayTeamScore).Should(gomega.Equal(86))
	g.Expect(*event.AwayQ1).Should(gomega.Equal(47))
	g.Expect(*event.AwayQ2).Should(gomega.Equal(39))
}

func TestGetEventSummaryNotFound(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	_, err := client.GetEventSummary(context.Background(), LeagueNCAAB, "999999999")
	g.Expect(err).ShouldNot(gomega.Succeed())
	g.Expect(err.Error()).Should(gomega.ContainSubstring("event not found"))
}

func TestGetEventSummaryInvalidLeague(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	client := NewClient(Config{})

	_, err := client.GetEventSummary(context.Background(), League("invalid"), "123")
	g.Expect(err).ShouldNot(gomega.Succeed())
}

func TestGetEventSummaryWithOT(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := espnSummaryResponse{
			Header: espnSummaryHeader{
				ID: "401809999",
				Season: espnSeason{
					Year: 2026,
					Type: 2,
				},
				Competitions: []espnSummaryCompetition{
					{
						ID:   "401809999",
						Date: "2026-02-02T00:00Z",
						Competitors: []espnSummaryCompetitor{
							{
								ID:       "1",
								HomeAway: "home",
								Team:     espnSummaryTeam{ID: "1", Abbreviation: "HOME"},
								Score:    "95",
								Linescores: []espnSummaryLinescore{
									{DisplayValue: "35"},
									{DisplayValue: "40"},
									{DisplayValue: "10"},
									{DisplayValue: "10"},
								},
							},
							{
								ID:       "2",
								HomeAway: "away",
								Team:     espnSummaryTeam{ID: "2", Abbreviation: "AWAY"},
								Score:    "90",
								Linescores: []espnSummaryLinescore{
									{DisplayValue: "40"},
									{DisplayValue: "35"},
									{DisplayValue: "5"},
									{DisplayValue: "10"},
								},
							},
						},
						Status: espnStatus{
							Period: 3,
							Type: espnStatusType{
								Name:      "STATUS_FINAL_OT",
								Completed: true,
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		RateLimit: 100,
	})

	event, err := client.GetEventSummary(context.Background(), LeagueNCAAB, "401809999")
	g.Expect(err).Should(gomega.Succeed())

	// Basketball halves + OT: Q1=H1, Q2=H2, Q3=OT1, Q4=OT2
	g.Expect(*event.HomeQ1).Should(gomega.Equal(35))
	g.Expect(*event.HomeQ2).Should(gomega.Equal(40))
	g.Expect(*event.HomeQ3).Should(gomega.Equal(10))
	g.Expect(*event.HomeQ4).Should(gomega.Equal(10))
	g.Expect(event.HomeOT).Should(gomega.BeNil()) // Only 4 periods, no extra OT

	g.Expect(*event.AwayQ1).Should(gomega.Equal(40))
	g.Expect(*event.AwayQ2).Should(gomega.Equal(35))
	g.Expect(*event.AwayQ3).Should(gomega.Equal(5))
	g.Expect(*event.AwayQ4).Should(gomega.Equal(10))
}
