package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/grokify/deppolicy/cli"
)

func newUICmd() *cobra.Command {
	var (
		policyPath   string
		graphPath    string
		baselinePath string
		outputDir    string
	)

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Render a self-contained multi-page HTML report (summary, findings, components, graph, tiers)",
		Long: `Render the policy, a scanned graph, and their evaluation as a
multi-page static HTML report: index.html (summary), findings.html
(filterable, including the classification backlog), components.html
(inventory with fan-in/out), graph.html (interactive dependency graph
grouped by org), and tiers.html (tier rules and members).

Every page is one self-contained file -- inline CSS/JS, data embedded as
JSON, no server or CDN -- so the report can be attached to CI runs or
opened from disk for architecture reviews.

With --baseline, violations are split into new versus baselined debt,
exactly as "deppolicy diff" classifies them. The report is strictly
read-only: policy changes flow through promote / policy-diff review.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := cli.UI(cli.UIOptions{
				PolicyPath:        policyPath,
				GraphPath:         graphPath,
				BaselineGraphPath: baselinePath,
				OutputDir:         outputDir,
				AsOf:              time.Now(),
			})
			if err != nil {
				return err
			}
			for _, p := range result.Pages {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), p); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&policyPath, "policy", "", "path to policy.json (required)")
	cmd.Flags().StringVar(&graphPath, "graph", "", "path to a previously scanned graph.json (required)")
	cmd.Flags().StringVar(&baselinePath, "baseline", "", "path to a baseline graph.json for ratchet classification (optional)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "directory to write the report pages into (required)")
	for _, name := range []string{"policy", "graph", "output-dir"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err) // programming error: flag was just registered above
		}
	}

	return cmd
}
