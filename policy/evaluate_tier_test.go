package policy

import (
	"testing"
	"time"
)

// tierPolicy models the real ecosystem's cloud-branch ladder:
// vertical-app -> horizontal-forge -> {forge-foundation, omni-adapter}.
func tierPolicy() Policy {
	return Policy{
		SchemaVersion: "1",
		Scope: Scope{
			Include: []string{"go:github.com/plexusone/*"},
		},
		Tiers: map[string]TierRule{
			"vertical-app":     {MayDependOn: []string{"horizontal-forge"}},
			"horizontal-forge": {MayDependOn: []string{"forge-foundation", "omni-adapter"}},
			"forge-foundation": {},
			"omni-adapter":     {},
		},
		Components: map[string]ComponentDeclaration{
			"go:github.com/plexusone/vertical-x":  {Tier: "vertical-app"},
			"go:github.com/plexusone/dashforge":   {Tier: "horizontal-forge"},
			"go:github.com/plexusone/actionforge": {Tier: "horizontal-forge"},
			"go:github.com/plexusone/systemforge": {Tier: "forge-foundation"},
			"go:github.com/plexusone/omnillm":     {Tier: "omni-adapter"},
			"go:github.com/plexusone/omnivoice":   {Tier: "omni-adapter"},
		},
	}
}

func TestEvaluateWithStatus_TierRuleAutoAllows(t *testing.T) {
	p := tierPolicy()
	g := edgeGraph(
		[2]string{"go:github.com/plexusone/vertical-x", "go:github.com/plexusone/dashforge"},
		[2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm"},
		[2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/systemforge"},
	)

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	for _, edge := range [][2]string{
		{"go:github.com/plexusone/vertical-x", "go:github.com/plexusone/dashforge"},
		{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm"},
		{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/systemforge"},
	} {
		f, ok := findingFor(t, findings, edge[0], edge[1])
		if !ok {
			t.Fatalf("expected a finding for %s -> %s", edge[0], edge[1])
		}
		if f.Status != FindingConforming {
			t.Errorf("%s -> %s Status = %q, want %q (tier rule)", edge[0], edge[1], f.Status, FindingConforming)
		}
	}
}

func TestEvaluateWithStatus_TierSkippingIsViolation(t *testing.T) {
	p := tierPolicy()
	// vertical-app may depend on horizontal-forge; omni-adapter is one
	// rung further down. Reaching it directly is an architecture bypass.
	g := edgeGraph([2]string{"go:github.com/plexusone/vertical-x", "go:github.com/plexusone/omnillm"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/vertical-x", "go:github.com/plexusone/omnillm")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q (tier rules must not compose transitively)", f.Status, FindingViolation)
	}
}

func TestEvaluateWithStatus_SameTierIsViolation(t *testing.T) {
	p := tierPolicy()
	g := edgeGraph(
		[2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/actionforge"}, // cross-forge
		[2]string{"go:github.com/plexusone/omnillm", "go:github.com/plexusone/omnivoice"},     // cross-omni
	)

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	for _, edge := range [][2]string{
		{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/actionforge"},
		{"go:github.com/plexusone/omnillm", "go:github.com/plexusone/omnivoice"},
	} {
		f, ok := findingFor(t, findings, edge[0], edge[1])
		if !ok {
			t.Fatalf("expected a finding for %s -> %s", edge[0], edge[1])
		}
		if f.Status != FindingViolation {
			t.Errorf("%s -> %s Status = %q, want %q (same-tier peers always need explicit wiring)", edge[0], edge[1], f.Status, FindingViolation)
		}
	}
}

func TestEvaluateWithStatus_SameTierExplicitRelationshipStillWorks(t *testing.T) {
	p := tierPolicy()
	p.Relationships = []Relationship{
		{
			Source:   "go:github.com/plexusone/dashforge",
			Relation: RelationCanDependOn,
			Target:   "go:github.com/plexusone/actionforge",
			Reason:   "dashforge embeds actionforge's pipeline runner (deliberate cross-forge wiring)",
		},
	}
	g := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/actionforge"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/plexusone/actionforge")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingConforming {
		t.Errorf("Status = %q, want %q (explicit wiring is the escape valve for same-tier)", f.Status, FindingConforming)
	}
}

func TestEvaluateWithStatus_UntieredComponentGetsNoTierAllow(t *testing.T) {
	p := tierPolicy()
	// Neither endpoint declared with a tier: nothing to match a rule on.
	g := edgeGraph([2]string{"go:github.com/plexusone/untiered-a", "go:github.com/plexusone/untiered-b"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/untiered-a", "go:github.com/plexusone/untiered-b")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q", f.Status, FindingViolation)
	}
}

func TestEvaluateWithStatus_TierRuleDoesNotCreateUnusedAllowances(t *testing.T) {
	p := tierPolicy()
	g := edgeGraph() // nothing observed

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none: blanket tier rules must not fabricate unused-allowance findings for every possible pair", findings)
	}
}

func TestEvaluateWithStatus_FoundationInvariantOverridesTierRule(t *testing.T) {
	p := tierPolicy()
	// A foundation component whose tier rule would allow the edge: the
	// invariant must still win.
	decl := p.Components["go:github.com/plexusone/dashforge"]
	decl.Status = "foundation"
	p.Components["go:github.com/plexusone/dashforge"] = decl

	g := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q (foundation invariant overrides tier rules)", f.Status, FindingViolation)
	}
}

func TestEvaluateWithStatus_DeprecatedOverridesTierRule(t *testing.T) {
	p := tierPolicy()
	decl := p.Components["go:github.com/plexusone/omnillm"]
	decl.Status = "deprecated"
	p.Components["go:github.com/plexusone/omnillm"] = decl

	g := edgeGraph([2]string{"go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm"})
	baseline := edgeGraph() // edge is new

	findings := p.EvaluateWithStatus(g, time.Now(), &baseline)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/dashforge", "go:github.com/plexusone/omnillm")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q (deprecated means no new dependents, tier rule or not)", f.Status, FindingViolation)
	}
}

func TestPolicy_Validate_TierRules(t *testing.T) {
	t.Run("valid tier config passes", func(t *testing.T) {
		if err := tierPolicy().Validate(); err != nil {
			t.Fatalf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("component referencing undeclared tier fails", func(t *testing.T) {
		p := tierPolicy()
		p.Components["go:github.com/plexusone/oops"] = ComponentDeclaration{Tier: "no-such-tier"}
		if err := p.Validate(); err == nil {
			t.Fatal("expected error for undeclared tier reference")
		}
	})

	t.Run("mayDependOn referencing undeclared tier fails", func(t *testing.T) {
		p := tierPolicy()
		p.Tiers["vertical-app"] = TierRule{MayDependOn: []string{"no-such-tier"}}
		if err := p.Validate(); err == nil {
			t.Fatal("expected error for undeclared mayDependOn tier")
		}
	})

	t.Run("tier listing itself fails", func(t *testing.T) {
		p := tierPolicy()
		p.Tiers["horizontal-forge"] = TierRule{MayDependOn: []string{"horizontal-forge"}}
		if err := p.Validate(); err == nil {
			t.Fatal("expected error for self-referencing tier")
		}
	})
}
