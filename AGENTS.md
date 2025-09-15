# Codebase Guidelines

## Build Commands
- Run TUI (default): `go run . [--storage=file|mongo]`
- Run web server: `go run . serve [--storage=file|mongo]`
- Run all tests: `go test ./...`
- Run specific test: `go test ./path/to/package -run TestFunctionName`
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
For the Build agent to ensure code quality, run these commands in order:
1. `go fmt ./...` - Format code
2. `go vet ./...` - Check for suspicious code
3. `go mod tidy` - Clean dependencies
4. `go build .` - Compile and check for build errors
5. `go test ./...` - Run all tests

If any command fails, analyze the errors, fix the issues, and repeat the workflow until all commands pass successfully.

## Code Style Guidelines
- **Architecture**: Domain-Driven Design (DDD) with clear separation between domain and impl
- **Error Handling**: Use detailed error messages with `fmt.Errorf`, always check and propagate errors
- **Context**: Pass context in all repository and service method signatures
- **Naming**:
  - Use `NewXxx` for constructor functions
  - Interfaces should end with `er` (e.g., `Repository`, `Service`)
  - Variables should be camelCase
- **Formatting**: Run `go fmt ./...` before committing (equivalent to gopls auto-format on save)
- **Imports**: Group standard library, external, and internal imports
- **Testing**: Write unit tests for all service methods, use mocks for dependencies

## Project Structure
- `main.go`: Root entry point handling TUI (default) and web server (serve) modes with --storage flag for file or mongo
- `internal/`: Core code (domain, impl, tui, ui)
- `internal/domain/`: Business entities, interfaces, services
- `internal/impl/`: External systems integration (config, database, repositories for JSON/Mongo, tools)
- `internal/tui/`: Terminal User Interface components using Bubble Tea
- `internal/ui/`: Web UI components

## Vimtea Configuration
The message view uses vimtea (github.com/kujtimiihoxha/vimtea) for Vim-like navigation in read-only mode:
- **Navigation**: h/j/k/l, w/W, b/B, gg/G, search (/), etc.
- **Visual Mode**: v, V, Ctrl+v for selection and clipboard copying
- **Line Numbers**: Ctrl+L to toggle line numbers on/off, or use `:zn` command
- **Commands**: :set number/:set nonumber to control line numbers
- **Disabled**: Insert mode (i, a, A, o, O) and all editing commands (d, c, s, etc.)
- **Clipboard**: Yank operations (y) copy to system clipboard
- **Real-time Updates**: Tool events append to editor content during processing
