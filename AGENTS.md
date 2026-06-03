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

Tested by Build Agent on 2026-02-23

---

# ADLC — Agent Development Life Cycle

## The 6 Pillars

Every reviewer scores work against these 6 pillars. A score >= 8/10 is required to advance.

| Pillar | Key Concerns |
|--------|-------------|
| **Maintainability** | DDD layer boundaries, naming conventions, struct tags, no duplication, clear interfaces |
| **Reliability** | Error handling, context propagation, concurrent safety, fault tolerance |
| **Scalability** | No N+1 queries, stateless services, indexed queries, both storage backends |
| **Usability** | RESTful API design, TUI/WebSocket behavior, clear error messages, performance |
| **Security** | API key protection, command injection, path traversal, MongoDB injection, input validation |
| **Quality** | Test coverage (happy + failure + edge), race-safe, acceptance criteria verified |

## Doer Agents

| Agent | Role | Tools | Model |
|-------|------|-------|-------|
| `story-refiner` | Interactive story clarification — explores codebase, asks question batches | Read, Grep, Glob, Bash | sonnet |
| `planner` | Implementation planning — Reuse Inventory + numbered steps | Read, Grep, Glob, Bash | sonnet |
| `architect` | Software design — DDD patterns, RESTful API standards | Bash, Read, Glob, Grep, WebSearch, WebFetch | opus |
| `developer` | Go implementation — step-by-step with test gates | Bash, Read, Edit, Write, Glob, Grep | sonnet |
| `tester` | Test writing — testify, mock at domain boundaries | Bash, Read, Edit, Write, Glob, Grep | sonnet |
| `security` | Threat modeling, vulnerability remediation | Bash, Read, Glob, Grep, WebSearch, WebFetch | sonnet |
| `dba` | Schema design, JSON+MongoDB storage duality | Bash, Read, Edit, Write, Glob, Grep | sonnet |
| `devops` | CI/CD pipelines, Docker, release automation | Bash, Read, Edit, Write, Glob, Grep | sonnet |

## Reviewer Agents (all read-only)

| Reviewer | Pairs With | Model | Focus |
|----------|-----------|-------|-------|
| `architect-reviewer` | architect | opus | Design correctness, DDD compliance, RESTful API |
| `planner-reviewer` | planner | sonnet | Plan completeness, file path accuracy, step ordering |
| `developer-reviewer` | developer | sonnet | Code quality, duplication check, layer boundaries |
| `tester-reviewer` | tester | sonnet | Coverage, failure paths, race safety |
| `security-reviewer` | security | sonnet | Injection, traversal, key leakage, MongoDB safety |
| `dba-reviewer` | dba | sonnet | Storage duality, struct tags, backward compatibility |
| `devops-reviewer` | devops | sonnet | Build targets, secret handling, go version consistency |

## ADLC Pipeline

```
Story Refinement (interactive loop with user)
       ↓
Design → architect-reviewer (loop up to 3x, need >= 8/10)
       ↓
Implementation Plan → planner-reviewer (loop up to 3x, need >= 8/10)
       ↓
  *** HUMAN APPROVAL CHECKPOINT — type "proceed" to continue ***
       ↓
Implementation → developer-reviewer (loop up to 3x, need >= 8/10)
       ↓
Testing → tester-reviewer (loop up to 3x, need >= 8/10)
       ↓
Security → security-reviewer (loop up to 3x, need >= 8/10)
       ↓
Database → dba-reviewer (conditional: if schema changed)
       ↓
DevOps → devops-reviewer (conditional: if pipeline/Docker changed)
       ↓
Acceptance Criteria Verification
       ↓
ADLC Summary Report
```

## Review Score Card Format

Every reviewer produces this exact format:

```
| Pillar         | Score | Key Findings |
|----------------|-------|-------------|
| Maintainability | X/10 | ...         |
| Reliability     | X/10 | ...         |
| Scalability     | X/10 | ...         |
| Usability       | X/10 | ...         |
| Security        | X/10 | ...         |
| Quality         | X/10 | ...         |
| Overall         | X.X/10 |           |

Verdict: APPROVED (>= 8.0) / REVISE (< 8.0)
```

## Skills Reference

### Full Pipeline
| Command | Description |
|---------|-------------|
| `/adlc <feature>` | Full pipeline: story → design → plan → implement → test → security → DB → DevOps |

### Pre-Coding
| Command | Description |
|---------|-------------|
| `/refine-story <idea>` | Interactive story clarification with codebase exploration |
| `/create-story <idea>` | Create a user story with acceptance criteria |
| `/plan-sprint <feature>` | Break a feature into ordered implementable stories |

### Review Loops
| Command | Description |
|---------|-------------|
| `/design-review <feature>` | architect → architect-reviewer loop |
| `/peer-review <task>` | developer → developer-reviewer loop |
| `/test-review <what>` | tester → tester-reviewer loop |
| `/security-review <what>` | security → security-reviewer loop |
| `/dba-review <what>` | dba → dba-reviewer loop |
| `/devops-review <what>` | devops → devops-reviewer loop |

### Standalone
| Command | Description |
|---------|-------------|
| `/review-code <file>` | One-shot code review (no loop) |
| `/write-tests <file>` | Write tests without review loop |
| `/security-scan <path>` | One-shot security scan |
| `/deploy <version>` | Release readiness check |
