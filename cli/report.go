package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

// evidenceFor returns the evidence recorded for the observed edge
// source -> target, if any. A finding with no matching graph edge (e.g. an
// unused allowance) simply has no evidence to show.
func evidenceFor(g graph.Graph, source, target string) []graph.Evidence {
	for _, d := range g.Dependencies {
		if d.Source == source && d.Target == target {
			return d.Evidence
		}
	}
	return nil
}

// FormatCheckReport renders result as a human-readable console report: one
// line per finding, with file:line evidence under each violation, followed
// by a summary count. It is a pure function of result, so it is testable
// without capturing stdout.
func FormatCheckReport(result CheckResult) string {
	var b strings.Builder

	counts := make(map[policy.FindingStatus]int)
	for _, f := range result.Findings {
		counts[f.Status]++

		switch f.Status {
		case policy.FindingViolation:
			fmt.Fprintf(&b, "FAIL  %s -> %s\n      %s\n", f.Source, f.Target, f.Reason)
			for _, ev := range evidenceFor(result.Graph, f.Source, f.Target) {
				if ev.File != "" {
					fmt.Fprintf(&b, "      %s:%d\n", ev.File, ev.Line)
				}
			}
		case policy.FindingConforming:
			fmt.Fprintf(&b, "PASS  %s -> %s\n", f.Source, f.Target)
		case policy.FindingUnusedAllowance:
			fmt.Fprintf(&b, "INFO  %s -> %s (unused allowance)\n", f.Source, f.Target)
		}
	}

	fmt.Fprintf(&b, "\n%d conforming, %d violation(s), %d unused allowance(s)\n",
		counts[policy.FindingConforming], counts[policy.FindingViolation], counts[policy.FindingUnusedAllowance])

	if result.ScanWarning != nil {
		fmt.Fprintf(&b, "\nWARNING: %v\n", result.ScanWarning)
	}

	return b.String()
}

// WriteFindings marshals findings as indented JSON and writes them to path.
func WriteFindings(findings []policy.Finding, path string) error {
	return writeJSON(findings, path)
}

// writeJSON marshals v as indented JSON and writes it to path.
func writeJSON(v any, path string) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("cli: marshal: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("cli: write %s: %w", path, err)
	}
	return nil
}
