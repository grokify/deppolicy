package policy

import (
	"fmt"
)

// Validate checks that the policy document is well-formed. It does not
// check referential integrity against a graph (e.g. whether a declared
// relationship's endpoints actually exist as scanned components) — that is
// the responsibility of the "validate" CLI command, which composes this
// structural check with graph-aware checks.
func (p Policy) Validate() error {
	if p.SchemaVersion == "" {
		return fmt.Errorf("policy: schemaVersion is required")
	}
	if len(p.Scope.Include) == 0 {
		return fmt.Errorf("policy: scope.include must declare at least one pattern")
	}

	for locator, decl := range p.Components {
		if !p.InScope(locator) {
			return fmt.Errorf("policy: component %q is declared but not in scope", locator)
		}
		if decl.Tier != "" {
			if _, ok := p.Tiers[decl.Tier]; !ok {
				return fmt.Errorf("policy: component %q references undeclared tier %q", locator, decl.Tier)
			}
		}
	}

	for tierName, rule := range p.Tiers {
		for _, dep := range rule.MayDependOn {
			if dep == tierName {
				return fmt.Errorf("policy: tier %q may not list itself in mayDependOn (same-tier dependencies always require explicit relationships)", tierName)
			}
			if _, ok := p.Tiers[dep]; !ok {
				return fmt.Errorf("policy: tier %q lists undeclared tier %q in mayDependOn", tierName, dep)
			}
		}
		for i, pattern := range rule.MayNotDependOn {
			if pattern == "" {
				return fmt.Errorf("policy: tier %q mayNotDependOn[%d] is empty", tierName, i)
			}
		}
	}

	seenRelationship := make(map[[2]string]bool, len(p.Relationships))
	for i, r := range p.Relationships {
		if r.Source == "" || r.Target == "" {
			return fmt.Errorf("policy: relationships[%d]: source and target are required", i)
		}
		if r.Relation != RelationCanDependOn {
			return fmt.Errorf("policy: relationships[%d]: unsupported relation %q (V1 supports only %q)", i, r.Relation, RelationCanDependOn)
		}
		if !p.InScope(r.Source) {
			return fmt.Errorf("policy: relationships[%d]: source %q is not in scope", i, r.Source)
		}
		if !p.InScope(r.Target) {
			return fmt.Errorf("policy: relationships[%d]: target %q is not in scope", i, r.Target)
		}
		if r.Reason == "" {
			return fmt.Errorf("policy: relationships[%d]: reason is required", i)
		}
		key := [2]string{r.Source, r.Target}
		if seenRelationship[key] {
			return fmt.Errorf("policy: relationships[%d]: duplicate relationship %s -> %s", i, r.Source, r.Target)
		}
		seenRelationship[key] = true
	}

	seenException := make(map[[2]string]bool, len(p.Exceptions))
	for i, e := range p.Exceptions {
		if e.Source == "" || e.Target == "" {
			return fmt.Errorf("policy: exceptions[%d]: source and target are required", i)
		}
		if e.Effect != EffectAllow {
			return fmt.Errorf("policy: exceptions[%d]: unsupported effect %q (V1 supports only %q)", i, e.Effect, EffectAllow)
		}
		if e.Reason == "" {
			return fmt.Errorf("policy: exceptions[%d]: reason is required", i)
		}
		if _, err := e.ExpiresAt(); err != nil {
			return fmt.Errorf("policy: exceptions[%d]: expires %q is not a valid date (want YYYY-MM-DD): %w", i, e.Expires, err)
		}
		key := [2]string{e.Source, e.Target}
		if seenException[key] {
			return fmt.Errorf("policy: exceptions[%d]: duplicate exception %s -> %s", i, e.Source, e.Target)
		}
		seenException[key] = true
	}

	return nil
}
