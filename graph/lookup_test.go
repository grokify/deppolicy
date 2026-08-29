package graph

import "testing"

func TestGraph_HasEdge(t *testing.T) {
	g := validGraph()
	if !g.HasEdge("go:github.com/plexusone/dashforge", "go:github.com/plexusone/omniagent") {
		t.Error("HasEdge() = false for observed edge, want true")
	}
	if g.HasEdge("go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm") {
		t.Error("HasEdge() = true for unobserved edge, want false")
	}
}
