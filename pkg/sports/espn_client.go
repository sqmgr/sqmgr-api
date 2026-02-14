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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

const (
	defaultESPNBaseURL = "https://site.api.espn.com/apis/site/v2/sports"
	defaultTimeout     = 30 * time.Second
)

// Client is an ESPN API client
type Client struct {
	httpClient  *http.Client
	baseURL     string
	rateLimiter *rate.Limiter
	logger      *logrus.Entry
}

// Config holds configuration for the ESPN client
type Config struct {
	BaseURL   string        // Optional, defaults to ESPN API
	RateLimit float64       // Requests per second, defaults to 10/s
	Timeout   time.Duration // HTTP timeout, defaults to 30s
	Logger    *logrus.Entry // Optional logger
}

// NewClient creates a new ESPN API client
func NewClient(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultESPNBaseURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	// Default rate limit: 10 requests per second (ESPN is public, no strict limits)
	rateLimit := cfg.RateLimit
	if rateLimit == 0 {
		rateLimit = 10.0
	}

	logger := cfg.Logger
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}

	return &Client{
		baseURL:     baseURL,
		rateLimiter: rate.NewLimiter(rate.Limit(rateLimit), 1),
		logger:      logger.WithField("component", "espn-client"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

const maxRetries = 3

// doRequest performs an HTTP request with rate limiting and retry logic
func (c *Client) doRequest(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wait for rate limiter
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		c.logger.WithFields(logrus.Fields{
			"url":     reqURL,
			"attempt": attempt + 1,
		}).Debug("making ESPN API request")

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			c.logger.WithError(err).WithField("url", reqURL).Warn("request failed")
			lastErr = fmt.Errorf("executing request: %w", err)
			continue
		}

		c.logger.WithFields(logrus.Fields{
			"url":    reqURL,
			"status": resp.StatusCode,
		}).Debug("received ESPN API response")

		// Handle rate limiting with retry
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()

			if attempt == maxRetries {
				return nil, fmt.Errorf("rate limited after %d retries", maxRetries)
			}

			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<attempt) * time.Second
			c.logger.WithFields(logrus.Fields{
				"url":     reqURL,
				"backoff": backoff,
				"attempt": attempt + 1,
			}).Warn("rate limited, backing off")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("request failed after %d attempts", maxRetries)
}

// GetTeams fetches all teams for a league
func (c *Client) GetTeams(ctx context.Context, league League) ([]Team, error) {
	if !league.IsValid() {
		return nil, fmt.Errorf("invalid league: %s", league)
	}

	path := fmt.Sprintf("/%s/teams", league.ESPNPath())

	// ESPN defaults to 50 teams, but college leagues have 700+
	query := url.Values{}
	query.Set("limit", "1000")

	resp, err := c.doRequest(ctx, path, query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var teamsResp espnTeamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&teamsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var teams []Team
	for _, sport := range teamsResp.Sports {
		for _, league := range sport.Leagues {
			for _, t := range league.Teams {
				teams = append(teams, Team{
					ID:             t.Team.ID,
					Name:           t.Team.Name,
					DisplayName:    t.Team.DisplayName,
					Abbreviation:   t.Team.Abbreviation,
					Location:       t.Team.Location,
					Color:          t.Team.Color,
					AlternateColor: t.Team.AlternateColor,
				})
			}
		}
	}

	return teams, nil
}

// GetScoreboard fetches the scoreboard (all games) for a league with optional date/week filters
func (c *Client) GetScoreboard(ctx context.Context, league League, opts ScoreboardOptions) ([]Event, error) {
	if !league.IsValid() {
		return nil, fmt.Errorf("invalid league: %s", league)
	}

	path := fmt.Sprintf("/%s/scoreboard", league.ESPNPath())
	query := url.Values{}

	if opts.Date != "" {
		query.Set("dates", opts.Date)
	}
	if opts.Week > 0 && (league == LeagueNFL || league == LeagueNCAAF) {
		query.Set("week", strconv.Itoa(opts.Week))
	}
	if opts.Season > 0 {
		query.Set("seasonYear", strconv.Itoa(opts.Season))
	}
	if opts.SeasonType > 0 {
		query.Set("seasontype", strconv.Itoa(int(opts.SeasonType)))
	}

	resp, err := c.doRequest(ctx, path, query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var scoreboardResp espnScoreboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&scoreboardResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	events := make([]Event, 0, len(scoreboardResp.Events))
	for _, e := range scoreboardResp.Events {
		event, err := c.parseEvent(e)
		if err != nil {
			c.logger.WithError(err).WithField("eventID", e.ID).Warn("failed to parse event")
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// GetScoreboardForDateRange fetches games for a date range by iterating through dates
func (c *Client) GetScoreboardForDateRange(ctx context.Context, league League, startDate, endDate time.Time) ([]Event, error) {
	var allEvents []Event
	seenIDs := make(map[string]bool)

	// Iterate through each date in the range
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("20060102")
		events, err := c.GetScoreboard(ctx, league, ScoreboardOptions{Date: dateStr})
		if err != nil {
			return nil, fmt.Errorf("fetching scoreboard for %s: %w", dateStr, err)
		}

		// Deduplicate events (ESPN may return same event for multiple dates)
		for _, e := range events {
			if !seenIDs[e.ID] {
				seenIDs[e.ID] = true
				allEvents = append(allEvents, e)
			}
		}
	}

	return allEvents, nil
}

// GetNFLSchedule fetches NFL games for a specific week and season
func (c *Client) GetNFLSchedule(ctx context.Context, season, week int, seasonType SeasonType) ([]Event, error) {
	return c.GetScoreboard(ctx, LeagueNFL, ScoreboardOptions{
		Season:     season,
		Week:       week,
		SeasonType: seasonType,
	})
}

// GetSeasonInfo fetches the current/upcoming season info for a league
func (c *Client) GetSeasonInfo(ctx context.Context, league League) (*SeasonInfo, error) {
	if !league.IsValid() {
		return nil, fmt.Errorf("invalid league: %s", league)
	}

	path := fmt.Sprintf("/%s/scoreboard", league.ESPNPath())

	resp, err := c.doRequest(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var scoreboardResp espnScoreboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&scoreboardResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(scoreboardResp.Leagues) == 0 {
		return nil, fmt.Errorf("no league info in response")
	}

	leagueInfo := scoreboardResp.Leagues[0]
	seasonData := leagueInfo.Season

	// Parse dates - ESPN uses format like "2025-07-31T07:00Z" (no seconds)
	startDate, err := parseESPNDate(seasonData.StartDate)
	if err != nil {
		return nil, fmt.Errorf("parsing start date %s: %w", seasonData.StartDate, err)
	}
	endDate, err := parseESPNDate(seasonData.EndDate)
	if err != nil {
		return nil, fmt.Errorf("parsing end date %s: %w", seasonData.EndDate, err)
	}

	now := time.Now()
	inSeason := now.After(startDate) && now.Before(endDate)

	return &SeasonInfo{
		Year:      seasonData.Year,
		StartDate: startDate,
		EndDate:   endDate,
		Type:      seasonData.Type.Name,
		InSeason:  inSeason,
	}, nil
}

// GetTeamSchedule fetches the full schedule for a specific team
func (c *Client) GetTeamSchedule(ctx context.Context, league League, teamID string) ([]Event, error) {
	if !league.IsValid() {
		return nil, fmt.Errorf("invalid league: %s", league)
	}

	path := fmt.Sprintf("/%s/teams/%s/schedule", league.ESPNPath(), teamID)

	resp, err := c.doRequest(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var scheduleResp espnTeamScheduleResponse
	if err := json.NewDecoder(resp.Body).Decode(&scheduleResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	events := make([]Event, 0, len(scheduleResp.Events))
	for _, e := range scheduleResp.Events {
		event, err := c.parseScheduleEvent(e)
		if err != nil {
			c.logger.WithError(err).WithField("eventID", e.ID).Warn("failed to parse schedule event")
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// parseScheduleEvent converts an ESPN team schedule event to our Event type
func (c *Client) parseScheduleEvent(e espnScheduleEvent) (Event, error) {
	event := Event{
		ID:         e.ID,
		Season:     e.Season.Year,
		SeasonType: SeasonType(e.SeasonType.Type),
	}

	// Parse date
	eventDate, err := time.Parse(time.RFC3339, e.Date)
	if err != nil {
		eventDate, err = time.Parse("2006-01-02T15:04Z", e.Date)
		if err != nil {
			return event, fmt.Errorf("parsing date %s: %w", e.Date, err)
		}
	}
	event.Date = eventDate

	// Parse week
	if e.Week != nil {
		week := e.Week.Number
		event.Week = &week
	}

	// Get competition details
	if len(e.Competitions) == 0 {
		return event, fmt.Errorf("no competitions found")
	}
	comp := e.Competitions[0]

	// Only set Name for special events (e.g., "Super Bowl LVIII", "Wild Card")
	// Regular season games will have no name (null)
	for _, note := range comp.Notes {
		if note.Headline != "" {
			event.Name = note.Headline
			break
		}
	}

	// Venue
	if comp.Venue != nil {
		event.Venue = comp.Venue.FullName
	}

	// Parse status
	switch comp.Status.Type.Name {
	case "STATUS_SCHEDULED":
		event.Status = EventStatusScheduled
	case "STATUS_FINAL", "STATUS_FINAL_OT":
		event.Status = EventStatusFinal
	default:
		if comp.Status.Type.Completed {
			event.Status = EventStatusFinal
		} else if comp.Status.Type.State == "pre" {
			event.Status = EventStatusScheduled
		} else {
			event.Status = EventStatusInProgress
		}
	}
	event.Period = comp.Status.Period
	event.Clock = comp.Status.DisplayClock
	event.StatusDetail = comp.Status.Type.Description

	// Parse competitors
	for _, competitor := range comp.Competitors {
		team := Team{
			ID:             competitor.Team.ID,
			Name:           competitor.Team.Name,
			DisplayName:    competitor.Team.DisplayName,
			Abbreviation:   competitor.Team.Abbreviation,
			Location:       competitor.Team.Location,
			Color:          competitor.Team.Color,
			AlternateColor: competitor.Team.AlternateColor,
		}

		// Parse score (team schedule uses score object, not string)
		var score *int
		if competitor.Score != nil {
			s := int(competitor.Score.Value)
			score = &s
		}

		// Note: Team schedule endpoint doesn't include linescores (quarter scores)
		// Those will be populated when we sync scores for in-progress/final games

		if competitor.HomeAway == "home" {
			event.HomeTeam = team
			event.HomeTeamScore = score
		} else {
			event.AwayTeam = team
			event.AwayTeamScore = score
		}
	}

	return event, nil
}

// GetEventSummary fetches details for a specific event by ESPN ID
// This is useful for events that don't appear in the daily scoreboard (e.g., smaller school games)
func (c *Client) GetEventSummary(ctx context.Context, league League, eventID string) (*Event, error) {
	if !league.IsValid() {
		return nil, fmt.Errorf("invalid league: %s", league)
	}

	path := fmt.Sprintf("/%s/summary", league.ESPNPath())
	query := url.Values{}
	query.Set("event", eventID)

	resp, err := c.doRequest(ctx, path, query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("event not found: %s", eventID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var summaryResp espnSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&summaryResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(summaryResp.Header.Competitions) == 0 {
		return nil, fmt.Errorf("no competitions in summary response")
	}

	event, err := c.parseSummaryEvent(summaryResp)
	if err != nil {
		return nil, fmt.Errorf("parsing summary event: %w", err)
	}

	return &event, nil
}

// parseSummaryEvent converts an ESPN summary response to our Event type
func (c *Client) parseSummaryEvent(resp espnSummaryResponse) (Event, error) {
	event := Event{
		ID:         resp.Header.ID,
		Season:     resp.Header.Season.Year,
		SeasonType: SeasonType(resp.Header.Season.Type),
	}

	comp := resp.Header.Competitions[0]

	// Parse date
	eventDate, err := time.Parse(time.RFC3339, comp.Date)
	if err != nil {
		eventDate, err = time.Parse("2006-01-02T15:04Z", comp.Date)
		if err != nil {
			return event, fmt.Errorf("parsing date %s: %w", comp.Date, err)
		}
	}
	event.Date = eventDate

	// Event name from notes
	for _, note := range comp.Notes {
		if note.Headline != "" {
			event.Name = note.Headline
			break
		}
	}

	// Venue
	if comp.Venue != nil {
		event.Venue = comp.Venue.FullName
	}

	// Parse status
	event.Period = comp.Status.Period
	event.Clock = comp.Status.DisplayClock
	event.StatusDetail = comp.Status.Type.Description
	switch comp.Status.Type.Name {
	case "STATUS_SCHEDULED":
		event.Status = EventStatusScheduled
	case "STATUS_FINAL", "STATUS_FINAL_OT":
		event.Status = EventStatusFinal
	default:
		if comp.Status.Type.Completed {
			event.Status = EventStatusFinal
		} else if comp.Status.Type.State == "pre" {
			event.Status = EventStatusScheduled
		} else {
			event.Status = EventStatusInProgress
		}
	}

	// Parse competitors
	for _, competitor := range comp.Competitors {
		team := Team{
			ID:             competitor.Team.ID,
			Name:           competitor.Team.Name,
			DisplayName:    competitor.Team.DisplayName,
			Abbreviation:   competitor.Team.Abbreviation,
			Location:       competitor.Team.Location,
			Color:          competitor.Team.Color,
			AlternateColor: competitor.Team.AlternateColor,
		}

		// Parse score
		var score *int
		if competitor.Score != "" {
			if s, err := strconv.Atoi(competitor.Score); err == nil {
				score = &s
			}
		}

		// Parse linescores (period scores - could be halves for basketball or quarters for football)
		var q1, q2, q3, q4, ot *int
		for i, ls := range competitor.Linescores {
			if ls.DisplayValue == "" {
				continue
			}
			s, err := strconv.Atoi(ls.DisplayValue)
			if err != nil {
				continue
			}
			switch i {
			case 0:
				q1 = &s
			case 1:
				q2 = &s
			case 2:
				q3 = &s
			case 3:
				q4 = &s
			default:
				// OT periods - sum them
				if ot == nil {
					ot = &s
				} else {
					*ot += s
				}
			}
		}

		if competitor.HomeAway == "home" {
			event.HomeTeam = team
			event.HomeTeamScore = score
			event.HomeQ1 = q1
			event.HomeQ2 = q2
			event.HomeQ3 = q3
			event.HomeQ4 = q4
			event.HomeOT = ot
		} else {
			event.AwayTeam = team
			event.AwayTeamScore = score
			event.AwayQ1 = q1
			event.AwayQ2 = q2
			event.AwayQ3 = q3
			event.AwayQ4 = q4
			event.AwayOT = ot
		}
	}

	return event, nil
}

// parseESPNDate parses dates from ESPN API which may have various formats
func parseESPNDate(dateStr string) (time.Time, error) {
	// Try formats in order of likelihood
	formats := []string{
		"2006-01-02T15:04Z",        // ESPN's typical format (no seconds)
		time.RFC3339,               // Standard format with seconds
		"2006-01-02T15:04:05Z",     // UTC with seconds, no offset
		"2006-01-02T15:04:05.000Z", // With milliseconds
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date with any known format")
}

// parseEvent converts an ESPN event to our Event type
func (c *Client) parseEvent(e espnEvent) (Event, error) {
	event := Event{
		ID:         e.ID,
		Season:     e.Season.Year,
		SeasonType: SeasonType(e.Season.Type),
	}

	// Parse date
	eventDate, err := time.Parse(time.RFC3339, e.Date)
	if err != nil {
		// Try alternate format
		eventDate, err = time.Parse("2006-01-02T15:04Z", e.Date)
		if err != nil {
			return event, fmt.Errorf("parsing date %s: %w", e.Date, err)
		}
	}
	event.Date = eventDate

	// Parse week (NFL/NCAAF)
	if e.Week != nil {
		week := e.Week.Number
		event.Week = &week
	}

	// Get competition details (there's usually just one)
	if len(e.Competitions) == 0 {
		return event, fmt.Errorf("no competitions found")
	}
	comp := e.Competitions[0]

	// Only set Name for special events (e.g., "Super Bowl LVIII", "Wild Card")
	// Regular season games will have no name (null)
	for _, note := range comp.Notes {
		if note.Headline != "" {
			event.Name = note.Headline
			break
		}
	}

	// Venue
	if comp.Venue != nil {
		event.Venue = comp.Venue.FullName
	}

	// Parse status
	event.Period = e.Status.Period
	event.Clock = e.Status.DisplayClock
	event.StatusDetail = e.Status.Type.Description
	switch e.Status.Type.Name {
	case "STATUS_SCHEDULED":
		event.Status = EventStatusScheduled
	case "STATUS_FINAL", "STATUS_FINAL_OT":
		event.Status = EventStatusFinal
	default:
		// Anything else (STATUS_IN_PROGRESS, STATUS_HALFTIME, etc.) is in progress
		if e.Status.Type.Completed {
			event.Status = EventStatusFinal
		} else if e.Status.Type.State == "pre" {
			event.Status = EventStatusScheduled
		} else {
			event.Status = EventStatusInProgress
		}
	}

	// Parse competitors
	for _, competitor := range comp.Competitors {
		team := Team{
			ID:             competitor.Team.ID,
			Name:           competitor.Team.Name,
			DisplayName:    competitor.Team.DisplayName,
			Abbreviation:   competitor.Team.Abbreviation,
			Location:       competitor.Team.Location,
			Color:          competitor.Team.Color,
			AlternateColor: competitor.Team.AlternateColor,
		}

		// Parse score
		var score *int
		if competitor.Score != "" {
			if s, err := strconv.Atoi(competitor.Score); err == nil {
				score = &s
			}
		}

		// Parse linescores (quarter scores)
		var q1, q2, q3, q4, ot *int
		if len(competitor.Linescores) > 0 {
			s := int(competitor.Linescores[0].Value)
			q1 = &s
		}
		if len(competitor.Linescores) > 1 {
			s := int(competitor.Linescores[1].Value)
			q2 = &s
		}
		if len(competitor.Linescores) > 2 {
			s := int(competitor.Linescores[2].Value)
			q3 = &s
		}
		if len(competitor.Linescores) > 3 {
			s := int(competitor.Linescores[3].Value)
			q4 = &s
		}
		if len(competitor.Linescores) > 4 {
			// Sum all OT periods
			otSum := 0
			for i := 4; i < len(competitor.Linescores); i++ {
				otSum += int(competitor.Linescores[i].Value)
			}
			ot = &otSum
		}

		if competitor.HomeAway == "home" {
			event.HomeTeam = team
			event.HomeTeamScore = score
			event.HomeQ1 = q1
			event.HomeQ2 = q2
			event.HomeQ3 = q3
			event.HomeQ4 = q4
			event.HomeOT = ot
		} else {
			event.AwayTeam = team
			event.AwayTeamScore = score
			event.AwayQ1 = q1
			event.AwayQ2 = q2
			event.AwayQ3 = q3
			event.AwayQ4 = q4
			event.AwayOT = ot
		}
	}

	return event, nil
}
