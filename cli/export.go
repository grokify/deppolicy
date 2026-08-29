package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

// ExportFormat selects a graph export syntax.
type ExportFormat string

const (
	FormatMermaid ExportFormat = "mermaid"
	FormatDOT     ExportFormat = "dot"
)

// exportModel is the normalized shape both syntaxes render: nodes grouped
// by org, edges flagged violating or not. Building it once keeps the two
// emitters trivially in sync on filtering/grouping semantics.
type exportModel struct {
	orgs       []string            // sorted
	nodesByOrg map[string][]string // org -> sorted locators
	edges      []exportEdge        // sorted (source, target)
}

type exportEdge struct {
	source, target string
	violation      bool
}

// buildExportModel filters and annotates g for rendering. With a policy,
// only in-scope components and in-scope-to-in-scope edges are included --
// a full ecosystem graph with hundreds of third-party targets is
// unreadable, and out-of-scope edges are by definition not architecture
// under governance -- and edges are marked as violations per
// EvaluateWithStatus (nil baseline). Without a policy, everything in the
// graph is included and nothing is marked.
func buildExportModel(g graph.Graph, p *policy.Policy, asOf time.Time) exportModel {
	include := func(locator string) bool {
		if p == nil {
			return true
		}
		return p.InScope(locator)
	}

	violations := map[[2]string]bool{}
	if p != nil {
		for _, f := range p.EvaluateWithStatus(g, asOf, nil) {
			if f.Status == policy.FindingViolation {
				violations[[2]string{f.Source, f.Target}] = true
			}
		}
	}

	nodeSet := map[string]bool{}
	seen := map[[2]string]bool{}
	var edges []exportEdge
	for _, d := range g.Dependencies {
		if !include(d.Source) || !include(d.Target) {
			continue
		}
		key := [2]string{d.Source, d.Target}
		if seen[key] {
			continue
		}
		seen[key] = true
		nodeSet[d.Source] = true
		nodeSet[d.Target] = true
		edges = append(edges, exportEdge{source: d.Source, target: d.Target, violation: violations[key]})
	}
	for _, c := range g.Components {
		if include(c.ID) {
			nodeSet[c.ID] = true
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].source != edges[j].source {
			return edges[i].source < edges[j].source
		}
		return edges[i].target < edges[j].target
	})

	nodesByOrg := map[string][]string{}
	for locator := range nodeSet {
		org := orgOf(locator)
		if org == "" {
			org = "other"
		}
		nodesByOrg[org] = append(nodesByOrg[org], locator)
	}
	orgs := make([]string, 0, len(nodesByOrg))
	for org := range nodesByOrg {
		sort.Strings(nodesByOrg[org])
		orgs = append(orgs, org)
	}
	sort.Strings(orgs)

	return exportModel{orgs: orgs, nodesByOrg: nodesByOrg, edges: edges}
}

// Export renders g in the requested format. p is optional; see
// buildExportModel for what it changes. asOf resolves exception expiry
// during violation marking.
func Export(g graph.Graph, p *policy.Policy, format ExportFormat, asOf time.Time) (string, error) {
	m := buildExportModel(g, p, asOf)
	switch format {
	case FormatMermaid:
		return renderMermaid(m), nil
	case FormatDOT:
		return renderDOT(m), nil
	default:
		return "", fmt.Errorf("cli: unknown export format %q (want %q or %q)", format, FormatMermaid, FormatDOT)
	}
}

// nodeID converts a locator into an identifier safe for both Mermaid and
// DOT bare identifiers.
func nodeID(locator string) string {
	var b strings.Builder
	for _, r := range locator {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// nodeLabel is the human-facing name: the locator without its scheme and
// host prefix, so "go:github.com/grokify/mogo" reads as "grokify/mogo".
func nodeLabel(locator string) string {
	label := strings.TrimPrefix(locator, "go:")
	return strings.TrimPrefix(label, "github.com/")
}

func renderMermaid(m exportModel) string {
	var b strings.Builder
	fmt.Fprintln(&b, "graph LR")
	for _, org := range m.orgs {
		fmt.Fprintf(&b, "  subgraph %s\n", org)
		for _, locator := range m.nodesByOrg[org] {
			fmt.Fprintf(&b, "    %s[\"%s\"]\n", nodeID(locator), nodeLabel(locator))
		}
		fmt.Fprintln(&b, "  end")
	}
	var violationIdx []int
	for i, e := range m.edges {
		fmt.Fprintf(&b, "  %s --> %s\n", nodeID(e.source), nodeID(e.target))
		if e.violation {
			violationIdx = append(violationIdx, i)
		}
	}
	for _, i := range violationIdx {
		fmt.Fprintf(&b, "  linkStyle %d stroke:#cc0000,stroke-width:2px\n", i)
	}
	return b.String()
}

func renderDOT(m exportModel) string {
	var b strings.Builder
	fmt.Fprintln(&b, "digraph deppolicy {")
	fmt.Fprintln(&b, "  rankdir=LR;")
	fmt.Fprintln(&b, "  node [shape=box];")
	for _, org := range m.orgs {
		fmt.Fprintf(&b, "  subgraph \"cluster_%s\" {\n", org)
		fmt.Fprintf(&b, "    label=\"%s\";\n", org)
		for _, locator := range m.nodesByOrg[org] {
			fmt.Fprintf(&b, "    %q [label=%q];\n", locator, nodeLabel(locator))
		}
		fmt.Fprintln(&b, "  }")
	}
	for _, e := range m.edges {
		if e.violation {
			fmt.Fprintf(&b, "  %q -> %q [color=\"#cc0000\", penwidth=2];\n", e.source, e.target)
		} else {
			fmt.Fprintf(&b, "  %q -> %q;\n", e.source, e.target)
		}
	}
	fmt.Fprintln(&b, "}")
	return b.String()
}
