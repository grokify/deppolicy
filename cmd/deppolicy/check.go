package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/grokify/deppolicy/cli"
)

func newCheckCmd() *cobra.Command {
	var (
		policyPath string
		output     string
		workers    int
	)

	cmd := &cobra.Command{
		Use:   "check <dir>",
		Short: "Evaluate a directory's scanned dependency graph against policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := cli.Check(cmd.Context(), cli.CheckOptions{
				PolicyPath: policyPath,
				ScanDir:    args[0],
				Scanner:    scannerVersion,
				Workers:    workers,
				AsOf:       time.Now(),
			})
			if err != nil {
				return err
			}

			if _, err := fmt.Fprint(cmd.OutOrStdout(), cli.FormatCheckReport(result)); err != nil {
				return err
			}

			if output != "" {
				if err := cli.WriteFindings(result.Findings, output); err != nil {
					return err
				}
			}

			if result.Violations() {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&policyPath, "policy", "", "path to policy.json (required)")
	cmd.Flags().StringVar(&output, "output", "", "write findings as JSON to this path")
	cmd.Flags().IntVar(&workers, "workers", 0, "scan concurrency (0 = GOMAXPROCS)")
	if err := cmd.MarkFlagRequired("policy"); err != nil {
		panic(err) // programming error: "policy" flag was just registered above
	}

	return cmd
}
