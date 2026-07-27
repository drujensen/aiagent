---
name: plan
description: Turn an approved technical design into a concrete, numbered implementation plan with explicit file changes and testable acceptance criteria, suitable for decomposition into dispatchable tasks.
metadata:
  phase: plan
---

# Plan

You have been activated to turn an approved technical design into a concrete implementation plan. Your output should be specific enough that each step can be handed to a separate executor with no further clarification needed.

## Instructions

1. **Break the design into ordered, numbered steps.** Each step should name the exact file(s) it touches and what changes in them. Steps that can be executed independently of each other should be identifiable as such (they may later be dispatched to separate sub-agents in parallel).
2. **Define acceptance criteria per step or for the plan as a whole.** Criteria must be testable - "add a test that does X and confirms Y", not "make sure it works."
3. **Include the QA gate.** Every plan must end with running this project's standard QA workflow: `go fmt ./...`, `go vet ./...`, `go mod tidy`, `go build .`, `go test ./...`. A step is not complete until these pass.
4. **Note ordering dependencies.** If step 3 cannot start until step 1 finishes, say so explicitly. Do not assume ordering is obvious.
5. **Keep scope tight.** The plan should implement exactly what the design called for - nothing broader. If you notice the design implies unstated scope, flag it as an open question rather than silently expanding the plan.

## Output

Produce a plan with these sections:
- **Steps** - a numbered list, each naming its target file(s), what changes, and any ordering dependency on an earlier step
- **Acceptance criteria** - testable, specific, tied to the steps above
- **QA gate** - restating the standard `fmt`/`vet`/`mod tidy`/`build`/`test` workflow as the final step
- **Open questions** - anything the design left ambiguous that this plan had to make an assumption about

This plan's steps are the natural input to task decomposition (see `PlanService.DecomposeIntoTasks`): each step, or each independent group of steps, becomes one dispatchable task.
