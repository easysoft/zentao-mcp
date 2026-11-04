package mcp

import (
	"context"
	"encoding/json"
	"github.com/merzzzl/openapi-mcp-server/internal/models"
	"github.com/merzzzl/openapi-mcp-server/internal/service/tool"
	"strings"
	"testing"
)

func TestExpectedJSONCannotSilentlySucceed(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		schema bool
		want   bool
	}{
		{200, "", true, true}, {200, "  ", true, true}, {200, `{"status":"success"}{"status":"success"}`, true, true},
		{200, "<html>error</html>", true, true}, {200, `{"status":"success"}`, true, false},
		{204, "", true, false}, {205, "", true, false}, {200, "plain text", false, false},
	} {
		td := &models.ToolDefinition{Method: "GET", Path: "/test"}
		if tc.schema {
			td.OutputSchema = json.RawMessage(`{"type":"object"}`)
		}
		c := New(nil, tool.New(errorTestProxy{tc.status, tc.body}, "https://example.test"), nil, false)
		result, structured, err := c.toolHandler(td)(context.Background(), nil, map[string]any{})
		if err != nil || result.IsError != tc.want {
			t.Fatalf("status %d body %q: %v, %v", tc.status, tc.body, result, err)
		}
		if tc.want {
			if structured != nil {
				t.Fatal("error has success output")
			}
			b, _ := json.Marshal(result.Content)
			if !strings.Contains(string(b), "上游接口未返回有效 JSON 数据") {
				t.Fatal("missing explanation")
			}
		}
	}
}
