package graph

// MergeDependencies collapses dependencies that share the same
// (source, relation, target) into a single edge, concatenating their
// evidence. This is what lets several packages within one component that
// all import another component collapse into one architecture-level edge
// while preserving every underlying import as evidence.
//
// Order is deterministic: edges appear in order of first occurrence, and
// evidence within an edge appears in input order.
func MergeDependencies(deps []Dependency) []Dependency {
	type key struct {
		source, target string
		relation       Relation
	}

	order := make([]key, 0, len(deps))
	byKey := make(map[key]*Dependency, len(deps))

	for _, d := range deps {
		k := key{source: d.Source, relation: d.Relation, target: d.Target}
		existing, ok := byKey[k]
		if !ok {
			merged := Dependency{Source: d.Source, Relation: d.Relation, Target: d.Target}
			byKey[k] = &merged
			order = append(order, k)
			existing = &merged
		}
		existing.Evidence = append(existing.Evidence, d.Evidence...)
	}

	out := make([]Dependency, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	return out
}
