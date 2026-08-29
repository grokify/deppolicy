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

func TestFormatDiffReport(t *testing.T) {
	result := DiffResult{
		CurrentGraph: graph.Graph{
			Dependencies: []graph.Dependency{
				{
					Source: "a", Relation: graph.RelationDependsOn, Target: "new-target",
					Evidence: []graph.Evidence{{Kind: graph.EvidenceKindSourceImport, File: "main.go", Line: 4}},
				},
			},
		},
		GraphDiff: graph.Diff{
			Added:     []graph.Dependency{{Source: "a", Target: "new-target"}},
			Removed:   []graph.Dependency{{Source: "a", Target: "old-target"}},
			Unchanged: []graph.Dependency{{Source: "a", Target: "stable-target"}},
		},
		Findings: []RatchetFinding{
			{Finding: policy.Finding{Source: "a", Target: "new-target", Status: policy.FindingViolation, Reason: "no allow rule; default-deny"}, Ratchet: RatchetNewViolation},
			{Finding: policy.Finding{Source: "a", Target: "old-debt", Status: policy.FindingViolation, Reason: "no allow rule; default-deny"}, Ratchet: RatchetExistingViolation},
			{Finding: policy.Finding{Source: "a", Target: "stable-target", Status: policy.FindingConforming, Reason: "explicit relationship"}, Ratchet: RatchetConforming},
		},
	}

	report := FormatDiffReport(result)

	if !strings.Contains(report, "FAIL  a -> new-target (new)") {
		t.Error("report missing FAIL line for new violation")
	}
	if !strings.Contains(report, "main.go:4") {
		t.Error("report missing evidence for the new violation")
	}
	if !strings.Contains(report, "WARN  a -> old-debt (existing debt)") {
		t.Error("report missing WARN line for existing violation")
	}
	if !strings.Contains(report, "PASS  a -> stable-target") {
		t.Error("report missing PASS line for conforming edge")
	}
	if !strings.Contains(report, "1 conforming, 1 new violation(s), 1 existing violation(s), 0 unused allowance(s)") {
		t.Errorf("report missing correct summary line, got:\n%s", report)
	}
	if !strings.Contains(report, "graph: 1 edge(s) added, 1 removed, 1 unchanged") {
		t.Errorf("report missing graph diff summary, got:\n%s", report)
	}
}

func TestFormatDiffReport_ScanWarning(t *testing.T) {
	report := FormatDiffReport(DiffResult{ScanWarning: errors.New("module x: parse error")})
	if !strings.Contains(report, "WARNING: module x: parse error") {
		t.Errorf("report missing scan warning, got:\n%s", report)
	}
}

func TestWriteRatchetFindings(t *testing.T) {
	findings := []RatchetFinding{
		{Finding: policy.Finding{Source: "a", Target: "b", Status: policy.FindingViolation}, Ratchet: RatchetNewViolation},
	}
	path := filepath.Join(t.TempDir(), "findings.json")

	if err := WriteRatchetFindings(findings, path); err != nil {
		t.Fatalf("WriteRatchetFindings() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), `"new_violation"`) {
		t.Errorf("written file missing expected ratchet status: %s", data)
	}
}
