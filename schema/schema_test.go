package schema

import (
	"encoding/json"
	"testing"
)

func TestPolicyJSON_IsValidJSONSchema(t *testing.T) {
	if len(PolicyJSON) == 0 {
		t.Fatal("PolicyJSON is empty; run `go generate ./schema/...`")
	}
	var doc map[string]any
	if err := json.Unmarshal(PolicyJSON, &doc); err != nil {
		t.Fatalf("PolicyJSON is not valid JSON: %v", err)
	}
	if doc["$schema"] == nil {
		t.Error("PolicyJSON missing $schema")
	}
	required, _ := doc["required"].([]any)
	if !containsString(required, "schemaVersion") || !containsString(required, "scope") {
		t.Errorf("PolicyJSON required = %v, want schemaVersion and scope", required)
	}
}

func TestGraphJSON_IsValidJSONSchema(t *testing.T) {
	if len(GraphJSON) == 0 {
		t.Fatal("GraphJSON is empty; run `go generate ./schema/...`")
	}
	var doc map[string]any
	if err := json.Unmarshal(GraphJSON, &doc); err != nil {
		t.Fatalf("GraphJSON is not valid JSON: %v", err)
	}
	if doc["$schema"] == nil {
		t.Error("GraphJSON missing $schema")
	}
}

func containsString(items []any, want string) bool {
	for _, item := range items {
		if s, ok := item.(string); ok && s == want {
			return true
		}
	}
	return false
}
