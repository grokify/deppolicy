package githubfleet

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grokify/deppolicy/component"
	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
	"github.com/grokify/deppolicy/scanner/golang"
)

// RepoManifestResult is one repository's successfully fetched and parsed
// go.mod.
type RepoManifestResult struct {
	Owner         string
	Repo          string
	ModuleLocator string
	Dependencies  []graph.Dependency
}

// ExcludedRepo records a repository that was not fetched at all, and why.
type ExcludedRepo struct {
	Owner, Repo string
	Reason      string
}

// RepoManifestError records a repository whose go.mod could not be fetched
// or parsed. One repository's failure must not abort the rest of the
// fleet, matching DiscoverModules' local-scan behavior.
type RepoManifestError struct {
	Owner, Repo string
	Err         error
}

// FleetReport is the outcome of scanning one GitHub org's repositories for
// manifest evidence, mirroring DiscoverModules' Included/Excluded/Errored
// shape so the two scan modes stay conceptually consistent.
type FleetReport struct {
	Included []RepoManifestResult
	Excluded []ExcludedRepo
	Errored  []RepoManifestError
}

// ScanOrgManifests lists every repository in org and fetches go.mod (if
// present) via the GitHub API contents endpoint -- no cloning.
//
// Archived repositories are excluded before any content fetch: per
// SPEC.md, GitHub's archived flag maps a repository to ungoverned, and
// skipping the fetch entirely (rather than fetching and marking it
// ungoverned afterward) also saves an API call per archived repo across a
// large fleet. A repository with no go.mod is not a Go module and is
// likewise excluded, not treated as an error.
//
// scope, if non-nil, additionally excludes a repository once its module
// locator is known (parsed from its go.mod) and falls outside
// scope.InScope -- e.g. a scope.exclude naming-convention pattern such as
// "sandbox-*". This can only happen after fetching, unlike the archived
// check: the module's locator isn't knowable from the repository name
// alone (a go.mod's declared module path need not match it), exactly as
// local scanning's DiscoverModules also fetches before classifying.
func ScanOrgManifests(ctx context.Context, client Client, org string, scope *policy.Policy) (FleetReport, error) {
	repos, err := client.ListOrgRepos(ctx, org)
	if err != nil {
		return FleetReport{}, fmt.Errorf("githubfleet: list repos for %s: %w", org, err)
	}

	var report FleetReport
	for _, r := range repos {
		if r.Archived {
			report.Excluded = append(report.Excluded, ExcludedRepo{Owner: org, Repo: r.Name, Reason: "archived"})
			continue
		}

		exists, err := client.FileExists(ctx, org, r.Name, "go.mod", nil)
		if err != nil {
			report.Errored = append(report.Errored, RepoManifestError{Owner: org, Repo: r.Name, Err: fmt.Errorf("githubfleet: check go.mod for %s/%s: %w", org, r.Name, err)})
			continue
		}
		if !exists {
			report.Excluded = append(report.Excluded, ExcludedRepo{Owner: org, Repo: r.Name, Reason: "no go.mod"})
			continue
		}

		data, err := client.GetFileContent(ctx, org, r.Name, "go.mod", nil)
		if err != nil {
			report.Errored = append(report.Errored, RepoManifestError{Owner: org, Repo: r.Name, Err: fmt.Errorf("githubfleet: fetch go.mod for %s/%s: %w", org, r.Name, err)})
			continue
		}

		moduleLocator, deps, err := golang.ParseManifest(org+"/"+r.Name+"/go.mod", data)
		if err != nil {
			report.Errored = append(report.Errored, RepoManifestError{Owner: org, Repo: r.Name, Err: err})
			continue
		}

		if scope != nil && !scope.InScope(moduleLocator) {
			report.Excluded = append(report.Excluded, ExcludedRepo{Owner: org, Repo: r.Name, Reason: "out of policy scope"})
			continue
		}

		report.Included = append(report.Included, RepoManifestResult{
			Owner:         org,
			Repo:          r.Name,
			ModuleLocator: moduleLocator,
			Dependencies:  deps,
		})
	}

	return report, nil
}

// ScanOrgsManifests scans multiple GitHub orgs via ScanOrgManifests and
// assembles one combined graph.Graph with Scan.Level = ScanLevelManifest,
// so callers never mistake this declared-dependency view -- one API call
// per repo, no cloning, no repository-local source parsing -- for the
// higher-assurance source-imports level local scanning produces.
//
// generated is passed explicitly rather than read from the wall clock, for
// the same reproducibility reason as golang.ScanRoots.
//
// The returned error is non-nil only when listing an org's repositories
// failed outright. A repository that individually failed to fetch or parse
// does not cause this error; it is folded into a single wrapped error
// alongside a successfully assembled graph, mirroring golang.ScanRoots.
func ScanOrgsManifests(ctx context.Context, client Client, orgs []string, generated time.Time, scannerID string, scope *policy.Policy) (graph.Graph, []FleetReport, error) {
	reports := make([]FleetReport, len(orgs))
	var deps []graph.Dependency
	var components []graph.ComponentObservation
	var errs []error

	for i, org := range orgs {
		report, err := ScanOrgManifests(ctx, client, org, scope)
		if err != nil {
			return graph.Graph{}, reports, err
		}
		reports[i] = report

		for _, r := range report.Included {
			components = append(components, graph.ComponentObservation{ID: r.ModuleLocator, Kind: component.KindGoModule})
			deps = append(deps, r.Dependencies...)
		}
		for _, e := range report.Errored {
			errs = append(errs, e.Err)
		}
	}

	var excluded []string
	for _, r := range reports {
		for _, e := range r.Excluded {
			excluded = append(excluded, fmt.Sprintf("%s/%s: %s", e.Owner, e.Repo, e.Reason))
		}
	}

	g := graph.Graph{
		Generated: generated,
		Scan: graph.ScanMetadata{
			Level:    graph.ScanLevelManifest,
			Scanner:  scannerID,
			Coverage: orgs,
			Excluded: excluded,
		},
		Components:   components,
		Dependencies: graph.MergeDependencies(deps),
	}

	if err := g.Validate(); err != nil {
		return graph.Graph{}, reports, err
	}

	if len(errs) > 0 {
		return g, reports, fmt.Errorf("githubfleet: %d repo(s) failed to fetch or parse: %w", len(errs), errors.Join(errs...))
	}

	return g, reports, nil
}
