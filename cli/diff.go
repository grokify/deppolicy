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

// RatchetStatus classifies a violation by whether it is newly introduced
// relative to a baseline graph or already existed there. This is the
// distinction ratchet CI needs: "existing architecture debt may remain, but
// a PR cannot introduce new architecture debt."
type RatchetStatus string

const (
	// RatchetNewViolation: the edge is absent from the baseline graph --
	// this PR introduced it. Blocks ratchet CI.
	RatchetNewViolation RatchetStatus = "new_violation"

	// RatchetExistingViolation: the edge already violated policy in the
	// baseline graph. Reported, but does not block ratchet CI.
	RatchetExistingViolation RatchetStatus = "existing_violation"

	// RatchetConforming and RatchetUnusedAllowance mirror the
	// corresponding policy.FindingStatus values; ratchet status only
	// distinguishes new from existing for violations.
	RatchetConforming      RatchetStatus = "conforming"
	RatchetUnusedAllowance RatchetStatus = "unused_allowance"
)

// RatchetFinding is one policy.Finding annotated with its ratchet status.
type RatchetFinding struct {
	policy.Finding
	Ratchet RatchetStatus `json:"ratchet"`
}

// ratchetAnnotate classifies each finding by whether its edge is among the
// added set (absent from baseline): a violating added edge is a
// RatchetNewViolation, a violating baselined edge is existing debt.
func ratchetAnnotate(findings []policy.Finding, added []graph.Dependency) []RatchetFinding {
	newEdges := make(map[[2]string]bool, len(added))
	for _, d := range added {
		newEdges[[2]string{d.Source, d.Target}] = true
	}
	ratchet := make([]RatchetFinding, 0, len(findings))
	for _, f := range findings {
		rf := RatchetFinding{Finding: f}
		switch f.Status {
		case policy.FindingViolation:
			if newEdges[[2]string{f.Source, f.Target}] {
				rf.Ratchet = RatchetNewViolation
			} else {
				rf.Ratchet = RatchetExistingViolation
			}
		case policy.FindingConforming:
			rf.Ratchet = RatchetConforming
		case policy.FindingUnusedAllowance:
			rf.Ratchet = RatchetUnusedAllowance
		}
		ratchet = append(ratchet, rf)
	}
	return ratchet
}

// DiffOptions configures Diff.
type DiffOptions struct {
	// PolicyPath is the path to a policy.json document. Required.
	PolicyPath string

	// BaselineGraphPath is the path to a previously saved graph.json --
	// typically produced by a prior `deppolicy scan` run and checked in
	// or cached, representing "what main looks like". Required.
	BaselineGraphPath string

	// CurrentDirs are scanned live (as one combined graph with one shared
	// component registry, exactly like Scan) to produce the graph being
	// compared against the baseline -- typically the PR's working tree,
	// or several org roots for a fleet-wide ratchet run. Required, at
	// least one.
	CurrentDirs []string

	// Scanner identifies this scanner build, recorded in the resulting
	// graph's scan metadata.
	Scanner string

	// Workers bounds scan concurrency; see golang.ScanModules.
	Workers int

	// AsOf is the evaluation time used to resolve exception expiry.
	AsOf time.Time
}

// DiffResult is Diff's structured output.
type DiffResult struct {
	Policy        *policy.Policy
	BaselineGraph graph.Graph
	CurrentGraph  graph.Graph
	GraphDiff     graph.Diff
	Findings      []RatchetFinding

	// ScanWarning is non-nil when some modules under CurrentDirs failed to
	// scan; Findings still reflect whatever did scan successfully.
	ScanWarning error
}

// NewViolations reports whether result contains any newly introduced
// violation -- the ratchet gate that should fail CI. Existing violations
// (already present in the baseline) do not count.
func (r DiffResult) NewViolations() bool {
	for _, f := range r.Findings {
		if f.Ratchet == RatchetNewViolation {
			return true
		}
	}
	return false
}

// Diff loads the policy and baseline graph, scans opts.CurrentDirs, and
// produces a graph-to-graph diff alongside ratchet-annotated findings.
//
// The same baseline graph drives two things at once: graph.DiffGraphs's
// structural new/existing classification, and
// Policy.EvaluateWithStatus's baseline-aware deprecated-target handling --
// both mean the same thing by "new" (absent from baseline), so passing one
// baseline to both keeps that meaning consistent rather than accidentally
// diverging.
func Diff(ctx context.Context, opts DiffOptions) (DiffResult, error) {
	policyData, err := os.ReadFile(opts.PolicyPath)
	if err != nil {
		return DiffResult{}, fmt.Errorf("cli: read policy %s: %w", opts.PolicyPath, err)
	}
	p, err := policy.Parse(policyData)
	if err != nil {
		return DiffResult{}, fmt.Errorf("cli: parse policy %s: %w", opts.PolicyPath, err)
	}

	baselineData, err := os.ReadFile(opts.BaselineGraphPath)
	if err != nil {
		return DiffResult{}, fmt.Errorf("cli: read baseline graph %s: %w", opts.BaselineGraphPath, err)
	}
	baseline, err := graph.Parse(baselineData)
	if err != nil {
		return DiffResult{}, fmt.Errorf("cli: parse baseline graph %s: %w", opts.BaselineGraphPath, err)
	}

	current, _, scanErr := golang.ScanRoots(ctx, opts.CurrentDirs, opts.AsOf, golang.ScanOptions{
		Scanner: opts.Scanner,
		Workers: opts.Workers,
		Scope:   p,
	})
	if scanErr != nil && current.Scan.Scanner == "" {
		return DiffResult{}, fmt.Errorf("cli: scan %v: %w", opts.CurrentDirs, scanErr)
	}

	gdiff := graph.DiffGraphs(*baseline, current)
	findings := p.EvaluateWithStatus(current, opts.AsOf, baseline)
	ratchet := ratchetAnnotate(findings, gdiff.Added)

	return DiffResult{
		Policy:        p,
		BaselineGraph: *baseline,
		CurrentGraph:  current,
		GraphDiff:     gdiff,
		Findings:      ratchet,
		ScanWarning:   scanErr,
	}, nil
}
