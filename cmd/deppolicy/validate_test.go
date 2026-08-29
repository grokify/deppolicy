package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCmd_Wiring(t *testing.T) {
	policyPath := filepath.Join(t.TempDir(), "policy.json")
	policyJSON := `{"schemaVersion":"1","scope":{"include":["go:github.com/orgA/*"]}}`
	if err := os.WriteFile(policyPath, []byte(policyJSON), 0o600); err != nil {
		t.Fatalf("write policy.json: %v", err)
	}

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"validate", "--policy", policyPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "policy is valid") {
		t.Errorf("unexpected output: %s", out.String())
	}
}

func TestValidateCmd_RequiresPolicyFlag(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"validate"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error when --policy is omitted")
	}
}
