---
name: test-review
description: Test writing + review loop for the aiagent project. The tester writes unit and integration tests using testify, then the tester-reviewer scores coverage and quality against the 6 pillars. Loops until score >= 8/10 or 3 iterations maximum.
user-invocable: true
argument-hint: "write tests for the new rate-limiting service"
context: fork
---

## Test Review

Write and review tests for: <skill-args>

### Instructions

You are orchestrating a Test Writing and Review cycle for the aiagent project.

**Phase 1: Test Writing**
1. Use the Agent tool with `subagent_type: "tester"` to write the tests
2. Pass it:
   - What to test (feature description or list of files changed)
   - The refined story's acceptance criteria (if available)
3. The tester will:
   - Read existing test files in the same packages for style reference
   - Write tests using testify (assert + mock)
   - Mock only at domain interface boundaries
   - Cover happy paths, failure paths, and edge cases
   - Run `go test ./... -race` to verify all tests pass

**Phase 2: Review**
4. Use the Agent tool with `subagent_type: "tester-reviewer"` to review the tests
5. Pass the tester's output (test files written, coverage results)
6. The reviewer will run `go test ./... -cover -race` and score against the 6 pillars

**Phase 3: Iterate**
7. If overall score **>= 8/10** (APPROVED): Report the test results and scores
8. If overall score **< 8/10** (REVISE):
   - Extract Critical Issues from the reviewer
   - Send issues back to the tester with specific gaps to fill
   - Re-review with the tester-reviewer
   - Repeat until APPROVED or 3 iterations maximum

**Phase 4: Report**
9. Present:
   - List of test files written/modified
   - Coverage report by package
   - Final pillar scores table
   - Acceptance criteria coverage: which criteria have tests, which don't

### Important
- Maximum 3 review iterations
- The tester WRITES TESTS; the reviewer only READS and SCORES
- Tests must use `t.TempDir()` for file-based tests, not hardcoded paths
- All tests must pass with `-race` flag
