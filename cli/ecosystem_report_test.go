package cli

import (
	"path/filepath"
	"testing"
)

const ecosystemTestPolicy = `{
  "schemaVersion": "1",
  "scope": {
    "include": ["go:github.com/orgA/*", "go:github.com/orgB/*"],
    "exclude": ["go:github.com/orgA/sandbox-*"]
  }
}`

func buildEcosystemFixture(t *testing.T) (policyPath, graphPath string) {
	t.Helper()
	root := t.TempDir()

	policyPath = filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, ecosystemTestPolicy)

	// Shape:
	//   orgA/appA  -> orgA/shared (in-org)
	//   orgA/appB  -> orgA/shared (in-org, duplicate-ish target)
	//   orgB/appC  -> orgA/shared (cross-org)
	//   orgA/appA  -> orgB/tool   (cross-org)
	//   orgA/leaf has zero fan-in and zero fan-out (rename candidate)
	//   orgA/shared has fan-in 3 (not a rename candidate; foundation-tier signal)
	graphPath = filepath.Join(root, "graph.json")
	graphJSON := `{
  "generated": "2026-01-01T00:00:00Z",
  "scan": { "level": "source-imports", "scanner": "test" },
  "components": [
    { "id": "go:github.com/orgA/appA", "kind": "go-module" },
    { "id": "go:github.com/orgA/appB", "kind": "go-module" },
    { "id": "go:github.com/orgB/appC", "kind": "go-module" },
    { "id": "go:github.com/orgA/shared", "kind": "go-module" },
    { "id": "go:github.com/orgB/tool", "kind": "go-module" },
    { "id": "go:github.com/orgA/leaf", "kind": "go-module" }
  ],
  "dependencies": [
    { "source": "go:github.com/orgA/appA", "relation": "depends_on", "target": "go:github.com/orgA/shared" },
    { "source": "go:github.com/orgA/appB", "relation": "depends_on", "target": "go:github.com/orgA/shared" },
    { "source": "go:github.com/orgB/appC", "relation": "depends_on", "target": "go:github.com/orgA/shared" },
    { "source": "go:github.com/orgA/appA", "relation": "depends_on", "target": "go:github.com/orgB/tool" }
  ]
}`
	writeCliFile(t, graphPath, graphJSON)

	return policyPath, graphPath
}

func TestEcosystemReport_Counts(t *testing.T) {
	policyPath, graphPath := buildEcosystemFixture(t)

	result, err := EcosystemReport(EcosystemReportOptions{PolicyPath: policyPath, GraphPath: graphPath})
	if err != nil {
		t.Fatalf("EcosystemReport() error = %v", err)
	}
	if result.TotalComponents != 6 {
		t.Errorf("TotalComponents = %d, want 6", result.TotalComponents)
	}
	if result.TotalEdges != 4 {
		t.Errorf("TotalEdges = %d, want 4", result.TotalEdges)
	}
}

func TestEcosystemReport_TopFanIn(t *testing.T) {
	policyPath, graphPath := buildEcosystemFixture(t)

	result, err := EcosystemReport(EcosystemReportOptions{PolicyPath: policyPath, GraphPath: graphPath})
	if err != nil {
		t.Fatalf("EcosystemReport() error = %v", err)
	}
	if len(result.TopFanIn) == 0 || result.TopFanIn[0].Locator != "go:github.com/orgA/shared" || result.TopFanIn[0].Count != 3 {
		t.Errorf("TopFanIn = %+v, want shared leading with count 3", result.TopFanIn)
	}
}

func TestEcosystemReport_TopFanIn_RespectsLimit(t *testing.T) {
	policyPath, graphPath := buildEcosystemFixture(t)

	result, err := EcosystemReport(EcosystemReportOptions{PolicyPath: policyPath, GraphPath: graphPath, TopN: 1})
	if err != nil {
		t.Fatalf("EcosystemReport() error = %v", err)
	}
	if len(result.TopFanIn) != 1 {
		t.Errorf("TopFanIn = %+v, want exactly 1 entry with TopN=1", result.TopFanIn)
	}
}

func TestEcosystemReport_CrossOrgEdges(t *testing.T) {
	policyPath, graphPath := buildEcosystemFixture(t)

	result, err := EcosystemReport(EcosystemReportOptions{PolicyPath: policyPath, GraphPath: graphPath})
	if err != nil {
		t.Fatalf("EcosystemReport() error = %v", err)
	}
	if len(result.CrossOrgEdges) != 2 {
		t.Fatalf("CrossOrgEdges = %+v, want 2 (orgB/appC->orgA/shared, orgA/appA->orgB/tool)", result.CrossOrgEdges)
	}
	for _, d := range result.CrossOrgEdges {
		if orgOf(d.Source) == orgOf(d.Target) {
			t.Errorf("edge %+v is not actually cross-org", d)
		}
	}
}

func TestEcosystemReport_RenameCandidates(t *testing.T) {
	policyPath, graphPath := buildEcosystemFixture(t)

	result, err := EcosystemReport(EcosystemReportOptions{PolicyPath: policyPath, GraphPath: graphPath})
	if err != nil {
		t.Fatalf("EcosystemReport() error = %v", err)
	}

	found := false
	for _, c := range result.RenameCandidates {
		if c.Locator == "go:github.com/orgA/leaf" {
			found = true
		}
		if c.Locator == "go:github.com/orgA/shared" {
			t.Error("shared has 3 dependents and must not be a rename candidate")
		}
	}
	if !found {
		t.Errorf("expected leaf (zero fan-in) to be a rename candidate, got %+v", result.RenameCandidates)
	}
}

func TestEcosystemReport_DeduplicatesRepeatedEdges(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, `{"schemaVersion":"1","scope":{"include":["go:github.com/orgA/*"]}}`)

	graphPath := filepath.Join(root, "graph.json")
	// The same edge appears twice -- a hand-edited or merged graph should
	// not be double-counted.
	graphJSON := `{
  "scan": { "level": "source-imports", "scanner": "test" },
  "dependencies": [
    { "source": "go:github.com/orgA/a", "relation": "depends_on", "target": "go:github.com/orgA/b" },
    { "source": "go:github.com/orgA/a", "relation": "depends_on", "target": "go:github.com/orgA/b" }
  ]
}`
	writeCliFile(t, graphPath, graphJSON)

	result, err := EcosystemReport(EcosystemReportOptions{PolicyPath: policyPath, GraphPath: graphPath})
	if err != nil {
		t.Fatalf("EcosystemReport() error = %v", err)
	}
	if len(result.TopFanIn) != 1 || result.TopFanIn[0].Count != 1 {
		t.Errorf("TopFanIn = %+v, want exactly 1 entry with count 1 (deduplicated)", result.TopFanIn)
	}
}

func TestEcosystemReport_MissingGraphFile(t *testing.T) {
	policyPath, _ := buildEcosystemFixture(t)
	_, err := EcosystemReport(EcosystemReportOptions{PolicyPath: policyPath, GraphPath: filepath.Join(t.TempDir(), "no-such.json")})
	if err == nil {
		t.Fatal("expected an error for a missing graph file")
	}
}

func TestEcosystemReport_MissingPolicyFile(t *testing.T) {
	_, graphPath := buildEcosystemFixture(t)
	_, err := EcosystemReport(EcosystemReportOptions{PolicyPath: filepath.Join(t.TempDir(), "no-such.json"), GraphPath: graphPath})
	if err == nil {
		t.Fatal("expected an error for a missing policy file")
	}
}

func TestOrgOf(t *testing.T) {
	tests := map[string]string{
		"go:github.com/orgA/repo":         "orgA",
		"go:github.com/orgA/repo/sub/pkg": "orgA",
		"go:github.com/orgA":              "orgA",
		"go:golang.org/x/mod":             "",
		"go:gopkg.in/yaml.v3":             "",
	}
	for locator, want := range tests {
		if got := orgOf(locator); got != want {
			t.Errorf("orgOf(%q) = %q, want %q", locator, got, want)
		}
	}
}
