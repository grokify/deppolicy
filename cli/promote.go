package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

// promotedReasonPlaceholder marks a proposed relationship's reason as
// generated rather than authored, so it stays greppable and obviously
// incomplete if a reviewer forgets to replace it before committing.
const promotedReasonPlaceholder = "PROMOTED: observed edge, needs review -- replace this reason before committing"

// ProposedRelationship is one candidate policy.Relationship generated from
// an observed, currently-unauthorized edge, plus evidence to help a
// reviewer judge it without re-reading the graph directly.
type ProposedRelationship struct {
	policy.Relationship
	Evidence []graph.Evidence `json:"evidence,omitempty"`
}

// PromoteOptions configures Promote.
type PromoteOptions struct {
	// PolicyPath is the path to the current policy.json. Required.
	PolicyPath string

	// GraphPath is the path to a previously scanned graph.json. Required.
	// Promote never scans live: bootstrap review is iterative, and
	// re-scanning a large fleet on every run would be wasteful --
	// scan once with `deppolicy scan`, then promote repeatedly against
	// that saved graph as relationships get reviewed and merged in.
	GraphPath string

	// AsOf is the evaluation time used to resolve exception expiry.
	AsOf time.Time
}

// PromoteResult is Promote's structured output.
type PromoteResult struct {
	Policy *policy.Policy
	Graph  graph.Graph

	// Proposed holds one candidate relationship per observed edge that
	// currently violates default-deny.
	Proposed []ProposedRelationship

	// ProposedPolicy is Policy with Proposed's relationships appended --
	// a complete, valid draft document, never written back to
	// PolicyPath. A reviewer diffs it against the original, edits each
	// proposed reason, and merges by hand.
	ProposedPolicy policy.Policy
}

// Promote loads a policy and a previously scanned graph, and proposes a
// new policy.Relationship for every observed edge the policy does not
// currently authorize. It never writes to PolicyPath: the original policy
// is intent, authored deliberately, and promotion only ever produces a
// separate, reviewable artifact -- never a silent change to it (see
// SPEC.md section 4: "policy is never generated from the graph").
func Promote(opts PromoteOptions) (PromoteResult, error) {
	policyData, err := os.ReadFile(opts.PolicyPath)
	if err != nil {
		return PromoteResult{}, fmt.Errorf("cli: read policy %s: %w", opts.PolicyPath, err)
	}
	p, err := policy.Parse(policyData)
	if err != nil {
		return PromoteResult{}, fmt.Errorf("cli: parse policy %s: %w", opts.PolicyPath, err)
	}

	graphData, err := os.ReadFile(opts.GraphPath)
	if err != nil {
		return PromoteResult{}, fmt.Errorf("cli: read graph %s: %w", opts.GraphPath, err)
	}
	g, err := graph.Parse(graphData)
	if err != nil {
		return PromoteResult{}, fmt.Errorf("cli: parse graph %s: %w", opts.GraphPath, err)
	}

	findings := p.EvaluateWithStatus(*g, opts.AsOf, nil)

	var proposed []ProposedRelationship
	var newRelationships []policy.Relationship
	for _, f := range findings {
		if f.Status != policy.FindingViolation {
			continue
		}
		rel := policy.Relationship{
			Source:   f.Source,
			Relation: policy.RelationCanDependOn,
			Target:   f.Target,
			Reason:   promotedReasonPlaceholder,
		}
		proposed = append(proposed, ProposedRelationship{
			Relationship: rel,
			Evidence:     evidenceFor(*g, f.Source, f.Target),
		})
		newRelationships = append(newRelationships, rel)
	}

	proposedPolicy := *p
	proposedPolicy.Relationships = append(append([]policy.Relationship{}, p.Relationships...), newRelationships...)

	return PromoteResult{
		Policy:         p,
		Graph:          *g,
		Proposed:       proposed,
		ProposedPolicy: proposedPolicy,
	}, nil
}
