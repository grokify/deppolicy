package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportCmd_Wiring(t *testing.T) {
	root := t.TempDir()

	policyPath := filepath.Join(root, "policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"schemaVersion":"1","scope":{"include":["go:github.com/orgA/*"]}}`), 0o600); err != nil {
		t.Fatalf("write policy.json: %v", err)
	}

	graphPath := filepath.Join(root, "graph.json")
	graphJSON := `{
  "scan": { "level": "source-imports", "scanner": "test" },
  "components": [
    { "id": "go:github.com/orgA/a" },
    { "id": "go:github.com/orgA/b" }
  ],
  "dependencies": [
    { "source": "go:github.com/orgA/a", "relation": "depends_on", "target": "go:github.com/orgA/b" }
  ]
}`
	if err := os.WriteFile(graphPath, []byte(graphJSON), 0o600); err != nil {
		t.Fatalf("write graph.json: %v", err)
	}

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"report", "--policy", policyPath, "--graph", graphPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "2 component(s), 1 direct dependency edge(s)") {
		t.Errorf("unexpected output: %s", out.String())
	}
	if !strings.Contains(out.String(), "go:github.com/orgA/a") {
		t.Errorf("expected the zero-fan-in component to be listed as a rename candidate: %s", out.String())
	}
}

func TestReportCmd_RequiresPolicyAndGraphFlags(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"report"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error when --policy and --graph are omitted")
	}
}
