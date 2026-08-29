package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
	"github.com/grokify/deppolicy/tabular"
)

func newTablesCmd() *cobra.Command {
	var (
		policyPath string
		graphPath  string
		format     string
		outputDir  string
	)

	cmd := &cobra.Command{
		Use:   "tables",
		Short: "Flatten policy, graph, and findings into queryable tables (Policy vs Actual vs Difference)",
		Long: `Flatten the three policy artifacts into four flat tables for analytics
platforms such as DashForge, spreadsheets, or notebooks:

  components      every declared or observed component with status/tier
  policy_edges    explicit relationships and exceptions (intent)
  observed_edges  every observed dependency with evidence summary (actual)
  findings        the evaluated comparison (difference)

Findings are computed here with the same evaluator as "check" (no
baseline). Without --output-dir, all tables print to stdout as one JSON
document; with it, one file per table is written in the chosen format.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := policy.ParseFile(policyPath)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(graphPath)
			if err != nil {
				return fmt.Errorf("read graph %s: %w", graphPath, err)
			}
			g, err := graph.Parse(data)
			if err != nil {
				return err
			}

			findings := p.EvaluateWithStatus(*g, time.Now(), nil)
			builtTables := tabular.Build(*p, *g, findings)

			if outputDir == "" {
				return tabular.WriteJSON(builtTables, cmd.OutOrStdout())
			}

			if err := os.MkdirAll(outputDir, 0o750); err != nil {
				return fmt.Errorf("create output dir %s: %w", outputDir, err)
			}
			for _, t := range builtTables {
				path := filepath.Join(outputDir, t.Name+"."+format)
				f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
				if err != nil {
					return fmt.Errorf("create %s: %w", path, err)
				}
				switch format {
				case "csv":
					err = tabular.WriteCSV(t, f)
				case "json":
					err = tabular.WriteJSON([]tabular.Table{t}, f)
				default:
					err = fmt.Errorf("unknown format %q (want csv or json)", format)
				}
				if cerr := f.Close(); cerr != nil && err == nil {
					err = cerr
				}
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&policyPath, "policy", "", "path to policy.json (required)")
	cmd.Flags().StringVar(&graphPath, "graph", "", "path to a previously scanned graph.json (required)")
	cmd.Flags().StringVar(&format, "format", "csv", "per-table file format when --output-dir is set: csv or json")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "write one file per table into this directory instead of JSON to stdout")
	for _, name := range []string{"policy", "graph"} {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err) // programming error: flag was just registered above
		}
	}

	return cmd
}
