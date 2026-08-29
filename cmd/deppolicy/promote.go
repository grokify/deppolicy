package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/grokify/deppolicy/cli"
)

func newPromoteCmd() *cobra.Command {
	var (
		policyPath string
		graphPath  string
		output     string
	)

	cmd := &cobra.Command{
		Use:   "promote",
		Short: "Propose new policy relationships for observed edges that currently violate default-deny",
		Long: `Propose new policy.Relationship entries for every edge in a graph
that the current policy does not authorize.

promote never writes to --policy: it only ever produces a separate,
reviewable draft (via --output) or console report. Each proposed
relationship carries a placeholder reason ("PROMOTED: ...") that must be
replaced with a real one before it is merged into the real policy --
promote discovers missing architecture decisions, it does not make them.

Unlike scan/check/diff, promote never scans live: pass a graph.json
produced by a prior "deppolicy scan" run. Bootstrap review is iterative,
and re-scanning a large fleet on every promote run would be wasteful.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := cli.Promote(cli.PromoteOptions{
				PolicyPath: policyPath,
				GraphPath:  graphPath,
				AsOf:       time.Now(),
			})
			if err != nil {
				return err
			}

			if _, err := fmt.Fprint(cmd.OutOrStdout(), cli.FormatPromoteReport(result)); err != nil {
				return err
			}

			if output != "" {
				if err := cli.WriteProposedPolicy(result.ProposedPolicy, output); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&policyPath, "policy", "", "path to the current policy.json (required)")
	cmd.Flags().StringVar(&graphPath, "graph", "", "path to a previously scanned graph.json (required)")
	cmd.Flags().StringVar(&output, "output", "", "write the proposed policy (original + proposed relationships) as JSON to this path")
	for _, name := range []string{"policy", "graph"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err) // programming error: flag was just registered above
		}
	}

	return cmd
}
