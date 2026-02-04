# Agentic Coding Guidelines for aiagent

This file provides guidelines for agentic coding agents working in this Go codebase. Focus on build/test commands, code quality, and style consistency.

## Build Commands
- Run TUI (default): `go run . [--storage=file|mongo]`
- Run web server: `go run . serve [--storage=file|mongo]`
- Run all tests: `go test ./...`
- Run specific test: `go test ./internal/domain -run TestUserService`
- Run tests with coverage: `go test ./internal/domain -cover`
- Build and run with Docker: `docker-compose up --build`

## Development Commands
- **Lint**: `go fmt ./...` - Format Go code
- **Vet**: `go vet ./...` - Report suspicious constructs
- **Mod tidy**: `go mod tidy` - Clean up module dependencies
- **Build**: `go build .` - Compile the project
- **Test with coverage**: `go test ./... -cover` - Run tests with coverage
- **Test verbose**: `go test ./... -v` - Run tests with verbose output
- **Race detection**: `go test ./... -race` - Run tests with race detection
- **Benchmark**: `go test ./... -bench=.` - Run benchmarks

## Quality Assurance Workflow
Agents should run this workflow after any code changes:
1. `go fmt ./...` - Format code
2. `go vet ./...` - Check for suspicious code
3. `go mod tidy` - Clean dependencies
4. `go build .` - Compile and check for build errors
5. `go test ./...` - Run all tests

If any command fails, analyze errors, fix issues, and repeat until all pass. Run this before committing changes.

## Code Style Guidelines
- **Architecture**: Domain-Driven Design (DDD) with clear separation between domain and impl
- **Error Handling**: Use detailed error messages with `fmt.Errorf`, always check and propagate errors. Wrap errors with context using `fmt.Errorf("failed to %s: %w", operation, err)`
- **Context**: Pass context.Context in all repository and service method signatures
- **Naming**:
  - Use `NewXxx` for constructor functions
  - Interfaces should end with `er` (e.g., `Repository`, `Service`)
  - Variables should be camelCase
  - Struct fields should be PascalCase for exported fields
- **Formatting**: Run `go fmt ./...` before committing (equivalent to gopls auto-format on save)
- **Imports**: Group standard library, external, and internal imports. Use blank lines to separate groups
- **Types and Structs**:
  - Use struct tags for JSON/BSON: `json:"fieldName" bson:"fieldName"`
  - Define interfaces in domain layer, implementations in impl
  - Use dependency injection for services
- **Testing**: Write unit tests for all service methods, use mocks for dependencies. Test files should end with `_test.go`
- **Logging**: Use structured logging with context. Avoid fmt.Printf; use proper logging libraries
- **Concurrency**: Use channels and goroutines carefully; avoid race conditions
- **Security**: Never log sensitive data (API keys, passwords). Use environment variables for secrets

## Project Structure
- `main.go`: Root entry point handling TUI (default) and web server (serve) modes with --storage flag for file or mongo
- `internal/`: Core code (domain, impl, tui, ui)
- `internal/domain/`: Business entities, interfaces, services
- `internal/impl/`: External systems integration (config, database, repositories for JSON/Mongo, tools)
- `internal/tui/`: Terminal User Interface components using Bubble Tea
- `internal/ui/`: Web UI components

## Commit and PR Practices
- Run QA workflow before committing
- Use descriptive commit messages focusing on "why" not "what"
- For PRs: Ensure tests pass, code is formatted, and vet checks succeed
- When using AI models: Test interactions thoroughly before merging

## Troubleshooting

### Common Development Issues

#### Build failures
**Problem**: `go build` fails
**Solution**:
1. Run `go mod tidy` to clean dependencies
2. Ensure Go version 1.23+ is installed
3. Check for missing dependencies
4. Run `go vet` and `go fmt` for code issues

#### Test failures
**Problem**: `go test` fails
**Solution**:
1. Check test environment setup
2. Verify all dependencies are available
3. Run tests individually to isolate issues: `go test ./package -run TestName`
4. Check for race conditions with `go test -race`

#### Settings not persisting
**Problem**: Changes don't save between restarts
**Solution**:
1. Verify storage path permissions
2. Check storage configuration (--storage=file or --storage=mongo)
3. Ensure MongoDB is running (if using mongo storage)
4. Check disk space availability

### Getting Help

If issues persist:
1. **Check Logs**: Enable debug logging for more details
2. **GitHub Issues**: Report bugs at https://github.com/drujensen/aiagent/issues
3. **Documentation**: Review [README.md](README.md) and [AIAGENT.md](AIAGENT.md)
