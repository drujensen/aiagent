---
description: Scaffold a new tool implementation for the aiagent project. Prompts for tool name and description, then creates the file following project conventions.
---

Create a new tool implementation for the aiagent project.

The user wants to add a tool named: $ARGUMENTS

## Steps

1. Read `internal/impl/tools/tool_factory.go` to understand how tools are registered
2. Read an existing simple tool (e.g., `internal/impl/tools/file_read.go`) to understand the interface
3. Read `internal/domain/interfaces/` to find the Tool interface definition
4. Create `internal/impl/tools/<toolname>.go` implementing the Tool interface
5. Register the new tool in `internal/impl/tools/tool_factory.go`
6. If it should be a default tool, add it to `internal/impl/defaults/defaults.go`
7. Write a test file at `internal/impl/tools/<toolname>_test.go`
8. Run the QA workflow: `go fmt ./... && go vet ./... && go build . && go test ./...`

## Naming Conventions
- File: `internal/impl/tools/<name>.go` (snake_case)
- Struct: `<Name>Tool` (PascalCase)
- Constructor: `New<Name>Tool`
- Tool name string (used in JSON): the human-readable name that agents will see

Follow error wrapping pattern: `fmt.Errorf("failed to %s: %w", operation, err)`
