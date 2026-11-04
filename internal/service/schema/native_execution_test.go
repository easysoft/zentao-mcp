package schema

import (
	"context"
	"github.com/getkin/kin-openapi/openapi3"
	"testing"
)

type nativeExecutionLoader struct{ doc *openapi3.T }

func (l nativeExecutionLoader) Load(context.Context) (*openapi3.T, error) { return l.doc, nil }

func TestNativeExecutionToolContracts(t *testing.T) {
	doc, err := openapi3.NewLoader().LoadFromFile("../../../docs/zentao-openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	tools, err := New(nativeExecutionLoader{doc}).Tools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	effortChecked := false
	for _, td := range tools {
		if td.OperationID == "get_executions_executionID" {
			effortChecked = true
			output := resolveInputJSONSchema(t, td.OutputSchema)
			if err := output.Validate(map[string]any{"status": "success", "execution": map[string]any{"totalEstimate": 6.25, "totalConsumed": 101.8, "totalLeft": 2.4}}); err != nil {
				t.Fatal(err)
			}
		}
		switch td.OperationID {
		case "get_testtasks_id_cases", "post_testtasks_id_cases", "get_testtasks_id_cases_caseID_results", "post_testtasks_id_cases_caseID_results", "get_testcases_id_results", "post_testcases_id_results":
			if found[td.OperationID] {
				t.Fatalf("duplicate tool %s", td.OperationID)
			}
			found[td.OperationID] = true
			input := resolveInputJSONSchema(t, td.InputSchema)
			if err := input.Validate(map[string]any{}); err == nil {
				t.Fatalf("missing path IDs accepted: %s", td.OperationID)
			}
			if td.OperationID == "post_testtasks_id_cases" {
				good := map[string]any{"id": 1, "payload": map[string]any{"cases": []any{map[string]any{"case": 2, "version": 1}}}}
				if err := input.Validate(good); err != nil {
					t.Fatal(err)
				}
				good["payload"] = map[string]any{"cases": []any{map[string]any{"case": 2}}}
				if err := input.Validate(good); err == nil {
					t.Fatal("missing explicit version accepted")
				}
			}
			if td.OperationID == "post_testtasks_id_cases_caseID_results" {
				good := map[string]any{"id": 1, "caseID": 2, "payload": map[string]any{"version": 1, "steps": []any{map[string]any{"result": "pass", "real": "synthetic"}}}}
				if err := input.Validate(good); err != nil {
					t.Fatal(err)
				}
				output := resolveInputJSONSchema(t, td.OutputSchema)
				if err := output.Validate(map[string]any{"status": "success", "resultID": 7, "runID": 8, "testtask": 1, "case": 2, "version": 1}); err != nil {
					t.Fatal(err)
				}
				if err := output.Validate(map[string]any{"status": "success"}); err == nil {
					t.Fatal("result IDs missing but accepted")
				}
			}
		}
	}
	if len(found) != 6 {
		t.Fatalf("expected 6 native tools, got %d", len(found))
	}
	if !effortChecked {
		t.Fatal("execution detail effort schema was not exercised")
	}
}
