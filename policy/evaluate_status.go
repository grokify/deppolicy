package policy

import (
	"fmt"
	"time"

	"github.com/grokify/deppolicy/component"
	"github.com/grokify/deppolicy/graph"
)

// effectiveStatus returns locator's status as declared by p, defaulting to
// governed when undeclared (see component.Component.EffectiveStatus).
func (p Policy) effectiveStatus(locator string) component.Status {
	if decl, ok := p.ComponentDeclaration(locator); ok && decl.Status != "" {
		return decl.Status
	}
	return component.StatusGoverned
}

// tierDenies reports whether the source component's tier prohibits a
// dependency on target via a MayNotDependOn pattern. Unlike tierAllows,
// only the source needs a declared tier -- the target is matched as a raw
// locator pattern, since deny rules exist chiefly to prohibit specific
// out-of-scope (third-party) dependencies that have no declaration.
func (p Policy) tierDenies(source, target string) (reason string, denied bool) {
	srcDecl, ok := p.ComponentDeclaration(source)
	if !ok || srcDecl.Tier == "" {
		return "", false
	}
	rule, ok := p.Tiers[srcDecl.Tier]
	if !ok {
		return "", false
	}
	for _, pattern := range rule.MayNotDependOn {
		if matchLocatorPattern(pattern, target) {
			return fmt.Sprintf("tier deny: %s may not depend on %s (pattern %s)", srcDecl.Tier, target, pattern), true
		}
	}
	return "", false
}

// tierAllows reports whether a tier rule authorizes source -> target: both
// components must carry a declared tier, and the source tier's MayDependOn
// must name the target tier directly. There is deliberately no
// transitivity and no self-tier match here -- Validate rejects self
// references, and a rule chain never composes (see TierRule).
func (p Policy) tierAllows(source, target string) (reason string, ok bool) {
	srcDecl, ok := p.ComponentDeclaration(source)
	if !ok || srcDecl.Tier == "" {
		return "", false
	}
	tgtDecl, ok := p.ComponentDeclaration(target)
	if !ok || tgtDecl.Tier == "" {
		return "", false
	}
	rule, ok := p.Tiers[srcDecl.Tier]
	if !ok {
		return "", false
	}
	for _, t := range rule.MayDependOn {
		if t == tgtDecl.Tier {
			return fmt.Sprintf("tier rule: %s may depend on %s", srcDecl.Tier, tgtDecl.Tier), true
		}
	}
	return "", false
}

// EvaluateWithStatus extends Evaluate with the component-status semantics
// from SPEC.md section 1.2:
//
//   - Foundation target: any governed source may depend on it without an
//     explicit relationship or exception.
//   - Foundation source: it may depend only on other foundation
//     components. This is an invariant, not a request -- an authored
//     relationship or exception that would violate it does not authorize
//     the edge.
//   - Deprecated target: an edge that already existed in baseline is
//     evaluated under the normal rules above, unaffected by deprecation
//     (deprecation does not revoke existing permission). An edge that is
//     newly observed (absent from baseline) is always a violation, even
//     if it is otherwise authorized (deprecation means no new dependents).
//
// baseline, if non-nil, is what distinguishes a deprecated target's
// pre-existing dependents from newly introduced ones; with baseline == nil,
// every inbound edge to a deprecated component is treated as new. asOf is
// passed explicitly for the same reason as in Evaluate: to keep evaluation
// a pure function of its arguments.
func (p Policy) EvaluateWithStatus(g graph.Graph, asOf time.Time, baseline *graph.Graph) []Finding {
	observed := observedEdges(g)
	authorized := authorizedEdges(p, asOf)
	edges := unionEdgeKeys(observed, authorized)

	var baselineObserved map[edgeKey]bool
	if baseline != nil {
		baselineObserved = observedEdges(*baseline)
	}

	findings := make([]Finding, 0, len(edges))
	for k := range edges {
		if !p.InScope(k.source) {
			continue // external source: not governed, not evaluated
		}

		// Tier denies run before everything else, and are the only rule
		// evaluated for out-of-scope targets: they exist precisely to
		// prohibit specific third-party dependencies (e.g. provider SDKs
		// in an abstraction core) that scope-based governance
		// deliberately ignores. A deny overrides every allow; the one
		// escape valve is an unexpired exception for the exact edge.
		if denyReason, denied := p.tierDenies(k.source, k.target); denied {
			if !observed[k] {
				continue // a denied edge that doesn't exist needs no finding
			}
			if e, ok := p.ActiveException(k.source, k.target, asOf); ok {
				findings = append(findings, Finding{
					Source: k.source, Target: k.target, Status: FindingConforming,
					Reason: "exception overriding tier deny: " + e.Reason,
				})
			} else {
				findings = append(findings, Finding{
					Source: k.source, Target: k.target, Status: FindingViolation,
					Reason: denyReason,
				})
			}
			continue
		}

		if !p.InScope(k.target) {
			continue // external target, not denied: not governed
		}

		isObserved := observed[k]
		reason, hasExplicitAuth := authorized[k]

		statusSource := p.effectiveStatus(k.source)
		statusTarget := p.effectiveStatus(k.target)

		invariantViolated := statusSource == component.StatusFoundation && statusTarget != component.StatusFoundation
		isNewToDeprecated := statusTarget == component.StatusDeprecated && !baselineObserved[k]

		switch {
		case isObserved && invariantViolated:
			findings = append(findings, Finding{
				Source: k.source, Target: k.target, Status: FindingViolation,
				Reason: "foundation component may only depend on other foundation components",
			})

		case isObserved && isNewToDeprecated:
			findings = append(findings, Finding{
				Source: k.source, Target: k.target, Status: FindingViolation,
				Reason: "deprecated component: no new dependents",
			})

		default:
			authorizedNow := hasExplicitAuth && !invariantViolated
			if !authorizedNow && !invariantViolated && statusTarget == component.StatusFoundation {
				authorizedNow = true
				if reason == "" {
					reason = "foundation component"
				}
			}
			if !authorizedNow && !invariantViolated {
				if tierReason, ok := p.tierAllows(k.source, k.target); ok {
					authorizedNow = true
					reason = tierReason
				}
			}

			switch {
			case isObserved && authorizedNow:
				findings = append(findings, Finding{Source: k.source, Target: k.target, Status: FindingConforming, Reason: reason})
			case isObserved && !authorizedNow:
				findings = append(findings, Finding{Source: k.source, Target: k.target, Status: FindingViolation, Reason: "no allow rule; default-deny"})
			case !isObserved && authorizedNow:
				findings = append(findings, Finding{Source: k.source, Target: k.target, Status: FindingUnusedAllowance, Reason: reason})
			}
		}
	}

	return sortFindings(findings)
}
