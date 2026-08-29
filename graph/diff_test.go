package graph

import "testing"

func dep(source, target string) Dependency {
	return Dependency{Source: source, Relation: RelationDependsOn, Target: target}
}

func TestDiffGraphs(t *testing.T) {
	baseline := Graph{Dependencies: []Dependency{
		dep("a", "b"),
		dep("a", "c"),
	}}
	current := Graph{Dependencies: []Dependency{
		dep("a", "b"), // unchanged
		dep("a", "d"), // added
	}}
	// dep("a", "c") is removed (present in baseline, absent from current)

	d := DiffGraphs(baseline, current)

	if len(d.Added) != 1 || d.Added[0].Target != "d" {
		t.Errorf("Added = %+v, want [a->d]", d.Added)
	}
	if len(d.Removed) != 1 || d.Removed[0].Target != "c" {
		t.Errorf("Removed = %+v, want [a->c]", d.Removed)
	}
	if len(d.Unchanged) != 1 || d.Unchanged[0].Target != "b" {
		t.Errorf("Unchanged = %+v, want [a->b]", d.Unchanged)
	}
}

func TestDiffGraphs_IdenticalGraphs(t *testing.T) {
	g := Graph{Dependencies: []Dependency{dep("a", "b")}}
	d := DiffGraphs(g, g)

	if len(d.Added) != 0 || len(d.Removed) != 0 {
		t.Errorf("expected no added/removed for identical graphs, got %+v", d)
	}
	if len(d.Unchanged) != 1 {
		t.Errorf("expected 1 unchanged edge, got %+v", d.Unchanged)
	}
}

func TestDiffGraphs_EmptyBaseline(t *testing.T) {
	current := Graph{Dependencies: []Dependency{dep("a", "b")}}
	d := DiffGraphs(Graph{}, current)

	if len(d.Added) != 1 {
		t.Errorf("expected every edge to be added against an empty baseline, got %+v", d)
	}
	if len(d.Removed) != 0 || len(d.Unchanged) != 0 {
		t.Errorf("expected no removed/unchanged edges, got %+v", d)
	}
}

func TestDiffGraphs_EmptyCurrent(t *testing.T) {
	baseline := Graph{Dependencies: []Dependency{dep("a", "b")}}
	d := DiffGraphs(baseline, Graph{})

	if len(d.Removed) != 1 {
		t.Errorf("expected every baseline edge to be removed against an empty current, got %+v", d)
	}
	if len(d.Added) != 0 || len(d.Unchanged) != 0 {
		t.Errorf("expected no added/unchanged edges, got %+v", d)
	}
}

func TestDiffGraphs_IgnoresEvidenceDifferences(t *testing.T) {
	baseline := Graph{Dependencies: []Dependency{
		{Source: "a", Relation: RelationDependsOn, Target: "b", Evidence: []Evidence{{Kind: EvidenceKindManifest, Source: "go.mod"}}},
	}}
	current := Graph{Dependencies: []Dependency{
		{Source: "a", Relation: RelationDependsOn, Target: "b", Evidence: []Evidence{{Kind: EvidenceKindSourceImport, File: "x.go", Line: 5}}},
	}}

	d := DiffGraphs(baseline, current)

	if len(d.Added) != 0 || len(d.Removed) != 0 {
		t.Errorf("edge with only evidence differences must be classified as unchanged, got %+v", d)
	}
	if len(d.Unchanged) != 1 {
		t.Errorf("expected 1 unchanged edge, got %+v", d.Unchanged)
	}
}
