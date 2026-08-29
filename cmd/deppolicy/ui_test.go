package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUICmd_Wiring(t *testing.T) {
	root := t.TempDir()

	policyPath := filepath.Join(root, "policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"schemaVersion":"1","scope":{"include":["go:github.com/orgA/*"]}}`), 0o600); err != nil {
		t.Fatalf("write policy.json: %v", err)
	}
	graphPath := filepath.Join(root, "graph.json")
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
	outDir := filepath.Join(root, "report")

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"ui", "--policy", policyPath, "--graph", graphPath, "--output-dir", outDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, name := range []string{"index.html", "findings.html", "components.html", "graph.html", "tiers.html"} {
		data, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
		if !strings.HasPrefix(string(data), "<!DOCTYPE html>") {
			t.Errorf("%s missing doctype", name)
		}
	}

	// The unauthorized edge must appear as a violation in the findings page.
	findings, err := os.ReadFile(filepath.Join(outDir, "findings.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(findings), `"status":"violation"`) {
		t.Error("findings.html missing violation data")
	}
}

func TestUICmd_RequiresFlags(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"ui"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error when required flags are omitted")
	}
}
