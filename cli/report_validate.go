package cli

import (
	"fmt"
	"strings"
)

// FormatValidateReport renders result as a human-readable console report:
// one line per issue, followed by a summary count.
func FormatValidateReport(result ValidateResult) string {
	var b strings.Builder

	var errorCount, warningCount int
	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityError:
			errorCount++
			fmt.Fprintf(&b, "ERROR  %s\n", issue.Message)
		case SeverityWarning:
			warningCount++
			fmt.Fprintf(&b, "WARN   %s\n", issue.Message)
		}
	}

	if len(result.Issues) == 0 {
		fmt.Fprintln(&b, "policy is valid")
	}

	fmt.Fprintf(&b, "\n%d error(s), %d warning(s)\n", errorCount, warningCount)

	return b.String()
}
