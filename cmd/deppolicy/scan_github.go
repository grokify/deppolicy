package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/grokify/gogithub/clientv1"

	"github.com/grokify/deppolicy/cli"
)

func newScanGitHubCmd() *cobra.Command {
	var (
		policyPath string
		output     string
		token      string
	)

	cmd := &cobra.Command{
		Use:   "scan-github <org>...",
		Short: "Scan one or more GitHub organizations for go.mod dependencies via the GitHub API (no cloning)",
		Long: `Scan one or more GitHub organizations' repositories for go.mod
dependencies via the GitHub contents API -- no cloning required.

This is manifest mode: it reads only go.mod, not source imports, so it is
much faster than a local source scan across a large fleet, at the cost of
lower assurance (see SPEC.md's evidence-provenance model). Archived
repositories are skipped without a fetch.

Requires a GitHub token via --token or the GITHUB_TOKEN environment
variable.

--policy is optional, exactly as with the local scan command: without it,
scan-github runs in bootstrap mode.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				token = os.Getenv("GITHUB_TOKEN")
			}
			if token == "" {
				return fmt.Errorf("a GitHub token is required: pass --token or set GITHUB_TOKEN")
			}

			client, err := clientv1.NewClient(cmd.Context(), token)
			if err != nil {
				return fmt.Errorf("create GitHub client: %w", err)
			}

			result, err := cli.ScanGitHub(cmd.Context(), client, cli.ScanGitHubOptions{
				Orgs:       args,
				PolicyPath: policyPath,
				Scanner:    scannerVersion,
				AsOf:       time.Now(),
			})
			if err != nil {
				return err
			}

			if _, err := fmt.Fprint(cmd.OutOrStdout(), cli.FormatScanGitHubReport(result)); err != nil {
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
	cmd.Flags().StringVar(&token, "token", "", "GitHub API token (defaults to GITHUB_TOKEN env var)")

	return cmd
}
