package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/grokify/deppolicy/cli"
)

func newDiffCmd() *cobra.Command {
	var (
		policyPath   string
		baselinePath string
		output       string
		workers      int
	)

	cmd := &cobra.Command{
		Use:   "diff <dir>...",
		Short: "Compare scanned graph(s) against a baseline graph.json for ratchet CI",
		Long: `Compare one or more directories' scanned dependency graph (combined with
one shared component registry, exactly like scan) against a baseline
graph.json.

Existing architecture debt (a violation already present in the baseline)
is reported but does not fail the command. A newly introduced violation --
absent from the baseline -- does. This is the "ratchet" model: a PR cannot
introduce new architecture debt, even before the whole ecosystem is fully
classified.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := cli.Diff(cmd.Context(), cli.DiffOptions{
				PolicyPath:        policyPath,
				BaselineGraphPath: baselinePath,
				CurrentDirs:       args,
				Scanner:           scannerVersion,
				Workers:           workers,
				AsOf:              time.Now(),
			})
			if err != nil {
				return err
			}

			if _, err := fmt.Fprint(cmd.OutOrStdout(), cli.FormatDiffReport(result)); err != nil {
				return err
			}

			if output != "" {
				if err := cli.WriteRatchetFindings(result.Findings, output); err != nil {
					return err
				}
			}

			if result.NewViolations() {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&policyPath, "policy", "", "path to policy.json (required)")
	cmd.Flags().StringVar(&baselinePath, "baseline", "", "path to a baseline graph.json (required)")
	cmd.Flags().StringVar(&output, "output", "", "write ratchet findings as JSON to this path")
	cmd.Flags().IntVar(&workers, "workers", 0, "scan concurrency (0 = GOMAXPROCS)")
	for _, name := range []string{"policy", "baseline"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err) // programming error: flag was just registered above
		}
	}

	return cmd
}
