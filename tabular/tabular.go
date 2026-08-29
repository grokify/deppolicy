// Package tabular flattens the three policy artifacts into neutral,
// queryable tables -- the "Policy vs Actual vs Difference" surface an
// analytics platform consumes. DashForge's datasource contract is a SQL
// connection interface, so an actual DashForge driver belongs in the
// dashforge repository; this package is the stable data surface such a
// driver (or any spreadsheet, notebook, or BI import) reads, exported
// here so the flattening semantics live next to the types they flatten.
package tabular

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

// Table is a named grid: column names plus rows of cells. Cells are `any`
// so numeric columns stay numeric in JSON; CSV rendering formats them with
// fmt.Sprint.
type Table struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

// Build flattens a policy (intent), graph (actual), and findings
// (difference) into four tables:
//
//	components      every declared or observed component with status/tier
//	policy_edges    explicit relationships and exceptions (what MAY exist);
//	                tier rules and foundation status are blanket permissions
//	                with no per-edge rows -- they appear via the components
//	                table's status/tier columns instead
//	observed_edges  every observed dependency with evidence summary
//	findings        the evaluated difference, one row per finding
//
// All tables are deterministically sorted.
func Build(p policy.Policy, g graph.Graph, findings []policy.Finding) []Table {
	return []Table{
		componentsTable(p, g),
		policyEdgesTable(p),
		observedEdgesTable(g),
		findingsTable(findings),
	}
}

func componentsTable(p policy.Policy, g graph.Graph) Table {
	declared := make(map[string]policy.ComponentDeclaration, len(p.Components))
	for locator, decl := range p.Components {
		declared[locator] = decl
	}
	observed := make(map[string]bool, len(g.Components))
	for _, c := range g.Components {
		observed[c.ID] = true
	}

	locators := make(map[string]bool, len(declared)+len(observed))
	for l := range declared {
		locators[l] = true
	}
	for l := range observed {
		locators[l] = true
	}

	sorted := make([]string, 0, len(locators))
	for l := range locators {
		sorted = append(sorted, l)
	}
	sort.Strings(sorted)

	t := Table{Name: "components", Columns: []string{"locator", "org", "status", "tier", "declared", "observed"}}
	for _, l := range sorted {
		decl, isDeclared := declared[l]
		status := string(decl.Status)
		if status == "" {
			status = "governed"
		}
		t.Rows = append(t.Rows, []any{l, orgOfLocator(l), status, decl.Tier, isDeclared, observed[l]})
	}
	return t
}

func policyEdgesTable(p policy.Policy) Table {
	t := Table{Name: "policy_edges", Columns: []string{"source", "target", "authorization", "reason", "decision", "expires"}}
	for _, r := range p.Relationships {
		t.Rows = append(t.Rows, []any{r.Source, r.Target, "relationship", r.Reason, r.Decision, ""})
	}
	for _, e := range p.Exceptions {
		t.Rows = append(t.Rows, []any{e.Source, e.Target, "exception", e.Reason, "", e.Expires})
	}
	sortRowsBySourceTarget(t.Rows)
	return t
}

func observedEdgesTable(g graph.Graph) Table {
	t := Table{Name: "observed_edges", Columns: []string{"source", "target", "evidence_count", "first_file", "first_line"}}
	for _, d := range g.Dependencies {
		firstFile, firstLine := "", 0
		for _, ev := range d.Evidence {
			if ev.File != "" {
				firstFile, firstLine = ev.File, ev.Line
				break
			}
		}
		t.Rows = append(t.Rows, []any{d.Source, d.Target, len(d.Evidence), firstFile, firstLine})
	}
	sortRowsBySourceTarget(t.Rows)
	return t
}

func findingsTable(findings []policy.Finding) Table {
	t := Table{Name: "findings", Columns: []string{"source", "target", "status", "reason"}}
	for _, f := range findings {
		t.Rows = append(t.Rows, []any{f.Source, f.Target, string(f.Status), f.Reason})
	}
	sortRowsBySourceTarget(t.Rows)
	return t
}

// sortRowsBySourceTarget sorts rows whose first two cells are source and
// target strings.
func sortRowsBySourceTarget(rows [][]any) {
	sort.Slice(rows, func(i, j int) bool {
		si, _ := rows[i][0].(string)
		sj, _ := rows[j][0].(string)
		if si != sj {
			return si < sj
		}
		ti, _ := rows[i][1].(string)
		tj, _ := rows[j][1].(string)
		return ti < tj
	})
}

// orgOfLocator extracts the GitHub organization segment from a
// "go:github.com/<org>/..." locator, or "" for any other shape.
func orgOfLocator(locator string) string {
	const prefix = "go:github.com/"
	if !strings.HasPrefix(locator, prefix) {
		return ""
	}
	rest := locator[len(prefix):]
	if idx := strings.Index(rest, "/"); idx >= 0 {
		return rest[:idx]
	}
	return rest
}

// WriteCSV renders one table as CSV with a header row.
func WriteCSV(t Table, w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(t.Columns); err != nil {
		return fmt.Errorf("tabular: write csv header for %s: %w", t.Name, err)
	}
	for _, row := range t.Rows {
		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = fmt.Sprint(cell)
		}
		if err := cw.Write(cells); err != nil {
			return fmt.Errorf("tabular: write csv row for %s: %w", t.Name, err)
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteJSON renders all tables as one indented JSON document.
func WriteJSON(tables []Table, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(tables)
}
