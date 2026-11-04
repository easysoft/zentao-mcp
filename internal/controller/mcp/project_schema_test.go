package mcp

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

func TestProgramTreePaginationDescription(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/zentao-openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	op := doc["paths"].(map[string]any)["/programs"].(map[string]any)["get"].(map[string]any)
	// 工具构建优先使用 summary，关键分页说明必须出现在实际暴露的描述中。
	for _, key := range []string{"summary", "description"} {
		for _, term := range []string{"顶层分支", "pageTotal", "bysearch", "父节点"} {
			if !strings.Contains(op[key].(string), term) {
				t.Fatalf("%s 缺少 %s", key, term)
			}
		}
	}
}

func TestBugListScopeDescription(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/zentao-openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"/products/:productID/bugs", "/projects/:projectID/bugs", "/executions/:executionID/bugs"} {
		op := doc["paths"].(map[string]any)[route].(map[string]any)["get"].(map[string]any)
		for _, term := range []string{"未解决（active）", "未关闭（active、resolved）", "不是全局"} {
			if !strings.Contains(op["summary"].(string), term) {
				t.Fatalf("%s 缺少 %s", route, term)
			}
		}
		found := false
		for _, value := range op["parameters"].([]any) {
			param := value.(map[string]any)
			if param["name"] == "browseType" {
				found = strings.Contains(param["description"].(string), "兼容 assigntome") && strings.Contains(param["description"].(string), "默认 all")
			}
		}
		if !found {
			t.Fatalf("%s 缺少人员筛选别名说明", route)
		}
	}
}

func TestProjectManagerSchema(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/zentao-openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err = json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	at := func(value any, keys ...string) any {
		for _, key := range keys {
			value = value.(map[string]any)[key]
		}
		return value
	}
	project := at(doc, "paths", "/projects", "get", "responses", "200", "content", "application/json", "schema", "properties", "projects", "items")
	schemaJSON, _ := json.Marshal(project)
	var schema jsonschema.Schema
	if err = json.Unmarshal(schemaJSON, &schema); err != nil {
		t.Fatal(err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []map[string]any{
		{"PM": "Manager", "PMAvatar": "", "PMUserID": "46", "invested": 1.25},
		{"PM": "Manager", "PMAvatar": "/avatar.png", "PMUserID": 46},
		{"PM": "", "PMAvatar": "", "PMUserID": 0},
		{"PM": false, "PMAvatar": false, "PMUserID": false},
	} {
		normalized := normalizeStructuredOutput(schemaJSON, row)
		if err = resolved.Validate(normalized); err != nil {
			t.Fatal(err)
		}
		if normalized.(map[string]any)["PM"] != row["PM"] {
			t.Fatal("manager value changed")
		}
		if normalized.(map[string]any)["invested"] != row["invested"] {
			t.Fatal("fractional work amount changed")
		}
	}
	for _, path := range []string{"/programs", "/executions"} {
		params := at(doc, "paths", path, "get", "parameters").([]any)
		found := map[string]bool{}
		for _, p := range params {
			found[p.(map[string]any)["name"].(string)] = true
		}
		if !found["status"] || !found["browseType"] {
			t.Fatalf("missing filter in %s", path)
		}
	}
	product := at(doc, "paths", "/products", "get", "responses", "200", "content", "application/json", "schema", "properties", "products", "items")
	productJSON, _ := json.Marshal(product)
	var productSchema jsonschema.Schema
	if err = json.Unmarshal(productJSON, &productSchema); err != nil {
		t.Fatal(err)
	}
	productResolved, err := productSchema.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	rates := map[string]any{"storyCompleteRate": 12.25, "requirementCompleteRate": 33.33, "bugFixedRate": 99.99}
	output := normalizeStructuredOutput(productJSON, rates)
	if err = productResolved.Validate(output); err != nil {
		t.Fatal(err)
	}
	for key, value := range rates {
		if output.(map[string]any)[key] != value {
			t.Fatal("rate precision changed")
		}
	}
}
