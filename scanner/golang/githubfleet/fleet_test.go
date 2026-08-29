package githubfleet

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/grokify/gogithub"

	"github.com/grokify/deppolicy/policy"
)

// fakeClient is a hand-written test double for Client. Client's narrow,
// two-method-plus-listing shape (rather than the full 60+-method
// clientv1.Client) is what makes this practical without a mocking
// framework or a real GitHub token.
type fakeClient struct {
	repos   []*gogithub.Repository
	listErr error

	fileExists     map[string]bool
	fileExistsErr  map[string]error
	fileExistsCall map[string]int

	fileContent    map[string][]byte
	fileContentErr map[string]error
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		fileExists:     map[string]bool{},
		fileExistsErr:  map[string]error{},
		fileExistsCall: map[string]int{},
		fileContent:    map[string][]byte{},
		fileContentErr: map[string]error{},
	}
}

func key(owner, repo string) string { return owner + "/" + repo }

func (f *fakeClient) ListOrgRepos(_ context.Context, _ string) ([]*gogithub.Repository, error) {
	return f.repos, f.listErr
}

func (f *fakeClient) FileExists(_ context.Context, owner, repo, _ string, _ *gogithub.ContentOptions) (bool, error) {
	k := key(owner, repo)
	f.fileExistsCall[k]++
	if err, ok := f.fileExistsErr[k]; ok {
		return false, err
	}
	return f.fileExists[k], nil
}

func (f *fakeClient) GetFileContent(_ context.Context, owner, repo, _ string, _ *gogithub.ContentOptions) ([]byte, error) {
	k := key(owner, repo)
	if err, ok := f.fileContentErr[k]; ok {
		return nil, err
	}
	return f.fileContent[k], nil
}

func TestScanOrgManifests_IncludesRepoWithGoMod(t *testing.T) {
	client := newFakeClient()
	client.repos = []*gogithub.Repository{{Name: "widgets"}}
	client.fileExists[key("orgA", "widgets")] = true
	client.fileContent[key("orgA", "widgets")] = []byte("module github.com/orgA/widgets\n\ngo 1.26\n\nrequire github.com/orgA/gizmos v1.0.0\n")

	report, err := ScanOrgManifests(context.Background(), client, "orgA", nil)
	if err != nil {
		t.Fatalf("ScanOrgManifests() error = %v", err)
	}

	if len(report.Included) != 1 {
		t.Fatalf("Included = %+v, want 1", report.Included)
	}
	r := report.Included[0]
	if r.ModuleLocator != "go:github.com/orgA/widgets" {
		t.Errorf("ModuleLocator = %q", r.ModuleLocator)
	}
	if len(r.Dependencies) != 1 || r.Dependencies[0].Target != "go:github.com/orgA/gizmos" {
		t.Errorf("Dependencies = %+v", r.Dependencies)
	}
}

func TestScanOrgManifests_ArchivedRepoExcludedWithoutFetch(t *testing.T) {
	client := newFakeClient()
	client.repos = []*gogithub.Repository{{Name: "old-thing", Archived: true}}

	report, err := ScanOrgManifests(context.Background(), client, "orgA", nil)
	if err != nil {
		t.Fatalf("ScanOrgManifests() error = %v", err)
	}

	if len(report.Excluded) != 1 || report.Excluded[0].Reason != "archived" {
		t.Errorf("Excluded = %+v, want 1 entry with reason \"archived\"", report.Excluded)
	}
	if client.fileExistsCall[key("orgA", "old-thing")] != 0 {
		t.Error("FileExists must not be called for an archived repo -- the whole point is skipping the fetch")
	}
}

func TestScanOrgManifests_NoGoModExcluded(t *testing.T) {
	client := newFakeClient()
	client.repos = []*gogithub.Repository{{Name: "docs-site"}}
	client.fileExists[key("orgA", "docs-site")] = false

	report, err := ScanOrgManifests(context.Background(), client, "orgA", nil)
	if err != nil {
		t.Fatalf("ScanOrgManifests() error = %v", err)
	}

	if len(report.Excluded) != 1 || report.Excluded[0].Reason != "no go.mod" {
		t.Errorf("Excluded = %+v, want 1 entry with reason \"no go.mod\"", report.Excluded)
	}
	if len(report.Errored) != 0 {
		t.Errorf("Errored = %+v, want none: a missing go.mod is not an error", report.Errored)
	}
}

func TestScanOrgManifests_FileExistsErrorIsRecorded(t *testing.T) {
	client := newFakeClient()
	client.repos = []*gogithub.Repository{{Name: "flaky"}}
	client.fileExistsErr[key("orgA", "flaky")] = errors.New("rate limited")

	report, err := ScanOrgManifests(context.Background(), client, "orgA", nil)
	if err != nil {
		t.Fatalf("ScanOrgManifests() error = %v", err)
	}
	if len(report.Errored) != 1 {
		t.Fatalf("Errored = %+v, want 1 entry", report.Errored)
	}
}

func TestScanOrgManifests_GetContentErrorIsRecorded(t *testing.T) {
	client := newFakeClient()
	client.repos = []*gogithub.Repository{{Name: "flaky"}}
	client.fileExists[key("orgA", "flaky")] = true
	client.fileContentErr[key("orgA", "flaky")] = errors.New("network error")

	report, err := ScanOrgManifests(context.Background(), client, "orgA", nil)
	if err != nil {
		t.Fatalf("ScanOrgManifests() error = %v", err)
	}
	if len(report.Errored) != 1 {
		t.Fatalf("Errored = %+v, want 1 entry", report.Errored)
	}
}

func TestScanOrgManifests_MalformedGoModIsRecorded(t *testing.T) {
	client := newFakeClient()
	client.repos = []*gogithub.Repository{{Name: "broken"}}
	client.fileExists[key("orgA", "broken")] = true
	client.fileContent[key("orgA", "broken")] = []byte("not a go.mod {{{")

	report, err := ScanOrgManifests(context.Background(), client, "orgA", nil)
	if err != nil {
		t.Fatalf("ScanOrgManifests() error = %v", err)
	}
	if len(report.Errored) != 1 {
		t.Fatalf("Errored = %+v, want 1 entry", report.Errored)
	}
}

func TestScanOrgManifests_OnePartialFailureDoesNotAbortOthers(t *testing.T) {
	client := newFakeClient()
	client.repos = []*gogithub.Repository{{Name: "broken"}, {Name: "good"}}
	client.fileExists[key("orgA", "broken")] = true
	client.fileContent[key("orgA", "broken")] = []byte("not a go.mod {{{")
	client.fileExists[key("orgA", "good")] = true
	client.fileContent[key("orgA", "good")] = []byte("module github.com/orgA/good\n\ngo 1.26\n")

	report, err := ScanOrgManifests(context.Background(), client, "orgA", nil)
	if err != nil {
		t.Fatalf("ScanOrgManifests() error = %v", err)
	}
	if len(report.Included) != 1 || len(report.Errored) != 1 {
		t.Errorf("Included = %+v, Errored = %+v, want 1 each", report.Included, report.Errored)
	}
}

func TestScanOrgManifests_ListReposError(t *testing.T) {
	client := newFakeClient()
	client.listErr = errors.New("unauthorized")

	_, err := ScanOrgManifests(context.Background(), client, "orgA", nil)
	if err == nil {
		t.Fatal("expected an error when ListOrgRepos fails")
	}
}

func TestScanOrgsManifests_CombinesOrgsIntoOneGraph(t *testing.T) {
	client := newFakeClient()
	client.repos = []*gogithub.Repository{{Name: "widgets"}}
	client.fileExists[key("orgA", "widgets")] = true
	client.fileContent[key("orgA", "widgets")] = []byte("module github.com/orgA/widgets\n\ngo 1.26\n")

	g, reports, err := ScanOrgsManifests(context.Background(), client, []string{"orgA", "orgB"}, time.Now(), "test", nil)
	if err != nil {
		t.Fatalf("ScanOrgsManifests() error = %v", err)
	}
	if g.Scan.Level != "manifest" {
		t.Errorf("Scan.Level = %q, want manifest", g.Scan.Level)
	}
	if len(g.Scan.Coverage) != 2 {
		t.Errorf("Scan.Coverage = %v, want both orgs", g.Scan.Coverage)
	}
	if len(reports) != 2 {
		t.Fatalf("reports = %+v, want one per org", reports)
	}
	// Both orgs use the same fake repo list, so components should reflect
	// two distinct orgs' worth of scanning even though the underlying
	// fake data is identical.
	if len(g.Components) == 0 {
		t.Error("expected at least one component in the combined graph")
	}
}

func TestScanOrgsManifests_PartialFailureStillReturnsUsableGraph(t *testing.T) {
	client := newFakeClient()
	client.repos = []*gogithub.Repository{{Name: "broken"}, {Name: "good"}}
	client.fileExists[key("orgA", "broken")] = true
	client.fileContent[key("orgA", "broken")] = []byte("not a go.mod {{{")
	client.fileExists[key("orgA", "good")] = true
	client.fileContent[key("orgA", "good")] = []byte("module github.com/orgA/good\n\ngo 1.26\n")

	g, _, err := ScanOrgsManifests(context.Background(), client, []string{"orgA"}, time.Now(), "test", nil)
	if err == nil {
		t.Fatal("expected a wrapped error reporting the broken repo")
	}
	if len(g.Components) != 1 {
		t.Errorf("expected the graph to still include the one good component, got %+v", g.Components)
	}
}

func TestScanOrgManifests_ScopeExcludesOutOfScopeModule(t *testing.T) {
	client := newFakeClient()
	client.repos = []*gogithub.Repository{{Name: "sandbox-scratch"}, {Name: "widgets"}}
	client.fileExists[key("orgA", "sandbox-scratch")] = true
	client.fileContent[key("orgA", "sandbox-scratch")] = []byte("module github.com/orgA/sandbox-scratch\n\ngo 1.26\n")
	client.fileExists[key("orgA", "widgets")] = true
	client.fileContent[key("orgA", "widgets")] = []byte("module github.com/orgA/widgets\n\ngo 1.26\n")

	scope := &policy.Policy{
		SchemaVersion: "1",
		Scope: policy.Scope{
			Include: []string{"go:github.com/orgA/*"},
			Exclude: []string{"go:github.com/orgA/sandbox-*"},
		},
	}

	report, err := ScanOrgManifests(context.Background(), client, "orgA", scope)
	if err != nil {
		t.Fatalf("ScanOrgManifests() error = %v", err)
	}

	if len(report.Included) != 1 || report.Included[0].ModuleLocator != "go:github.com/orgA/widgets" {
		t.Errorf("Included = %+v, want only widgets", report.Included)
	}

	found := false
	for _, e := range report.Excluded {
		if e.Repo == "sandbox-scratch" && e.Reason == "out of policy scope" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected sandbox-scratch to be excluded by scope, Excluded = %+v", report.Excluded)
	}
}
