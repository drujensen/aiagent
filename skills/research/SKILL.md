---
name: research
description: Investigate a problem or feature request before any design or implementation work begins - explore the existing codebase, identify the relevant files and patterns, and produce a findings summary grounded in what was actually found.
metadata:
  phase: research
---

# Research

You have been activated to research a problem or feature request before any design or implementation work begins. Your job is to build an accurate, evidence-based picture of the current state - not to propose a solution and not to write code.

## Instructions

1. **Restate the request** in your own words in one or two sentences, so downstream steps can confirm you understood it correctly.
2. **Locate the relevant code.** Use Read, Grep, and Glob to find the files, packages, and existing patterns that this request touches. Cite specific file paths and line numbers for every claim you make - do not describe code you have not actually read.
3. **Identify existing patterns to follow.** This is a Go project using Domain-Driven Design with a strict separation between `internal/domain/` (business logic, no external imports) and `internal/impl/` (external integrations). Note which existing entities, services, repositories, or tools are analogous to what's being asked for.
4. **Surface constraints and risks.** Note anything that could make this harder than it looks: concurrency hazards, existing tests that encode assumptions, storage-format implications (JSON file repositories and MongoDB repositories must both stay consistent), or backward-compatibility concerns.
5. **List open questions.** If something is genuinely ambiguous and you cannot resolve it by reading the code, say so explicitly rather than guessing.

## Output

Produce a findings summary with these sections:
- **Request** - the one/two-sentence restatement
- **Relevant files** - a table of file paths with a one-line description of why each is relevant
- **Existing patterns** - what precedent already exists in the codebase for this kind of change
- **Constraints and risks** - anything that could complicate implementation
- **Open questions** - anything unresolved that the design phase needs to address

Do not propose a solution here. Do not write or edit any code. Your output is the input to the design phase, not a substitute for it.
