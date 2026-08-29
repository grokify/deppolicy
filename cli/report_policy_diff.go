package cli

import (
	"fmt"
	"strings"
)

// FormatPolicyDiffReport renders result as a human-readable architecture
// change review.
func FormatPolicyDiffReport(result PolicyDiffResult) string {
	var b strings.Builder

	if len(result.Added) == 0 && len(result.Removed) == 0 && len(result.ComponentChanges) == 0 {
		fmt.Fprintln(&b, "no policy changes")
		return b.String()
	}

	for _, r := range result.Added {
		fmt.Fprintf(&b, "ADDED    %s -> %s\n         %s\n", r.Source, r.Target, r.Reason)
	}
	for _, r := range result.Removed {
		if r.StillObserved {
			fmt.Fprintf(&b, "REMOVED  %s -> %s (BREAKING: edge still observed; next fleet scan will report it as a violation)\n", r.Source, r.Target)
		} else {
			fmt.Fprintf(&b, "REMOVED  %s -> %s\n", r.Source, r.Target)
		}
	}
	for _, c := range result.ComponentChanges {
		fmt.Fprintf(&b, "CHANGED  %s", c.Locator)
		if c.OldStatus != c.NewStatus {
			fmt.Fprintf(&b, "  status: %s -> %s", orNone(c.OldStatus), orNone(c.NewStatus))
		}
		if c.OldTier != c.NewTier {
			fmt.Fprintf(&b, "  tier: %s -> %s", orNone(c.OldTier), orNone(c.NewTier))
		}
		fmt.Fprintln(&b)
	}

	var breaking int
	for _, r := range result.Removed {
		if r.StillObserved {
			breaking++
		}
	}
	fmt.Fprintf(&b, "\n%d added, %d removed (%d breaking), %d component change(s)\n",
		len(result.Added), len(result.Removed), breaking, len(result.ComponentChanges))

	return b.String()
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// WritePolicyDiff marshals result as indented JSON and writes it to path.
func WritePolicyDiff(result PolicyDiffResult, path string) error {
	return writeJSON(result, path)
}
