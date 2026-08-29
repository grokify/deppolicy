package policy

import (
	"testing"
	"time"
)

func statusPolicy() Policy {
	return Policy{
		SchemaVersion: "1",
		Scope: Scope{
			Include: []string{"go:github.com/plexusone/*"},
		},
		Components: map[string]ComponentDeclaration{
			"go:github.com/plexusone/mogo":     {Status: "foundation"},
			"go:github.com/plexusone/legacy":   {Status: "deprecated"},
			"go:github.com/plexusone/badactor": {Status: "foundation"}, // for invariant test
		},
	}
}

func TestEvaluateWithStatus_FoundationTargetAutoAllowed(t *testing.T) {
	p := statusPolicy()
	g := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/mogo"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/plexusone/mogo")
	if !ok {
		t.Fatal("expected a finding for the observed edge to a foundation component")
	}
	if f.Status != FindingConforming {
		t.Errorf("Status = %q, want %q (foundation targets are auto-authorized)", f.Status, FindingConforming)
	}
}

func TestEvaluateWithStatus_FoundationSourceInvariant(t *testing.T) {
	p := statusPolicy()
	// badactor is foundation but depends on a non-foundation component:
	// the invariant must make this a violation even though nothing else
	// is unusual about the edge.
	g := edgeGraph([2]string{"go:github.com/plexusone/badactor", "go:github.com/plexusone/dashforge"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/badactor", "go:github.com/plexusone/dashforge")
	if !ok {
		t.Fatal("expected a finding for the invariant-violating edge")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q", f.Status, FindingViolation)
	}
}

func TestEvaluateWithStatus_FoundationInvariant_ExplicitRelationshipDoesNotOverride(t *testing.T) {
	p := statusPolicy()
	p.Relationships = []Relationship{
		{
			Source:   "go:github.com/plexusone/badactor",
			Relation: RelationCanDependOn,
			Target:   "go:github.com/plexusone/dashforge",
		},
	}
	g := edgeGraph([2]string{"go:github.com/plexusone/badactor", "go:github.com/plexusone/dashforge"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/badactor", "go:github.com/plexusone/dashforge")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q (invariant must override an explicit relationship)", f.Status, FindingViolation)
	}
}

func TestEvaluateWithStatus_DeprecatedNewEdge_AlwaysViolation(t *testing.T) {
	p := statusPolicy()
	p.Relationships = []Relationship{
		{
			Source:   "go:github.com/plexusone/dashforge",
			Relation: RelationCanDependOn,
			Target:   "go:github.com/plexusone/legacy",
		},
	}
	g := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/legacy"})
	baseline := edgeGraph() // empty: this edge is new

	findings := p.EvaluateWithStatus(g, time.Now(), &baseline)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/plexusone/legacy")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q (new dependent on deprecated component, even though authorized)", f.Status, FindingViolation)
	}
}

func TestEvaluateWithStatus_DeprecatedBaselinedEdge_NormalRules(t *testing.T) {
	p := statusPolicy()
	p.Relationships = []Relationship{
		{
			Source:   "go:github.com/plexusone/dashforge",
			Relation: RelationCanDependOn,
			Target:   "go:github.com/plexusone/legacy",
		},
	}
	g := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/legacy"})
	baseline := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/legacy"}) // already existed

	findings := p.EvaluateWithStatus(g, time.Now(), &baseline)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/plexusone/legacy")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingConforming {
		t.Errorf("Status = %q, want %q (existing authorized dependent: permission not revoked)", f.Status, FindingConforming)
	}
}

func TestEvaluateWithStatus_DeprecatedBaselinedEdge_UnauthorizedStillViolation(t *testing.T) {
	p := statusPolicy() // no relationship authorizing dashforge -> legacy
	g := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/legacy"})
	baseline := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/legacy"})

	findings := p.EvaluateWithStatus(g, time.Now(), &baseline)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/plexusone/legacy")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q (baseline tolerance does not grant authorization it never had)", f.Status, FindingViolation)
	}
}

func TestEvaluateWithStatus_NilBaselineTreatsDeprecatedEdgeAsNew(t *testing.T) {
	p := statusPolicy()
	p.Relationships = []Relationship{
		{
			Source:   "go:github.com/plexusone/dashforge",
			Relation: RelationCanDependOn,
			Target:   "go:github.com/plexusone/legacy",
		},
	}
	g := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/legacy"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/plexusone/legacy")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q (nil baseline must treat every edge as new)", f.Status, FindingViolation)
	}
}

func TestEvaluateWithStatus_UnaffectedGovernedEdgesMatchBaseEvaluate(t *testing.T) {
	p := evalPolicy()
	g := edgeGraph(
		[2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/omniagent"},
		[2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm"},
	)
	asOf := time.Now()

	base := p.Evaluate(g, asOf)
	withStatus := p.EvaluateWithStatus(g, asOf, nil)

	if len(base) != len(withStatus) {
		t.Fatalf("finding count differs: base=%d withStatus=%d", len(base), len(withStatus))
	}
	for i := range base {
		if base[i] != withStatus[i] {
			t.Errorf("finding %d differs: base=%+v withStatus=%+v", i, base[i], withStatus[i])
		}
	}
}

func TestEvaluateWithStatus_Deterministic(t *testing.T) {
	p := statusPolicy()
	g := edgeGraph(
		[2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/mogo"},
		[2]string{"go:github.com/plexusone/badactor", "go:github.com/plexusone/dashforge"},
	)

	first := p.EvaluateWithStatus(g, time.Now(), nil)
	second := p.EvaluateWithStatus(g, time.Now(), nil)

	if len(first) != len(second) {
		t.Fatalf("non-deterministic finding count")
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("non-deterministic order at %d: %+v vs %+v", i, first[i], second[i])
		}
	}
}
