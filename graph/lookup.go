package graph

// HasEdge reports whether the graph records an observed direct dependency
// from source to target.
func (g Graph) HasEdge(source, target string) bool {
	for _, d := range g.Dependencies {
		if d.Relation == RelationDependsOn && d.Source == source && d.Target == target {
			return true
		}
	}
	return false
}
