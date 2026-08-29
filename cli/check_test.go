package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grokify/deppolicy/policy"
)

func writeCliFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

const checkTestPolicy = `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"] },
  "relationships": [
    {
      "source": "go:github.com/orgA/consumer",
      "relation": "can_depend_on",
      "target": "go:github.com/orgA/provider",
      "reason": "test fixture"
    }
  ]
}`

func buildCheckFixture(t *testing.T) (scanDir, policyPath string) {
	t.Helper()
	root := t.TempDir()

	writeCliFile(t, filepath.Join(root, "consumer/go.mod"), "module github.com/orgA/consumer\n\ngo 1.26\n")
	writeCliFile(t, filepath.Join(root, "consumer/main.go"), `package main

import "github.com/orgA/badtarget"

func main() { _ = badtarget.X }
`)

	writeCliFile(t, filepath.Join(root, "badtarget/go.mod"), "module github.com/orgA/badtarget\n\ngo 1.26\n")
	writeCliFile(t, filepath.Join(root, "badtarget/main.go"), "package main\n\nfunc main() {}\n")

	policyPath = filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, checkTestPolicy)

	return root, policyPath
}

func TestCheck_ReportsViolation(t *testing.T) {
	scanDir, policyPath := buildCheckFixture(t)

	result, err := Check(context.Background(), CheckOptions{
		PolicyPath: policyPath,
		ScanDir:    scanDir,
		Scanner:    "test",
		AsOf:       time.Now(),
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if !result.Violations() {
		t.Fatal("expected a violation for the unauthorized consumer -> badtarget edge")
	}

	found := false
	for _, f := range result.Findings {
		if f.Source == "go:github.com/orgA/consumer" && f.Target == "go:github.com/orgA/badtarget" {
			found = true
			if f.Status != policy.FindingViolation {
				t.Errorf("Status = %q, want %q", f.Status, policy.FindingViolation)
			}
		}
	}
	if !found {
		t.Fatal("expected a finding for consumer -> badtarget")
	}
}

func TestCheck_MissingPolicyFile(t *testing.T) {
	scanDir, _ := buildCheckFixture(t)

	_, err := Check(context.Background(), CheckOptions{
		PolicyPath: filepath.Join(scanDir, "no-such-policy.json"),
		ScanDir:    scanDir,
		Scanner:    "test",
		AsOf:       time.Now(),
	})
	if err == nil {
		t.Fatal("expected an error for a missing policy file")
	}
}

func TestCheck_InvalidPolicy(t *testing.T) {
	scanDir, _ := buildCheckFixture(t)

	badPolicyPath := filepath.Join(t.TempDir(), "policy.json")
	writeCliFile(t, badPolicyPath, `{"schemaVersion":"1","scope":{"include":[]}}`)

	_, err := Check(context.Background(), CheckOptions{
		PolicyPath: badPolicyPath,
		ScanDir:    scanDir,
		Scanner:    "test",
		AsOf:       time.Now(),
	})
	if err == nil {
		t.Fatal("expected an error for a structurally invalid policy")
	}
}

func TestCheckResult_Violations(t *testing.T) {
	clean := CheckResult{Findings: []policy.Finding{{Status: policy.FindingConforming}}}
	if clean.Violations() {
		t.Error("Violations() = true for an all-conforming result, want false")
	}

	dirty := CheckResult{Findings: []policy.Finding{{Status: policy.FindingViolation}}}
	if !dirty.Violations() {
		t.Error("Violations() = false for a result containing a violation, want true")
	}
}
