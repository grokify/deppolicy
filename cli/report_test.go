package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

func TestFormatCheckReport(t *testing.T) {
	result := CheckResult{
		Graph: graph.Graph{
			Dependencies: []graph.Dependency{
				{
					Source: "go:github.com/orgA/consumer", Relation: graph.RelationDependsOn, Target: "go:github.com/orgA/badtarget",
					Evidence: []graph.Evidence{{Kind: graph.EvidenceKindSourceImport, File: "main.go", Line: 3}},
				},
			},
		},
		Findings: []policy.Finding{
			{Source: "go:github.com/orgA/consumer", Target: "go:github.com/orgA/provider", Status: policy.FindingConforming, Reason: "explicit relationship"},
			{Source: "go:github.com/orgA/consumer", Target: "go:github.com/orgA/badtarget", Status: policy.FindingViolation, Reason: "no allow rule; default-deny"},
			{Source: "go:github.com/orgA/dashforge", Target: "go:github.com/orgA/unused", Status: policy.FindingUnusedAllowance, Reason: "explicit relationship"},
		},
	}

	report := FormatCheckReport(result)

	if !strings.Contains(report, "PASS  go:github.com/orgA/consumer -> go:github.com/orgA/provider") {
		t.Error("report missing PASS line for conforming finding")
	}
	if !strings.Contains(report, "FAIL  go:github.com/orgA/consumer -> go:github.com/orgA/badtarget") {
		t.Error("report missing FAIL line for violation")
	}
	if !strings.Contains(report, "main.go:3") {
		t.Error("report missing file:line evidence for the violation")
	}
	if !strings.Contains(report, "INFO  go:github.com/orgA/dashforge -> go:github.com/orgA/unused") {
		t.Error("report missing INFO line for unused allowance")
	}
	if !strings.Contains(report, "1 conforming, 1 violation(s), 1 unused allowance(s)") {
		t.Errorf("report missing correct summary line, got:\n%s", report)
	}
}

func TestFormatCheckReport_ScanWarning(t *testing.T) {
	result := CheckResult{ScanWarning: errors.New("module x: parse error")}
	report := FormatCheckReport(result)
	if !strings.Contains(report, "WARNING: module x: parse error") {
		t.Errorf("report missing scan warning, got:\n%s", report)
	}
}

func TestFormatCheckReport_NoScanWarning(t *testing.T) {
	report := FormatCheckReport(CheckResult{})
	if strings.Contains(report, "WARNING") {
		t.Error("report should not mention WARNING when ScanWarning is nil")
	}
}

func TestWriteFindings(t *testing.T) {
	findings := []policy.Finding{
		{Source: "a", Target: "b", Status: policy.FindingViolation, Reason: "no allow rule; default-deny"},
	}
	path := filepath.Join(t.TempDir(), "findings.json")

	if err := WriteFindings(findings, path); err != nil {
		t.Fatalf("WriteFindings() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), `"violation"`) {
		t.Errorf("written findings.json missing expected content: %s", data)
	}
}

func TestWriteFindings_InvalidPath(t *testing.T) {
	err := WriteFindings(nil, filepath.Join(t.TempDir(), "no-such-dir", "findings.json"))
	if err == nil {
		t.Fatal("expected an error writing to a nonexistent directory")
	}
}
