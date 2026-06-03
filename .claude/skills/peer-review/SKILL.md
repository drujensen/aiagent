---
name: peer-review
description: Code implementation + review loop for the aiagent project. The developer implements changes following the approved plan, then the developer-reviewer scores the code against the 6 pillars including a mandatory duplication check. Loops until score >= 8/10 or 3 iterations maximum.
user-invocable: true
argument-hint: "implement the rate-limiting feature per the approved plan"
context: fork
---

## Peer Review

Implement and review: <skill-args>

### Instructions

You are orchestrating a Code Implementation and Review cycle for the aiagent project.

**Phase 1: Implementation**
1. Use the Agent tool with `subagent_type: "developer"` to implement the changes
2. Pass it:
   - The approved implementation plan (Reuse Inventory + numbered steps)
   - The approved design document
   - The refined story (acceptance criteria)
3. The developer will:
   - First establish a green test baseline: `go test ./...`
   - Read the Reuse Inventory and 2–3 similar existing features
   - Implement one step at a time, running `go build . && go test ./...` after each step
   - Run the full QA workflow when done: `go fmt ./... && go vet ./... && go mod tidy && go build . && go test ./... -race`

**Phase 2: Review**
4. Use the Agent tool with `subagent_type: "developer-reviewer"` to review the implementation
5. Pass the developer's summary of changes to the reviewer
6. The reviewer will:
   - First perform a mandatory duplication check (before scoring any pillar)
   - Run `go vet ./... && go build . && go test ./... -race`
   - Score against all 6 pillars

**Phase 3: Iterate**
7. If overall score **>= 8/10** (APPROVED): Report the implementation and scores
8. If overall score **< 8/10** (REVISE):
   - Extract Critical Issues from the reviewer
   - Send issues back to the developer via Agent tool with specific fix instructions
   - Re-review with the developer-reviewer
   - Repeat until APPROVED or 3 iterations maximum

**Phase 4: Report**
9. Present:
   - Summary of files changed
   - Final pillar scores table
   - Iteration count and what changed
   - Any remaining warnings (non-blocking)

### Important
- Maximum 3 review iterations
- If not approved after 3 iterations, present work with remaining issues flagged
- The developer WRITES CODE; the reviewer only READS and SCORES
- The developer never skips test gates between steps
