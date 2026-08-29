package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/grokify/deppolicy/cli"
)

func newReportCmd() *cobra.Command {
	var (
		policyPath string
		graphPath  string
		output     string
		topN       int
	)

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate an ecosystem summary and rename-safety report from a scanned graph",
		Long: `Generate an ecosystem summary from a previously scanned graph.json:
component/edge counts, cross-org coupling, fan-in leaders (foundation-tier
candidates -- components many others depend on), and a rename-safety list
of components with zero observed dependents.

The rename-safety list only means "nothing in this graph depends on it, so
renaming or reclassifying it will not break an importer." It is not a
dead-code or unused-repository detector: any leaf application nothing
else imports will show up here too.

Like promote, report never scans live: pass a graph.json produced by
"deppolicy scan".`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := cli.EcosystemReport(cli.EcosystemReportOptions{
				PolicyPath: policyPath,
				GraphPath:  graphPath,
				TopN:       topN,
			})
			if err != nil {
				return err
			}

			if _, err := fmt.Fprint(cmd.OutOrStdout(), cli.FormatEcosystemReport(result)); err != nil {
				return err
			}

			if output != "" {
				if err := cli.WriteEcosystemReport(result, output); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&policyPath, "policy", "", "path to policy.json (required)")
	cmd.Flags().StringVar(&graphPath, "graph", "", "path to a previously scanned graph.json (required)")
	cmd.Flags().StringVar(&output, "output", "", "write the report as JSON to this path")
	cmd.Flags().IntVar(&topN, "top", 20, "number of fan-in leaders to include")
	for _, name := range []string{"policy", "graph"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err) // programming error: flag was just registered above
		}
	}

	return cmd
}
