package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
	"github.com/grokify/deppolicy/scanner/golang"
)

// ScanOptions configures Scan.
type ScanOptions struct {
	// RootDirs are the directory trees to scan, combined into one graph
	// with one shared component registry. Required, at least one.
	//
	// Discovering across all roots before scanning is what lets a deep
	// import in a module under one root resolve to its owning module
	// under a different root -- e.g. an org-B module importing an org-A
	// module's subpackage -- rather than being recorded as an unresolved
	// raw import path just because that org wasn't scanned together with
	// this one.
	RootDirs []string

	// PolicyPath, if set, is used both for scope-based exclusion during
	// discovery and to seed the component registry that resolves deep
	// import paths (see golang.ScanOptions.Scope). Omit to scan in
	// bootstrap mode: no policy exists yet, so nothing is excluded and
	// the registry is seeded purely from discovered modules.
	PolicyPath string

	// Scanner identifies this scanner build, recorded in the resulting
	// graph's scan metadata.
	Scanner string

	// Workers bounds scan concurrency; see golang.ScanModules.
	Workers int

	// AsOf is recorded as the graph's Generated timestamp.
	AsOf time.Time
}

// ScanResult is Scan's structured output.
type ScanResult struct {
	// Policy is non-nil only when opts.PolicyPath was set.
	Policy *policy.Policy
	Graph  graph.Graph

	// Reports holds one DiscoveryReport per root, in the same order as
	// opts.RootDirs.
	Reports []golang.DiscoveryReport

	// ScanWarning is non-nil when some modules failed to scan; Graph
	// still reflects whatever did scan successfully.
	ScanWarning error
}

// Scan discovers and scans every Go module under opts.RootDirs as one
// combined graph, optionally scoped and registry-seeded by a policy
// document.
func Scan(ctx context.Context, opts ScanOptions) (ScanResult, error) {
	var p *policy.Policy
	if opts.PolicyPath != "" {
		data, err := os.ReadFile(opts.PolicyPath)
		if err != nil {
			return ScanResult{}, fmt.Errorf("cli: read policy %s: %w", opts.PolicyPath, err)
		}
		parsed, err := policy.Parse(data)
		if err != nil {
			return ScanResult{}, fmt.Errorf("cli: parse policy %s: %w", opts.PolicyPath, err)
		}
		p = parsed
	}

	g, reports, scanErr := golang.ScanRoots(ctx, opts.RootDirs, opts.AsOf, golang.ScanOptions{
		Scanner: opts.Scanner,
		Workers: opts.Workers,
		Scope:   p,
	})
	if scanErr != nil && g.Scan.Scanner == "" {
		return ScanResult{}, fmt.Errorf("cli: scan %v: %w", opts.RootDirs, scanErr)
	}

	return ScanResult{Policy: p, Graph: g, Reports: reports, ScanWarning: scanErr}, nil
}
