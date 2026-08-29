package cli

import (
	"fmt"
	"strings"
)

// FormatEcosystemReport renders result as a human-readable console
// report.
func FormatEcosystemReport(result EcosystemReportResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%d component(s), %d direct dependency edge(s)\n", result.TotalComponents, result.TotalEdges)
	fmt.Fprintf(&b, "%d cross-org edge(s)\n", len(result.CrossOrgEdges))

	if len(result.TopFanIn) > 0 {
		fmt.Fprintf(&b, "\nTop fan-in (foundation-tier candidates):\n")
		for _, c := range result.TopFanIn {
			fmt.Fprintf(&b, "  %4d  %s\n", c.Count, c.Locator)
		}
	}

	fmt.Fprintf(&b, "\n%d component(s) with no observed dependents (safe to rename; not evidence of being unused):\n", len(result.RenameCandidates))
	for _, c := range result.RenameCandidates {
		fmt.Fprintf(&b, "  %s\n", c.Locator)
	}

	return b.String()
}

// WriteEcosystemReport marshals result as indented JSON and writes it to
// path.
func WriteEcosystemReport(result EcosystemReportResult, path string) error {
	return writeJSON(result, path)
}
