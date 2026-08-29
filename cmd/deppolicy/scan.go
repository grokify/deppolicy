package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/grokify/deppolicy/cli"
)

func newScanCmd() *cobra.Command {
	var (
		policyPath string
		output     string
		workers    int
	)

	cmd := &cobra.Command{
		Use:   "scan <dir>...",
		Short: "Scan one or more directory trees for Go modules and their direct dependencies",
		Long: `Scan one or more directory trees for Go modules and their direct
dependencies, producing a graph.json describing what actually exists.

Multiple directories are combined into one graph with one shared component
registry, which is what lets a deep import in a module under one directory
resolve to its owning module under another -- e.g. scanning several org
roots together so a cross-org import attributes correctly instead of being
recorded as an unresolved raw import path.

--policy is optional: without it, scan runs in bootstrap mode -- nothing
is excluded, since no scope has been declared yet. This is the mode used
to discover an ecosystem's current shape before a policy exists to
author against. With --policy, scope.exclude patterns and declared
components apply exactly as they do during check/diff.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := cli.Scan(cmd.Context(), cli.ScanOptions{
				RootDirs:   args,
				PolicyPath: policyPath,
				Scanner:    scannerVersion,
				Workers:    workers,
				AsOf:       time.Now(),
			})
			if err != nil {
				return err
			}

			if _, err := fmt.Fprint(cmd.OutOrStdout(), cli.FormatScanReport(result)); err != nil {
				return err
			}

			if output != "" {
				if err := result.Graph.WriteFile(output); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&policyPath, "policy", "", "path to policy.json for scope-based exclusion (optional)")
	cmd.Flags().StringVar(&output, "output", "", "write the scanned graph as JSON to this path")
	cmd.Flags().IntVar(&workers, "workers", 0, "scan concurrency (0 = GOMAXPROCS)")

	return cmd
}
