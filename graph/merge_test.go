package graph

import "testing"

func TestMergeDependencies(t *testing.T) {
	deps := []Dependency{
		{
			Source: "dashforge", Relation: RelationDependsOn, Target: "omniagent",
			Evidence: []Evidence{{Kind: EvidenceKindSourceImport, File: "a.go", Line: 1}},
		},
		{
			Source: "dashforge", Relation: RelationDependsOn, Target: "omnillm",
			Evidence: []Evidence{{Kind: EvidenceKindSourceImport, File: "b.go", Line: 2}},
		},
		{
			Source: "dashforge", Relation: RelationDependsOn, Target: "omniagent",
			Evidence: []Evidence{{Kind: EvidenceKindSourceImport, File: "c.go", Line: 3}},
		},
	}

	merged := MergeDependencies(deps)

	if len(merged) != 2 {
		t.Fatalf("MergeDependencies() returned %d edges, want 2", len(merged))
	}
	if merged[0].Target != "omniagent" || merged[1].Target != "omnillm" {
		t.Errorf("MergeDependencies() order = [%s, %s], want first-occurrence order [omniagent, omnillm]", merged[0].Target, merged[1].Target)
	}
	if len(merged[0].Evidence) != 2 {
		t.Errorf("merged omniagent edge has %d evidence entries, want 2", len(merged[0].Evidence))
	}
	if merged[0].Evidence[0].File != "a.go" || merged[0].Evidence[1].File != "c.go" {
		t.Errorf("merged evidence not in input order: %+v", merged[0].Evidence)
	}
}

func TestMergeDependencies_Empty(t *testing.T) {
	if got := MergeDependencies(nil); len(got) != 0 {
		t.Errorf("MergeDependencies(nil) = %v, want empty", got)
	}
}
