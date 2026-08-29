package golang

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/grokify/deppolicy/policy"
)

// DiscoveredModule is one Go module found during a filesystem walk.
type DiscoveredModule struct {
	Dir           string
	ModuleLocator string
}

// ExcludedModule records a discovered module that was excluded from
// scanning, and why. Excluded is not the same as invisible: a report must
// be able to say what was skipped and why.
type ExcludedModule struct {
	Dir           string
	ModuleLocator string
	Reason        string
}

// DiscoveryError records a module directory whose go.mod could not be
// parsed. A malformed go.mod in one repository must not abort discovery of
// the rest of a fleet.
type DiscoveryError struct {
	Dir string
	Err error
}

// DiscoveryReport is the result of walking a directory tree for Go
// modules.
type DiscoveryReport struct {
	Included []DiscoveredModule
	Excluded []ExcludedModule
	Errored  []DiscoveryError
}

// ExcludedSummary formats each excluded module as one human-readable line,
// suitable for graph.ScanMetadata.Excluded or console reporting (e.g. a
// future `--show-excluded` CLI flag).
func (r DiscoveryReport) ExcludedSummary() []string {
	out := make([]string, 0, len(r.Excluded))
	for _, e := range r.Excluded {
		out = append(out, fmt.Sprintf("%s (%s): %s", e.ModuleLocator, e.Dir, e.Reason))
	}
	return out
}

// DiscoverModules walks root looking for Go modules (directories
// containing a go.mod), skipping built-in noise directories (vendor,
// testdata, node_modules, .git, and any hidden directory) during the walk
// -- these are never real components regardless of policy. The walk
// continues into a module's own subdirectories after finding its go.mod,
// so nested modules (e.g. "tools/go.mod") are discovered too.
//
// If scope is non-nil, each discovered module's locator is additionally
// classified against scope.InScope: a module outside the governed
// ecosystem, or matching a scope.exclude naming-convention pattern (such as
// "sandbox-*", "*-archive", "experiment-*"), is reported as excluded
// rather than included. With scope == nil, every discovered module is
// included -- this mode is for bootstrap discovery before a policy exists.
//
// A malformed go.mod is recorded in Errored rather than aborting the walk.
// The returned error is non-nil only for filesystem-level walk failures
// (e.g. a permission error), not for individual module problems.
func DiscoverModules(root string, scope *policy.Policy) (DiscoveryReport, error) {
	var report DiscoveryReport

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && (skipDirNames[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}

		moduleLocator, _, parseErr := ManifestDependencies(path)
		if parseErr != nil {
			if errors.Is(parseErr, os.ErrNotExist) {
				return nil // no go.mod in this directory
			}
			report.Errored = append(report.Errored, DiscoveryError{Dir: path, Err: parseErr})
			return nil
		}

		if scope != nil && !scope.InScope(moduleLocator) {
			report.Excluded = append(report.Excluded, ExcludedModule{
				Dir:           path,
				ModuleLocator: moduleLocator,
				Reason:        "out of policy scope",
			})
			return nil
		}

		report.Included = append(report.Included, DiscoveredModule{Dir: path, ModuleLocator: moduleLocator})
		return nil
	})

	return report, err
}
