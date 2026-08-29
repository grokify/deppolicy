# Tiers

Tiers group components into architectural archetypes and attach **blanket rules** to the archetype, so the explicit relationship list stays reserved for individual decisions. A component opts in via its declaration (`"tier": "..."`); the policy's `tiers` section defines the vocabulary and rules.

```json
"tiers": {
  "library":          { },
  "adapter":          { "mayDependOn": ["core"] },
  "core":             { "mayNotDependOn": ["go:github.com/some-provider/*"] },
  "platform-app":     { "mayDependOn": ["adapter", "core", "library"] }
}
```

## Allow rules: `mayDependOn`

An observed edge A → B is tier-authorized when both components carry declared tiers and A's tier lists B's tier. Three properties are deliberate and non-configurable:

- **No transitivity.** `app → adapter` plus `adapter → core` does **not** authorize `app → core`. Skipping a layer is an architecture bypass and always requires an explicit relationship.
- **No self-tier match.** A tier may never list itself (`validate` rejects it): same-tier peers always require explicit wiring, which keeps peers from coupling by default.
- **Foundation is orthogonal.** There is no blanket "→ everything in tier X for everyone" shape. Universal consumability is the per-component [`foundation` status](model.md#status) — adding a component to a tier never silently widens what *unrelated* components may depend on.

A useful pattern is **two levels of foundation**: `foundation` status for the handful of utility libraries *everyone* (including other libraries) may use, and a `library` tier that only *application* tiers may blanket-consume — membership in each being a separate, reviewable declaration.

## Deny rules: `mayNotDependOn`

Deny rules take locator **patterns**, and they are the model's only rule that governs **third-party** targets — everything else deliberately ignores out-of-scope dependencies. The motivating case: an abstraction core must not import provider SDKs; those belong in provider adapters.

```json
"core": {
  "mayNotDependOn": ["go:github.com/anthropics/*", "go:github.com/openai/*"]
}
```

A deny **overrides every allow**, explicit relationships included. The one escape valve is an unexpired [exception](policy.md#exceptions) for the exact edge — visible, justified, time-bounded. Denied-but-unobserved pairs produce no findings.

## Evaluation order

For an observed in-scope edge: deny rules run first; then the foundation invariant and deprecated no-new-dependents checks; then authorization via explicit relationship → foundation target → tier allow → exception. Blanket tier rules never generate unused-allowance findings — they are permissions, not per-edge declarations.
