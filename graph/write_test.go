package graph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGraph_WriteFile(t *testing.T) {
	g := validGraph()
	path := filepath.Join(t.TempDir(), "graph.json")

	if err := g.WriteFile(path); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	roundTripped, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse(written file) error = %v", err)
	}
	if !roundTripped.HasEdge("go:github.com/plexusone/dashforge", "go:github.com/plexusone/omniagent") {
		t.Error("round-tripped graph is missing the expected edge")
	}
}

func TestGraph_WriteFile_InvalidPath(t *testing.T) {
	g := validGraph()
	if err := g.WriteFile(filepath.Join(t.TempDir(), "no-such-dir", "graph.json")); err == nil {
		t.Fatal("expected error writing to a nonexistent directory")
	}
}
