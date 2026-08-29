package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const promoteTestPolicy = `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"] },
  "relationships": [
    {
      "source": "go:github.com/orgA/consumer",
      "relation": "can_depend_on",
      "target": "go:github.com/orgA/authorized",
      "reason": "already reviewed"
    }
  ]
}`

const promoteTestGraph = `{
  "generated": "2026-01-01T00:00:00Z",
  "scan": { "level": "source-imports", "scanner": "test" },
  "dependencies": [
    {
      "source": "go:github.com/orgA/consumer",
      "relation": "depends_on",
      "target": "go:github.com/orgA/authorized"
    },
    {
      "source": "go:github.com/orgA/consumer",
      "relation": "depends_on",
      "target": "go:github.com/orgA/unauthorized",
      "evidence": [
        { "kind": "source-import", "file": "main.go", "line": 7 }
      ]
    }
  ]
}`

func buildPromoteFixture(t *testing.T) (policyPath, graphPath string) {
	t.Helper()
	root := t.TempDir()

	policyPath = filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, promoteTestPolicy)

	graphPath = filepath.Join(root, "graph.json")
	writeCliFile(t, graphPath, promoteTestGraph)

	return policyPath, graphPath
}

func TestPromote_ProposesOnlyUnauthorizedEdges(t *testing.T) {
	policyPath, graphPath := buildPromoteFixture(t)

	result, err := Promote(PromoteOptions{PolicyPath: policyPath, GraphPath: graphPath, AsOf: time.Now()})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	if len(result.Proposed) != 1 {
		t.Fatalf("Proposed = %+v, want 1 (only the unauthorized edge)", result.Proposed)
	}
	p := result.Proposed[0]
	if p.Source != "go:github.com/orgA/consumer" || p.Target != "go:github.com/orgA/unauthorized" {
		t.Errorf("Proposed[0] = %+v, want consumer -> unauthorized", p)
	}
	if p.Relation != "can_depend_on" {
		t.Errorf("Proposed[0].Relation = %q, want can_depend_on", p.Relation)
	}
	if p.Reason == "" {
		t.Error("Proposed[0].Reason is empty, want a placeholder reason")
	}
	if len(p.Evidence) != 1 || p.Evidence[0].File != "main.go" || p.Evidence[0].Line != 7 {
		t.Errorf("Proposed[0].Evidence = %+v, want main.go:7", p.Evidence)
	}
}

func TestPromote_DoesNotModifyOriginalPolicyFile(t *testing.T) {
	policyPath, graphPath := buildPromoteFixture(t)

	before, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatalf("read policy before Promote: %v", err)
	}

	if _, err := Promote(PromoteOptions{PolicyPath: policyPath, GraphPath: graphPath, AsOf: time.Now()}); err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	after, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatalf("read policy after Promote: %v", err)
	}
	if string(before) != string(after) {
		t.Error("Promote must never modify the original policy file")
	}
}

func TestPromote_ProposedPolicyIsValidAndIncludesOriginal(t *testing.T) {
	policyPath, graphPath := buildPromoteFixture(t)

	result, err := Promote(PromoteOptions{PolicyPath: policyPath, GraphPath: graphPath, AsOf: time.Now()})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}

	if err := result.ProposedPolicy.Validate(); err != nil {
		t.Errorf("ProposedPolicy.Validate() error = %v, want nil (must be a usable draft)", err)
	}

	if len(result.ProposedPolicy.Relationships) != 2 {
		t.Errorf("ProposedPolicy.Relationships = %+v, want 2 (1 original + 1 proposed)", result.ProposedPolicy.Relationships)
	}
	if !result.ProposedPolicy.HasRelationship("go:github.com/orgA/consumer", "go:github.com/orgA/authorized") {
		t.Error("ProposedPolicy must retain the original relationship")
	}
	if !result.ProposedPolicy.HasRelationship("go:github.com/orgA/consumer", "go:github.com/orgA/unauthorized") {
		t.Error("ProposedPolicy must include the newly proposed relationship")
	}
}

func TestPromote_NoViolations_NoProposals(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, `{"schemaVersion":"1","scope":{"include":["go:github.com/orgA/*"]}}`)
	graphPath := filepath.Join(root, "graph.json")
	writeCliFile(t, graphPath, `{"scan":{"level":"source-imports","scanner":"test"}}`)

	result, err := Promote(PromoteOptions{PolicyPath: policyPath, GraphPath: graphPath, AsOf: time.Now()})
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	if len(result.Proposed) != 0 {
		t.Errorf("Proposed = %+v, want none", result.Proposed)
	}
	if len(result.ProposedPolicy.Relationships) != 0 {
		t.Errorf("ProposedPolicy.Relationships = %+v, want none", result.ProposedPolicy.Relationships)
	}
}

func TestPromote_MissingPolicyFile(t *testing.T) {
	_, graphPath := buildPromoteFixture(t)
	_, err := Promote(PromoteOptions{PolicyPath: filepath.Join(t.TempDir(), "no-such.json"), GraphPath: graphPath, AsOf: time.Now()})
	if err == nil {
		t.Fatal("expected an error for a missing policy file")
	}
}

func TestPromote_MissingGraphFile(t *testing.T) {
	policyPath, _ := buildPromoteFixture(t)
	_, err := Promote(PromoteOptions{PolicyPath: policyPath, GraphPath: filepath.Join(t.TempDir(), "no-such.json"), AsOf: time.Now()})
	if err == nil {
		t.Fatal("expected an error for a missing graph file")
	}
}
