package cli

import (
	"strings"
	"testing"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

func TestFormatPromoteReport_WithProposals(t *testing.T) {
	result := PromoteResult{
		Proposed: []ProposedRelationship{
			{
				Relationship: policy.Relationship{Source: "a", Target: "b", Relation: policy.RelationCanDependOn, Reason: promotedReasonPlaceholder},
				Evidence:     []graph.Evidence{{Kind: graph.EvidenceKindSourceImport, File: "main.go", Line: 7}},
			},
		},
	}

	report := FormatPromoteReport(result)

	if !strings.Contains(report, "PROPOSE  a -> b") {
		t.Errorf("report missing PROPOSE line, got:\n%s", report)
	}
	if !strings.Contains(report, "main.go:7") {
		t.Errorf("report missing evidence line, got:\n%s", report)
	}
	if !strings.Contains(report, "1 relationship(s) proposed") {
		t.Errorf("report missing summary, got:\n%s", report)
	}
}

func TestFormatPromoteReport_NoProposals(t *testing.T) {
	report := FormatPromoteReport(PromoteResult{})
	if !strings.Contains(report, "no unauthorized edges found") {
		t.Errorf("report missing clean-run message, got:\n%s", report)
	}
}

func TestWriteProposedPolicy(t *testing.T) {
	p := policy.Policy{
		SchemaVersion: "1",
		Scope:         policy.Scope{Include: []string{"go:github.com/orgA/*"}},
	}
	path := t.TempDir() + "/proposed.json"

	if err := WriteProposedPolicy(p, path); err != nil {
		t.Fatalf("WriteProposedPolicy() error = %v", err)
	}
}
