---
name: plan-sprint
description: Break a feature or epic into ordered, implementable user stories for the aiagent project. Uses the planner agent to analyze the feature and produce a sprint plan with story dependencies.
user-invocable: true
argument-hint: "feature: real-time collaboration on chats"
context: fork
---

## Sprint Planning

Plan the following feature into stories: <skill-args>

### Instructions

Use the Agent tool with `subagent_type: "planner"` to:

1. Explore the aiagent codebase to understand the current state of related functionality
2. Break the feature into ordered, implementable user stories
3. Identify dependencies between stories
4. Estimate complexity (S/M/L) based on number of layers touched

### Expected Output

The planner will produce a sprint plan:

```markdown
## Sprint Plan: [Feature Name]

### Stories (ordered by dependency)

**Story 1: [Title]** — Size: S/M/L
- Goal: [one sentence]
- Acceptance Criteria: [2–5 bullet points]
- Depends on: [none or Story N]
- Layers touched: [entities / interfaces / services / repositories / TUI / Web UI]

**Story 2: ...**
```

### Important
- Stories should be independently shippable where possible
- Each story should be small enough to implement and test in one session
- Dependencies must be explicit — a story that requires another story's entities or interfaces must list it
- Use `/refine-story` on each story before starting implementation
