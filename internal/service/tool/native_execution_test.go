package tool

import "testing"

func TestNativeExecutionEvidenceIsNotLogged(t *testing.T) {
	input := map[string]any{"id": 1, "payload": map[string]any{"steps": []any{map[string]any{"real": "private evidence"}}}, "Token": "secret"}
	safe := redactInput(input)
	if safe["payload"] != redactedValue || safe["Token"] != redactedValue || safe["id"] != 1 {
		t.Fatal("private payload leaked or resource ID lost")
	}
	if _, ok := input["payload"].(map[string]any); !ok {
		t.Fatal("request input mutated")
	}
}
