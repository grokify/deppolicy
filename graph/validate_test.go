package graph

import "testing"

func validGraph() Graph {
	return Graph{
		Scan: ScanMetadata{
			Level:   ScanLevelSourceImports,
			Scanner: "deppolicy-go vtest",
		},
		Dependencies: []Dependency{
			{
				Source:   "go:github.com/plexusone/dashforge",
				Relation: RelationDependsOn,
				Target:   "go:github.com/plexusone/omniagent",
				Evidence: []Evidence{
					{Kind: EvidenceKindSourceImport, File: "internal/agent/client.go", Line: 12},
				},
			},
		},
	}
}

func TestGraph_Validate_Valid(t *testing.T) {
	if err := validGraph().Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestGraph_Validate_UnknownScanLevel(t *testing.T) {
	g := validGraph()
	g.Scan.Level = "bogus"
	if err := g.Validate(); err == nil {
		t.Fatal("expected error for unknown scan level")
	}
}

func TestGraph_Validate_MissingScanner(t *testing.T) {
	g := validGraph()
	g.Scan.Scanner = ""
	if err := g.Validate(); err == nil {
		t.Fatal("expected error for missing scanner")
	}
}

func TestGraph_Validate_MissingSourceOrTarget(t *testing.T) {
	g := validGraph()
	g.Dependencies[0].Target = ""
	if err := g.Validate(); err == nil {
		t.Fatal("expected error for missing target")
	}
}

func TestGraph_Validate_UnsupportedRelation(t *testing.T) {
	g := validGraph()
	g.Dependencies[0].Relation = "can_call"
	if err := g.Validate(); err == nil {
		t.Fatal("expected error for unsupported relation")
	}
}

func TestGraph_Validate_UnrecognizedEvidenceKind(t *testing.T) {
	g := validGraph()
	g.Dependencies[0].Evidence[0].Kind = "guesswork"
	if err := g.Validate(); err == nil {
		t.Fatal("expected error for unrecognized evidence kind")
	}
}
