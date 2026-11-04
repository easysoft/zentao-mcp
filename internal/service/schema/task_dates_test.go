package schema

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// Exercise both the OpenAPI loader and the JSON Schema advertised to MCP clients.
func TestExecutionTaskOutputAllowsNullableDates(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromFile("../../../docs/zentao-openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	op := doc.Paths.Value("/executions/:executionID/tasks").Get
	raw, _ := New(nil).buildToolOutputSchema(context.Background(), doc, op)
	resolved := resolveInputJSONSchema(t, raw)
	fields := []string{"activatedDate", "canceledDate", "closedDate", "deadline", "estStarted", "finishedDate", "realStarted"}
	for _, field := range fields {
		t.Run(field, func(t *testing.T) {
			property := op.Responses.Status(200).Value.Content["application/json"].Schema.Value.Properties["tasks"].Value.Items.Value.Properties[field].Value
			if err := property.Validate(context.Background()); err != nil {
				t.Fatal(err)
			}
			for _, value := range []any{nil, "", "2026-01-01", "2026-01-01 12:00:00"} {
				payload := map[string]any{"status": "success", "tasks": []any{map[string]any{field: value}}}
				if err := resolved.Validate(payload); err != nil {
					t.Errorf("value %#v rejected: %v", value, err)
				}
			}
			for _, value := range []any{42, true, map[string]any{}, []any{}} {
				payload := map[string]any{"status": "success", "tasks": []any{map[string]any{field: value}}}
				if err := resolved.Validate(payload); err == nil {
					t.Errorf("invalid value %#v accepted", value)
				}
			}
		})
	}
}
