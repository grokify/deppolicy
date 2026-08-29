package golang

import (
	"context"
	"testing"
	"time"
)

// TestScanRoots_DuplicateModuleLocator is the regression case for a real
// bug found scanning the actual grokify ecosystem: two unrelated
// directories declared the same go.mod module path (an unfinished
// code-generation template scaffold using the placeholder
// "github.com/GIT_USER_ID/GIT_REPO_ID", copied into four different
// locations without being customized). The resulting graph had duplicate
// graph.ComponentObservation entries for the same locator, which corrupted
// downstream fan-in analysis.
func TestScanRoots_DuplicateModuleLocator(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "first-copy", "github.com/GIT_USER_ID/GIT_REPO_ID")
	writeModule(t, root, "second-copy", "github.com/GIT_USER_ID/GIT_REPO_ID")
	writeModule(t, root, "distinct", "github.com/orgA/distinct")

	g, reports, err := ScanRoots(context.Background(), []string{root}, time.Now(), ScanOptions{Scanner: "test"})
	if err != nil {
		t.Fatalf("ScanRoots() error = %v", err)
	}

	count := 0
	for _, c := range g.Components {
		if c.ID == "go:github.com/GIT_USER_ID/GIT_REPO_ID" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("got %d components for the duplicate locator, want exactly 1", count)
	}
	if len(g.Components) != 2 {
		t.Errorf("Components = %+v, want 2 (one deduplicated placeholder, one distinct)", g.Components)
	}

	found := false
	for _, r := range reports {
		for _, e := range r.Excluded {
			if e.ModuleLocator == "go:github.com/GIT_USER_ID/GIT_REPO_ID" {
				found = true
				if e.Reason == "" {
					t.Error("dropped duplicate has no reason recorded")
				}
			}
		}
	}
	if !found {
		t.Error("the dropped duplicate must be recorded in DiscoveryReport.Excluded, not silently dropped")
	}
}

func TestScanRoots_DuplicateAcrossDifferentRoots(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	writeModule(t, rootA, "copy1", "github.com/GIT_USER_ID/GIT_REPO_ID")
	writeModule(t, rootB, "copy2", "github.com/GIT_USER_ID/GIT_REPO_ID")

	g, reports, err := ScanRoots(context.Background(), []string{rootA, rootB}, time.Now(), ScanOptions{Scanner: "test"})
	if err != nil {
		t.Fatalf("ScanRoots() error = %v", err)
	}
	if len(g.Components) != 1 {
		t.Errorf("Components = %+v, want 1 (deduplicated across roots)", g.Components)
	}
	if len(reports) != 2 {
		t.Fatalf("reports = %+v, want one per root", reports)
	}
	if len(reports[0].Excluded) != 0 {
		t.Errorf("reports[0] (first root, kept) Excluded = %+v, want none", reports[0].Excluded)
	}
	if len(reports[1].Excluded) != 1 {
		t.Errorf("reports[1] (second root, dropped) Excluded = %+v, want 1 entry", reports[1].Excluded)
	}
}
