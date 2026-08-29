package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTablesFixture(t *testing.T) (policyPath, graphPath string) {
	t.Helper()
	root := t.TempDir()
	policyPath = filepath.Join(root, "policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"schemaVersion":"1","scope":{"include":["go:github.com/orgA/*"]}}`), 0o600); err != nil {
		t.Fatalf("write policy.json: %v", err)
	}
	graphPath = filepath.Join(root, "graph.json")
	graphJSON := `{
  "scan": { "level": "source-imports", "scanner": "test" },
  "components": [ { "id": "go:github.com/orgA/a" } ],
  "dependencies": [
    { "source": "go:github.com/orgA/a", "relation": "depends_on", "target": "go:github.com/orgA/b" }
  ]
}`
	if err := os.WriteFile(graphPath, []byte(graphJSON), 0o600); err != nil {
		t.Fatalf("write graph.json: %v", err)
	}
	return policyPath, graphPath
}

func TestTablesCmd_StdoutJSON(t *testing.T) {
	policyPath, graphPath := writeTablesFixture(t)

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"tables", "--policy", policyPath, "--graph", graphPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var tables []map[string]any
	if err := json.Unmarshal(out.Bytes(), &tables); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, out.String())
	}
	if len(tables) != 4 {
		t.Errorf("got %d tables, want 4", len(tables))
	}
	// The unauthorized observed edge must appear as a violation finding.
	if !strings.Contains(out.String(), `"violation"`) {
		t.Errorf("expected a violation row in findings, got:\n%s", out.String())
	}
}

func TestTablesCmd_OutputDirCSV(t *testing.T) {
	policyPath, graphPath := writeTablesFixture(t)
	outDir := filepath.Join(t.TempDir(), "tables")

	cmd := newRootCmd()
	cmd.SetArgs([]string{"tables", "--policy", policyPath, "--graph", graphPath, "--output-dir", outDir, "--format", "csv"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, name := range []string{"components", "policy_edges", "observed_edges", "findings"} {
		path := filepath.Join(outDir, name+".csv")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
		if len(data) == 0 {
			t.Errorf("%s is empty", path)
		}
	}
}
