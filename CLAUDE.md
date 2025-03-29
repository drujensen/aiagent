# AI Agent Codebase Guide

## Build Commands
- Run HTTP server: `go run ./cmd/http`
- Run Console: `go run ./cmd/console`
- Run all tests: `go test ./...`
- Run specific test: `go test ./path/to/package -run TestFunctionName`
- Build and run with Docker: `docker-compose up --build`

## Code Style Guidelines
- **Architecture**: Domain-Driven Design (DDD) with clear separation between domain and infrastructure
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
- `cmd/`: Entry points for applications
- `internal/`: Core code (domain, infrastructure, UI)
- `domain/`: Business entities, interfaces, services
- `infrastructure/`: External systems integration
- `ui/`: User interface components
