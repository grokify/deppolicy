package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/htmlreport"
	"github.com/grokify/deppolicy/policy"
)

// UIOptions configures UI.
type UIOptions struct {
	// PolicyPath and GraphPath are required. Like report/promote/tables,
	// UI never scans live: pass a graph.json from a prior scan.
	PolicyPath string
	GraphPath  string

	// BaselineGraphPath, if set, enables ratchet classification: violation
	// findings are split into new (absent from baseline) versus baselined
	// debt, exactly as `deppolicy diff` does.
	BaselineGraphPath string

	// OutputDir receives one HTML file per report page. Required.
	OutputDir string

	// AsOf is the evaluation time and the report's generation stamp.
	AsOf time.Time
}

// UIResult is UI's structured output.
type UIResult struct {
	Pages []string // file paths written, in page order
}

// UI renders the multi-page static HTML report into opts.OutputDir.
func UI(opts UIOptions) (UIResult, error) {
	p, err := policy.ParseFile(opts.PolicyPath)
	if err != nil {
		return UIResult{}, err
	}
	g, err := graphFromFile(opts.GraphPath)
	if err != nil {
		return UIResult{}, err
	}

	var baseline *graph.Graph
	if opts.BaselineGraphPath != "" {
		baseline, err = graphFromFile(opts.BaselineGraphPath)
		if err != nil {
			return UIResult{}, err
		}
	}

	findings := p.EvaluateWithStatus(*g, opts.AsOf, baseline)

	var reportFindings []htmlreport.Finding
	if baseline != nil {
		gdiff := graph.DiffGraphs(*baseline, *g)
		for _, rf := range ratchetAnnotate(findings, gdiff.Added) {
			reportFindings = append(reportFindings, toReportFinding(rf.Finding, string(rf.Ratchet), *g))
		}
	} else {
		for _, f := range findings {
			reportFindings = append(reportFindings, toReportFinding(f, "", *g))
		}
	}

	pages, err := htmlreport.Build(htmlreport.Input{
		GeneratedAt: opts.AsOf,
		PolicyPath:  opts.PolicyPath,
		GraphPath:   opts.GraphPath,
		Baseline:    baseline != nil,
		Policy:      *p,
		Graph:       *g,
		Findings:    reportFindings,
	})
	if err != nil {
		return UIResult{}, err
	}

	if err := os.MkdirAll(opts.OutputDir, 0o750); err != nil {
		return UIResult{}, fmt.Errorf("cli: create output dir %s: %w", opts.OutputDir, err)
	}
	var written []string
	for _, page := range pages {
		path := filepath.Join(opts.OutputDir, page.Name)
		if err := os.WriteFile(path, page.HTML, 0o600); err != nil {
			return UIResult{}, fmt.Errorf("cli: write %s: %w", path, err)
		}
		written = append(written, path)
	}
	return UIResult{Pages: written}, nil
}

// toReportFinding maps an evaluated finding (plus optional ratchet
// classification) into the report's display row. Ratchet statuses collapse
// to "new"/"existing" for violations only -- conforming/unused rows carry
// no ratchet label, since baselining only changes what a violation means.
func toReportFinding(f policy.Finding, ratchet string, g graph.Graph) htmlreport.Finding {
	out := htmlreport.Finding{
		Source: f.Source,
		Target: f.Target,
		Status: string(f.Status),
		Reason: f.Reason,
	}
	switch ratchet {
	case string(RatchetNewViolation):
		out.Ratchet = "new"
	case string(RatchetExistingViolation):
		out.Ratchet = "existing"
	}
	for _, ev := range evidenceFor(g, f.Source, f.Target) {
		if ev.File != "" {
			out.EvidenceFile, out.EvidenceLine = ev.File, ev.Line
			break
		}
	}
	return out
}
