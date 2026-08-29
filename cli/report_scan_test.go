package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/scanner/golang"
)

func TestFormatScanReport(t *testing.T) {
	result := ScanResult{
		Graph: graph.Graph{
			Components:   []graph.ComponentObservation{{ID: "a"}, {ID: "b"}},
			Dependencies: []graph.Dependency{{Source: "a", Target: "b"}},
		},
		Reports: []golang.DiscoveryReport{{
			Excluded: []golang.ExcludedModule{{Dir: "/x/sandbox-c", ModuleLocator: "c", Reason: "out of policy scope"}},
			Errored:  []golang.DiscoveryError{{Dir: "/x/broken", Err: errors.New("bad go.mod")}},
		}},
		ScanWarning: errors.New("module d: parse error"),
	}

	report := FormatScanReport(result)

	if !strings.Contains(report, "2 component(s) discovered, 1 direct dependency edge(s)") {
		t.Errorf("report missing count summary, got:\n%s", report)
	}
	if !strings.Contains(report, "c (/x/sandbox-c): out of policy scope") {
		t.Errorf("report missing excluded entry, got:\n%s", report)
	}
	if !strings.Contains(report, "/x/broken: bad go.mod") {
		t.Errorf("report missing errored entry, got:\n%s", report)
	}
	if !strings.Contains(report, "WARNING: module d: parse error") {
		t.Errorf("report missing scan warning, got:\n%s", report)
	}
}

func TestFormatScanReport_Clean(t *testing.T) {
	report := FormatScanReport(ScanResult{})
	if strings.Contains(report, "excluded") || strings.Contains(report, "failed to parse") || strings.Contains(report, "WARNING") {
		t.Errorf("report should omit empty sections, got:\n%s", report)
	}
}
