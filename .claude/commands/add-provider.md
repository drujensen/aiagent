---
description: Scaffold a new AI provider integration. Creates the integration file, registers it in the factory, and adds default provider data.
---

Add a new AI provider integration to aiagent.

Provider details: $ARGUMENTS

## Steps

1. Read `internal/impl/integrations/aimodel_factory.go` to understand how providers are registered
2. Read an existing provider (e.g., `internal/impl/integrations/xai.go`) to understand the pattern
3. Read `internal/impl/integrations/generic.go` for the base `AIModelIntegration` struct
4. Create `internal/impl/integrations/<provider>.go`:
   - Embed or extend `AIModelIntegration`
   - Override only what differs from OpenAI-compatible behavior
   - Handle provider-specific auth headers, request/response formats
5. Register the new type in `internal/impl/integrations/aimodel_factory.go`
6. Add provider defaults to `internal/impl/defaults/defaults.go`:
   - Use a stable UUID for the ID
   - Set `Type`, `BaseURL`, `APIKeyName`
7. Update `.env.example` with the new API key variable
8. Run QA: `go fmt ./... && go vet ./... && go build . && go test ./...`

## Provider Struct Pattern

```go
type <Provider>Integration struct {
    *AIModelIntegration
}

func New<Provider>Integration(model *entities.Model) *<Provider>Integration {
    base := NewAIModelIntegration(model)
    // customize base if needed
    return &<Provider>Integration{AIModelIntegration: base}
}
```
