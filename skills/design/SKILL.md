---
name: design
description: Turn a research findings summary into a concrete technical design - define the entities, interfaces, and services involved, following this project's DDD/Clean Architecture conventions and the standing CLAUDE.md/AGENTS.md rules.
metadata:
  phase: design
---

# Design

You have been activated to produce a technical design from a prior research findings summary (or from the request directly, if no research step preceded this one). Your job is to decide the shape of the solution, not to implement it.

## Instructions

1. **Enforce project conventions first.** Before proposing anything, read the repository's `CLAUDE.md` (root) and any relevant `AGENTS.md` files for the directories you expect to touch. Every design decision you make must be consistent with what those files say - constructor naming (`NewXxx`), interface naming (`...Repository`/`...Service`), the `context.Context`-first method signature convention, struct tag conventions (`json:"..." bson:"..."`), and the strict rule that `internal/domain/` code never imports `internal/impl/`.
2. **Reuse before inventing.** Identify existing entities, interfaces, services, or tools that already do something close to what's needed. Prefer extending or composing over duplicating. State explicitly what you are reusing and what is genuinely new.
3. **Define the shape of the change:**
   - New or modified entities (`internal/domain/entities/`)
   - New or modified interfaces (`internal/domain/interfaces/`)
   - New or modified services (`internal/domain/services/`)
   - New or modified implementations (`internal/impl/...`), noting that JSON and MongoDB repositories must both be updated when a repository interface changes
   - Any new tool, following the existing `entities.Tool` interface and `ToolFactoryEntry` registration pattern
4. **Call out concurrency and storage-duality implications.** If the change touches a repository, state how it will stay safe under concurrent access (the existing repositories use `sync.RWMutex` with a deep-copy-on-read/write contract) and how JSON and Mongo storage will be kept consistent.
5. **Keep it minimal.** Do not propose abstractions, patterns, or generality beyond what the request actually needs. Three similar lines of code is preferable to a premature abstraction, per this project's stated engineering principles.

## Output

Produce a design document with these sections:
- **Summary** - what is being built, in a few sentences
- **Reuse inventory** - existing code being reused or extended, with file paths
- **New/changed components** - entities, interfaces, services, tools, each with its target file path and a short description of its responsibility
- **Storage/concurrency notes** - how this stays consistent across JSON/Mongo and safe under concurrent access, if applicable
- **Convention compliance** - a short explicit note confirming the design follows the CLAUDE.md/AGENTS.md rules you read in step 1

Do not write implementation code here. Your output is the input to the plan phase.
