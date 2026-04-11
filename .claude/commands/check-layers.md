---
description: Verify that the DDD layer boundaries are intact — domain never imports impl. Reports any violations.
---

Check that the DDD dependency rules are not violated in the aiagent project.

## Rule

`internal/domain/` must NEVER import from `internal/impl/`.
`internal/impl/` may import from `internal/domain/`.
`internal/tui/` and `internal/ui/` may import from both.

## Check Commands

Run these to find violations:

```bash
# Find any domain file that imports from impl
grep -r "drujensen/aiagent/internal/impl" internal/domain/ --include="*.go" -l

# Show full import context for any violations
grep -r "drujensen/aiagent/internal/impl" internal/domain/ --include="*.go" -n
```

Also verify:
1. All domain interfaces are defined in `internal/domain/interfaces/` (not in impl)
2. All entity constructors follow `NewXxx` naming
3. All entities have both `json` and `bson` struct tags
4. No naked errors (errors without wrapping context)

Report any violations found and suggest the correct fix (e.g., move the dependency to an interface, inject it via constructor).
