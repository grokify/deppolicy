package policy

import (
	"testing"
	"time"

	"github.com/grokify/deppolicy/graph"
)

func evalPolicy() Policy {
	return Policy{
		SchemaVersion: "1",
		Scope: Scope{
			Include: []string{"go:github.com/plexusone/*"},
		},
		Relationships: []Relationship{
			{
				Source:   "go:github.com/plexusone/dashforge",
				Relation: RelationCanDependOn,
				Target:   "go:github.com/plexusone/omniagent",
			},
			{
				Source:   "go:github.com/plexusone/dashforge",
				Relation: RelationCanDependOn,
				Target:   "go:github.com/plexusone/unused",
			},
		},
		Exceptions: []Exception{
			{
				Source:  "go:github.com/plexusone/legacy",
				Target:  "go:github.com/plexusone/omnillm",
				Effect:  EffectAllow,
				Expires: "2099-12-31",
				Reason:  "migration in progress",
			},
			{
				Source:  "go:github.com/plexusone/oldthing",
				Target:  "go:github.com/plexusone/omnillm",
				Effect:  EffectAllow,
				Expires: "2020-01-01", // already expired relative to asOf below
				Reason:  "stale exception",
			},
		},
	}
}

func edgeGraph(edges ...[2]string) graph.Graph {
	g := graph.Graph{
		Scan: graph.ScanMetadata{Level: graph.ScanLevelSourceImports, Scanner: "test"},
	}
	for _, e := range edges {
		g.Dependencies = append(g.Dependencies, graph.Dependency{
			Source: e[0], Relation: graph.RelationDependsOn, Target: e[1],
		})
	}
	return g
}

func findingFor(t *testing.T, findings []Finding, source, target string) (Finding, bool) {
	t.Helper()
	for _, f := range findings {
		if f.Source == source && f.Target == target {
			return f, true
		}
	}
	return Finding{}, false
}

func TestEvaluate_Conforming(t *testing.T) {
	p := evalPolicy()
	g := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/omniagent"})

	findings := p.Evaluate(g, time.Now())

	f, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/plexusone/omniagent")
	if !ok {
		t.Fatal("expected a finding for the authorized, observed edge")
	}
	if f.Status != FindingConforming {
		t.Errorf("Status = %q, want %q", f.Status, FindingConforming)
	}
}

func TestEvaluate_Violation_DefaultDeny(t *testing.T) {
	p := evalPolicy()
	g := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm"})

	findings := p.Evaluate(g, time.Now())

	f, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm")
	if !ok {
		t.Fatal("expected a finding for the unauthorized, observed edge")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q", f.Status, FindingViolation)
	}
}

func TestEvaluate_UnusedAllowance(t *testing.T) {
	p := evalPolicy()
	g := edgeGraph() // no edges observed at all

	findings := p.Evaluate(g, time.Now())

	f, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/plexusone/unused")
	if !ok {
		t.Fatal("expected a finding for the authorized-but-unobserved edge")
	}
	if f.Status != FindingUnusedAllowance {
		t.Errorf("Status = %q, want %q", f.Status, FindingUnusedAllowance)
	}
}

func TestEvaluate_NeitherObservedNorAuthorized_NoFinding(t *testing.T) {
	p := evalPolicy()
	g := edgeGraph()

	findings := p.Evaluate(g, time.Now())

	if _, ok := findingFor(t, findings, "go:github.com/plexusone/x", "go:github.com/plexusone/y"); ok {
		t.Error("expected no finding for an edge that is neither observed nor authorized")
	}
}

func TestEvaluate_ExternalEdgesProduceNoFinding(t *testing.T) {
	p := evalPolicy()
	g := edgeGraph(
		[2]string{"go:github.com/plexusone/dashforge", "go:github.com/google/uuid"},
		[2]string{"go:github.com/other/foo", "go:github.com/plexusone/omniagent"},
	)

	findings := p.Evaluate(g, time.Now())

	if _, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/google/uuid"); ok {
		t.Error("expected no finding for an edge with an out-of-scope target")
	}
	if _, ok := findingFor(t, findings, "go:github.com/other/foo", "go:github.com/plexusone/omniagent"); ok {
		t.Error("expected no finding for an edge with an out-of-scope source")
	}
}

func TestEvaluate_UnexpiredException_Conforming(t *testing.T) {
	p := evalPolicy()
	asOf := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	g := edgeGraph([2]string{"go:github.com/plexusone/legacy", "go:github.com/plexusone/omnillm"})

	findings := p.Evaluate(g, asOf)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/legacy", "go:github.com/plexusone/omnillm")
	if !ok {
		t.Fatal("expected a finding for the exception-authorized edge")
	}
	if f.Status != FindingConforming {
		t.Errorf("Status = %q, want %q", f.Status, FindingConforming)
	}
}

func TestEvaluate_ExpiredException_Violation(t *testing.T) {
	p := evalPolicy()
	asOf := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	g := edgeGraph([2]string{"go:github.com/plexusone/oldthing", "go:github.com/plexusone/omnillm"})

	findings := p.Evaluate(g, asOf)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/oldthing", "go:github.com/plexusone/omnillm")
	if !ok {
		t.Fatal("expected a finding for the edge with an expired exception")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q (expired exceptions must not authorize)", f.Status, FindingViolation)
	}
}

func TestEvaluate_DeterministicOrder(t *testing.T) {
	p := evalPolicy()
	g := edgeGraph(
		[2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm"},
		[2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/omniagent"},
	)

	first := p.Evaluate(g, time.Now())
	second := p.Evaluate(g, time.Now())

	if len(first) != len(second) {
		t.Fatalf("non-deterministic finding count: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("non-deterministic order at index %d: %+v vs %+v", i, first[i], second[i])
		}
	}

	for i := 1; i < len(first); i++ {
		if first[i-1].Source > first[i].Source {
			t.Errorf("findings not sorted by source: %+v", first)
		}
	}
}

func TestEvaluate_Pure_DoesNotMutateInputs(t *testing.T) {
	p := evalPolicy()
	g := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/omniagent"})

	relCountBefore := len(p.Relationships)
	depCountBefore := len(g.Dependencies)

	_ = p.Evaluate(g, time.Now())

	if len(p.Relationships) != relCountBefore {
		t.Error("Evaluate mutated Policy.Relationships")
	}
	if len(g.Dependencies) != depCountBefore {
		t.Error("Evaluate mutated Graph.Dependencies")
	}
}
