---
name: write-tests
description: Write tests for the aiagent project without a review loop. The tester agent writes unit and integration tests using testify following the project's mock-at-domain-boundary pattern.
user-invocable: true
argument-hint: "write tests for internal/domain/services/model_service.go"
context: fork
---

## Write Tests

Write tests for: <skill-args>

### Instructions

Use the Agent tool with `subagent_type: "tester"` to write tests.

Pass it:
- What to test (file path, function name, or feature description)
- Acceptance criteria (if available — each criterion should map to at least one test)

The tester will:
1. Read the code under test thoroughly
2. Read existing test files in the same package for style reference
3. Write tests using testify (assert + mock)
4. Cover: happy paths, all error paths, edge cases
5. Mock only at domain interface boundaries
6. Run `go test ./... -race` to verify tests pass

### Test Patterns in This Project

- **Domain service tests**: mock repositories implementing domain interfaces via `testify/mock`
- **Tool tests**: real behavior, `t.TempDir()` for file isolation
- **Integration tests**: guarded with `if os.Getenv("API_KEY") == "" { t.Skip(...) }`
- **Test names**: `TestXxx_scenarioDescription`
- **Table-driven**: preferred for multiple input/output scenarios

### Output

The tester will report:
- Test files written/modified
- Coverage result: `go test ./pkg -cover`
- Race detection result: `go test ./pkg -race`
- List of acceptance criteria covered

### Important
- This is a standalone skill — use `/test-review` for the full test + review loop
- All new tests must pass with `-race` flag
