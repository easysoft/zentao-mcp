package tool

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/merzzzl/openapi-mcp-server/internal/models"
)

type listFilterProxy struct{ query url.Values }

func (p *listFilterProxy) Do(_ context.Context, req *http.Request) (*http.Response, error) {
	p.query = req.URL.Query()
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
}

func TestListFilterAlias(t *testing.T) {
	for _, tc := range []struct{ method, path, query, want string }{
		{"GET", "/programs", "status=doing", "browseType=doing"},
		{"GET", "/executions", "status=all", "browseType=all"},
		{"GET", "/executions", "status=all&browseType=closed", "browseType=closed"},
		{"GET", "/programs", "browseType=&status=wait", "browseType=wait"},
		{"GET", "/programs", "", ""},
		{"GET", "/projects", "status=doing", "status=doing"},
		{"GET", "/executions/{executionID}/tasks", "status=all", "status=all"},
		{"POST", "/executions", "status=doing", "status=doing"},
	} {
		t.Run(tc.method+tc.path+tc.query, func(t *testing.T) {
			q, _ := url.ParseQuery(tc.query)
			applyListFilterAlias(tc.method, tc.path, q)
			if q.Encode() != tc.want {
				t.Fatalf("got %s, want %s", q.Encode(), tc.want)
			}
		})
	}
}

func TestExecuteAppliesListFilterAlias(t *testing.T) {
	proxy := &listFilterProxy{}
	svc := New(proxy, "https://example.test/api.php/v2")
	td := &models.ToolDefinition{Method: "GET", Path: "/executions", Params: []models.ToolParam{{Name: "status", In: "query"}, {Name: "browseType", In: "query"}}}
	_, _, err := svc.Execute(context.Background(), td, map[string]any{"status": "all", "browseType": "closed"})
	if err != nil {
		t.Fatal(err)
	}
	if proxy.query.Encode() != "browseType=closed" {
		t.Fatal(proxy.query)
	}
}
