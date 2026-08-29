package analyzer_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/grokify/deppolicy/analyzer"
)

// TestAnalyzer drives the real analyzer through analysistest's GOPATH
// driver: the consumer package imports one explicitly authorized target
// (provider), one foundation target (foundlib), and one unauthorized
// target (badtarget) -- only the last carries a `want` diagnostic, so the
// test simultaneously asserts the violation fires at the right line AND
// that the two authorized imports produce no diagnostics at all
// (analysistest fails on unexpected diagnostics, not just missing ones).
func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()

	if err := analyzer.Analyzer.Flags.Set("policy", filepath.Join(testdata, "policy.json")); err != nil {
		t.Fatalf("set policy flag: %v", err)
	}
	// analysistest's driver supplies no module info (pass.Module is nil),
	// so the module override flag is required here; a real `go vet` run
	// gets the module path from the build system instead.
	if err := analyzer.Analyzer.Flags.Set("module", "github.com/orgA/consumer"); err != nil {
		t.Fatalf("set module flag: %v", err)
	}

	analysistest.Run(t, testdata, analyzer.Analyzer, "consumer")
}

// TestAnalyzer_OutOfScopeModuleIsSilent re-runs the same package with a
// module identity outside the governed ecosystem: every diagnostic must
// disappear, since third-party modules are not subject to policy.
func TestAnalyzer_OutOfScopeModuleIsSilent(t *testing.T) {
	testdata := analysistest.TestData()

	if err := analyzer.Analyzer.Flags.Set("policy", filepath.Join(testdata, "policy.json")); err != nil {
		t.Fatalf("set policy flag: %v", err)
	}
	if err := analyzer.Analyzer.Flags.Set("module", "github.com/other/tool"); err != nil {
		t.Fatalf("set module flag: %v", err)
	}

	analysistest.Run(t, testdata, analyzer.Analyzer, "github.com/orgA/provider")
}
