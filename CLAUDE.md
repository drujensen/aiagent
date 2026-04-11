# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build .

# Run TUI (default)
go run . [--storage=file|mongo] [--global]

# Run web server
go run . serve [--storage=file|mongo]

# Refresh models from models.dev
go run . refresh

# Test
go test ./...
go test ./... -cover
go test ./... -race
go test ./internal/domain -run TestName   # single test

# QA workflow (run after any code changes before committing)
go fmt ./...
go vet ./...
go mod tidy
go build .
go test ./...

# Docker
docker-compose up --build
```

## Architecture

This project is a framework for building and interacting with AI agents. It follows Domain-Driven Design (DDD) with a strict separation between `internal/domain/` (business logic) and `internal/impl/` (external integrations).

### Core Separation

**Agent vs. Model**: The key design distinction — an **Agent** defines *behavior* (system prompt, tools, name), while a **Model** defines *inference* (provider, model name, temperature, context window). A **Chat** links one Agent + one Model and holds the message history. This allows switching models or agents mid-conversation independently.

### Package Layout

- `internal/domain/entities/` — core data types: `Agent`, `Model`, `Chat`, `Message`, `Provider`, `ToolData`, `Skill`
- `internal/domain/interfaces/` — repository and service interfaces (all implementations must satisfy these)
- `internal/domain/services/` — business logic: `ChatService`, `AgentService`, `ModelService`, `ProviderService`, `ToolService`, `SkillService`, `ModelRefreshService`
- `internal/impl/integrations/` — AI provider clients; `AIModelIntegration` is the base OpenAI-compatible implementation, with provider-specific subclasses for Anthropic, Google, xAI, etc.
- `internal/impl/repositories/json/` — file-based storage to `.aiagent/storage/` (default) or `~/.aiagent/storage/` (with `--global`)
- `internal/impl/repositories/mongo/` — MongoDB-backed storage; both sets implement identical domain interfaces
- `internal/impl/tools/` — tool implementations: `BashTool`, `FileReadTool`, `FileWriteTool`, `WebSearchTool`, `BrowserTool`, `VisionTool`, `CompressionTool`, `TodoTool`, etc.
- `internal/impl/config/` — loads `~/.aiagent/config.yaml` for API keys and provider settings
- `internal/impl/defaults/` — seed data for built-in providers, agents, and tools
- `internal/tui/` — Bubble Tea terminal UI
- `internal/ui/` — Echo web server with embedded static files and WebSocket for real-time updates

### Dependency Flow

```
main.go → wires up repositories + services → passes to tui/ or ui/
domain/services → domain/interfaces (never import impl directly)
impl/ → implements domain/interfaces
```

### Storage

- Default: `.aiagent/storage/*.json` in the current directory
- Global: `~/.aiagent/storage/` (pass `--global` flag)
- MongoDB: set `MONGO_URI` env var and pass `--storage=mongo`

### Environment Variables

Copy `.env.example` to `.env`. Key variables: `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GOOGLE_API_KEY`, `XAI_API_KEY`, `TAVILY_API_KEY` (web search), `MONGO_URI`.

## Code Conventions

- Constructors named `NewXxx`; interfaces end with `Repository` or `Service`
- All repository/service methods take `context.Context` as first argument
- Struct tags: `json:"fieldName" bson:"fieldName"` on all entities
- Error wrapping: `fmt.Errorf("failed to %s: %w", operation, err)`
- Interfaces defined in `domain/`, implementations in `impl/`; never reverse this dependency
