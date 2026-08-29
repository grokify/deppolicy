// Package graph defines the generated dependency graph document: what
// direct dependency edges are actually observed to exist, as evidenced by
// one or more scans. Graph is the "reality" side of the policy/graph/
// findings contract; unlike policy, it is never hand-authored.
package graph

import (
	"time"

	"github.com/grokify/deppolicy/component"
)

// Relation identifies the kind of fact a graph edge records. V1 has exactly
// one relation: the observed-fact counterpart to policy.RelationCanDependOn.
type Relation string

// RelationDependsOn is the only relation V1 scanners emit.
const RelationDependsOn Relation = "depends_on"

// EvidenceKind identifies how a dependency edge was discovered. Confidence
// and provenance never affect policy authorization -- only reporting.
type EvidenceKind string

const (
	// EvidenceKindManifest evidence comes from a dependency manifest such
	// as go.mod, without inspecting source. Assurance: declared.
	EvidenceKindManifest EvidenceKind = "manifest"

	// EvidenceKindSourceImport evidence comes from an actual source-level
	// import statement. Assurance: observed. This is V1's authoritative
	// evidence kind for architecture decisions.
	EvidenceKindSourceImport EvidenceKind = "source-import"

	// Reserved for future scanners; not emitted by any V1 scanner.
	EvidenceKindIaC         EvidenceKind = "iac"
	EvidenceKindLLMAnalysis EvidenceKind = "llm-analysis"
	EvidenceKindTelemetry   EvidenceKind = "telemetry"
)

// Evidence records one place a dependency edge was observed.
type Evidence struct {
	Kind EvidenceKind `json:"kind"`

	// File and Line locate source-import evidence.
	File string `json:"file,omitempty"`
	Line int    `json:"line,omitempty"`

	// Source names the manifest a manifest-kind evidence entry came
	// from, e.g. "go.mod".
	Source string `json:"source,omitempty"`
}

// Dependency is one observed direct edge: Source depends_on Target. A single
// Dependency may carry multiple Evidence entries -- e.g. several packages
// within one module importing another module all collapse to one edge with
// multiple evidence entries underneath it.
type Dependency struct {
	Source   string     `json:"source"`
	Relation Relation   `json:"relation"`
	Target   string     `json:"target"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

// ComponentObservation records a component as it appeared in this scan.
type ComponentObservation struct {
	ID     string         `json:"id"`
	Kind   component.Kind `json:"kind,omitempty"`
	Commit string         `json:"commit,omitempty"`
}

// ScanLevel identifies the assurance level of a scan, so consumers do not
// mistake a fast manifest-only fleet scan for a full source audit.
type ScanLevel string

const (
	ScanLevelManifest      ScanLevel = "manifest"
	ScanLevelSourceImports ScanLevel = "source-imports"
)

// ScanMetadata describes the scan that produced a Graph.
type ScanMetadata struct {
	Level    ScanLevel `json:"level"`
	Scanner  string    `json:"scanner"` // e.g. "deppolicy-go v0.3.0"
	Coverage []string  `json:"coverage,omitempty"`
	Excluded []string  `json:"excluded,omitempty"` // patterns/locators skipped, for visibility
}

// Graph is the generated, observed dependency graph produced by one scan.
// It is never hand-edited: it is compared against a Policy to produce
// Findings.
type Graph struct {
	Generated    time.Time              `json:"generated"`
	Scan         ScanMetadata           `json:"scan"`
	Components   []ComponentObservation `json:"components,omitempty"`
	Dependencies []Dependency           `json:"dependencies,omitempty"`
}
