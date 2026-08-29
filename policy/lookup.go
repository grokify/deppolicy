package policy

import "time"

// HasRelationship reports whether the policy explicitly authorizes
// "source can_depend_on target" via an authored Relationship. It does not
// account for Scope, foundation status, or Exceptions — those compose in
// the evaluator, which makes the full authorization decision for an
// observed edge.
func (p Policy) HasRelationship(source, target string) bool {
	for _, r := range p.Relationships {
		if r.Relation == RelationCanDependOn && r.Source == source && r.Target == target {
			return true
		}
	}
	return false
}

// ActiveException returns the exception authorizing source -> target as of
// asOf, if one exists and has not expired by that time.
func (p Policy) ActiveException(source, target string, asOf time.Time) (Exception, bool) {
	for _, e := range p.Exceptions {
		if e.Source == source && e.Target == target && !e.IsExpired(asOf) {
			return e, true
		}
	}
	return Exception{}, false
}

// ComponentDeclaration returns the authored declaration for locator, if the
// policy declares one. A locator with no declaration defaults to governed
// status.
func (p Policy) ComponentDeclaration(locator string) (ComponentDeclaration, bool) {
	decl, ok := p.Components[locator]
	return decl, ok
}
