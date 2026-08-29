package golang

import (
	"context"
	"testing"
	"time"
)

// TestScanRoots_CrossRootRegistryResolution is the regression case for the
// exact problem found scanning the real ecosystem: scanning each org
// separately meant a cross-org deep import couldn't be coarsened to its
// owning module, because that module's registry entry only existed in a
// different root's own discovery pass. ScanRoots must discover across all
// roots before building the registry so this resolves correctly.
func TestScanRoots_CrossRootRegistryResolution(t *testing.T) {
	orgA := t.TempDir()
	orgB := t.TempDir()

	writeModule(t, orgA, "consumer", "github.com/orgA/consumer", "github.com/orgB/provider/deep/pkg")
	writeModule(t, orgB, "provider", "github.com/orgB/provider")

	g, reports, err := ScanRoots(context.Background(), []string{orgA, orgB}, time.Now(), ScanOptions{Scanner: "test"})
	if err != nil {
		t.Fatalf("ScanRoots() error = %v", err)
	}

	if len(reports) != 2 {
		t.Fatalf("reports = %+v, want 2 (one per root)", reports)
	}

	if !g.HasEdge("go:github.com/orgA/consumer", "go:github.com/orgB/provider") {
		t.Errorf("expected the deep cross-root import to resolve to its owning module, deps = %+v", g.Dependencies)
	}
	if g.HasEdge("go:github.com/orgA/consumer", "go:github.com/orgB/provider/deep/pkg") {
		t.Error("deep import must not remain unresolved when its owning module is discovered under a different root")
	}

	if len(g.Scan.Coverage) != 2 {
		t.Errorf("Scan.Coverage = %v, want both roots listed", g.Scan.Coverage)
	}
}

func TestScanRoots_CombinesModulesAndReportsFromAllRoots(t *testing.T) {
	orgA := t.TempDir()
	orgB := t.TempDir()

	writeModule(t, orgA, "keep", "github.com/orgA/keep")
	writeModule(t, orgB, "keep", "github.com/orgB/keep")

	g, reports, err := ScanRoots(context.Background(), []string{orgA, orgB}, time.Now(), ScanOptions{Scanner: "test"})
	if err != nil {
		t.Fatalf("ScanRoots() error = %v", err)
	}
	if len(g.Components) != 2 {
		t.Errorf("Components = %+v, want 2", g.Components)
	}
	if len(reports[0].Included) != 1 || len(reports[1].Included) != 1 {
		t.Errorf("expected each root's report to include exactly its own module, reports = %+v", reports)
	}
}
