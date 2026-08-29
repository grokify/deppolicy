package htmlreport

import (
	"strings"
	"testing"
	"time"

	"github.com/grokify/deppolicy/graph"
	"github.com/grokify/deppolicy/policy"
)

func fixtureInput() Input {
	return Input{
		GeneratedAt: time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC),
		PolicyPath:  "ecosystem/policy.json",
		GraphPath:   "graph.json",
		Baseline:    true,
		Policy: policy.Policy{
			SchemaVersion: "1",
			Scope:         policy.Scope{Include: []string{"go:github.com/orgA/*"}},
			Tiers: map[string]policy.TierRule{
				"library":      {},
				"flagship-app": {MayDependOn: []string{"library"}},
				"omni-core":    {MayNotDependOn: []string{"go:github.com/anthropics/*"}},
			},
			Components: map[string]policy.ComponentDeclaration{
				"go:github.com/orgA/mogo": {Status: "foundation"},
				"go:github.com/orgA/app":  {Tier: "flagship-app"},
				"go:github.com/orgA/lib":  {Tier: "library"},
			},
		},
		Graph: graph.Graph{
			Scan: graph.ScanMetadata{Level: graph.ScanLevelSourceImports, Scanner: "test"},
			Components: []graph.ComponentObservation{
				{ID: "go:github.com/orgA/app"}, {ID: "go:github.com/orgA/lib"}, {ID: "go:github.com/orgA/rogue"},
			},
			Dependencies: []graph.Dependency{
				{Source: "go:github.com/orgA/app", Relation: graph.RelationDependsOn, Target: "go:github.com/orgA/lib"},
				{Source: "go:github.com/orgA/rogue", Relation: graph.RelationDependsOn, Target: "go:github.com/orgA/app"},
			},
		},
		Findings: []Finding{
			{Source: "go:github.com/orgA/app", Target: "go:github.com/orgA/lib", Status: "conforming", Reason: "tier rule: flagship-app may depend on library"},
			{Source: "go:github.com/orgA/rogue", Target: "go:github.com/orgA/app", Status: "violation", Ratchet: "new", Reason: "no allow rule; default-deny", EvidenceFile: "main.go", EvidenceLine: 7},
		},
	}
}

func TestBuild_AllPages(t *testing.T) {
	pages, err := Build(fixtureInput())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	want := []string{"index.html", "findings.html", "components.html", "graph.html", "tiers.html"}
	if len(pages) != len(want) {
		t.Fatalf("got %d pages, want %d", len(pages), len(want))
	}
	for i, name := range want {
		if pages[i].Name != name {
			t.Errorf("pages[%d].Name = %q, want %q", i, pages[i].Name, name)
		}
		html := string(pages[i].HTML)
		if !strings.HasPrefix(html, "<!DOCTYPE html>") {
			t.Errorf("%s does not start with doctype", name)
		}
		if !strings.Contains(html, `class="active"`) {
			t.Errorf("%s has no active nav item", name)
		}
		if !strings.Contains(html, "read-only report") {
			t.Errorf("%s missing read-only footer", name)
		}
	}
}

func TestBuild_PageContents(t *testing.T) {
	pages, err := Build(fixtureInput())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	byName := map[string]string{}
	for _, p := range pages {
		byName[p.Name] = string(p.HTML)
	}

	if !strings.Contains(byName["findings.html"], `"ratchet":"new"`) {
		t.Error("findings.html missing new-violation ratchet data")
	}
	if !strings.Contains(byName["findings.html"], `"evidenceFile":"main.go"`) {
		t.Error("findings.html missing evidence data")
	}
	if !strings.Contains(byName["components.html"], `"locator":"go:github.com/orgA/mogo"`) {
		t.Error("components.html missing declared-only foundation component")
	}
	if !strings.Contains(byName["graph.html"], `"status":"new-violation"`) {
		t.Error("graph.html missing new-violation edge status")
	}
	if !strings.Contains(byName["tiers.html"], `"mayNotDependOn":["go:github.com/anthropics/*"]`) {
		t.Error("tiers.html missing deny rule data")
	}
	if !strings.Contains(byName["index.html"], `"newViolations":1`) {
		t.Error("index.html missing new-violation count")
	}
}

func TestBuild_FanCounts(t *testing.T) {
	pages, err := Build(fixtureInput())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	var components string
	for _, p := range pages {
		if p.Name == "components.html" {
			components = string(p.HTML)
		}
	}
	// app: fanIn 1 (from rogue), fanOut 1 (to lib)
	if !strings.Contains(components, `"locator":"go:github.com/orgA/app","org":"orgA","status":"governed","tier":"flagship-app","declared":true,"observed":true,"fanIn":1,"fanOut":1`) {
		t.Errorf("components.html missing expected app row; got:\n%s", components[strings.Index(components, "const DATA"):][:600])
	}
}
