package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/scanner/golang/githubfleet"
)

func TestFormatScanGitHubReport(t *testing.T) {
	result := ScanGitHubResult{
		Graph: graph.Graph{
			Components:   []graph.ComponentObservation{{ID: "a"}},
			Dependencies: []graph.Dependency{{Source: "a", Target: "b"}},
		},
		Reports: []githubfleet.FleetReport{{
			Excluded: []githubfleet.ExcludedRepo{{Owner: "orgA", Repo: "old-thing", Reason: "archived"}},
			Errored:  []githubfleet.RepoManifestError{{Owner: "orgA", Repo: "flaky", Err: errors.New("network error")}},
		}},
		ScanWarning: errors.New("1 repo(s) failed"),
	}

	report := FormatScanGitHubReport(result)

	if !strings.Contains(report, "1 component(s) discovered, 1 direct dependency edge(s)") {
		t.Errorf("report missing count summary, got:\n%s", report)
	}
	if !strings.Contains(report, "orgA/old-thing: archived") {
		t.Errorf("report missing excluded entry, got:\n%s", report)
	}
	if !strings.Contains(report, "orgA/flaky: network error") {
		t.Errorf("report missing errored entry, got:\n%s", report)
	}
	if !strings.Contains(report, "WARNING: 1 repo(s) failed") {
		t.Errorf("report missing scan warning, got:\n%s", report)
	}
}

func TestFormatScanGitHubReport_Clean(t *testing.T) {
	report := FormatScanGitHubReport(ScanGitHubResult{})
	if strings.Contains(report, "excluded") || strings.Contains(report, "failed") || strings.Contains(report, "WARNING") {
		t.Errorf("report should omit empty sections, got:\n%s", report)
	}
}
