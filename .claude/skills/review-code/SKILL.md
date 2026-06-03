---
name: review-code
description: Quick one-shot code review for the aiagent project. The developer-reviewer checks DDD layer boundaries, Go idioms, error handling, storage duality, and test coverage. No iteration loop — immediate feedback.
user-invocable: true
argument-hint: "review internal/impl/tools/new_tool.go"
context: fork
---

## Code Review

Review the following: <skill-args>

### Instructions

Use the Agent tool with `subagent_type: "developer-reviewer"` to perform a one-shot code review.

Pass it:
- The file(s) or description of what to review
- Any relevant context (what feature this is for, what the acceptance criteria are)

The reviewer will:
1. Perform a mandatory duplication check first
2. Run automated checks: `go vet ./... && go build . && go test ./... -race`
3. Score against the 6 pillars (Maintainability, Reliability, Scalability, Usability, Security, Quality)
4. Produce a verdict: APPROVED or REVISE

### Output Format

The reviewer produces:
- Duplication check results
- Pillar scores table
- Verdict: APPROVED / REVISE
- Critical Issues (blocking)
- Warnings (non-blocking)
- Strengths

### Important
- This is a one-shot review with no iteration — use `/peer-review` for the full implementation + review loop
- Critical issues must be addressed before merging
- Warnings are recommendations
