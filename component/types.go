// Package component defines the ecosystem-neutral unit governed by dependency
// policy: a component. A component may be a Go module, a directory within a
// Go module (a "proto-module"), an npm package, a Rust crate, a service, or
// any other unit identified by a locator string.
package component

// ID is a stable component identity. It survives repository moves, module
// renames, and promotion from a proto-module (a package-level component) to
// a full module, because policy and graph documents reference components by
// ID rather than by their current locator.
type ID string

// Kind identifies the ecosystem-specific packaging mechanism a component
// uses. It is metadata only: policy and graph semantics never branch on Kind.
type Kind string

// Recognized component kinds. This list is expected to grow as scanners for
// additional ecosystems are added; it is not exhaustive by design.
const (
	KindGoModule       Kind = "go-module"
	KindGoPackage      Kind = "go-package" // a directory within a Go module declared as a proto-module
	KindNPMPackage     Kind = "npm-package"
	KindCrate          Kind = "crate"
	KindGem            Kind = "gem"
	KindPyPIDist       Kind = "pypi-distribution"
	KindMavenArtifact  Kind = "maven-artifact"
	KindContainerImage Kind = "container-image"
	KindService        Kind = "service"
)

// Status governs how a component participates in policy evaluation.
type Status string

const (
	// StatusGoverned is the default: dependency edges to and from this
	// component require explicit authorization (or a foundation-tier
	// target) under default-deny.
	StatusGoverned Status = "governed"

	// StatusFoundation marks a component that any governed component may
	// depend on without an explicit allow edge. Foundation components may
	// themselves only depend on other foundation components.
	StatusFoundation Status = "foundation"

	// StatusDeprecated tolerates existing dependents (baselined) but
	// treats any new inbound edge as a violation.
	StatusDeprecated Status = "deprecated"

	// StatusUngoverned is treated like a third-party target/source for
	// evaluation purposes, but remains visible in the graph and reports
	// (unlike a scope-excluded component, which is dropped entirely).
	StatusUngoverned Status = "ungoverned"
)

// Component is a governed unit of the ecosystem: something that can be a
// source or target of a dependency relationship in policy and graph
// documents.
type Component struct {
	// ID is the stable identity referenced by policy relationships and
	// graph edges. Required.
	ID ID `json:"id"`

	// Kind identifies the ecosystem-specific packaging mechanism this
	// component uses. Required.
	Kind Kind `json:"kind"`

	// Locator is the current ecosystem address for this component, e.g.
	// "go:github.com/plexusone/omnillm" or a purl-style identifier.
	// Required. For proto-modules this is a path prefix beneath a parent
	// module's locator.
	Locator string `json:"locator"`

	// Status governs evaluation behavior. Empty is equivalent to
	// StatusGoverned; use EffectiveStatus to read the resolved value.
	Status Status `json:"status,omitempty"`

	// Reason documents why a non-default status or classification was
	// assigned, for human and agent readability.
	Reason string `json:"reason,omitempty"`

	// Decision references an ADR or other durable record backing this
	// component's classification.
	Decision string `json:"decision,omitempty"`
}

// EffectiveStatus returns the component's status, defaulting to
// StatusGoverned when unset.
func (c Component) EffectiveStatus() Status {
	if c.Status == "" {
		return StatusGoverned
	}
	return c.Status
}

// IsFoundation reports whether the component has foundation status.
func (c Component) IsFoundation() bool {
	return c.EffectiveStatus() == StatusFoundation
}

// IsDeprecated reports whether the component has deprecated status.
func (c Component) IsDeprecated() bool {
	return c.EffectiveStatus() == StatusDeprecated
}

// IsUngoverned reports whether the component has ungoverned status.
func (c Component) IsUngoverned() bool {
	return c.EffectiveStatus() == StatusUngoverned
}
