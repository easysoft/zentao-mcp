package tool

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/merzzzl/openapi-mcp-server/internal/models"
)

func TestExecutionTaskFilterAlias(t *testing.T) {
	for _, path := range []string{"/executions/:executionID/tasks", "/executions/{executionID}/tasks"} {
		for _, tc := range []struct{ input, want string }{
			{"status=all&pageID=2", "browseType=all&pageID=2"},
			{"status=closed", "browseType=closed"},
			{"status=all&browseType=closed", "browseType=closed"},
			{"status=all&browseType=", "browseType=all"},
			{"browseType=all&recPerPage=50&orderBy=id_desc", "browseType=all&orderBy=id_desc&recPerPage=50"},
			{"pageID=1", "pageID=1"},
			{"status=", ""},
		} {
			q, _ := url.ParseQuery(tc.input)
			applyExecutionTaskFilterAlias("GET", path, q)
			if q.Encode() != tc.want {
				t.Errorf("%s %s: got %s want %s", path, tc.input, q.Encode(), tc.want)
			}
		}
	}
	for _, tc := range []struct{ method, path string }{
		{"GET", "/tasks"},
		{"POST", "/executions/:executionID/tasks"},
		{"PUT", "/executions/{executionID}/tasks"},
	} {
		q := url.Values{"status": {"all"}}
		applyExecutionTaskFilterAlias(tc.method, tc.path, q)
		if q.Encode() != "status=all" {
			t.Errorf("unrelated operation modified: %s %s", tc.method, tc.path)
		}
	}
}

type taskFilterCaptureProxy struct{ request *http.Request }

func (p *taskFilterCaptureProxy) Do(_ context.Context, request *http.Request) (*http.Response, error) {
	p.request = request
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status":"success","tasks":[]}`))}, nil
}

// Load query parameters from the real specification and verify the wire request.
func TestExecutionTaskFilterForwarding(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromFile("../../../docs/zentao-openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	path := "/executions/:executionID/tasks"
	definition := &models.ToolDefinition{Method: "GET", Path: path}
	foundBrowseType := false
	for _, parameter := range doc.Paths.Value(path).Get.Parameters {
		p := parameter.Value
		definition.Params = append(definition.Params, models.ToolParam{Name: p.Name, In: p.In})
		if p.Name == "browseType" && p.In == "query" {
			foundBrowseType = true
		}
	}
	if !foundBrowseType {
		t.Fatal("browseType missing from bundled specification")
	}
	for _, input := range []map[string]any{
		{"executionID": 123, "status": "all", "pageID": "2"},
		{"executionID": 123, "status": "closed", "browseType": "all", "pageID": "2"},
	} {
		proxy := &taskFilterCaptureProxy{}
		_, _, err := New(proxy, "https://example.invalid/api.php/v2").Execute(context.Background(), definition, input)
		if err != nil {
			t.Fatal(err)
		}
		if proxy.request.URL.Path != "/api.php/v2/executions/123/tasks" {
			t.Fatal(proxy.request.URL.Path)
		}
		if proxy.request.URL.RawQuery != "browseType=all&pageID=2" {
			t.Fatal(proxy.request.URL.RawQuery)
		}
	}
}
