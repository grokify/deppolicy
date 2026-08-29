# Policy Documents

The policy is the **intent** side of the three-artifact contract: what dependency edges *may* exist, independent of what the code currently does. It is JSON in git, authored and reviewed like an API contract — a policy diff that adds `a can_depend_on b` is an architecture change and should be reviewed as one (see [`policy-diff`](../reference/commands.md#policy-diff-old-policyjson-new-policyjson)).

## Default deny

Within the governed scope, a direct dependency from A to B requires one of:

- an explicit `can_depend_on` **relationship**, or
- B has [`foundation` status](model.md#status), or
- A's declared tier lists B's tier in a [tier rule](tiers.md), or
- an unexpired **exception**.

Absence of authorization is denial. There is no `"default": "deny"` knob — it is the specification, not a setting.

## Relationships

```json
{
  "source": "go:github.com/yourorg/app",
  "relation": "can_depend_on",
  "target": "go:github.com/yourorg/engine",
  "reason": "the app drives the engine's public API",
  "decision": "ADR-0042"
}
```

`reason` is **required** — every permanent boundary crossing must justify itself. `decision` optionally points at the ADR or record behind it, turning documentation into an executable constraint.

## Exceptions

A time-bounded allowance for exactly one edge, for migrations and other temporary states:

```json
{
  "source": "go:github.com/yourorg/app",
  "target": "go:github.com/yourorg/legacy-store",
  "effect": "allow",
  "expires": "2027-03-31",
  "reason": "migration to the new storage API, tracked in ..."
}
```

Both `expires` and `reason` are required, so temporary debt stays visible instead of quietly becoming permanent. Expired exceptions stop authorizing and are flagged by `validate`. An unexpired exception is also the sole escape valve for a [tier deny rule](tiers.md#deny-rules-maynotdependon).

## Transitive dependencies are not governed

If A → B and B → C are each authorized, A needs no authorization to reach C transitively:

> We govern boundaries, not reachability. Every **direct** internal boundary crossing must be explicitly authorized.

Governing reachability would make the policy the transitive closure of the graph — noisy and impossible to reason about. Global correctness composes from locally checked direct edges.

## One central policy

The policy is centrally authored; repositories contribute *facts* (scans), never local overrides. Local escape hatches would distribute the effective architecture across hundreds of repos — the opposite of the goal, which is that one document tells you how the ecosystem fits together.
