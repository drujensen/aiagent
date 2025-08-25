# CRUSH.md

## Build/Lint/Test Commands
- Build: go build -o aiagent main.go
- Run TUI: go run . [--storage=file|mongo]
- Run web server: go run . serve [--storage=file|mongo]
- Test all: go test ./...
- Test single: go test ./path/to/package -run TestFunctionName
- Lint: go vet ./... (no dedicated linter found; consider adding golangci-lint)
- Generate Swagger: swag init --dir ./internal/api --output ./internal/api
- Docker: docker-compose up --build

## Code Style Guidelines
- Architecture: Domain-Driven Design (DDD) with separation of domain, impl, tui, ui
- Formatting: Run `go fmt ./...` before commits
- Imports: Group into standard library, third-party, local; sort alphabetically within groups
- Naming: CamelCase for exported, lowerCamelCase for unexported; interfaces end with 'er'; constructors as NewXxx
- Types: Strong typing; use interfaces for abstractions in domain/interfaces
- Error Handling: Check err != nil; use fmt.Errorf for messages; log with zap; propagate errors
- Context: Pass context.Context to all repo/service methods
- Logging: Use zap for all logging, with Debug/Info/Error levels
- Testing: Write unit tests for services/repositories; use mocks

## Codebase Structure
- main.go: Entry point for TUI/web modes
- internal/domain/: Entities, services, interfaces, errors
- internal/impl/: Config, DB, repositories (JSON/Mongo), tools, integrations
- internal/tui/: Bubble Tea components
- internal/ui/: Web UI with Echo framework
- internal/api/: Swagger API docs

## Additional Notes
- No Cursor or Copilot rules found
- Follow Go best practices; keep code modular and extensible