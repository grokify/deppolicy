package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

func exportFixture() (graph.Graph, *policy.Policy) {
	g := graph.Graph{
		Scan: graph.ScanMetadata{Level: graph.ScanLevelSourceImports, Scanner: "test"},
		Components: []graph.ComponentObservation{
			{ID: "go:github.com/orgA/app"},
			{ID: "go:github.com/orgA/allowed"},
			{ID: "go:github.com/orgB/lib"},
		},
		Dependencies: []graph.Dependency{
			{Source: "go:github.com/orgA/app", Relation: graph.RelationDependsOn, Target: "go:github.com/orgA/allowed"},
			{Source: "go:github.com/orgA/app", Relation: graph.RelationDependsOn, Target: "go:github.com/orgB/lib"},
			{Source: "go:github.com/orgA/app", Relation: graph.RelationDependsOn, Target: "go:github.com/google/uuid"},
		},
	}
	p := &policy.Policy{
		SchemaVersion: "1",
		Scope:         policy.Scope{Include: []string{"go:github.com/orgA/*", "go:github.com/orgB/*"}},
		Relationships: []policy.Relationship{
			{Source: "go:github.com/orgA/app", Relation: policy.RelationCanDependOn, Target: "go:github.com/orgA/allowed", Reason: "test fixture"},
		},
	}
	return g, p
}

func TestExport_Mermaid_WithPolicy(t *testing.T) {
	g, p := exportFixture()

	out, err := Export(g, p, FormatMermaid, time.Now())
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !strings.HasPrefix(out, "graph LR\n") {
		t.Errorf("mermaid output missing header:\n%s", out)
	}
	for _, want := range []string{"subgraph orgA", "subgraph orgB", `["orgA/app"]`, "-->"} {
		if !strings.Contains(out, want) {
			t.Errorf("mermaid output missing %q:\n%s", want, out)
		}
	}
	// orgA/app -> orgB/lib has no relationship: it must be styled as a
	// violation; the third-party uuid edge must be filtered out entirely.
	if !strings.Contains(out, "linkStyle") {
		t.Errorf("mermaid output missing violation linkStyle:\n%s", out)
	}
	if strings.Contains(out, "uuid") {
		t.Errorf("out-of-scope target must be filtered when a policy is given:\n%s", out)
	}
}

func TestExport_Mermaid_WithoutPolicy_IncludesEverythingUnstyled(t *testing.T) {
	g, _ := exportFixture()

	out, err := Export(g, nil, FormatMermaid, time.Now())
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if !strings.Contains(out, "uuid") {
		t.Errorf("without a policy every edge should be rendered:\n%s", out)
	}
	if strings.Contains(out, "linkStyle") {
		t.Errorf("without a policy no violation styling should appear:\n%s", out)
	}
}

func TestExport_DOT(t *testing.T) {
	g, p := exportFixture()

	out, err := Export(g, p, FormatDOT, time.Now())
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	for _, want := range []string{
		"digraph deppolicy {",
		`subgraph "cluster_orgA"`,
		`subgraph "cluster_orgB"`,
		`"go:github.com/orgA/app" -> "go:github.com/orgB/lib" [color="#cc0000", penwidth=2];`,
		`"go:github.com/orgA/app" -> "go:github.com/orgA/allowed";`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dot output missing %q:\n%s", want, out)
		}
	}
}

func TestExport_UnknownFormat(t *testing.T) {
	g, _ := exportFixture()
	if _, err := Export(g, nil, "svg", time.Now()); err == nil {
		t.Fatal("expected an error for an unknown format")
	}
}

func TestNodeID_Sanitizes(t *testing.T) {
	got := nodeID("go:github.com/orgA/app-x")
	if strings.ContainsAny(got, ":/.-") {
		t.Errorf("nodeID() = %q contains unsafe characters", got)
	}
}
