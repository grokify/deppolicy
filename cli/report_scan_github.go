package cli

import (
	"fmt"
	"strings"
)

// FormatScanGitHubReport renders result as a human-readable console
// summary, mirroring FormatScanReport's shape for the local scanner:
// component and edge counts, plus what was excluded or failed, always
// listed by name rather than just counted.
func FormatScanGitHubReport(result ScanGitHubResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%d component(s) discovered, %d direct dependency edge(s)\n",
		len(result.Graph.Components), len(result.Graph.Dependencies))

	var excludedCount, erroredCount int
	for _, r := range result.Reports {
		excludedCount += len(r.Excluded)
		erroredCount += len(r.Errored)
	}

	if excludedCount > 0 {
		fmt.Fprintf(&b, "\n%d repo(s) excluded:\n", excludedCount)
		for _, r := range result.Reports {
			for _, e := range r.Excluded {
				fmt.Fprintf(&b, "  %s/%s: %s\n", e.Owner, e.Repo, e.Reason)
			}
		}
	}

	if erroredCount > 0 {
		fmt.Fprintf(&b, "\n%d repo(s) failed to fetch or parse:\n", erroredCount)
		for _, r := range result.Reports {
			for _, e := range r.Errored {
				fmt.Fprintf(&b, "  %s/%s: %v\n", e.Owner, e.Repo, e.Err)
			}
		}
	}

	if result.ScanWarning != nil {
		fmt.Fprintf(&b, "\nWARNING: %v\n", result.ScanWarning)
	}

	return b.String()
}
