package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
	"github.com/grokify/deppolicy/scanner/golang/githubfleet"
)

// ScanGitHubOptions configures ScanGitHub.
type ScanGitHubOptions struct {
	// Orgs are the GitHub organizations to scan. Required, at least one.
	Orgs []string

	// PolicyPath, if set, is used for scope-based exclusion, exactly as
	// it is for a local Scan. Omit to scan in bootstrap mode.
	PolicyPath string

	// Scanner identifies this scanner build, recorded in the resulting
	// graph's scan metadata.
	Scanner string

	// AsOf is recorded as the graph's Generated timestamp.
	AsOf time.Time
}

// ScanGitHubResult is ScanGitHub's structured output.
type ScanGitHubResult struct {
	// Policy is non-nil only when opts.PolicyPath was set.
	Policy *policy.Policy
	Graph  graph.Graph

	// Reports holds one FleetReport per org, in the same order as
	// opts.Orgs.
	Reports []githubfleet.FleetReport

	// ScanWarning is non-nil when some repositories failed to fetch or
	// parse; Graph still reflects whatever did scan successfully.
	ScanWarning error
}

// ScanGitHub scans opts.Orgs via the GitHub API's contents endpoint (no
// cloning), optionally scoped by a policy document.
//
// client is injected rather than constructed here, so this function stays
// testable with a fake -- constructing a real client from a token, which
// needs network/oauth2 setup, belongs in cmd/deppolicy where untestable
// wiring concerns live, not in this package.
func ScanGitHub(ctx context.Context, client githubfleet.Client, opts ScanGitHubOptions) (ScanGitHubResult, error) {
	var p *policy.Policy
	if opts.PolicyPath != "" {
		data, err := os.ReadFile(opts.PolicyPath)
		if err != nil {
			return ScanGitHubResult{}, fmt.Errorf("cli: read policy %s: %w", opts.PolicyPath, err)
		}
		parsed, err := policy.Parse(data)
		if err != nil {
			return ScanGitHubResult{}, fmt.Errorf("cli: parse policy %s: %w", opts.PolicyPath, err)
		}
		p = parsed
	}

	g, reports, scanErr := githubfleet.ScanOrgsManifests(ctx, client, opts.Orgs, opts.AsOf, opts.Scanner, p)
	if scanErr != nil && g.Scan.Scanner == "" {
		return ScanGitHubResult{}, fmt.Errorf("cli: scan github orgs %v: %w", opts.Orgs, scanErr)
	}

	return ScanGitHubResult{Policy: p, Graph: g, Reports: reports, ScanWarning: scanErr}, nil
}
