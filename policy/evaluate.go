package policy

import (
	"sort"
	"time"

	"github.com/grokify/deppolicy/graph"
)

// FindingStatus classifies one edge under evaluation. See SPEC.md section 5
// for the four-state model this implements.
type FindingStatus string

const (
	// FindingConforming: the edge is observed in the graph and authorized
	// by policy.
	FindingConforming FindingStatus = "conforming"

	// FindingViolation: the edge is observed in the graph but not
	// authorized by policy -- a default-deny violation.
	FindingViolation FindingStatus = "violation"

	// FindingUnusedAllowance: the edge is authorized by policy but not
	// observed in the graph -- architecture debt of the opposite kind
	// from a violation, worth pruning from the policy.
	FindingUnusedAllowance FindingStatus = "unused_allowance"
)

// Finding is one row of evaluation output: the comparison, for one edge, of
// what policy authorizes against what the graph observes. An edge that is
// neither observed nor authorized produces no Finding.
type Finding struct {
	Source string        `json:"source"`
	Target string        `json:"target"`
	Status FindingStatus `json:"status"`
	Reason string        `json:"reason"`
}

// edgeKey identifies one directed dependency edge for map-based lookup.
type edgeKey struct{ source, target string }

// observedEdges indexes g's depends_on edges by (source, target).
func observedEdges(g graph.Graph) map[edgeKey]bool {
	observed := make(map[edgeKey]bool, len(g.Dependencies))
	for _, d := range g.Dependencies {
		if d.Relation != graph.RelationDependsOn {
			continue
		}
		observed[edgeKey{d.Source, d.Target}] = true
	}
	return observed
}

// authorizedEdges indexes p's explicit relationships and unexpired
// exceptions by (source, target), mapping each to a human-readable reason.
func authorizedEdges(p Policy, asOf time.Time) map[edgeKey]string {
	authorized := make(map[edgeKey]string, len(p.Relationships)+len(p.Exceptions))
	for _, r := range p.Relationships {
		if r.Relation != RelationCanDependOn {
			continue
		}
		authorized[edgeKey{r.Source, r.Target}] = "explicit relationship"
	}
	for _, e := range p.Exceptions {
		if e.IsExpired(asOf) {
			continue
		}
		authorized[edgeKey{e.Source, e.Target}] = "exception: " + e.Reason
	}
	return authorized
}

// unionEdgeKeys returns every key present in either map.
func unionEdgeKeys(a map[edgeKey]bool, b map[edgeKey]string) map[edgeKey]bool {
	edges := make(map[edgeKey]bool, len(a)+len(b))
	for k := range a {
		edges[k] = true
	}
	for k := range b {
		edges[k] = true
	}
	return edges
}

// sortFindings orders findings by (source, target, status) for
// deterministic output.
func sortFindings(findings []Finding) []Finding {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Source != findings[j].Source {
			return findings[i].Source < findings[j].Source
		}
		if findings[i].Target != findings[j].Target {
			return findings[i].Target < findings[j].Target
		}
		return findings[i].Status < findings[j].Status
	})
	return findings
}

// Evaluate compares p against g and returns one Finding per edge that is
// either observed, authorized, or both, sorted by (source, target, status)
// for deterministic output.
//
// asOf is the evaluation time used to resolve exception expiry; it is
// passed explicitly, not read from the clock, so Evaluate remains a pure
// function of its arguments -- the same (p, g, asOf) always produces the
// same findings.
//
// Evaluate implements only the base scope and default-deny rules (SPEC.md
// sections 2 and 3): explicit relationships and unexpired exceptions
// authorize an edge; nothing else does. Foundation-tier auto-authorization
// and deprecated no-new-dependents semantics are layered on top of this by
// EvaluateWithStatus, not inside Evaluate.
func (p Policy) Evaluate(g graph.Graph, asOf time.Time) []Finding {
	observed := observedEdges(g)
	authorized := authorizedEdges(p, asOf)
	edges := unionEdgeKeys(observed, authorized)

	findings := make([]Finding, 0, len(edges))
	for k := range edges {
		if !p.InScope(k.source) || !p.InScope(k.target) {
			continue // external: not governed, not evaluated
		}

		isObserved := observed[k]
		reason, isAuthorized := authorized[k]

		switch {
		case isObserved && isAuthorized:
			findings = append(findings, Finding{Source: k.source, Target: k.target, Status: FindingConforming, Reason: reason})
		case isObserved && !isAuthorized:
			findings = append(findings, Finding{Source: k.source, Target: k.target, Status: FindingViolation, Reason: "no allow rule; default-deny"})
		case !isObserved && isAuthorized:
			findings = append(findings, Finding{Source: k.source, Target: k.target, Status: FindingUnusedAllowance, Reason: reason})
		}
	}

	return sortFindings(findings)
}
