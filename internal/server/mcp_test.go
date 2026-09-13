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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

// expectedMCPTools is every tool the analytics server must expose.
var expectedMCPTools = []string{
	"get_site_stats",
	"get_time_series",
	"get_pool_fill_rates",
	"get_pool_breakdown",
	"get_engagement_summary",
	"list_pools",
	"get_pool",
	"list_pool_squares",
	"list_pool_members",
	"list_pool_activity",
	"list_users",
	"get_user",
	"get_top_pool_creators",
	"list_popular_events",
	"list_sports_sync_runs",
	"describe_schema",
	"run_sql_query",
}

// newMCPTestServer builds a Server with a mocked database and its MCP server.
func newMCPTestServer(t *testing.T) (*Server, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	s := &Server{
		model:   model.New(db),
		version: "test",
		broker:  NewPoolBroker(),
	}
	s.mcpServer = s.newMCPServer()
	return s, mock
}

// connectMCPClient connects an in-memory MCP client to the server's MCP server,
// using ctx as the server-side context.
func connectMCPClient(t *testing.T, ctx context.Context, s *Server) *mcp.ClientSession {
	t.Helper()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := s.mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestMCPServer_ListsReadOnlyTools(t *testing.T) {
	g := gomega.NewWithT(t)
	s, _ := newMCPTestServer(t)
	ctx := context.Background()
	session := connectMCPClient(t, ctx, s)

	res, err := session.ListTools(ctx, nil)
	g.Expect(err).Should(gomega.Succeed())

	names := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
		g.Expect(tool.Description).ShouldNot(gomega.BeEmpty(), tool.Name)
		g.Expect(tool.Annotations).ShouldNot(gomega.BeNil(), tool.Name)
		g.Expect(tool.Annotations.ReadOnlyHint).Should(gomega.BeTrue(), tool.Name)
		g.Expect(tool.InputSchema).ShouldNot(gomega.BeNil(), tool.Name)
	}
	g.Expect(names).Should(gomega.ConsistOf(expectedMCPTools))

	// Input schemas carry the field descriptions and required fields
	var query *mcp.Tool
	for _, tool := range res.Tools {
		if tool.Name == "run_sql_query" {
			query = tool
		}
	}
	g.Expect(query).ShouldNot(gomega.BeNil())
	schema, err := json.Marshal(query.InputSchema)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(string(schema)).Should(gomega.ContainSubstring(`"required":["query"]`))
	g.Expect(string(schema)).Should(gomega.ContainSubstring("max_rows"))
	g.Expect(string(schema)).Should(gomega.ContainSubstring("trailing semicolon"))

	g.Expect(session.InitializeResult().Instructions).Should(gomega.ContainSubstring("READ ONLY"))
	g.Expect(session.InitializeResult().ServerInfo.Name).Should(gomega.Equal("sqmgr-admin-analytics"))
}

func TestMCPServer_ToolRunsInReadOnlyTransaction(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newMCPTestServer(t)

	// Capture log entries so the admin's identity can be checked
	hook := logrustest.NewGlobal()
	defer hook.Reset()
	prevLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetLevel(prevLevel)

	admin := &model.User{ID: 42, IsSiteAdmin: true}
	ctx := context.WithValue(context.Background(), ctxUserKey, admin)
	session := connectMCPClient(t, ctx, s)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = 30000")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("information_schema.columns").WillReturnRows(
		sqlmock.NewRows([]string{"table_name", "column_name", "udt_name", "nullable"}).
			AddRow("pools", "id", "int8", false).
			AddRow("pools", "name", "text", false).
			AddRow("users", "email", "text", true),
	)
	mock.ExpectQuery("pg_enum").WillReturnRows(
		sqlmock.NewRows([]string{"typname", "enumlabel"}).
			AddRow("stores", "sqmgr").
			AddRow("stores", "auth0"),
	)
	mock.ExpectRollback()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "describe_schema"})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(res.IsError).Should(gomega.BeFalse())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	var schema model.Schema
	decodeStructuredContent(t, res, &schema)
	g.Expect(schema.Tables).Should(gomega.HaveLen(2))
	g.Expect(schema.Tables[0].Name).Should(gomega.Equal("pools"))
	g.Expect(schema.Tables[0].Columns).Should(gomega.HaveLen(2))
	g.Expect(schema.Tables[1].Columns[0].Nullable).Should(gomega.BeTrue())
	g.Expect(schema.Enums).Should(gomega.Equal([]model.SchemaEnum{{Name: "stores", Values: []string{"sqmgr", "auth0"}}}))

	// The text content mirrors the structured content for older clients
	g.Expect(res.Content).Should(gomega.HaveLen(1))
	text, ok := res.Content[0].(*mcp.TextContent)
	g.Expect(ok).Should(gomega.BeTrue())
	g.Expect(text.Text).Should(gomega.ContainSubstring(`"pools"`))

	// The call was logged against the admin who made it
	var logged bool
	for _, entry := range hook.AllEntries() {
		if entry.Data["tool"] == "describe_schema" && entry.Data["userID"] == int64(42) {
			logged = true
		}
	}
	g.Expect(logged).Should(gomega.BeTrue(), "expected a log entry with the tool name and user ID")
}

func TestMCPServer_ToolErrorsAreReportedAsToolErrors(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newMCPTestServer(t)
	ctx := context.Background()
	session := connectMCPClient(t, ctx, s)

	// Bad input never reaches the database beyond the transaction itself
	mock.ExpectBegin()
	mock.ExpectExec("SET LOCAL statement_timeout").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_site_stats", Arguments: map[string]interface{}{"start": "last tuesday"}})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(res.IsError).Should(gomega.BeTrue())
	g.Expect(res.Content[0].(*mcp.TextContent).Text).Should(gomega.ContainSubstring("invalid date"))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	// A missing pool is a tool error, not a protocol error
	mock.ExpectBegin()
	mock.ExpectExec("SET LOCAL statement_timeout").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT id, user_id FROM pools WHERE token").WithArgs("missing").WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}))
	mock.ExpectRollback()
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "list_pool_squares", Arguments: map[string]interface{}{"token": "missing"}})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(res.IsError).Should(gomega.BeTrue())
	g.Expect(res.Content[0].(*mcp.TextContent).Text).Should(gomega.ContainSubstring("pool not found"))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	// A rejected ad hoc query is a tool error too
	mock.ExpectBegin()
	mock.ExpectExec("SET LOCAL statement_timeout").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "run_sql_query", Arguments: map[string]interface{}{"query": "SELECT 1; DROP TABLE pools"}})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(res.IsError).Should(gomega.BeTrue())
	g.Expect(res.Content[0].(*mcp.TextContent).Text).Should(gomega.ContainSubstring("single statement"))

	// Schema validation rejects a missing required argument before any query runs
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "get_pool", Arguments: map[string]interface{}{}})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(res.IsError).Should(gomega.BeTrue())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	// Database failures surface as tool errors without leaking a protocol error
	mock.ExpectBegin().WillReturnError(errors.New("database down"))
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "get_site_stats"})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(res.IsError).Should(gomega.BeTrue())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())
}

func TestMCPServer_TimeSeriesTool(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newMCPTestServer(t)
	ctx := context.Background()
	session := connectMCPClient(t, ctx, s)

	mock.ExpectBegin()
	mock.ExpectExec("SET LOCAL statement_timeout").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT date_trunc('month', created) AS period, COUNT(*) FROM pools WHERE created >= $1 AND created < $2 GROUP BY 1 ORDER BY 1")).
		WithArgs(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)).
		WillReturnRows(sqlmock.NewRows([]string{"period", "count"}).
			AddRow(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), int64(3)).
			AddRow(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), int64(5)))
	mock.ExpectRollback()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_time_series", Arguments: map[string]interface{}{
		"metric": "pools_created",
		"start":  "2024-01-01",
		"end":    "2024-03-31",
	}})
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(res.IsError).Should(gomega.BeFalse(), "%v", res.Content)
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	var out struct {
		Metric   string                  `json:"metric"`
		Interval string                  `json:"interval"`
		Total    int64                   `json:"total"`
		Points   []model.TimeSeriesPoint `json:"points"`
	}
	decodeStructuredContent(t, res, &out)
	g.Expect(out.Metric).Should(gomega.Equal("pools_created"))
	g.Expect(out.Interval).Should(gomega.Equal("month"), "month is the default interval")
	g.Expect(out.Total).Should(gomega.Equal(int64(8)))
	g.Expect(out.Points).Should(gomega.Equal([]model.TimeSeriesPoint{
		{Period: "2024-01-01", Count: 3},
		{Period: "2024-02-01", Count: 0},
		{Period: "2024-03-01", Count: 5},
	}))
}

// decodeStructuredContent unmarshals a tool result's structured content into out.
func decodeStructuredContent(t *testing.T, res *mcp.CallToolResult, out interface{}) {
	t.Helper()
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decoding structured content %s: %v", raw, err)
	}
}

// mcpRequest builds a streamable-HTTP POST carrying one JSON-RPC message.
func mcpRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/admin/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	return req
}

func TestMCPHandler_RequiresSiteAdmin(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newMCPTestServer(t)
	handler := s.adminHandler(s.mcpHandler())
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`

	// No authenticated user in context: the auth middleware never ran
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, mcpRequest(t, body))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusInternalServerError))

	// Authenticated but not a site admin
	rec = httptest.NewRecorder()
	req := mcpRequest(t, body)
	ctx := context.WithValue(req.Context(), ctxUserKey, &model.User{ID: 7, IsSiteAdmin: false})
	handler.ServeHTTP(rec, req.WithContext(ctx))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusForbidden))
	var errResp ErrorResponse
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &errResp)).Should(gomega.Succeed())
	g.Expect(errResp.Code).Should(gomega.Equal(ErrCodeForbidden))

	// Site admin: the request reaches the MCP server
	rec = httptest.NewRecorder()
	req = mcpRequest(t, body)
	ctx = context.WithValue(req.Context(), ctxUserKey, &model.User{ID: 8, IsSiteAdmin: true})
	handler.ServeHTTP(rec, req.WithContext(ctx))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK), rec.Body.String())
	g.Expect(rec.Header().Get("Content-Type")).Should(gomega.HavePrefix("application/json"))

	var rpc struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &rpc)).Should(gomega.Succeed())
	names := make([]string, 0, len(rpc.Result.Tools))
	for _, tool := range rpc.Result.Tools {
		names = append(names, tool.Name)
	}
	g.Expect(names).Should(gomega.ConsistOf(expectedMCPTools))
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed(), "listing tools must not touch the database")
}

func TestMCPHandler_StatelessToolCallOverHTTP(t *testing.T) {
	g := gomega.NewWithT(t)
	s, mock := newMCPTestServer(t)
	handler := s.adminHandler(s.mcpHandler())

	mock.ExpectBegin()
	mock.ExpectExec("SET LOCAL statement_timeout").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM (\nSELECT 1 AS one\n) AS mcp_query LIMIT $1")).WithArgs(6).
		WillReturnRows(sqlmock.NewRows([]string{"one"}).AddRow(int64(1)))
	mock.ExpectRollback()

	body := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"run_sql_query","arguments":{"query":"SELECT 1 AS one","max_rows":5}}}`
	rec := httptest.NewRecorder()
	req := mcpRequest(t, body)
	ctx := context.WithValue(req.Context(), ctxUserKey, &model.User{ID: 9, IsSiteAdmin: true})
	handler.ServeHTTP(rec, req.WithContext(ctx))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusOK), rec.Body.String())
	g.Expect(mock.ExpectationsWereMet()).Should(gomega.Succeed())

	var rpc struct {
		Result struct {
			IsError           bool              `json:"isError"`
			StructuredContent model.QueryResult `json:"structuredContent"`
		} `json:"result"`
	}
	g.Expect(json.Unmarshal(rec.Body.Bytes(), &rpc)).Should(gomega.Succeed())
	g.Expect(rpc.Result.IsError).Should(gomega.BeFalse())
	g.Expect(rpc.Result.StructuredContent.Columns).Should(gomega.Equal([]string{"one"}))
	g.Expect(rpc.Result.StructuredContent.RowCount).Should(gomega.Equal(1))

	// Stateless servers do not offer the SSE listening stream
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/mcp", nil)
	req.Header.Set("Accept", "text/event-stream")
	ctx = context.WithValue(req.Context(), ctxUserKey, &model.User{ID: 9, IsSiteAdmin: true})
	handler.ServeHTTP(rec, req.WithContext(ctx))
	g.Expect(rec.Code).Should(gomega.Equal(http.StatusMethodNotAllowed))
}

func TestParseDateBound(t *testing.T) {
	g := gomega.NewWithT(t)

	zero, err := parseDateBound("  ", true)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(zero.IsZero()).Should(gomega.BeTrue())

	start, err := parseDateBound("2024-03-15", false)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(start).Should(gomega.Equal(time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)))

	// A calendar end date includes the whole day
	end, err := parseDateBound("2024-03-15", true)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(end).Should(gomega.Equal(time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC)))

	// RFC3339 timestamps are taken as-is, normalized to UTC
	ts, err := parseDateBound("2024-03-15T10:30:00-05:00", true)
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(ts).Should(gomega.Equal(time.Date(2024, 3, 15, 15, 30, 0, 0, time.UTC)))

	_, err = parseDateBound("March 15", false)
	g.Expect(errors.Is(err, errInvalidDate)).Should(gomega.BeTrue())
	g.Expect(isMCPUserError(err)).Should(gomega.BeTrue())
}

func TestParseDateRange(t *testing.T) {
	g := gomega.NewWithT(t)

	r, err := parseDateRange("", "")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(r.IsZero()).Should(gomega.BeTrue())

	r, err = parseDateRange("2024-01-01", "2024-01-01")
	g.Expect(err).Should(gomega.Succeed())
	g.Expect(r.Start).Should(gomega.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))
	g.Expect(r.End).Should(gomega.Equal(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)))

	_, err = parseDateRange("2024-02-01", "2024-01-01")
	g.Expect(err).Should(gomega.MatchError(gomega.ContainSubstring("start must be before end")))

	_, err = parseDateRange("2024-01-01", "soon")
	g.Expect(errors.Is(err, errInvalidDate)).Should(gomega.BeTrue())
}

func TestIsMCPUserError(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(isMCPUserError(model.ErrPoolNotFound)).Should(gomega.BeTrue())
	g.Expect(isMCPUserError(model.ErrUserNotFound)).Should(gomega.BeTrue())
	g.Expect(isMCPUserError(model.ErrInvalidQuery)).Should(gomega.BeTrue())
	g.Expect(isMCPUserError(model.ErrQueryFailed)).Should(gomega.BeTrue())
	g.Expect(isMCPUserError(errors.New("connection refused"))).Should(gomega.BeFalse())
}

func TestDescribeBreakdownDimensions(t *testing.T) {
	g := gomega.NewWithT(t)

	desc := describeBreakdownDimensions()
	for name := range model.BreakdownDimensions() {
		g.Expect(desc).Should(gomega.ContainSubstring(name + " ("))
	}
	// Sorted output keeps the tool description stable
	g.Expect(strings.Index(desc, "archived (")).Should(gomega.BeNumerically("<", strings.Index(desc, "grid_type (")))
}
