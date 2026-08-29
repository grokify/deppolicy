package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/grokify/deppolicy/cli"
	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

func newExportCmd() *cobra.Command {
	var (
		graphPath  string
		policyPath string
		format     string
		output     string
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a scanned graph as Mermaid or DOT, grouped by org, with violations highlighted",
		Long: `Export a previously scanned graph.json as a Mermaid or Graphviz DOT
diagram, with components grouped by GitHub org.

With --policy, only in-scope components and edges are included and
policy-violating edges are highlighted in red; without it, everything in
the graph is rendered unstyled.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(graphPath)
			if err != nil {
				return fmt.Errorf("read graph %s: %w", graphPath, err)
			}
			g, err := graph.Parse(data)
			if err != nil {
				return err
			}

			var p *policy.Policy
			if policyPath != "" {
				p, err = policy.ParseFile(policyPath)
				if err != nil {
					return err
				}
			}

			rendered, err := cli.Export(*g, p, cli.ExportFormat(format), time.Now())
			if err != nil {
				return err
			}

			if output != "" {
				return os.WriteFile(output, []byte(rendered), 0o600) //nolint:gosec // G703: output path comes from the user's own CLI flag
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), rendered)
			return err
		},
	}

	cmd.Flags().StringVar(&graphPath, "graph", "", "path to a previously scanned graph.json (required)")
	cmd.Flags().StringVar(&policyPath, "policy", "", "path to policy.json for scoping and violation highlighting (optional)")
	cmd.Flags().StringVar(&format, "format", "mermaid", "output format: mermaid or dot")
	cmd.Flags().StringVar(&output, "output", "", "write the diagram to this path instead of stdout")
	if err := cmd.MarkFlagRequired("graph"); err != nil {
		panic(err) // programming error: "graph" flag was just registered above
	}

	return cmd
}
