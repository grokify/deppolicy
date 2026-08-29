package golang

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/grokify/deppolicy/component"
	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

// ScanOptions configures a full ecosystem scan.
type ScanOptions struct {
	// Scanner identifies this scanner build, e.g. "deppolicy-go v0.1.0",
	// and is recorded in the resulting graph's scan metadata.
	Scanner string

	// Workers bounds scan concurrency; see ScanModules.
	Workers int

	// Scope, if set, is used both to exclude out-of-scope modules during
	// discovery (see DiscoverModules) and to seed the component registry
	// used to resolve deep import paths to their owning module (see
	// ImportDependencies).
	Scope *policy.Policy
}

// ScanRoots discovers Go modules under every root and produces one combined
// graph.Graph: discovery across all roots, concurrent import scanning,
// aggregation, and scan metadata.
//
// Discovering across all roots before building the component registry --
// rather than scanning each root independently and merging results -- is
// what lets a deep import from a module under one root correctly resolve
// to its owning module under a different root (e.g. a plexusone module
// importing a grokify module's subpackage): the registry needs to know
// about every governed module across the whole fleet, not just the ones
// under whichever root happens to contain the importing code.
//
// generated is passed explicitly rather than read from the wall clock, so
// that everything upstream of policy evaluation remains reproducible given
// the same inputs.
//
// The returned error is non-nil only when the scan could not produce a
// usable graph at all (a filesystem walk failure, or the assembled graph
// fails structural validation). A module that individually failed to scan
// does not cause this error; it is instead reported via the returned
// per-root DiscoveryReport.Errored and folded into a single wrapped error,
// which callers can inspect without losing the rest of the scan.
//
// If two directories -- anywhere across all roots -- declare the same
// module locator, only the first encountered (in root, then discovery,
// order) is scanned and included; the rest are recorded in their own
// root's DiscoveryReport.Excluded rather than silently dropped or left to
// produce duplicate graph.ComponentObservation entries. This is common
// enough in practice to be worth handling explicitly rather than treating
// as a should-never-happen case: an unfinished code-generation template
// scaffold whose go.mod placeholder (e.g. "github.com/GIT_USER_ID/
// GIT_REPO_ID") was never customized will do exactly this across every
// copy of that template in a large tree.
func ScanRoots(ctx context.Context, roots []string, generated time.Time, opts ScanOptions) (graph.Graph, []DiscoveryReport, error) {
	reports := make([]DiscoveryReport, len(roots))
	type rootModule struct {
		DiscoveredModule
		rootIndex int
	}
	var discovered []rootModule

	for i, root := range roots {
		report, err := DiscoverModules(root, opts.Scope)
		if err != nil {
			return graph.Graph{}, reports, fmt.Errorf("golang: discover %s: %w", root, err)
		}
		reports[i] = report
		for _, m := range report.Included {
			discovered = append(discovered, rootModule{DiscoveredModule: m, rootIndex: i})
		}
	}

	var included []DiscoveredModule
	keptDirByLocator := make(map[string]string, len(discovered))
	for _, m := range discovered {
		if keptDir, dup := keptDirByLocator[m.ModuleLocator]; dup {
			reports[m.rootIndex].Excluded = append(reports[m.rootIndex].Excluded, ExcludedModule{
				Dir:           m.Dir,
				ModuleLocator: m.ModuleLocator,
				Reason:        fmt.Sprintf("duplicate module locator (already scanned from %s)", keptDir),
			})
			continue
		}
		keptDirByLocator[m.ModuleLocator] = m.Dir
		included = append(included, m.DiscoveredModule)
	}

	registry, err := buildRegistry(included, opts.Scope)
	if err != nil {
		return graph.Graph{}, reports, err
	}

	dirs := make([]string, len(included))
	for i, m := range included {
		dirs[i] = m.Dir
	}

	results := ScanModules(ctx, dirs, registry, opts.Workers)
	deps, scanErrs := MergeResults(results)

	components := make([]graph.ComponentObservation, 0, len(included))
	for _, m := range included {
		components = append(components, graph.ComponentObservation{
			ID:     m.ModuleLocator,
			Kind:   component.KindGoModule,
			Commit: gitCommit(ctx, m.Dir),
		})
	}

	var excluded []string
	for _, r := range reports {
		excluded = append(excluded, r.ExcludedSummary()...)
	}

	g := graph.Graph{
		Generated: generated,
		Scan: graph.ScanMetadata{
			Level:    graph.ScanLevelSourceImports,
			Scanner:  opts.Scanner,
			Coverage: roots,
			Excluded: excluded,
		},
		Components:   components,
		Dependencies: deps,
	}

	if err := g.Validate(); err != nil {
		return graph.Graph{}, reports, err
	}

	if len(scanErrs) > 0 {
		return g, reports, fmt.Errorf("golang: %d module(s) failed to scan: %w", len(scanErrs), errors.Join(scanErrs...))
	}

	return g, reports, nil
}

// Scan is ScanRoots for a single root, returning that root's
// DiscoveryReport directly instead of a single-element slice.
func Scan(ctx context.Context, root string, generated time.Time, opts ScanOptions) (graph.Graph, DiscoveryReport, error) {
	g, reports, err := ScanRoots(ctx, []string{root}, generated, opts)
	return g, reports[0], err
}

// buildRegistry seeds a component.Registry from every discovered module
// (as a go-module component with no explicit status) and then overlays any
// policy-declared components on top: a declaration at a discovered
// module's own locator overrides its kind/status/reason/decision, while a
// declaration at a finer locator (a proto-module carve-out with no
// matching discovered module) is added as its own component. Overlaying
// into a map keyed by locator, rather than concatenating two slices, is
// what avoids a duplicate-locator error from component.NewRegistry when a
// discovered module and its policy declaration share a locator.
func buildRegistry(included []DiscoveredModule, scope *policy.Policy) (*component.Registry, error) {
	byLocator := make(map[string]component.Component, len(included))
	for _, m := range included {
		byLocator[m.ModuleLocator] = component.Component{
			ID:      component.ID(m.ModuleLocator),
			Kind:    component.KindGoModule,
			Locator: m.ModuleLocator,
		}
	}

	if scope != nil {
		for locator, decl := range scope.Components {
			c := byLocator[locator]
			c.ID = component.ID(locator)
			c.Locator = locator
			if decl.Kind != "" {
				c.Kind = decl.Kind
			}
			c.Status = decl.Status
			c.Reason = decl.Reason
			c.Decision = decl.Decision
			byLocator[locator] = c
		}
	}

	components := make([]component.Component, 0, len(byLocator))
	for _, c := range byLocator {
		components = append(components, c)
	}
	return component.NewRegistry(components)
}

// gitCommit returns dir's current HEAD commit SHA, or "" if dir is not
// inside a git working tree or git is unavailable. This is best-effort
// provenance, not a scan precondition: DepPolicy must be able to scan
// module directories that are not under git at all.
func gitCommit(ctx context.Context, dir string) string {
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
