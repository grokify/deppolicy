package cli

import (
	"fmt"
	"strings"
)

// FormatScanReport renders result as a human-readable console summary:
// component and edge counts, plus what was excluded or failed to parse.
// Excluded and errored modules are always listed, never just counted, so a
// fleet scan's "what did I skip and why" question is answered inline
// rather than requiring a separate lookup.
func FormatScanReport(result ScanResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%d component(s) discovered, %d direct dependency edge(s)\n",
		len(result.Graph.Components), len(result.Graph.Dependencies))

	var excludedCount, erroredCount int
	for _, r := range result.Reports {
		excludedCount += len(r.Excluded)
		erroredCount += len(r.Errored)
	}

	if excludedCount > 0 {
		fmt.Fprintf(&b, "\n%d component(s) excluded:\n", excludedCount)
		for _, r := range result.Reports {
			for _, e := range r.Excluded {
				fmt.Fprintf(&b, "  %s (%s): %s\n", e.ModuleLocator, e.Dir, e.Reason)
			}
		}
	}

	if erroredCount > 0 {
		fmt.Fprintf(&b, "\n%d module(s) failed to parse:\n", erroredCount)
		for _, r := range result.Reports {
			for _, e := range r.Errored {
				fmt.Fprintf(&b, "  %s: %v\n", e.Dir, e.Err)
			}
		}
	}

	if result.ScanWarning != nil {
		fmt.Fprintf(&b, "\nWARNING: %v\n", result.ScanWarning)
	}

	return b.String()
}
