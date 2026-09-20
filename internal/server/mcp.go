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

package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

// mcpStatementTimeout bounds how long any single analytics query may run.
const mcpStatementTimeout = 30 * time.Second

const mcpInstructions = `Read-only analytics for SqMGR site admins. Every tool runs inside a PostgreSQL READ ONLY transaction, so nothing here can change data.

Dates: "start" and "end" accept YYYY-MM-DD (interpreted as UTC calendar days; "end" includes the whole day) or RFC3339 timestamps (where "end" is exclusive). Omit both for all time. Timestamps in the database are UTC.

Start with get_site_stats or get_time_series for high-level questions, list_pools/get_pool for a specific pool, and run_sql_query (after describe_schema) for anything the other tools do not cover.`

// readOnlyHint marks every tool as read-only for clients that surface it.
var readOnlyHint = &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}

// newMCPServer builds the admin analytics MCP server with every tool registered.
func (s *Server) newMCPServer() *mcp.Server {
	srv := mcp.NewServer(
		&mcp.Implementation{Name: "sqmgr-admin-analytics", Version: s.version},
		&mcp.ServerOptions{Instructions: mcpInstructions},
	)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_site_stats",
		Description: "Site-wide totals: pools (all/active/archived), registered users, guest users, site admins, active grids, claimed squares, and pool memberships. With a date range, each count covers records created in that range (claimed squares use the time they were last modified).",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in dateRangeInput) (interface{}, error) {
		r, err := in.dateRange()
		if err != nil {
			return nil, err
		}
		return a.SiteStats(ctx, r)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name: "get_time_series",
		Description: "Count a metric per day, week, month, or year, suitable for charting. Buckets with no activity are included with a count of 0. " +
			"Metrics: pools_created, users_registered (Auth0 accounts), guest_users_created, squares_claimed (claim events from the square log), grids_created, pool_members_joined. " +
			"Summing the counts answers questions like \"how many pools were created in the last six months\".",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in timeSeriesInput) (interface{}, error) {
		r, err := in.dateRange()
		if err != nil {
			return nil, err
		}
		interval := in.Interval
		if interval == "" {
			interval = model.IntervalMonth
		}
		points, err := a.TimeSeries(ctx, in.Metric, interval, r)
		if err != nil {
			return nil, err
		}
		var total int64
		for _, p := range points {
			total += p.Count
		}
		return map[string]interface{}{
			"metric":   in.Metric,
			"interval": interval,
			"total":    total,
			"points":   points,
		}, nil
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name: "get_pool_fill_rates",
		Description: "How full pools get: the number and percentage of pools with every square claimed, partially claimed, or untouched, plus average/median fill percentage and a histogram of fill buckets. " +
			"The date range applies to when the pool was created. Optionally restrict to one grid type (std100, std50, std25, roll100) or to archived/unarchived pools.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in poolFillRatesInput) (interface{}, error) {
		r, err := in.dateRange()
		if err != nil {
			return nil, err
		}
		return a.PoolFillRates(ctx, model.PoolFillRateFilter{Created: r, GridType: model.GridType(in.GridType), Archived: in.Archived})
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_pool_breakdown",
		Description: "Group pools (or grids/squares) by one dimension and return the count and percentage in each group. Dimensions: " + describeBreakdownDimensions() + ". The date range applies to when the record was created (last modified for square_state).",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in breakdownInput) (interface{}, error) {
		r, err := in.dateRange()
		if err != nil {
			return nil, err
		}
		return a.PoolBreakdown(ctx, in.Dimension, r)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name: "get_engagement_summary",
		Description: "User engagement within a date range: new registered and guest users, how many distinct users created pools, repeat creators (2+ pools in range), returning creators (also created a pool before the range start), " +
			"pools that received claims or were linked to a sports event, average members and grids per pool, and square claim events split by registered users, guest users, and anonymous claims.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in dateRangeInput) (interface{}, error) {
		r, err := in.dateRange()
		if err != nil {
			return nil, err
		}
		return a.EngagementSummary(ctx, r)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name: "list_pools",
		Description: "Search and page through pools with owner, member count, grid count, and fill percentage. Filter by name, owner email, creation date range, grid type, archived flag, and min/max fill percentage. " +
			"Sort by created (default), name, member_count, grid_count, total_squares, claimed_squares, or fill_percent. Returns the total number of matches for paging.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in listPoolsInput) (interface{}, error) {
		r, err := in.dateRange()
		if err != nil {
			return nil, err
		}
		return a.ListPools(ctx, model.PoolListFilter{
			Search:         in.Search,
			OwnerEmail:     in.OwnerEmail,
			Created:        r,
			GridType:       model.GridType(in.GridType),
			Archived:       in.Archived,
			MinFillPercent: in.MinFillPercent,
			MaxFillPercent: in.MaxFillPercent,
			SortBy:         in.SortBy,
			SortDir:        in.SortDir,
			Limit:          in.Limit,
			Offset:         in.Offset,
		})
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_pool",
		Description: "Full details for one pool by token: settings, owner, member/grid counts, squares by state (unclaimed, claimed, paid-partial, paid-full), active invite links, and every grid with its linked sports event.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in poolTokenInput) (interface{}, error) {
		return a.PoolDetails(ctx, in.Token)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_pool_squares",
		Description: "Every claimed square in a pool with the claimant name, claim state, and (when the claim was made by a registered user) the user's ID, account type, and email address. Set include_unclaimed to also list open squares.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in poolSquaresInput) (interface{}, error) {
		return a.PoolSquares(ctx, in.Token, in.IncludeUnclaimed)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_pool_members",
		Description: "Everyone with access to a pool: the owner, managers, and members, with email address, account type, when they joined, and how many squares each has claimed.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in poolTokenInput) (interface{}, error) {
		return a.PoolMembers(ctx, in.Token)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_pool_activity",
		Description: "The pool's square change log, newest first: each claim, unclaim, or payment-state change with who made it (including email when known), a note, and the remote address.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in poolActivityInput) (interface{}, error) {
		return a.PoolActivity(ctx, in.Token, in.Offset, in.Limit)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name: "list_users",
		Description: "Search and page through users with pools owned, pools joined, and squares claimed. store selects registered users (auth0, the default), guest users (sqmgr), or all. " +
			"Filter by email substring and sign-up date range; sort by created (default), id, pools_owned, pools_joined, or squares_claimed.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in listUsersInput) (interface{}, error) {
		r, err := in.dateRange()
		if err != nil {
			return nil, err
		}
		return a.ListUsers(ctx, model.UserListFilter{
			Search:  in.Search,
			Store:   in.Store,
			Created: r,
			SortBy:  in.SortBy,
			SortDir: in.SortDir,
			Limit:   in.Limit,
			Offset:  in.Offset,
		})
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_user",
		Description: "One user's profile and activity by user ID or exact email address: account type, admin flag, sign-up date, counts, and the pools they own and belong to.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in getUserInput) (interface{}, error) {
		return a.UserDetails(ctx, in.ID, in.Email)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_top_pool_creators",
		Description: "Users ranked by the number of pools they created in the date range, with the squares claimed and members across those pools. Useful for finding power users and commissioners.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in rankedInput) (interface{}, error) {
		r, err := in.dateRange()
		if err != nil {
			return nil, err
		}
		return a.TopPoolCreators(ctx, r, in.Limit)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_popular_events",
		Description: "Sports events ranked by how many active grids are linked to them, with the number of pools and claimed squares involved. Optionally filter by league (nfl, nba, wnba, ncaab, ncaaf) and event date range.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in popularEventsInput) (interface{}, error) {
		r, err := in.dateRange()
		if err != nil {
			return nil, err
		}
		return a.PopularEvents(ctx, in.League, r, in.Limit)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_sports_sync_runs",
		Description: "Recent runs of the sports data sync job (teams, schedule, scores) with timing, records processed, and any error, for checking that live scores are healthy.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in limitInput) (interface{}, error) {
		return a.SportsSyncRuns(ctx, in.Limit)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "describe_schema",
		Description: "The database tables, columns, and enum types, for writing run_sql_query queries.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, _ struct{}) (interface{}, error) {
		return a.Schema(ctx)
	}))

	mcp.AddTool(srv, &mcp.Tool{
		Name: "run_sql_query",
		Description: "Run an ad hoc PostgreSQL SELECT for questions the other tools do not answer. The query runs in a READ ONLY transaction with a 30 second timeout, must be a single statement (no semicolons), " +
			"is wrapped as a subquery so only SELECT/WITH/VALUES work, and is capped at max_rows rows (default 100, max 1000). Queries that reference password_hash or use escaped identifiers are rejected. Call describe_schema first to see the tables.",
		Annotations: readOnlyHint,
	}, mcpTool(s, func(ctx context.Context, a *model.Analytics, in sqlQueryInput) (interface{}, error) {
		return a.Query(ctx, in.Query, in.MaxRows)
	}))

	return srv
}

// mcpHandler serves the MCP server over streamable HTTP. It must be mounted
// behind the auth and admin middleware. Sessions are stateless and responses
// are plain JSON so the server's write timeout applies cleanly.
func (s *Server) mcpHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return s.mcpServer
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
}

// mcpToolFunc is the shape of an analytics tool: it receives an Analytics
// bound to a read-only transaction and returns a JSON-serializable result.
type mcpToolFunc[In any] func(ctx context.Context, a *model.Analytics, in In) (interface{}, error)

// mcpTool adapts an mcpToolFunc into an MCP tool handler, running it inside a
// read-only transaction and logging the call.
func mcpTool[In any](s *Server, fn mcpToolFunc[In]) mcp.ToolHandlerFor[In, interface{}] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, interface{}, error) {
		toolName := ""
		if req != nil && req.Params != nil {
			toolName = req.Params.Name
		}
		entry := logrus.WithField("tool", toolName)
		if user, ok := userFromContext(ctx); ok {
			entry = entry.WithField("userID", user.ID)
		}

		var out interface{}
		err := model.RunReadOnly(ctx, s.model.DB, mcpStatementTimeout, func(a *model.Analytics) error {
			var err error
			out, err = fn(ctx, a, in)
			return err
		})
		if err != nil {
			if isMCPUserError(err) {
				entry.WithError(err).Debug("mcp tool rejected input")
			} else {
				entry.WithError(err).Error("mcp tool failed")
			}
			return nil, nil, err
		}

		entry.Debug("mcp tool succeeded")
		return nil, out, nil
	}
}

// isMCPUserError reports whether an error describes bad input rather than a
// server problem.
func isMCPUserError(err error) bool {
	return errors.Is(err, model.ErrPoolNotFound) ||
		errors.Is(err, model.ErrUserNotFound) ||
		errors.Is(err, model.ErrInvalidQuery) ||
		errors.Is(err, model.ErrQueryFailed) ||
		errors.Is(err, errInvalidDate)
}

var errInvalidDate = errors.New("invalid date")

// parseDateBound parses a tool date argument. Calendar days are UTC; when the
// value is an end bound, the whole day is included by moving to the next
// midnight. RFC3339 timestamps are used as-is.
func parseDateBound(value string, isEnd bool) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		if isEnd {
			t = t.AddDate(0, 0, 1)
		}
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("%w %q; expected YYYY-MM-DD or an RFC3339 timestamp", errInvalidDate, value)
}

// parseDateRange builds a model.DateRange from start/end tool arguments.
func parseDateRange(start, end string) (model.DateRange, error) {
	var r model.DateRange
	var err error
	if r.Start, err = parseDateBound(start, false); err != nil {
		return r, err
	}
	if r.End, err = parseDateBound(end, true); err != nil {
		return r, err
	}
	if !r.Start.IsZero() && !r.End.IsZero() && !r.End.After(r.Start) {
		return r, fmt.Errorf("%w: start must be before end", errInvalidDate)
	}
	return r, nil
}

func describeBreakdownDimensions() string {
	dims := model.BreakdownDimensions()
	names := make([]string, 0, len(dims))
	for name := range dims {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, name+" ("+dims[name]+")")
	}
	return strings.Join(parts, "; ")
}

// Tool inputs. Descriptions become the JSON schema shown to MCP clients.

type dateRangeInput struct {
	Start string `json:"start,omitempty" jsonschema:"Inclusive start: YYYY-MM-DD (UTC day) or RFC3339 timestamp. Omit for no lower bound."`
	End   string `json:"end,omitempty" jsonschema:"End: YYYY-MM-DD (UTC day, the whole day is included) or an exclusive RFC3339 timestamp. Omit for no upper bound."`
}

func (in dateRangeInput) dateRange() (model.DateRange, error) {
	return parseDateRange(in.Start, in.End)
}

type timeSeriesInput struct {
	Metric   string `json:"metric" jsonschema:"One of: pools_created, users_registered, guest_users_created, squares_claimed, grids_created, pool_members_joined."`
	Interval string `json:"interval,omitempty" jsonschema:"Bucket size: day, week (starting Monday), month (default), or year."`
	Start    string `json:"start,omitempty" jsonschema:"Inclusive start: YYYY-MM-DD (UTC day) or RFC3339 timestamp. Omit for no lower bound."`
	End      string `json:"end,omitempty" jsonschema:"End: YYYY-MM-DD (UTC day, the whole day is included) or an exclusive RFC3339 timestamp. Omit for no upper bound."`
}

func (in timeSeriesInput) dateRange() (model.DateRange, error) {
	return parseDateRange(in.Start, in.End)
}

type poolFillRatesInput struct {
	Start    string `json:"start,omitempty" jsonschema:"Inclusive start of the pool creation range: YYYY-MM-DD (UTC day) or RFC3339 timestamp."`
	End      string `json:"end,omitempty" jsonschema:"End of the pool creation range: YYYY-MM-DD (UTC day, the whole day is included) or an exclusive RFC3339 timestamp."`
	GridType string `json:"grid_type,omitempty" jsonschema:"Only pools of this grid type: std100, std50, std25, or roll100."`
	Archived *bool  `json:"archived,omitempty" jsonschema:"true for only archived pools, false for only unarchived pools. Omit for both."`
}

func (in poolFillRatesInput) dateRange() (model.DateRange, error) {
	return parseDateRange(in.Start, in.End)
}

type breakdownInput struct {
	Dimension string `json:"dimension" jsonschema:"One of: grid_type, number_set_config, archived, password_required, open_access_on_lock, owner_store, league, square_state."`
	Start     string `json:"start,omitempty" jsonschema:"Inclusive start: YYYY-MM-DD (UTC day) or RFC3339 timestamp. Omit for no lower bound."`
	End       string `json:"end,omitempty" jsonschema:"End: YYYY-MM-DD (UTC day, the whole day is included) or an exclusive RFC3339 timestamp. Omit for no upper bound."`
}

func (in breakdownInput) dateRange() (model.DateRange, error) {
	return parseDateRange(in.Start, in.End)
}

type listPoolsInput struct {
	Search         string   `json:"search,omitempty" jsonschema:"Case-insensitive substring of the pool name, or an exact pool token."`
	OwnerEmail     string   `json:"owner_email,omitempty" jsonschema:"Case-insensitive substring of the pool owner's email address."`
	Start          string   `json:"start,omitempty" jsonschema:"Inclusive start of the creation date range: YYYY-MM-DD (UTC day) or RFC3339 timestamp."`
	End            string   `json:"end,omitempty" jsonschema:"End of the creation date range: YYYY-MM-DD (UTC day, the whole day is included) or an exclusive RFC3339 timestamp."`
	GridType       string   `json:"grid_type,omitempty" jsonschema:"Only pools of this grid type: std100, std50, std25, or roll100."`
	Archived       *bool    `json:"archived,omitempty" jsonschema:"true for only archived pools, false for only unarchived pools. Omit for both."`
	MinFillPercent *float64 `json:"min_fill_percent,omitempty" jsonschema:"Only pools with at least this percentage of squares claimed (0-100)."`
	MaxFillPercent *float64 `json:"max_fill_percent,omitempty" jsonschema:"Only pools with at most this percentage of squares claimed (0-100)."`
	SortBy         string   `json:"sort_by,omitempty" jsonschema:"created (default), name, member_count, grid_count, total_squares, claimed_squares, or fill_percent."`
	SortDir        string   `json:"sort_dir,omitempty" jsonschema:"asc or desc (default)."`
	Limit          int      `json:"limit,omitempty" jsonschema:"Page size, default 25, max 500."`
	Offset         int64    `json:"offset,omitempty" jsonschema:"Number of pools to skip for paging."`
}

func (in listPoolsInput) dateRange() (model.DateRange, error) {
	return parseDateRange(in.Start, in.End)
}

type poolTokenInput struct {
	Token string `json:"token" jsonschema:"The pool's token (the identifier in its URL)."`
}

type poolSquaresInput struct {
	Token            string `json:"token" jsonschema:"The pool's token (the identifier in its URL)."`
	IncludeUnclaimed bool   `json:"include_unclaimed,omitempty" jsonschema:"Also include squares that have not been claimed."`
}

type poolActivityInput struct {
	Token  string `json:"token" jsonschema:"The pool's token (the identifier in its URL)."`
	Limit  int    `json:"limit,omitempty" jsonschema:"Page size, default 25, max 500."`
	Offset int64  `json:"offset,omitempty" jsonschema:"Number of entries to skip for paging."`
}

type listUsersInput struct {
	Search  string `json:"search,omitempty" jsonschema:"Case-insensitive substring of the email address."`
	Store   string `json:"store,omitempty" jsonschema:"auth0 (registered users, default), sqmgr (guest users), or all."`
	Start   string `json:"start,omitempty" jsonschema:"Inclusive start of the sign-up date range: YYYY-MM-DD (UTC day) or RFC3339 timestamp."`
	End     string `json:"end,omitempty" jsonschema:"End of the sign-up date range: YYYY-MM-DD (UTC day, the whole day is included) or an exclusive RFC3339 timestamp."`
	SortBy  string `json:"sort_by,omitempty" jsonschema:"created (default), id, pools_owned, pools_joined, or squares_claimed."`
	SortDir string `json:"sort_dir,omitempty" jsonschema:"asc or desc (default)."`
	Limit   int    `json:"limit,omitempty" jsonschema:"Page size, default 25, max 500."`
	Offset  int64  `json:"offset,omitempty" jsonschema:"Number of users to skip for paging."`
}

func (in listUsersInput) dateRange() (model.DateRange, error) {
	return parseDateRange(in.Start, in.End)
}

type getUserInput struct {
	ID    int64  `json:"id,omitempty" jsonschema:"The user's numeric ID. Takes precedence over email."`
	Email string `json:"email,omitempty" jsonschema:"The user's exact (case-insensitive) email address."`
}

type rankedInput struct {
	Start string `json:"start,omitempty" jsonschema:"Inclusive start of the pool creation range: YYYY-MM-DD (UTC day) or RFC3339 timestamp."`
	End   string `json:"end,omitempty" jsonschema:"End of the pool creation range: YYYY-MM-DD (UTC day, the whole day is included) or an exclusive RFC3339 timestamp."`
	Limit int    `json:"limit,omitempty" jsonschema:"How many users to return, default 25, max 500."`
}

func (in rankedInput) dateRange() (model.DateRange, error) {
	return parseDateRange(in.Start, in.End)
}

type popularEventsInput struct {
	League string `json:"league,omitempty" jsonschema:"Only events in this league: nfl, nba, wnba, ncaab, or ncaaf."`
	Start  string `json:"start,omitempty" jsonschema:"Inclusive start of the event date range: YYYY-MM-DD (UTC day) or RFC3339 timestamp."`
	End    string `json:"end,omitempty" jsonschema:"End of the event date range: YYYY-MM-DD (UTC day, the whole day is included) or an exclusive RFC3339 timestamp."`
	Limit  int    `json:"limit,omitempty" jsonschema:"How many events to return, default 25, max 500."`
}

func (in popularEventsInput) dateRange() (model.DateRange, error) {
	return parseDateRange(in.Start, in.End)
}

type limitInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"How many rows to return, default 25, max 500."`
}

type sqlQueryInput struct {
	Query   string `json:"query" jsonschema:"A single PostgreSQL SELECT (or WITH/VALUES) statement without a trailing semicolon."`
	MaxRows int    `json:"max_rows,omitempty" jsonschema:"Maximum rows to return, default 100, max 1000."`
}
