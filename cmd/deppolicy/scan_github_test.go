package main

import "testing"

// TestScanGitHubCmd_RequiresToken exercises only the pre-network-call
// error path: a real successful run needs a live GitHub token and network
// access, which cli.ScanGitHub's own tests already cover via a fake
// client (see cli/scan_github_test.go) -- there is no value in duplicating
// that here against a real API.
func TestScanGitHubCmd_RequiresToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")

	cmd := newRootCmd()
	cmd.SetArgs([]string{"scan-github", "someorg"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error when no token is available")
	}
}

func TestScanGitHubCmd_RequiresAtLeastOneOrg(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"scan-github"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error when no organization is given")
	}
}
