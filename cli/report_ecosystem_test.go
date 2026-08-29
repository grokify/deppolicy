package cli

import (
	"strings"
	"testing"

	"github.com/grokify/deppolicy/graph"
)

func TestFormatEcosystemReport(t *testing.T) {
	result := EcosystemReportResult{
		TotalComponents: 6,
		TotalEdges:      4,
		CrossOrgEdges:   []graph.Dependency{{Source: "go:github.com/orgA/appA", Target: "go:github.com/orgB/tool"}},
		TopFanIn:        []ComponentFanIn{{Locator: "go:github.com/orgA/shared", Count: 3}},
		RenameCandidates: []RenameCandidate{
			{Locator: "go:github.com/orgA/leaf"},
		},
	}

	report := FormatEcosystemReport(result)

	if !strings.Contains(report, "6 component(s), 4 direct dependency edge(s)") {
		t.Errorf("report missing count summary, got:\n%s", report)
	}
	if !strings.Contains(report, "1 cross-org edge(s)") {
		t.Errorf("report missing cross-org count, got:\n%s", report)
	}
	if !strings.Contains(report, "3  go:github.com/orgA/shared") {
		t.Errorf("report missing fan-in leader, got:\n%s", report)
	}
	if !strings.Contains(report, "go:github.com/orgA/leaf") {
		t.Errorf("report missing rename candidate, got:\n%s", report)
	}
	if !strings.Contains(report, "not evidence of being unused") {
		t.Errorf("report missing safety-framing caveat, got:\n%s", report)
	}
}

func TestWriteEcosystemReport(t *testing.T) {
	path := t.TempDir() + "/report.json"
	if err := WriteEcosystemReport(EcosystemReportResult{TotalComponents: 1}, path); err != nil {
		t.Fatalf("WriteEcosystemReport() error = %v", err)
	}
}
