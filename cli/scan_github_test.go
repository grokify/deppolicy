package cli

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/grokify/gogithub"
)

// fakeGitHubClient is a minimal test double for githubfleet.Client.
type fakeGitHubClient struct {
	repos       map[string][]*gogithub.Repository
	fileExists  map[string]bool
	fileContent map[string][]byte
}

func newFakeGitHubClient() *fakeGitHubClient {
	return &fakeGitHubClient{
		repos:       map[string][]*gogithub.Repository{},
		fileExists:  map[string]bool{},
		fileContent: map[string][]byte{},
	}
}

func (f *fakeGitHubClient) ListOrgRepos(_ context.Context, org string) ([]*gogithub.Repository, error) {
	return f.repos[org], nil
}

func (f *fakeGitHubClient) FileExists(_ context.Context, owner, repo, _ string, _ *gogithub.ContentOptions) (bool, error) {
	return f.fileExists[owner+"/"+repo], nil
}

func (f *fakeGitHubClient) GetFileContent(_ context.Context, owner, repo, _ string, _ *gogithub.ContentOptions) ([]byte, error) {
	return f.fileContent[owner+"/"+repo], nil
}

func TestScanGitHub_NoPolicy_BootstrapMode(t *testing.T) {
	client := newFakeGitHubClient()
	client.repos["orgA"] = []*gogithub.Repository{{Name: "widgets"}}
	client.fileExists["orgA/widgets"] = true
	client.fileContent["orgA/widgets"] = []byte("module github.com/orgA/widgets\n\ngo 1.26\n")

	result, err := ScanGitHub(context.Background(), client, ScanGitHubOptions{
		Orgs:    []string{"orgA"},
		Scanner: "test",
		AsOf:    time.Now(),
	})
	if err != nil {
		t.Fatalf("ScanGitHub() error = %v", err)
	}
	if result.Policy != nil {
		t.Error("Policy = non-nil, want nil when PolicyPath is empty")
	}
	if len(result.Graph.Components) != 1 {
		t.Errorf("Components = %+v, want 1", result.Graph.Components)
	}
	if result.Graph.Scan.Level != "manifest" {
		t.Errorf("Scan.Level = %q, want manifest", result.Graph.Scan.Level)
	}
}

func TestScanGitHub_WithPolicy_ExcludesOutOfScope(t *testing.T) {
	client := newFakeGitHubClient()
	client.repos["orgA"] = []*gogithub.Repository{{Name: "sandbox-x"}, {Name: "widgets"}}
	client.fileExists["orgA/sandbox-x"] = true
	client.fileContent["orgA/sandbox-x"] = []byte("module github.com/orgA/sandbox-x\n\ngo 1.26\n")
	client.fileExists["orgA/widgets"] = true
	client.fileContent["orgA/widgets"] = []byte("module github.com/orgA/widgets\n\ngo 1.26\n")

	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	writeCliFile(t, policyPath, `{
  "schemaVersion": "1",
  "scope": { "include": ["go:github.com/orgA/*"], "exclude": ["go:github.com/orgA/sandbox-*"] }
}`)

	result, err := ScanGitHub(context.Background(), client, ScanGitHubOptions{
		Orgs:       []string{"orgA"},
		PolicyPath: policyPath,
		Scanner:    "test",
		AsOf:       time.Now(),
	})
	if err != nil {
		t.Fatalf("ScanGitHub() error = %v", err)
	}
	if result.Policy == nil {
		t.Fatal("Policy = nil, want non-nil when PolicyPath is set")
	}
	if len(result.Graph.Components) != 1 || result.Graph.Components[0].ID != "go:github.com/orgA/widgets" {
		t.Errorf("Components = %+v, want only widgets", result.Graph.Components)
	}
}

func TestScanGitHub_MissingPolicyFile(t *testing.T) {
	client := newFakeGitHubClient()
	_, err := ScanGitHub(context.Background(), client, ScanGitHubOptions{
		Orgs:       []string{"orgA"},
		PolicyPath: filepath.Join(t.TempDir(), "no-such-policy.json"),
		Scanner:    "test",
		AsOf:       time.Now(),
	})
	if err == nil {
		t.Fatal("expected an error for a missing policy file")
	}
}

func TestScanGitHub_MultipleOrgs(t *testing.T) {
	client := newFakeGitHubClient()
	client.repos["orgA"] = []*gogithub.Repository{{Name: "widgets"}}
	client.fileExists["orgA/widgets"] = true
	client.fileContent["orgA/widgets"] = []byte("module github.com/orgA/widgets\n\ngo 1.26\n")
	client.repos["orgB"] = []*gogithub.Repository{{Name: "gizmos"}}
	client.fileExists["orgB/gizmos"] = true
	client.fileContent["orgB/gizmos"] = []byte("module github.com/orgB/gizmos\n\ngo 1.26\n")

	result, err := ScanGitHub(context.Background(), client, ScanGitHubOptions{
		Orgs:    []string{"orgA", "orgB"},
		Scanner: "test",
		AsOf:    time.Now(),
	})
	if err != nil {
		t.Fatalf("ScanGitHub() error = %v", err)
	}
	if len(result.Reports) != 2 {
		t.Fatalf("Reports = %+v, want one per org", result.Reports)
	}
	if len(result.Graph.Components) != 2 {
		t.Errorf("Components = %+v, want 2", result.Graph.Components)
	}
}
