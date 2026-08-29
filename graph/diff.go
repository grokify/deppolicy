package graph

// Diff is the structural comparison of two graphs' dependency edges,
// classified purely by (source, target) presence -- differing evidence or
// relation on an edge that exists in both graphs does not affect
// classification.
type Diff struct {
	// Added holds edges present in current but not in baseline.
	Added []Dependency
	// Removed holds edges present in baseline but not in current.
	Removed []Dependency
	// Unchanged holds edges present in both graphs (as they appear in
	// current).
	Unchanged []Dependency
}

// DiffGraphs compares baseline against current. This is what makes ratchet
// CI possible: "this dependency did not exist on main and has no
// architecture policy" is a graph-level fact, independent of policy
// evaluation -- see the cli package's Diff, which composes this with
// Policy.EvaluateWithStatus to classify each violation as new or existing.
func DiffGraphs(baseline, current Graph) Diff {
	baselineSet := edgeKeySet(baseline)
	currentSet := edgeKeySet(current)

	var d Diff
	for _, dep := range current.Dependencies {
		if baselineSet[edgeKeyOf(dep)] {
			d.Unchanged = append(d.Unchanged, dep)
		} else {
			d.Added = append(d.Added, dep)
		}
	}
	for _, dep := range baseline.Dependencies {
		if !currentSet[edgeKeyOf(dep)] {
			d.Removed = append(d.Removed, dep)
		}
	}

	return d
}

type diffEdgeKey struct{ source, target string }

func edgeKeyOf(d Dependency) diffEdgeKey {
	return diffEdgeKey{d.Source, d.Target}
}

func edgeKeySet(g Graph) map[diffEdgeKey]bool {
	set := make(map[diffEdgeKey]bool, len(g.Dependencies))
	for _, d := range g.Dependencies {
		set[edgeKeyOf(d)] = true
	}
	return set
}
