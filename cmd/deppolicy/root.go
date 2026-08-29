package main

import "github.com/spf13/cobra"

// scannerVersion identifies this scanner build in scan metadata. It has no
// release process yet, so it is a static placeholder until tagging exists.
const scannerVersion = "deppolicy-go (dev)"

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deppolicy",
		Short: "Statically analyzable dependency policy for the module ecosystem",
	}
	cmd.AddCommand(newScanCmd())
	cmd.AddCommand(newScanGitHubCmd())
	cmd.AddCommand(newCheckCmd())
	cmd.AddCommand(newDiffCmd())
	cmd.AddCommand(newPolicyDiffCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newPromoteCmd())
	cmd.AddCommand(newReportCmd())
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newTablesCmd())
	cmd.AddCommand(newUICmd())
	return cmd
}
