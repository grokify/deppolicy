// Package htmlreport renders the three policy artifacts as a
// self-contained, multi-page static HTML report: summary, filterable
// findings, component inventory, an interactive dependency graph, and the
// tier structure. Every page is a single file with inline CSS/JS and its
// data embedded as JSON -- no server, no CDN, no build step -- so a report
// can be attached to CI runs or opened from disk for architecture reviews.
//
// The report is strictly read-only by design: the canonical policy is
// reviewed JSON in git, and anything that edits policy must flow through
// the promote / policy-diff review path, never a UI save button.
package htmlreport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

// Finding is one evaluated edge as the report displays it. Ratchet is
// empty when no baseline was supplied, "new" or "existing" for violations
// when one was.
type Finding struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	Status       string `json:"status"`
	Ratchet      string `json:"ratchet,omitempty"`
	Reason       string `json:"reason"`
	EvidenceFile string `json:"evidenceFile,omitempty"`
	EvidenceLine int    `json:"evidenceLine,omitempty"`
}

// Input is everything Build needs. Findings are supplied by the caller
// (already evaluated and, if a baseline was used, ratchet-annotated) so
// this package stays purely presentational.
type Input struct {
	GeneratedAt time.Time
	PolicyPath  string
	GraphPath   string
	Baseline    bool
	Policy      policy.Policy
	Graph       graph.Graph
	Findings    []Finding
}

// Page is one rendered HTML file.
type Page struct {
	Name string // file name, e.g. "index.html"
	HTML []byte
}

type componentRow struct {
	Locator  string `json:"locator"`
	Org      string `json:"org"`
	Status   string `json:"status"`
	Tier     string `json:"tier"`
	Declared bool   `json:"declared"`
	Observed bool   `json:"observed"`
	FanIn    int    `json:"fanIn"`
	FanOut   int    `json:"fanOut"`
}

type graphNode struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Org        string `json:"org"`
	Tier       string `json:"tier"`
	Status     string `json:"status"`
	Violations int    `json:"violations"`
}

type graphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Status string `json:"status"`
}

type tierView struct {
	Name           string   `json:"name"`
	MayDependOn    []string `json:"mayDependOn"`
	MayNotDependOn []string `json:"mayNotDependOn"`
	Members        []string `json:"members"`
}

type summary struct {
	Components      int            `json:"components"`
	InScopeEdges    int            `json:"inScopeEdges"`
	Conforming      int            `json:"conforming"`
	Violations      int            `json:"violations"`
	NewViolations   int            `json:"newViolations"`
	UnusedAllowance int            `json:"unusedAllowance"`
	Foundation      []string       `json:"foundation"`
	Orgs            map[string]int `json:"orgs"`
	ScanLevel       string         `json:"scanLevel"`
	Scanner         string         `json:"scanner"`
}

// pageData is the template payload shared by every page.
type pageData struct {
	Title       string
	Active      string
	GeneratedAt string
	PolicyPath  string
	GraphPath   string
	Baseline    bool
	JSON        template.JS
}

// Build renders all report pages.
func Build(in Input) ([]Page, error) {
	rows, nodes, edges := assemble(in)
	tiers := tierViews(in.Policy)
	sum := summarize(in, rows)

	type spec struct {
		name, title, tmpl string
		data              any
	}
	specs := []spec{
		{"index.html", "Summary", "index", map[string]any{"summary": sum, "tiers": tiers}},
		{"findings.html", "Findings", "findings", map[string]any{"findings": in.Findings}},
		{"components.html", "Components", "components", map[string]any{"components": rows}},
		{"graph.html", "Graph", "graph", map[string]any{"nodes": nodes, "edges": edges}},
		{"tiers.html", "Tiers", "tiers", map[string]any{"tiers": tiers}},
	}

	pages := make([]Page, 0, len(specs))
	for _, s := range specs {
		payload, err := json.Marshal(s.data)
		if err != nil {
			return nil, fmt.Errorf("htmlreport: marshal %s data: %w", s.name, err)
		}
		var buf bytes.Buffer
		err = tmpl.ExecuteTemplate(&buf, s.tmpl, pageData{
			Title:       s.title,
			Active:      s.name,
			GeneratedAt: in.GeneratedAt.UTC().Format("2006-01-02 15:04 UTC"),
			PolicyPath:  in.PolicyPath,
			GraphPath:   in.GraphPath,
			Baseline:    in.Baseline,
			JSON:        template.JS(payload), //nolint:gosec // payload is json.Marshal output embedded as a JS literal, not user-authored script
		})
		if err != nil {
			return nil, fmt.Errorf("htmlreport: render %s: %w", s.name, err)
		}
		pages = append(pages, Page{Name: s.name, HTML: buf.Bytes()})
	}
	return pages, nil
}

// assemble computes component rows, graph nodes, and graph edges from the
// in-scope portion of the graph plus policy declarations.
func assemble(in Input) ([]componentRow, []graphNode, []graphEdge) {
	inScope := func(l string) bool { return in.Policy.InScope(l) }

	findingByEdge := make(map[[2]string]Finding, len(in.Findings))
	violationsBySource := map[string]int{}
	for _, f := range in.Findings {
		findingByEdge[[2]string{f.Source, f.Target}] = f
		if f.Status == "violation" {
			violationsBySource[f.Source]++
		}
	}

	fanIn, fanOut := map[string]int{}, map[string]int{}
	seen := map[[2]string]bool{}
	var edges []graphEdge
	nodeSet := map[string]bool{}
	for _, d := range in.Graph.Dependencies {
		if !inScope(d.Source) || !inScope(d.Target) {
			continue
		}
		k := [2]string{d.Source, d.Target}
		if seen[k] {
			continue
		}
		seen[k] = true
		fanOut[d.Source]++
		fanIn[d.Target]++
		nodeSet[d.Source] = true
		nodeSet[d.Target] = true
		status := "unevaluated"
		if f, ok := findingByEdge[k]; ok {
			status = f.Status
			if f.Ratchet == "new" {
				status = "new-violation"
			}
		}
		edges = append(edges, graphEdge{Source: d.Source, Target: d.Target, Status: status})
	}
	for _, c := range in.Graph.Components {
		if inScope(c.ID) {
			nodeSet[c.ID] = true
		}
	}
	for locator := range in.Policy.Components {
		nodeSet[locator] = true
	}

	locators := make([]string, 0, len(nodeSet))
	for l := range nodeSet {
		locators = append(locators, l)
	}
	sort.Strings(locators)

	observed := make(map[string]bool, len(in.Graph.Components))
	for _, c := range in.Graph.Components {
		observed[c.ID] = true
	}

	rows := make([]componentRow, 0, len(locators))
	nodes := make([]graphNode, 0, len(locators))
	for _, l := range locators {
		decl, isDeclared := in.Policy.Components[l]
		status := string(decl.Status)
		if status == "" {
			status = "governed"
		}
		rows = append(rows, componentRow{
			Locator: l, Org: orgOf(l), Status: status, Tier: decl.Tier,
			Declared: isDeclared, Observed: observed[l],
			FanIn: fanIn[l], FanOut: fanOut[l],
		})
		nodes = append(nodes, graphNode{
			ID: l, Label: label(l), Org: orgOf(l), Tier: decl.Tier, Status: status,
			Violations: violationsBySource[l],
		})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		return edges[i].Target < edges[j].Target
	})
	return rows, nodes, edges
}

func tierViews(p policy.Policy) []tierView {
	members := map[string][]string{}
	for locator, decl := range p.Components {
		if decl.Tier != "" {
			members[decl.Tier] = append(members[decl.Tier], locator)
		}
	}
	names := make([]string, 0, len(p.Tiers))
	for name := range p.Tiers {
		names = append(names, name)
	}
	sort.Strings(names)

	views := make([]tierView, 0, len(names))
	for _, name := range names {
		rule := p.Tiers[name]
		sort.Strings(members[name])
		views = append(views, tierView{
			Name:           name,
			MayDependOn:    rule.MayDependOn,
			MayNotDependOn: rule.MayNotDependOn,
			Members:        members[name],
		})
	}
	return views
}

func summarize(in Input, rows []componentRow) summary {
	sum := summary{
		Components: len(rows),
		Orgs:       map[string]int{},
		ScanLevel:  string(in.Graph.Scan.Level),
		Scanner:    in.Graph.Scan.Scanner,
	}
	for _, r := range rows {
		sum.Orgs[r.Org]++
		if r.Status == "foundation" {
			sum.Foundation = append(sum.Foundation, r.Locator)
		}
	}
	sort.Strings(sum.Foundation)
	for _, f := range in.Findings {
		switch f.Status {
		case "conforming":
			sum.Conforming++
		case "violation":
			sum.Violations++
			if f.Ratchet == "new" {
				sum.NewViolations++
			}
		case "unused_allowance":
			sum.UnusedAllowance++
		}
	}
	sum.InScopeEdges = sum.Conforming + sum.Violations
	return sum
}

// orgOf extracts the GitHub organization from a "go:github.com/<org>/..."
// locator, or "other" for any other shape.
func orgOf(locator string) string {
	const prefix = "go:github.com/"
	if !strings.HasPrefix(locator, prefix) {
		return "other"
	}
	rest := locator[len(prefix):]
	if idx := strings.Index(rest, "/"); idx >= 0 {
		return rest[:idx]
	}
	return rest
}

// label strips the scheme and host for display.
func label(locator string) string {
	l := strings.TrimPrefix(locator, "go:")
	return strings.TrimPrefix(l, "github.com/")
}
