package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/grokify/deppolicy/cli"
)

func newValidateCmd() *cobra.Command {
	var (
		policyPath string
		graphPath  string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a policy document: schema, structure, expired exceptions, and (optionally) referential integrity",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := cli.Validate(cli.ValidateOptions{
				PolicyPath: policyPath,
				GraphPath:  graphPath,
				AsOf:       time.Now(),
			})
			if err != nil {
				return err
			}

			if _, err := fmt.Fprint(cmd.OutOrStdout(), cli.FormatValidateReport(result)); err != nil {
				return err
			}

			if !result.OK() {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&policyPath, "policy", "", "path to policy.json (required)")
	cmd.Flags().StringVar(&graphPath, "graph", "", "path to a graph.json for referential-integrity checks (optional)")
	if err := cmd.MarkFlagRequired("policy"); err != nil {
		panic(err) // programming error: "policy" flag was just registered above
	}

	return cmd
}
