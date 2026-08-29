package cli

import (
	"fmt"
	"strings"
)

// FormatDiffReport renders result as a human-readable console report for
// ratchet CI: new violations (FAIL, blocking), existing violations (WARN,
// informational debt), conforming/unused-allowance findings, and the
// underlying graph-level added/removed edge counts.
func FormatDiffReport(result DiffResult) string {
	var b strings.Builder

	for _, f := range result.Findings {
		switch f.Ratchet {
		case RatchetNewViolation:
			fmt.Fprintf(&b, "FAIL  %s -> %s (new)\n      %s\n", f.Source, f.Target, f.Reason)
			for _, ev := range evidenceFor(result.CurrentGraph, f.Source, f.Target) {
				if ev.File != "" {
					fmt.Fprintf(&b, "      %s:%d\n", ev.File, ev.Line)
				}
			}
		case RatchetExistingViolation:
			fmt.Fprintf(&b, "WARN  %s -> %s (existing debt)\n      %s\n", f.Source, f.Target, f.Reason)
		case RatchetConforming:
			fmt.Fprintf(&b, "PASS  %s -> %s\n", f.Source, f.Target)
		case RatchetUnusedAllowance:
			fmt.Fprintf(&b, "INFO  %s -> %s (unused allowance)\n", f.Source, f.Target)
		}
	}

	var newViolations, existingViolations, conforming, unused int
	for _, f := range result.Findings {
		switch f.Ratchet {
		case RatchetNewViolation:
			newViolations++
		case RatchetExistingViolation:
			existingViolations++
		case RatchetConforming:
			conforming++
		case RatchetUnusedAllowance:
			unused++
		}
	}

	fmt.Fprintf(&b, "\n%d conforming, %d new violation(s), %d existing violation(s), %d unused allowance(s)\n",
		conforming, newViolations, existingViolations, unused)
	fmt.Fprintf(&b, "graph: %d edge(s) added, %d removed, %d unchanged\n",
		len(result.GraphDiff.Added), len(result.GraphDiff.Removed), len(result.GraphDiff.Unchanged))

	if result.ScanWarning != nil {
		fmt.Fprintf(&b, "\nWARNING: %v\n", result.ScanWarning)
	}

	return b.String()
}

// WriteRatchetFindings marshals ratchet-annotated findings as indented JSON
// and writes them to path.
func WriteRatchetFindings(findings []RatchetFinding, path string) error {
	return writeJSON(findings, path)
}
