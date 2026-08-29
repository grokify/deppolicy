package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

// ComponentFanIn is one governed component's distinct-dependent count.
type ComponentFanIn struct {
	Locator string
	Count   int
}

// RenameCandidate is a governed component with no observed dependents
// anywhere in the graph. It is safe to rename to a naming-convention-
// excluded form (e.g. "sandbox-*", "*-archive") without breaking any
// importer -- but that is the only claim this makes. A leaf application
// nothing else imports is expected to show up here just as often as an
// abandoned library; this report is a safety check for a rename you
// already have another reason to make, not a dead-code detector.
type RenameCandidate struct {
	Locator string
}

// EcosystemReportOptions configures EcosystemReport.
type EcosystemReportOptions struct {
	// GraphPath is a previously scanned graph.json. Required.
	GraphPath string

	// PolicyPath is required. Rename-safety candidates are, by
	// construction, governed components the policy did NOT already
	// exclude by a naming-convention pattern (scope.exclude) -- those
	// patterns are always policy-authored in this design, never
	// hardcoded in the tool (SPEC.md section 2), so without a policy
	// there is no basis for "not already matching naming patterns."
	PolicyPath string

	// TopN bounds how many fan-in leaders are returned. <= 0 defaults to
	// 20.
	TopN int
}

// EcosystemReportResult is EcosystemReport's structured output.
type EcosystemReportResult struct {
	Policy *policy.Policy
	Graph  graph.Graph

	TotalComponents int
	TotalEdges      int

	// CrossOrgEdges are edges whose source and target locators belong to
	// different GitHub orgs, sorted by (source, target).
	CrossOrgEdges []graph.Dependency

	// TopFanIn holds the components with the most distinct in-scope
	// dependents, sorted descending -- exactly the shape a `foundation`
	// status declaration exists for (SPEC.md section 1.2).
	TopFanIn []ComponentFanIn

	// RenameCandidates holds every in-scope graph component with zero
	// observed dependents, sorted by locator.
	RenameCandidates []RenameCandidate
}

// EcosystemReport loads a policy and a previously scanned graph and
// computes fleet-wide summary statistics: total size, cross-org coupling,
// fan-in leaders (foundation-tier candidates), and rename-safety
// candidates (SPEC.md-adjacent analysis, not itself part of the spec).
//
// This never scans live, for the same reason Promote doesn't: report
// generation is meant to be run repeatedly against one saved graph while
// reviewing it, not to re-pay a multi-minute fleet scan each time.
func EcosystemReport(opts EcosystemReportOptions) (EcosystemReportResult, error) {
	graphData, err := os.ReadFile(opts.GraphPath)
	if err != nil {
		return EcosystemReportResult{}, fmt.Errorf("cli: read graph %s: %w", opts.GraphPath, err)
	}
	g, err := graph.Parse(graphData)
	if err != nil {
		return EcosystemReportResult{}, fmt.Errorf("cli: parse graph %s: %w", opts.GraphPath, err)
	}

	policyData, err := os.ReadFile(opts.PolicyPath)
	if err != nil {
		return EcosystemReportResult{}, fmt.Errorf("cli: read policy %s: %w", opts.PolicyPath, err)
	}
	p, err := policy.Parse(policyData)
	if err != nil {
		return EcosystemReportResult{}, fmt.Errorf("cli: parse policy %s: %w", opts.PolicyPath, err)
	}

	topN := opts.TopN
	if topN <= 0 {
		topN = 20
	}

	fanIn := make(map[string]int, len(g.Components))
	var crossOrg []graph.Dependency
	seen := make(map[[2]string]bool, len(g.Dependencies))
	for _, d := range g.Dependencies {
		key := [2]string{d.Source, d.Target}
		if seen[key] {
			continue
		}
		seen[key] = true

		if p.InScope(d.Target) {
			fanIn[d.Target]++
		}

		sourceOrg, targetOrg := orgOf(d.Source), orgOf(d.Target)
		if sourceOrg != "" && targetOrg != "" && sourceOrg != targetOrg {
			crossOrg = append(crossOrg, d)
		}
	}

	leaders := make([]ComponentFanIn, 0, len(fanIn))
	for locator, count := range fanIn {
		leaders = append(leaders, ComponentFanIn{Locator: locator, Count: count})
	}
	sort.Slice(leaders, func(i, j int) bool {
		if leaders[i].Count != leaders[j].Count {
			return leaders[i].Count > leaders[j].Count
		}
		return leaders[i].Locator < leaders[j].Locator
	})
	if len(leaders) > topN {
		leaders = leaders[:topN]
	}

	var renameCandidates []RenameCandidate
	for _, c := range g.Components {
		if !p.InScope(c.ID) {
			continue
		}
		if fanIn[c.ID] == 0 {
			renameCandidates = append(renameCandidates, RenameCandidate{Locator: c.ID})
		}
	}
	sort.Slice(renameCandidates, func(i, j int) bool { return renameCandidates[i].Locator < renameCandidates[j].Locator })

	sort.Slice(crossOrg, func(i, j int) bool {
		if crossOrg[i].Source != crossOrg[j].Source {
			return crossOrg[i].Source < crossOrg[j].Source
		}
		return crossOrg[i].Target < crossOrg[j].Target
	})

	return EcosystemReportResult{
		Policy:           p,
		Graph:            *g,
		TotalComponents:  len(g.Components),
		TotalEdges:       len(g.Dependencies),
		CrossOrgEdges:    crossOrg,
		TopFanIn:         leaders,
		RenameCandidates: renameCandidates,
	}, nil
}

// orgOf extracts the GitHub organization segment from a "go:github.com/
// <org>/<repo...>" locator, or "" if locator doesn't have that shape (e.g.
// a third-party module on a non-GitHub host, or a non-Go component kind).
func orgOf(locator string) string {
	const prefix = "go:github.com/"
	if !strings.HasPrefix(locator, prefix) {
		return ""
	}
	rest := locator[len(prefix):]
	if idx := strings.Index(rest, "/"); idx >= 0 {
		return rest[:idx]
	}
	return rest
}
