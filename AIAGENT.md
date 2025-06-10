# AI Agent Codebase Guide

## Build Commands
- Run CLI: `go run ./cmd/aiagent/main.go [-storage=file|mongo]`
- Run web server: `go run ./cmd/aiagent/main.go serve [-storage=file|mongo]`
- Run all tests: `go test ./...`
- Run specific test: `go test ./path/to/package -run TestFunctionName`
- Build and run with Docker: `docker-compose up --build`

## Code Style Guidelines
- **Architecture**: Domain-Driven Design (DDD) with clear separation between domain and impl
- **Error Handling**: Use detailed error messages with `fmt.Errorf`, always check and propagate errors
- **Context**: Pass context in all repository and service method signatures
- **Naming**:
  - Use `NewXxx` for constructor functions
  - Interfaces should end with `er` (e.g., `Repository`, `Service`)
  - Variables should be camelCase
- **Formatting**: Run `go fmt ./...` before committing
- **Imports**: Group standard library, external, and internal imports
- **Testing**: Write unit tests for all service methods, use mocks for dependencies

## Project Structure
- `cmd/server/main.go`: Entry points for applications
- `internal/`: Core code (api, domain, impl, UI)
- `internal/api/`: API handlers and routes
- `internal/domain/`: Business entities, interfaces, services
- `internal/impl/`: External systems integration
- `internal/ui/`: User interface components
