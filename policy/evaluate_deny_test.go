package policy

import (
	"strings"
	"testing"
	"time"
)

// denyPolicy models the real omni-family constraint this mechanism exists
// for: an abstraction core must not import provider SDKs -- those belong
// in omni-<provider> adapters.
func denyPolicy() Policy {
	return Policy{
		SchemaVersion: "1",
		Scope: Scope{
			Include: []string{"go:github.com/plexusone/*"},
		},
		Tiers: map[string]TierRule{
			"omni-core": {
				MayNotDependOn: []string{
					"go:github.com/anthropics/*",
					"go:github.com/openai/*",
				},
			},
			"omni-provider": {MayDependOn: []string{"omni-core"}},
		},
		Components: map[string]ComponentDeclaration{
			"go:github.com/plexusone/omnillm-core":   {Tier: "omni-core"},
			"go:github.com/plexusone/omni-anthropic": {Tier: "omni-provider"},
		},
	}
}

func TestEvaluateWithStatus_TierDeny_ExternalTarget(t *testing.T) {
	p := denyPolicy()
	g := edgeGraph([2]string{"go:github.com/plexusone/omnillm-core", "go:github.com/anthropics/anthropic-sdk-go"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/omnillm-core", "go:github.com/anthropics/anthropic-sdk-go")
	if !ok {
		t.Fatal("expected a finding for the denied external dependency")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q", f.Status, FindingViolation)
	}
	if !strings.Contains(f.Reason, "tier deny") {
		t.Errorf("Reason = %q, want a tier-deny reason", f.Reason)
	}
}

func TestEvaluateWithStatus_TierDeny_ProviderMayUseSDK(t *testing.T) {
	p := denyPolicy()
	// The provider adapter wrapping the SDK is exactly where the SDK
	// belongs: omni-provider has no deny rule, and the SDK is external,
	// so no finding at all.
	g := edgeGraph([2]string{"go:github.com/plexusone/omni-anthropic", "go:github.com/anthropics/anthropic-sdk-go"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none: providers wrap SDKs by design", findings)
	}
}

func TestEvaluateWithStatus_TierDeny_UnobservedProducesNoFinding(t *testing.T) {
	p := denyPolicy()
	g := edgeGraph() // nothing observed

	findings := p.EvaluateWithStatus(g, time.Now(), nil)
	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none for unobserved denied pairs", findings)
	}
}

func TestEvaluateWithStatus_TierDeny_ExceptionOverrides(t *testing.T) {
	p := denyPolicy()
	p.Exceptions = []Exception{
		{
			Source:  "go:github.com/plexusone/omnillm-core",
			Target:  "go:github.com/anthropics/anthropic-sdk-go",
			Effect:  EffectAllow,
			Expires: "2099-12-31",
			Reason:  "migrating token counting into omni-anthropic",
		},
	}
	g := edgeGraph([2]string{"go:github.com/plexusone/omnillm-core", "go:github.com/anthropics/anthropic-sdk-go"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/omnillm-core", "go:github.com/anthropics/anthropic-sdk-go")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingConforming {
		t.Errorf("Status = %q, want %q (unexpired exception is the deny escape valve)", f.Status, FindingConforming)
	}
	if !strings.Contains(f.Reason, "exception overriding tier deny") {
		t.Errorf("Reason = %q, want exception-overriding-deny reason", f.Reason)
	}
}

func TestEvaluateWithStatus_TierDeny_ExpiredExceptionDoesNotOverride(t *testing.T) {
	p := denyPolicy()
	p.Exceptions = []Exception{
		{
			Source:  "go:github.com/plexusone/omnillm-core",
			Target:  "go:github.com/anthropics/anthropic-sdk-go",
			Effect:  EffectAllow,
			Expires: "2020-01-01",
			Reason:  "long-expired migration window",
		},
	}
	g := edgeGraph([2]string{"go:github.com/plexusone/omnillm-core", "go:github.com/anthropics/anthropic-sdk-go"})

	findings := p.EvaluateWithStatus(g, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/omnillm-core", "go:github.com/anthropics/anthropic-sdk-go")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q", f.Status, FindingViolation)
	}
}

func TestEvaluateWithStatus_TierDeny_OverridesExplicitRelationship(t *testing.T) {
	p := denyPolicy()
	// A deny pattern matching an in-scope target must override even an
	// authored relationship -- deny is absolute except for exceptions.
	tier := p.Tiers["omni-core"]
	tier.MayNotDependOn = append(tier.MayNotDependOn, "go:github.com/plexusone/forbidden")
	p.Tiers["omni-core"] = tier
	p.Relationships = []Relationship{
		{
			Source:   "go:github.com/plexusone/omnillm-core",
			Relation: RelationCanDependOn,
			Target:   "go:github.com/plexusone/forbidden",
			Reason:   "authored in error",
		},
	}
	g := edgeGraph([2]string{"go:github.com/plexusone/omnillm-core", "go:github.com/plexusone/forbidden"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)

	f, ok := findingFor(t, findings, "go:github.com/plexusone/omnillm-core", "go:github.com/plexusone/forbidden")
	if !ok {
		t.Fatal("expected a finding")
	}
	if f.Status != FindingViolation {
		t.Errorf("Status = %q, want %q (deny overrides explicit relationships)", f.Status, FindingViolation)
	}
}

func TestEvaluateWithStatus_TierDeny_UntieredSourceUnaffected(t *testing.T) {
	p := denyPolicy()
	g := edgeGraph([2]string{"go:github.com/plexusone/untiered", "go:github.com/anthropics/anthropic-sdk-go"})

	findings := p.EvaluateWithStatus(g, time.Now(), nil)
	if len(findings) != 0 {
		t.Errorf("findings = %+v, want none: deny rules attach to tiers, and this source has none", findings)
	}
}

func TestPolicy_Validate_EmptyDenyPatternRejected(t *testing.T) {
	p := denyPolicy()
	tier := p.Tiers["omni-core"]
	tier.MayNotDependOn = append(tier.MayNotDependOn, "")
	p.Tiers["omni-core"] = tier

	if err := p.Validate(); err == nil {
		t.Fatal("expected error for empty mayNotDependOn pattern")
	}
}
