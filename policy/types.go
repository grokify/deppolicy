// Package policy defines the authored dependency policy document: the
// architectural intent for a governed ecosystem. Policy is data, not code —
// it is deliberately constrained to remain statically and deterministically
// evaluable (see the evaluator in a later package), with no expressions,
// callbacks, or runtime lookups.
package policy

import (
	"time"

	"github.com/grokify/deppolicy/component"
)

// Relation identifies the kind of permission a policy relationship grants.
// V1 supports exactly one relation; typed relations such as can_call or
// can_publish_to are reserved for a future version once a real use case
// requires them.
type Relation string

// RelationCanDependOn is the only relation V1 supports: it authorizes a
// direct dependency from Source on Target.
const RelationCanDependOn Relation = "can_depend_on"

// Effect identifies the kind of permission an exception grants. V1 supports
// only temporary allow exceptions to the default-deny rule; there is no
// "deny exception" because default-deny already denies unauthorized edges.
type Effect string

// EffectAllow is the only exception effect V1 supports.
const EffectAllow Effect = "allow"

// Scope defines the boundary of the governed ecosystem via locator patterns.
// A locator is in scope if it matches at least one Include pattern and no
// Exclude pattern. Dependencies with an out-of-scope source or target are
// not evaluated by policy (they are treated like third-party dependencies).
//
// Patterns support a single wildcard form: "*" matches any sequence of
// characters within one "/"-delimited segment (as in path.Match), and a
// pattern ending in "/*" additionally matches any locator nested arbitrarily
// deep beneath the prefix before "/*" — this is what lets
// "go:github.com/grokify/*" cover every component under that org, while
// "go:github.com/*/*-archive" matches only repos whose name ends in
// "-archive" without also matching their subpaths.
type Scope struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude,omitempty"`
}

// ComponentDeclaration is the policy-authored metadata for a component,
// keyed by its locator in Policy.Components. It assigns non-default status
// (foundation, deprecated, ungoverned), an optional tier (see
// Policy.Tiers), and records the rationale.
type ComponentDeclaration struct {
	Kind     component.Kind   `json:"kind,omitempty"`
	Status   component.Status `json:"status,omitempty"`
	Tier     string           `json:"tier,omitempty"`
	Reason   string           `json:"reason,omitempty"`
	Decision string           `json:"decision,omitempty"`
}

// TierRule declares which tiers a source tier's components may
// automatically depend on, without per-edge relationships. The lists are
// explicit and non-transitive by design: a rule chain vertical-app ->
// horizontal-forge -> omni-adapter does NOT let a vertical-app reach an
// omni-adapter directly -- skipping a layer is an architecture bypass and
// always requires an explicit Relationship. A tier may never list itself:
// same-tier dependencies (cross-forge, cross-omni) always require explicit
// wiring, which is what keeps peers from coupling by default.
type TierRule struct {
	MayDependOn []string `json:"mayDependOn,omitempty"`

	// MayNotDependOn holds locator patterns (Scope pattern syntax) whose
	// matches this tier's components are prohibited from depending on --
	// including out-of-scope targets, which is the whole point: it is the
	// only rule in the model that governs third-party dependencies, and
	// it exists for constraints like "an abstraction core must not import
	// provider SDKs" (e.g. omnillm-core must not import an Anthropic or
	// OpenAI SDK; that belongs in the omni-anthropic provider adapter).
	//
	// A deny overrides every allow -- explicit relationships and
	// mayDependOn rules included. The one escape valve is an unexpired
	// Exception for the exact edge, so a denied dependency that must
	// exist temporarily is visible, justified, and time-bounded.
	MayNotDependOn []string `json:"mayNotDependOn,omitempty"`
}

// Relationship is an authored, explicit direct-dependency authorization:
// Source can_depend_on Target. Reason is required -- every permanent
// architectural boundary crossing this permissive should be able to say
// why in one sentence; Decision optionally points to the ADR or other
// durable record backing it.
type Relationship struct {
	Source   string   `json:"source"`
	Relation Relation `json:"relation"`
	Target   string   `json:"target"`
	Reason   string   `json:"reason"`
	Decision string   `json:"decision,omitempty"`
}

// Exception is a centrally-declared, time-bounded allowance that overrides
// default-deny for one edge without adding a permanent Relationship. Unlike
// a Relationship, an Exception must carry an expiration and a reason, so it
// remains visible as temporary debt rather than a silent permanent edge.
type Exception struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Effect  Effect `json:"effect"`
	Expires string `json:"expires"` // date, layout "2006-01-02"
	Reason  string `json:"reason"`
}

// expiresLayout is the required layout for Exception.Expires.
const expiresLayout = "2006-01-02"

// ExpiresAt parses Expires using the required date layout.
func (e Exception) ExpiresAt() (time.Time, error) {
	return time.Parse(expiresLayout, e.Expires)
}

// IsExpired reports whether the exception has expired as of the given time.
// The evaluation time is passed explicitly rather than read from the clock
// so that evaluation remains a pure function of its inputs. An unparseable
// Expires value is treated as expired (fail closed).
func (e Exception) IsExpired(asOf time.Time) bool {
	expiresAt, err := e.ExpiresAt()
	if err != nil {
		return true
	}
	return !asOf.Before(expiresAt.AddDate(0, 0, 1))
}

// Policy is the authored, centrally-owned dependency policy document for a
// governed ecosystem. It is the "intent" side of the policy/graph/findings
// contract: it describes what dependency edges MAY exist, independent of
// what the code currently does.
type Policy struct {
	// SchemaVersion identifies the shape of this document. PolicyVersion
	// identifies its content; the two evolve independently.
	SchemaVersion string `json:"schemaVersion"`
	PolicyVersion string `json:"policyVersion,omitempty"`

	Scope Scope `json:"scope"`

	// Components declares non-default status/kind/tier metadata, keyed by
	// locator. A locator not present here defaults to governed status
	// with an unspecified kind and no tier.
	Components map[string]ComponentDeclaration `json:"components,omitempty"`

	// Tiers declares the archetype vocabulary and the per-tier
	// auto-allow rules, keyed by tier name. A component participates in
	// tier rules only if its declaration names one of these tiers.
	// Foundation status is deliberately orthogonal to tiers: there is no
	// blanket "-> library" rule shape, so making a library universally
	// consumable is always a deliberate per-component status change
	// (e.g. mogo, goauth), never a side effect of tier membership.
	Tiers map[string]TierRule `json:"tiers,omitempty"`

	Relationships []Relationship `json:"relationships,omitempty"`
	Exceptions    []Exception    `json:"exceptions,omitempty"`
}
