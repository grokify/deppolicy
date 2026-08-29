package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/grokify/deppolicy/cli"
)

func newPolicyDiffCmd() *cobra.Command {
	var (
		graphPath   string
		output      string
		failOnBreak bool
	)

	cmd := &cobra.Command{
		Use:   "policy-diff <old-policy.json> <new-policy.json>",
		Short: "Review a policy change: added/removed relationships and component status/tier changes",
		Long: `Compare two policy documents as an architecture change review.

An added relationship is a newly permitted dependency; a removed one is a
revoked permission. With --graph, a removal whose edge is still observed
in the scanned graph is flagged BREAKING: by design, merging it does not
break unrelated repos' CI -- the next fleet scan reports those edges as
violations instead -- so policy review is the moment to decide whether
the code gets fixed first.

Exit code is 0 even with breaking removals unless --fail-on-breaking is
set (for use in the policy repo's own PR CI).`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := cli.PolicyDiff(cli.PolicyDiffOptions{
				OldPolicyPath: args[0],
				NewPolicyPath: args[1],
				GraphPath:     graphPath,
			})
			if err != nil {
				return err
			}

			if _, err := fmt.Fprint(cmd.OutOrStdout(), cli.FormatPolicyDiffReport(result)); err != nil {
				return err
			}

			if output != "" {
				if err := cli.WritePolicyDiff(result, output); err != nil {
					return err
				}
			}

			if failOnBreak && result.BreakingRemovals() {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&graphPath, "graph", "", "path to a scanned graph.json to classify removals as breaking (optional)")
	cmd.Flags().StringVar(&output, "output", "", "write the diff as JSON to this path")
	cmd.Flags().BoolVar(&failOnBreak, "fail-on-breaking", false, "exit non-zero if any removed relationship's edge is still observed")

	return cmd
}
