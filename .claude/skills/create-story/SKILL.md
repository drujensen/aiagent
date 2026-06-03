---
name: create-story
description: Create a detailed user story for the aiagent project with acceptance criteria and technical breakdown. Uses the story-refiner agent to explore the codebase and produce a structured story artifact.
user-invocable: true
argument-hint: "as a user I want to X so that Y"
context: fork
---

## Create Story

Create a user story for: <skill-args>

### Instructions

Use the Agent tool with `subagent_type: "story-refiner"` to:

1. Explore the aiagent codebase to understand what already exists related to this story
2. Ask the user focused clarifying questions in batches of 3–6 until all unknowns are resolved
3. Produce a complete user story artifact

### Expected Output

The story-refiner will produce:

**Refined Story:**
- Goal: one-sentence description of what the user wants and why
- Acceptance Criteria: specific, testable bullet points
- Scope Boundaries: in scope / out of scope
- Known Constraints: technical and business constraints
- Resolved Unknowns: questions asked → answers given
- Definition of Done: checklist including `go test ./... -race` and storage path verification

**Relevant Codebase Context:**
- Table of files, services, interfaces, and patterns directly relevant to this story
- Patterns to follow (2–3 existing implementations to read as examples)
- Watch-out-for: gotchas and near-misses

### Important
- This is the input to the `/design-review` skill
- The story-refiner does NOT propose solutions — it only clarifies
- Save the output — it is passed verbatim to the architect and developer
