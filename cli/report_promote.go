package cli

import (
	"fmt"
	"strings"

	"github.com/grokify/deppolicy/policy"
)

// FormatPromoteReport renders result as a human-readable console report:
// one line per proposed relationship, with evidence, followed by a
// summary. It is a pure function of result, so it is testable without
// capturing stdout.
func FormatPromoteReport(result PromoteResult) string {
	var b strings.Builder

	if len(result.Proposed) == 0 {
		fmt.Fprintln(&b, "no unauthorized edges found; policy is fully classified against this graph")
		return b.String()
	}

	for _, p := range result.Proposed {
		fmt.Fprintf(&b, "PROPOSE  %s -> %s\n", p.Source, p.Target)
		for _, ev := range p.Evidence {
			if ev.File != "" {
				fmt.Fprintf(&b, "         %s:%d\n", ev.File, ev.Line)
			}
		}
	}

	fmt.Fprintf(&b, "\n%d relationship(s) proposed\n", len(result.Proposed))

	return b.String()
}

// WriteProposedPolicy marshals p as indented JSON and writes it to path.
// This is always a separate artifact from the original policy document --
// see Promote's doc comment.
func WriteProposedPolicy(p policy.Policy, path string) error {
	return writeJSON(p, path)
}
