package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/merzzzl/openapi-mcp-server/internal/models"
	"github.com/merzzzl/openapi-mcp-server/internal/service/tool"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type errorTestProxy struct {
	status int
	body   string
}

func (p errorTestProxy) Do(context.Context, *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: p.status, Body: io.NopCloser(strings.NewReader(p.body))}, nil
}

// Exercise SDK output validation, not just the handler's return value.
func TestToolFailureSkipsSuccessOutputSchema(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		failure bool
	}{
		{"forbidden", 403, `{"error":"Access not allowed"}`, true},
		{"unauthorized", 401, `{"error":"Unauthorized"}`, true},
		{"nonJSON", 502, `Bad gateway`, true},
		{"applicationFailure", 200, `{"status":"fail","message":"Program is not allowed."}`, true},
		{"success", 200, `{"status":"success","items":[]}`, false},
		{"objectStatus", 200, `{"status":{"value":"fail"},"items":[]}`, false},
	} {
		for _, wrap := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{true: "Wrapped", false: ""}[wrap], func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				schema := json.RawMessage(`{"type":"object","required":["items"],"properties":{"items":{"type":"array"}}}`)
				if wrap {
					schema = json.RawMessage(`{"type":"object","required":["result"],"properties":{"result":{"type":"object","required":["items"]}}}`)
				}
				td := &models.ToolDefinition{OperationID: "test", Method: "GET", Path: "/test", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: schema, WrapOutput: wrap}
				controller := New(nil, tool.New(errorTestProxy{tc.status, tc.body}, "https://example.test"), nil, false)
				server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "1"}, nil)
				controller.registerTool(ctx, server, td)
				st, ct := mcpsdk.NewInMemoryTransports()
				ss, err := server.Connect(ctx, st, nil)
				if err != nil {
					t.Fatal(err)
				}
				defer ss.Close()
				client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "client", Version: "1"}, nil)
				cs, err := client.Connect(ctx, ct, nil)
				if err != nil {
					t.Fatal(err)
				}
				defer cs.Close()
				result, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "test", Arguments: map[string]any{}})
				if err != nil {
					t.Fatal(err)
				}
				if result.IsError != tc.failure {
					t.Fatalf("isError=%v", result.IsError)
				}
				if tc.failure {
					if result.StructuredContent != nil {
						t.Fatal("failure must not use success schema")
					}
					got := result.Content[1].(*mcpsdk.TextContent).Text
					var original, returned any
					if json.Unmarshal([]byte(tc.body), &original) == nil {
						if json.Unmarshal([]byte(got), &returned) != nil || !reflect.DeepEqual(original, returned) {
							t.Fatal("upstream JSON error changed")
						}
					} else if got != tc.body {
						t.Fatal("upstream text error changed")
					}
				} else if result.StructuredContent == nil {
					t.Fatal("success missing structured output")
				}
			})
		}
	}
}
