package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

const policyDiffOld = `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"] },
  "components": {
    "go:github.com/orgA/mogo": { "status": "foundation" }
  },
  "relationships": [
    { "source": "go:github.com/orgA/a", "relation": "can_depend_on", "target": "go:github.com/orgA/kept", "reason": "kept" },
    { "source": "go:github.com/orgA/a", "relation": "can_depend_on", "target": "go:github.com/orgA/removed-live", "reason": "will be removed while still used" },
    { "source": "go:github.com/orgA/a", "relation": "can_depend_on", "target": "go:github.com/orgA/removed-dead", "reason": "will be removed after code cleanup" }
  ]
}`

const policyDiffNew = `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"] },
  "components": {
    "go:github.com/orgA/mogo": { "status": "foundation" },
    "go:github.com/orgA/legacy": { "status": "deprecated" }
  },
  "relationships": [
    { "source": "go:github.com/orgA/a", "relation": "can_depend_on", "target": "go:github.com/orgA/kept", "reason": "kept" },
    { "source": "go:github.com/orgA/b", "relation": "can_depend_on", "target": "go:github.com/orgA/kept", "reason": "newly permitted" }
  ]
}`

const policyDiffGraph = `{
  "scan": { "level": "source-imports", "scanner": "test" },
  "dependencies": [
    { "source": "go:github.com/orgA/a", "relation": "depends_on", "target": "go:github.com/orgA/removed-live" }
  ]
}`

func buildPolicyDiffFixture(t *testing.T) (oldPath, newPath, graphPath string) {
	t.Helper()
	root := t.TempDir()
	oldPath = filepath.Join(root, "old.json")
	newPath = filepath.Join(root, "new.json")
	graphPath = filepath.Join(root, "graph.json")
	writeCliFile(t, oldPath, policyDiffOld)
	writeCliFile(t, newPath, policyDiffNew)
	writeCliFile(t, graphPath, policyDiffGraph)
	return oldPath, newPath, graphPath
}

func TestPolicyDiff_AddedRemovedAndBreaking(t *testing.T) {
	oldPath, newPath, graphPath := buildPolicyDiffFixture(t)

	result, err := PolicyDiff(PolicyDiffOptions{OldPolicyPath: oldPath, NewPolicyPath: newPath, GraphPath: graphPath})
	if err != nil {
		t.Fatalf("PolicyDiff() error = %v", err)
	}

	if len(result.Added) != 1 || result.Added[0].Source != "go:github.com/orgA/b" {
		t.Errorf("Added = %+v, want the b -> kept relationship", result.Added)
	}

	if len(result.Removed) != 2 {
		t.Fatalf("Removed = %+v, want 2", result.Removed)
	}
	byTarget := map[string]RemovedRelationship{}
	for _, r := range result.Removed {
		byTarget[r.Target] = r
	}
	if !byTarget["go:github.com/orgA/removed-live"].StillObserved {
		t.Error("removed-live must be flagged StillObserved (its edge is in the graph)")
	}
	if byTarget["go:github.com/orgA/removed-dead"].StillObserved {
		t.Error("removed-dead must not be flagged StillObserved")
	}
	if !result.BreakingRemovals() {
		t.Error("BreakingRemovals() = false, want true")
	}

	if len(result.ComponentChanges) != 1 || result.ComponentChanges[0].Locator != "go:github.com/orgA/legacy" || result.ComponentChanges[0].NewStatus != "deprecated" {
		t.Errorf("ComponentChanges = %+v, want the legacy deprecation", result.ComponentChanges)
	}
}

func TestPolicyDiff_NoGraph_NothingBreaking(t *testing.T) {
	oldPath, newPath, _ := buildPolicyDiffFixture(t)

	result, err := PolicyDiff(PolicyDiffOptions{OldPolicyPath: oldPath, NewPolicyPath: newPath})
	if err != nil {
		t.Fatalf("PolicyDiff() error = %v", err)
	}
	if result.BreakingRemovals() {
		t.Error("BreakingRemovals() = true without a graph, want false")
	}
}

func TestPolicyDiff_IdenticalPolicies(t *testing.T) {
	oldPath, _, _ := buildPolicyDiffFixture(t)

	result, err := PolicyDiff(PolicyDiffOptions{OldPolicyPath: oldPath, NewPolicyPath: oldPath})
	if err != nil {
		t.Fatalf("PolicyDiff() error = %v", err)
	}
	if len(result.Added) != 0 || len(result.Removed) != 0 || len(result.ComponentChanges) != 0 {
		t.Errorf("diff of a policy against itself = %+v, want empty", result)
	}
}

func TestPolicyDiff_MissingFile(t *testing.T) {
	oldPath, _, _ := buildPolicyDiffFixture(t)
	if _, err := PolicyDiff(PolicyDiffOptions{OldPolicyPath: oldPath, NewPolicyPath: filepath.Join(t.TempDir(), "nope.json")}); err == nil {
		t.Fatal("expected an error for a missing policy file")
	}
}

func TestFormatPolicyDiffReport(t *testing.T) {
	oldPath, newPath, graphPath := buildPolicyDiffFixture(t)
	result, err := PolicyDiff(PolicyDiffOptions{OldPolicyPath: oldPath, NewPolicyPath: newPath, GraphPath: graphPath})
	if err != nil {
		t.Fatalf("PolicyDiff() error = %v", err)
	}

	report := FormatPolicyDiffReport(result)

	if !strings.Contains(report, "ADDED    go:github.com/orgA/b -> go:github.com/orgA/kept") {
		t.Errorf("report missing ADDED line, got:\n%s", report)
	}
	if !strings.Contains(report, "REMOVED  go:github.com/orgA/a -> go:github.com/orgA/removed-live (BREAKING") {
		t.Errorf("report missing BREAKING removal, got:\n%s", report)
	}
	if !strings.Contains(report, "CHANGED  go:github.com/orgA/legacy  status: (none) -> deprecated") {
		t.Errorf("report missing component change, got:\n%s", report)
	}
	if !strings.Contains(report, "1 added, 2 removed (1 breaking), 1 component change(s)") {
		t.Errorf("report missing summary, got:\n%s", report)
	}
}

func TestFormatPolicyDiffReport_NoChanges(t *testing.T) {
	report := FormatPolicyDiffReport(PolicyDiffResult{})
	if !strings.Contains(report, "no policy changes") {
		t.Errorf("report missing no-changes message, got:\n%s", report)
	}
}
