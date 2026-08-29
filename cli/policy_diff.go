package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

// RemovedRelationship is one relationship present in the old policy but
// absent from the new one, annotated with whether the edge it authorized
// is still observed in a supplied graph -- a "breaking" removal, since the
// next fleet scan will report that edge as a violation.
type RemovedRelationship struct {
	policy.Relationship
	StillObserved bool `json:"stillObserved"`
}

// ComponentChange records a component declaration whose status or tier
// differs between the two policies (including appearing/disappearing).
type ComponentChange struct {
	Locator   string `json:"locator"`
	OldStatus string `json:"oldStatus,omitempty"`
	NewStatus string `json:"newStatus,omitempty"`
	OldTier   string `json:"oldTier,omitempty"`
	NewTier   string `json:"newTier,omitempty"`
}

// PolicyDiffOptions configures PolicyDiff.
type PolicyDiffOptions struct {
	// OldPolicyPath and NewPolicyPath are the two policy documents to
	// compare -- typically the base branch's and the PR's. Required.
	OldPolicyPath string
	NewPolicyPath string

	// GraphPath, if set, is a previously scanned graph.json used to
	// classify removed relationships as breaking (edge still observed)
	// or clean (edge no longer exists in code). Without it, removals are
	// reported but StillObserved is always false.
	GraphPath string
}

// PolicyDiffResult is PolicyDiff's structured output.
type PolicyDiffResult struct {
	Added            []policy.Relationship `json:"added,omitempty"`
	Removed          []RemovedRelationship `json:"removed,omitempty"`
	ComponentChanges []ComponentChange     `json:"componentChanges,omitempty"`
}

// BreakingRemovals reports whether any removed relationship's edge is
// still observed.
func (r PolicyDiffResult) BreakingRemovals() bool {
	for _, rm := range r.Removed {
		if rm.StillObserved {
			return true
		}
	}
	return false
}

// PolicyDiff compares two policy documents as an architecture change
// review: a relationship added is a newly permitted dependency, a
// relationship removed is a revoked one, and a component status/tier
// change alters what the rules mean for every edge touching it.
//
// This exists because of a deliberate operational decision (SPEC-adjacent,
// recorded in the PRD): a policy removal must not break unrelated repos'
// CI runs the moment the policy repo merges -- instead, the removal is
// surfaced here, at policy-review time, as "still observed" when a graph
// shows code that depends on the permission being revoked. The reviewer
// decides whether to fix the code first or accept that the next fleet
// scan will report those edges as violations.
func PolicyDiff(opts PolicyDiffOptions) (PolicyDiffResult, error) {
	oldPolicy, err := policy.ParseFile(opts.OldPolicyPath)
	if err != nil {
		return PolicyDiffResult{}, fmt.Errorf("cli: old policy: %w", err)
	}
	newPolicy, err := policy.ParseFile(opts.NewPolicyPath)
	if err != nil {
		return PolicyDiffResult{}, fmt.Errorf("cli: new policy: %w", err)
	}

	var g *graph.Graph
	if opts.GraphPath != "" {
		parsed, err := graphFromFile(opts.GraphPath)
		if err != nil {
			return PolicyDiffResult{}, err
		}
		g = parsed
	}

	type edgeKey struct{ source, target string }
	oldRels := make(map[edgeKey]policy.Relationship, len(oldPolicy.Relationships))
	for _, r := range oldPolicy.Relationships {
		oldRels[edgeKey{r.Source, r.Target}] = r
	}
	newRels := make(map[edgeKey]policy.Relationship, len(newPolicy.Relationships))
	for _, r := range newPolicy.Relationships {
		newRels[edgeKey{r.Source, r.Target}] = r
	}

	var result PolicyDiffResult
	for k, r := range newRels {
		if _, ok := oldRels[k]; !ok {
			result.Added = append(result.Added, r)
		}
	}
	for k, r := range oldRels {
		if _, ok := newRels[k]; !ok {
			removed := RemovedRelationship{Relationship: r}
			if g != nil {
				removed.StillObserved = g.HasEdge(r.Source, r.Target)
			}
			result.Removed = append(result.Removed, removed)
		}
	}

	locators := make(map[string]bool, len(oldPolicy.Components)+len(newPolicy.Components))
	for l := range oldPolicy.Components {
		locators[l] = true
	}
	for l := range newPolicy.Components {
		locators[l] = true
	}
	for l := range locators {
		oldDecl := oldPolicy.Components[l]
		newDecl := newPolicy.Components[l]
		if oldDecl.Status != newDecl.Status || oldDecl.Tier != newDecl.Tier {
			result.ComponentChanges = append(result.ComponentChanges, ComponentChange{
				Locator:   l,
				OldStatus: string(oldDecl.Status),
				NewStatus: string(newDecl.Status),
				OldTier:   oldDecl.Tier,
				NewTier:   newDecl.Tier,
			})
		}
	}

	sort.Slice(result.Added, func(i, j int) bool {
		if result.Added[i].Source != result.Added[j].Source {
			return result.Added[i].Source < result.Added[j].Source
		}
		return result.Added[i].Target < result.Added[j].Target
	})
	sort.Slice(result.Removed, func(i, j int) bool {
		if result.Removed[i].Source != result.Removed[j].Source {
			return result.Removed[i].Source < result.Removed[j].Source
		}
		return result.Removed[i].Target < result.Removed[j].Target
	})
	sort.Slice(result.ComponentChanges, func(i, j int) bool {
		return result.ComponentChanges[i].Locator < result.ComponentChanges[j].Locator
	})

	return result, nil
}

// graphFromFile reads and parses a graph document from path.
func graphFromFile(path string) (*graph.Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cli: read graph %s: %w", path, err)
	}
	g, err := graph.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("cli: parse graph %s: %w", path, err)
	}
	return g, nil
}
