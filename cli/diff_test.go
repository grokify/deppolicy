package cli

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/grokify/deppolicy/policy"
)

const diffTestPolicy = `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"] },
  "relationships": [
    {
      "source": "go:github.com/orgA/consumer",
      "relation": "can_depend_on",
      "target": "go:github.com/orgA/authorized-target",
      "reason": "test fixture"
    }
  ]
}`

const diffTestBaselineGraph = `{
  "generated": "2026-01-01T00:00:00Z",
  "scan": { "level": "source-imports", "scanner": "test" },
  "dependencies": [
    {
      "source": "go:github.com/orgA/consumer",
      "relation": "depends_on",
      "target": "go:github.com/orgA/authorized-target"
    },
    {
      "source": "go:github.com/orgA/consumer",
      "relation": "depends_on",
      "target": "go:github.com/orgA/existing-violation-target"
    }
  ]
}`

func buildDiffFixture(t *testing.T) (currentDir, policyPath, baselinePath string) {
	t.Helper()
	root := t.TempDir()

	writeCliFile(t, filepath.Join(root, "consumer/go.mod"), "module github.com/orgA/consumer\n\ngo 1.26\n")
	writeCliFile(t, filepath.Join(root, "consumer/main.go"), `package main

import (
	"github.com/orgA/authorized-target"
	"github.com/orgA/existing-violation-target"
	"github.com/orgA/new-violation-target"
)

func main() {
	_ = authorizedtarget.X
	_ = existingviolationtarget.X
	_ = newviolationtarget.X
}
`)

	for _, name := range []string{"authorized-target", "existing-violation-target", "new-violation-target"} {
		writeCliFile(t, filepath.Join(root, name, "go.mod"), "module github.com/orgA/"+name+"\n\ngo 1.26\n")
		writeCliFile(t, filepath.Join(root, name, "main.go"), "package main\n\nfunc main() {}\n")
	}

	policyPath = filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, diffTestPolicy)

	baselinePath = filepath.Join(root, "baseline-graph.json")
	writeCliFile(t, baselinePath, diffTestBaselineGraph)

	return root, policyPath, baselinePath
}

func ratchetFindingFor(findings []RatchetFinding, source, target string) (RatchetFinding, bool) {
	for _, f := range findings {
		if f.Source == source && f.Target == target {
			return f, true
		}
	}
	return RatchetFinding{}, false
}

func TestDiff_ClassifiesNewVsExistingViolations(t *testing.T) {
	currentDir, policyPath, baselinePath := buildDiffFixture(t)

	result, err := Diff(context.Background(), DiffOptions{
		PolicyPath:        policyPath,
		BaselineGraphPath: baselinePath,
		CurrentDirs:       []string{currentDir},
		Scanner:           "test",
		AsOf:              time.Now(),
	})
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	existing, ok := ratchetFindingFor(result.Findings, "go:github.com/orgA/consumer", "go:github.com/orgA/existing-violation-target")
	if !ok {
		t.Fatal("expected a finding for the existing violation")
	}
	if existing.Ratchet != RatchetExistingViolation {
		t.Errorf("existing violation Ratchet = %q, want %q", existing.Ratchet, RatchetExistingViolation)
	}

	fresh, ok := ratchetFindingFor(result.Findings, "go:github.com/orgA/consumer", "go:github.com/orgA/new-violation-target")
	if !ok {
		t.Fatal("expected a finding for the new violation")
	}
	if fresh.Ratchet != RatchetNewViolation {
		t.Errorf("new violation Ratchet = %q, want %q", fresh.Ratchet, RatchetNewViolation)
	}

	conforming, ok := ratchetFindingFor(result.Findings, "go:github.com/orgA/consumer", "go:github.com/orgA/authorized-target")
	if !ok {
		t.Fatal("expected a finding for the conforming edge")
	}
	if conforming.Ratchet != RatchetConforming {
		t.Errorf("conforming Ratchet = %q, want %q", conforming.Ratchet, RatchetConforming)
	}

	if !result.NewViolations() {
		t.Error("NewViolations() = false, want true (new-violation-target must block ratchet CI)")
	}
}

func TestDiff_GraphDiffCounts(t *testing.T) {
	currentDir, policyPath, baselinePath := buildDiffFixture(t)

	result, err := Diff(context.Background(), DiffOptions{
		PolicyPath:        policyPath,
		BaselineGraphPath: baselinePath,
		CurrentDirs:       []string{currentDir},
		Scanner:           "test",
		AsOf:              time.Now(),
	})
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	if len(result.GraphDiff.Added) != 1 {
		t.Errorf("GraphDiff.Added = %+v, want 1 edge (new-violation-target)", result.GraphDiff.Added)
	}
	if len(result.GraphDiff.Unchanged) != 2 {
		t.Errorf("GraphDiff.Unchanged = %+v, want 2 edges", result.GraphDiff.Unchanged)
	}
	if len(result.GraphDiff.Removed) != 0 {
		t.Errorf("GraphDiff.Removed = %+v, want none", result.GraphDiff.Removed)
	}
}

func TestDiffResult_NewViolations_FalseWhenOnlyExistingDebt(t *testing.T) {
	result := DiffResult{Findings: []RatchetFinding{
		{Finding: policy.Finding{Status: policy.FindingViolation}, Ratchet: RatchetExistingViolation},
		{Finding: policy.Finding{Status: policy.FindingConforming}, Ratchet: RatchetConforming},
	}}
	if result.NewViolations() {
		t.Error("NewViolations() = true for a result with only existing debt, want false")
	}
}

func TestDiff_MissingBaselineFile(t *testing.T) {
	currentDir, policyPath, _ := buildDiffFixture(t)

	_, err := Diff(context.Background(), DiffOptions{
		PolicyPath:        policyPath,
		BaselineGraphPath: filepath.Join(currentDir, "no-such-baseline.json"),
		CurrentDirs:       []string{currentDir},
		Scanner:           "test",
		AsOf:              time.Now(),
	})
	if err == nil {
		t.Fatal("expected an error for a missing baseline graph file")
	}
}
