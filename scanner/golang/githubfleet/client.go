// Package githubfleet scans Go modules across GitHub organizations via the
// GitHub API's contents endpoint -- no cloning -- for fast, fleet-wide
// manifest-mode dependency evidence. It is a separate package from
// scanner/golang so that a local-only consumer (e.g. the check/diff/
// validate CLI commands) never has to pull in the GitHub API client
// dependency chain it doesn't need.
package githubfleet

import (
	"context"

	"github.com/grokify/gogithub"
)

// Client is the minimal interface githubfleet needs from
// github.com/grokify/gogithub/clientv1.Client, which has 60+ methods for
// operations this package never performs. Accepting this narrow interface
// at the point of use -- rather than the full clientv1.Client -- is what
// lets tests use a two-method fake instead of a real GitHub client.
// clientv1.Client already satisfies this interface structurally.
type Client interface {
	// ListOrgRepos lists every repository in org.
	ListOrgRepos(ctx context.Context, org string) ([]*gogithub.Repository, error)

	// FileExists reports whether path exists in owner/repo, without
	// erroring on a 404 -- used to distinguish "not a Go module" from a
	// real fetch failure before calling GetFileContent, since
	// GetFileContent's own 404 case is not distinguishable from other
	// errors via errors.Is/errors.As.
	FileExists(ctx context.Context, owner, repo, path string, opts *gogithub.ContentOptions) (bool, error)

	// GetFileContent fetches a file's content from a repository.
	GetFileContent(ctx context.Context, owner, repo, path string, opts *gogithub.ContentOptions) ([]byte, error)
}
