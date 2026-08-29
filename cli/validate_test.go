package cli

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidate_Valid(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"] }
}`)

	result, err := Validate(ValidateOptions{PolicyPath: policyPath, AsOf: time.Now()})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !result.OK() {
		t.Errorf("OK() = false for a valid policy, issues = %+v", result.Issues)
	}
	if result.Policy == nil {
		t.Error("Policy = nil for a valid document")
	}
}

func TestValidate_SchemaViolation_UnknownField(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	// "extraField" is not part of the schema (additionalProperties: false),
	// but Go's json.Unmarshal silently ignores it -- this is exactly the
	// gap schema-level validation exists to close.
	writeCliFile(t, policyPath, `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"] },
  "extraField": "oops"
}`)

	result, err := Validate(ValidateOptions{PolicyPath: policyPath, AsOf: time.Now()})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if result.OK() {
		t.Fatal("OK() = true, want false: unknown top-level field should be a schema error")
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Errorf("expected at least one SeverityError issue, got %+v", result.Issues)
	}

	// The document is still structurally valid at the Go level (the
	// unknown field is just ignored by Unmarshal), so Policy should still
	// be populated -- schema errors don't block structural parsing.
	if result.Policy == nil {
		t.Error("Policy = nil, want non-nil (structural parse should still succeed)")
	}
}

func TestValidate_StructuralViolation_NotCaughtBySchema(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	// "relation" is schema-typed as a plain string, so the schema alone
	// accepts this; only policy.Validate's semantic check rejects it.
	writeCliFile(t, policyPath, `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"] },
  "relationships": [
    { "source": "go:github.com/orgA/a", "relation": "can_call", "target": "go:github.com/orgA/b", "reason": "test fixture" }
  ]
}`)

	result, err := Validate(ValidateOptions{PolicyPath: policyPath, AsOf: time.Now()})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if result.OK() {
		t.Fatal("OK() = true, want false: unsupported relation must fail structural validation")
	}
	if result.Policy != nil {
		t.Error("Policy = non-nil, want nil: structural parse failed")
	}
}

func TestValidate_ExpiredException_Warning(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"] },
  "exceptions": [
    { "source": "go:github.com/orgA/a", "target": "go:github.com/orgA/b", "effect": "allow", "expires": "2020-01-01", "reason": "old migration" }
  ]
}`)

	result, err := Validate(ValidateOptions{PolicyPath: policyPath, AsOf: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !result.OK() {
		t.Errorf("OK() = false, want true: an expired exception is a warning, not an error, issues = %+v", result.Issues)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Severity == SeverityWarning {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a warning for the expired exception, got %+v", result.Issues)
	}
}

func TestValidate_ReferentialIntegrity(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"] },
  "relationships": [
    { "source": "go:github.com/orgA/a", "relation": "can_depend_on", "target": "go:github.com/orgA/b", "reason": "test fixture" },
    { "source": "go:github.com/orgA/a", "relation": "can_depend_on", "target": "go:github.com/orgA/ghost", "reason": "test fixture" }
  ]
}`)

	graphPath := filepath.Join(root, "graph.json")
	writeCliFile(t, graphPath, `{
  "generated": "2026-01-01T00:00:00Z",
  "scan": { "level": "source-imports", "scanner": "test" },
  "dependencies": [
    { "source": "go:github.com/orgA/a", "relation": "depends_on", "target": "go:github.com/orgA/b" }
  ]
}`)

	result, err := Validate(ValidateOptions{PolicyPath: policyPath, GraphPath: graphPath, AsOf: time.Now()})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !result.OK() {
		t.Errorf("OK() = false, want true: a dangling reference is a warning, not an error, issues = %+v", result.Issues)
	}

	foundGhost := false
	for _, issue := range result.Issues {
		if issue.Severity == SeverityWarning && strings.Contains(issue.Message, "ghost") {
			foundGhost = true
		}
	}
	if !foundGhost {
		t.Errorf("expected a warning about the dangling 'ghost' reference, got %+v", result.Issues)
	}
}

func TestValidate_MissingPolicyFile(t *testing.T) {
	_, err := Validate(ValidateOptions{PolicyPath: filepath.Join(t.TempDir(), "no-such-file.json"), AsOf: time.Now()})
	if err == nil {
		t.Fatal("expected an error for a missing policy file")
	}
}

func TestValidate_MissingGraphFile(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, `{"schemaVersion":"1","scope":{"include":["go:github.com/orgA/*"]}}`)

	_, err := Validate(ValidateOptions{PolicyPath: policyPath, GraphPath: filepath.Join(root, "no-such-graph.json"), AsOf: time.Now()})
	if err == nil {
		t.Fatal("expected an error for a missing graph file")
	}
}
