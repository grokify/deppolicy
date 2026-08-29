package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromoteCmd_Wiring(t *testing.T) {
	root := t.TempDir()

	policyPath := filepath.Join(root, "policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"schemaVersion":"1","scope":{"include":["go:github.com/orgA/*"]}}`), 0o600); err != nil {
		t.Fatalf("write policy.json: %v", err)
	}

	graphPath := filepath.Join(root, "graph.json")
	graphJSON := `{
  "scan": { "level": "source-imports", "scanner": "test" },
  "dependencies": [
    { "source": "go:github.com/orgA/a", "relation": "depends_on", "target": "go:github.com/orgA/b" }
  ]
}`
	if err := os.WriteFile(graphPath, []byte(graphJSON), 0o600); err != nil {
		t.Fatalf("write graph.json: %v", err)
	}

	outputPath := filepath.Join(root, "proposed.json")

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"promote", "--policy", policyPath, "--graph", graphPath, "--output", outputPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "PROPOSE  go:github.com/orgA/a -> go:github.com/orgA/b") {
		t.Errorf("unexpected output: %s", out.String())
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("expected proposed policy to be written: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("written proposed.json is not valid JSON: %v", err)
	}

	originalAfter, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatalf("read policy after run: %v", err)
	}
	if strings.Contains(string(originalAfter), "go:github.com/orgA/b") {
		t.Error("promote must never modify the original --policy file")
	}
}

func TestPromoteCmd_RequiresPolicyAndGraphFlags(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"promote"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error when --policy and --graph are omitted")
	}
}
