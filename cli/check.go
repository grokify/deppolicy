// Package cli holds the reusable orchestration behind deppolicy's CLI
// commands: load a policy, scan or load a graph, evaluate, and produce a
// structured result. cmd/deppolicy is a thin Cobra adapter over this
// package -- flag parsing and process exit codes live there; everything
// else lives here so it stays testable without invoking the binary.
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

// CheckOptions configures Check.
type CheckOptions struct {
	// PolicyPath is the path to a policy.json document. Required.
	PolicyPath string

	// ScanDir is the directory to scan: a single Go module root, or a
	// root containing several (e.g. an org directory under a pre-cloned
	// workspace). Required.
	ScanDir string

	// Scanner identifies this scanner build, recorded in the resulting
	// graph's scan metadata.
	Scanner string

	// Workers bounds scan concurrency; see golang.ScanModules.
	Workers int

	// AsOf is the evaluation time used to resolve exception expiry. The
	// CLI layer supplies time.Now() here; Check itself never reads the
	// clock, so it stays a thin, testable wrapper around pure evaluation.
	AsOf time.Time
}

// CheckResult is Check's structured output.
type CheckResult struct {
	Policy   *policy.Policy
	Graph    graph.Graph
	Findings []policy.Finding

	// ScanWarning is non-nil when some modules under ScanDir failed to
	// scan; Findings still reflect whatever did scan successfully. It is
	// surfaced rather than silently discarded so a caller can decide how
	// to report it (e.g. a warning line, not a check failure).
	ScanWarning error
}

// Violations reports whether result contains any policy.FindingViolation.
func (r CheckResult) Violations() bool {
	for _, f := range r.Findings {
		if f.Status == policy.FindingViolation {
			return true
		}
	}
	return false
}

// Check loads the policy at opts.PolicyPath, scans opts.ScanDir, and
// evaluates the policy against the resulting graph (without a baseline --
// see the diff command for baseline-aware deprecated-target handling).
func Check(ctx context.Context, opts CheckOptions) (CheckResult, error) {
	data, err := os.ReadFile(opts.PolicyPath)
	if err != nil {
		return CheckResult{}, fmt.Errorf("cli: read policy %s: %w", opts.PolicyPath, err)
	}
	p, err := policy.Parse(data)
	if err != nil {
		return CheckResult{}, fmt.Errorf("cli: parse policy %s: %w", opts.PolicyPath, err)
	}

	g, _, scanErr := golang.Scan(ctx, opts.ScanDir, opts.AsOf, golang.ScanOptions{
		Scanner: opts.Scanner,
		Workers: opts.Workers,
		Scope:   p,
	})
	if scanErr != nil && g.Scan.Scanner == "" {
		// No usable graph was produced at all -- this is fatal.
		return CheckResult{}, fmt.Errorf("cli: scan %s: %w", opts.ScanDir, scanErr)
	}

	findings := p.EvaluateWithStatus(g, opts.AsOf, nil)

	return CheckResult{
		Policy:      p,
		Graph:       g,
		Findings:    findings,
		ScanWarning: scanErr,
	}, nil
}
