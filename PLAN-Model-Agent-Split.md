# Plan: Split Agent into Agent + Model with Provider Refresh

**Status**: 🔄 In Progress  
**Last Updated**: 2026-01-11  
**No Backward Compatibility Required** ✓

---

## Executive Summary

**Objective**: Split current `Agent` entity into two independent components:
- **Agent**: Manages behavior (system prompt, toolset)
- **Model**: Manages inference configuration (provider, model name, temperature, context window, etc.)

**Benefits**:
- Switch models independently of agents without changing system prompt or tools
- Maintain consistent agent behavior across different models
- Easy provider/model refresh from models.dev API
- Test same prompt across different models quickly

**Key Decisions**:
- Model entity will be independent and reference Provider (not 1:1)
- Chat history is preserved when switching models
- Use models.dev API for centralized provider/model data (like opencode)
- TUI: Ctrl+M for model switching (consistent with Ctrl+A for agents)
- **No backward compatibility** - users will need to re-create their data

---

## Architecture Overview

### Current State

**Agent Entity** (tightly coupled):
```go
type Agent struct {
    ID              string       // Unique ID
    Name            string       // Display name
    ProviderID      string       // Reference to Provider
    ProviderType    ProviderType // Denormalized
    Endpoint        string       // API endpoint (from Provider)
    Model           string       // Model name
    APIKey          string       // Provider API key
    SystemPrompt    string       // Behavioral instructions
    Temperature     *float64     // Inference parameter
    MaxTokens       *int         // Inference parameter
    ContextWindow   *int         // Inference parameter
    ReasoningEffort string       // Inference parameter
    Tools           []string     // Available tool names
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**Problems**:
- Tight coupling between behavior and inference
- Changing model requires creating new agent or editing existing one
- Redundancy: Multiple agents needed for same behavior with different models
- No easy way to update available models/pricing from providers

---

### Proposed State

**New Entity: Model** (inference only):
```go
type Model struct {
    ID              string       // Unique ID
    Name            string       // Display name (e.g., "GPT-4o High Temp")
    ProviderID      string       // Reference to Provider
    ProviderType    ProviderType // Denormalized for access
    ModelName       string       // Actual model name (e.g., "gpt-4o")
    APIKey          string       // Provider API key
    Temperature     *float64     // Inference parameter
    MaxTokens       *int         // Inference parameter
    ContextWindow   *int         // Inference parameter
    ReasoningEffort string       // Inference parameter
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**Updated Agent Entity** (behavior only):
```go
type Agent struct {
    ID              string       // Unique ID
    Name            string       // Display name
    SystemPrompt    string       // Behavioral instructions
    Tools           []string     // Available tool names
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**Updated Chat Entity** (references both):
```go
type Chat struct {
    ID              string       // Unique ID
    AgentID         string       // Reference to Agent (behavior)
    ModelID         string       // Reference to Model (inference) - NEW
    Name            string       // Chat name
    Messages        []Message    // Message history (preserved on model switch)
    Usage           *ChatUsage   // Token/cost tracking
    CreatedAt       time.Time
    UpdatedAt       time.Time
    Active          bool
}
```

**Relationships**:
```
┌─────────────┐
│   Provider  │
│             │
├─────────────┤
│ ID          │
│ Name        │
│ Type        │
│ BaseURL     │
│ Models[]    │
└─────────────┘
        │
        │ (1 to many)
        │
┌─────┴────────┐
│              │
│  ┌──────────┴──────────┐
│  │                     │
│┌─▼──────────┐     ┌───▼──────────┐
││   Agent     │     │    Model     │
││            │     │             │
│├────────────┤     ├────────────┤
││ ID         │     │ ID          │
││ Name       │     │ Name        │
││ System...  │     │ ProviderID  │
││ Tools[]    │     │ ModelName   │
│└────────────┘     │ Temp, etc. │
│    │              └────────────┘
│    │                   │
│    │ (1 to many)       │ (1 to many)
│    │                   │
┌───┴──────────┐     ┌────▼──────────┐
│     Chat      │     │    Chat       │
│               │     │               │
├───────────────┤     ├───────────────┤
│ ID            │     │ ID            │
│ AgentID       │────▶│ AgentID       │
│ ModelID       │────▶│ ModelID       │
│ Name          │     │ Name          │
│ Messages[]    │     │ Messages[]    │
│ Usage         │     │ Usage         │
└───────────────┘     └───────────────┘

Note: Chat references both Agent (behavior) and Model (inference)
```

---

## Implementation Phases

### Phase 1: Entity & Repository Layer ✅ Estimated: 2-3 days

- [x] **1.1** Create `Model` entity in `internal/domain/entities/model.go`
  - [x] Define struct with all inference fields
  - [x] Add `NewModel()` constructor
  - [x] Implement `list.Item` interface methods (FilterValue, Title, Description)
  - [x] Write entity unit tests in `entities_test.go`

- [x] **1.2** Update `Agent` entity in `internal/domain/entities/agent.go`
  - [x] Remove: ProviderID, ProviderType, Endpoint, Model, APIKey, Temperature, MaxTokens, ContextWindow, ReasoningEffort
  - [x] Keep: ID, Name, SystemPrompt, Tools, CreatedAt, UpdatedAt
  - [x] Update `NewAgent()` constructor (remove removed parameters)
  - [x] Update entity unit tests

- [x] **1.3** Update `Chat` entity in `internal/domain/entities/chat.go`
  - [x] Add: ModelID field
  - [x] Update entity unit tests

- [x] **1.4** Create `ModelRepository` interface in `internal/domain/interfaces/model_repository.go`
  - [x] Define: CreateModel, UpdateModel, DeleteModel, GetModel, ListModels, GetModelsByProvider

- [x] **1.5** Implement JSON `ModelRepository` in `internal/impl/repositories/json/model_repository.go`
  - [x] Implement all interface methods
  - [x] Handle storage in `.aiagent/models.json`
  - [x] Write unit tests

- [ ] **1.6** Implement MongoDB `ModelRepository` in `internal/impl/repositories/mongo/model_repository.go`
  - [ ] Implement all interface methods
  - [ ] Create appropriate indexes
  - [ ] Write unit tests

- [x] **1.7** Update `ChatRepository` implementations
  - [x] Add `model_id` field to all CRUD operations
  - [x] Update JSON repository
  - [ ] Update MongoDB repository
  - [ ] Update unit tests

**Phase 1 Complete**: 6/7 tasks complete (86%)

---

### Phase 2: Service Layer 🔄 Estimated: 2-3 days

- [x] **2.1** Create `ModelService` in `internal/domain/services/model_service.go`
  - [x] Define interface
  - [x] Implement all methods
  - [ ] Add validation logic
  - [ ] Write unit tests

- [x] **2.2** Update `AgentService` in `internal/domain/services/agent_service.go`
  - [x] Remove provider/model field validation from CreateAgent
  - [x] Remove provider/model field validation from UpdateAgent
  - [ ] Update unit tests

- [x] **2.3** Update `ChatService` in `internal/domain/services/chat_service.go`
  - [x] Add `modelService` to struct
  - [x] Update `NewChat()` to accept model_id parameter
  - [x] Update `CreateChat()` to validate and store model_id
  - [x] Update `UpdateChat()` to handle model_id changes
  - [ ] **Critical**: Modify `SendMessage()`:
    - [ ] Get Model instead of Agent for inference settings
    - [ ] Use model.ProviderID to get Provider
    - [ ] Use model.ModelName, model.Temperature, etc. for AI calls
    - [ ] Keep agent.SystemPrompt and agent.Tools for behavior
  - [ ] Update unit tests
  - NOTE: `models.dev Client` and `ModelRefreshService` created separately (2.5 and 2.6)

- [ ] **2.4** Create `models.dev Client` in `internal/impl/modelsdev/client.go`
  - [x] Define structs for models.dev API response
  - [x] Implement `Fetch()` method
  - [x] Implement `GetCached()` method
  - [x] Implement caching logic in `.aiagent/providers-cache.json`
  - [x] Add User-Agent header
  - [x] Write unit tests

- [ ] **2.5** Create `ModelRefreshService` in `internal/domain/services/model_refresh_service.go`
  - [ ] Define interface
  - [ ] Implement RefreshAllProviders()
  - [ ] Implement RefreshProvider()
  - [ ] Add 24-hour cache check
  - [ ] Map models.dev data to Provider entities
  - [ ] Upsert Providers with new models
  - [ ] Write unit tests

- [ ] Write unit tests

**Phase 2 Status**: 4/5 tasks complete (80%)

---

### Phase 3: TUI Implementation ⏱️ Estimated: 2-3 days

- [ ] **3.1** Create `ModelView` component in `internal/tui/model_view.go`
  - [ ] Define struct (similar to AgentView)
  - [ ] Implement Init() with model fetching
  - [ ] Implement Update() with key handling
  - [ ] Implement View() with proper styling
  - [ ] Add mode support ("view" vs "switch")

- [ ] **3.2** Update `messages.go` in `internal/tui/`
  - [ ] Add: `startModelSwitchMsg` struct
  - [ ] Add: `modelSelectedMsg` struct with modelID field
  - [ ] Add: `modelsFetchedMsg` struct with models array

- [ ] **3.3** Update `TUI` in `internal/tui/tui.go`
  - [ ] Add `modelService` to struct
  - [ ] Add `modelView` component
  - [ ] Handle `startModelSwitchMsg` (state transition to models/list)
  - [ ] Handle `modelSelectedMsg` (update chat's model_id)
  - [ ] Initialize modelView in Init()
  - [ ] Add model state to Update() switch

- [ ] **3.4** Update `ChatView` in `internal/tui/chat_view.go`
  - [ ] Add `currentModel` field to struct
  - [ ] Add Ctrl+M key handler → return startModelSwitchMsg
  - [ ] Update footer display to show both Agent and Model info
  - [ ] Update `SetChatActive` to load Model based on model_id
  - [ ] Update chat initialization to load Model

- [ ] **3.5** Update `AgentView` in `internal/tui/agent_view.go`
  - [ ] Remove model-related fields from display
  - [ ] Simplify to show only behavior fields (name, system prompt, tools)

- [ ] **3.6** Update `HelpView` in `internal/tui/help_view.go`
  - [ ] Add Ctrl+M shortcut to help text
  - [ ] Document model switching flow

- [ ] **3.7** Manual TUI Testing
  - [ ] Verify Ctrl+M opens model list
  - [ ] Test model filtering/searching
  - [ ] Test model selection switches chat's model
  - [ ] Verify message history preserved after switch
  - [ ] Verify footer shows correct model info (Agent: X | Model: Y)
  - [ ] Test agent creation (no model fields shown)

**Phase 3 Complete**: TUI model switching works, history preserved

---

### Phase 4: Web UI Implementation ⏱️ Estimated: 2-3 days

- [ ] **4.1** Create Model Controller in `internal/ui/controllers/model_controller.go`
  - [ ] Implement `ListModels()` handler
  - [ ] Implement `CreateModel()` handler
  - [ ] Implement `EditModel()` handler
  - [ ] Implement `DeleteModel()` handler
  - [ ] Implement `GetModelsByProvider()` handler

- [ ] **4.2** Create Model Pages in `internal/ui/templates/`
  - [ ] Create `models.html` - list all models
  - [ ] Create `model_new.html` - create new model form
  - [ ] Create `model_edit.html` - edit model form
  - [ ] Add model selection dropdown from Provider templates

- [ ] **4.3** Update Agent Pages
  - [ ] Remove model selection from agent creation form
  - [ ] Remove model display from agent list
  - [ ] Simplify to show only behavior fields (name, system prompt, tools)

- [ ] **4.4** Update Chat Pages
  - [ ] Add model selector to new chat creation
  - [ ] Add model selection button to chat view
  - [ ] Display current model in chat header
  - [ ] Update templates to show agent + model info

- [ ] **4.5** Create API Endpoints
  - [ ] `GET /api/models` - list all models
  - [ ] `POST /api/models` - create model
  - [ ] `GET /api/models/:id` - get model
  - [ ] `PUT /api/models/:id` - update model
  - [ ] `DELETE /api/models/:id` - delete model
  - [ ] `GET /api/models/provider/:id` - list models by provider

- [ ] **4.6** Update Chat API Endpoints
  - [ ] `POST /api/chats` - require model_id parameter
  - [ ] `PUT /api/chats/:id` - allow updating model_id

- [ ] **4.7** Add Provider Refresh Endpoints
  - [ ] `POST /api/providers/refresh` - refresh all providers
  - [ ] `POST /api/providers/:id/refresh` - refresh specific provider
  - [ ] `GET /api/providers/refresh/status` - get last refresh time

- [ ] **4.8** Manual Web UI Testing
  - [ ] Test model list page
  - [ ] Test model creation from template
  - [ ] Test model editing
  - [ ] Test model deletion
  - [ ] Test chat creation with both agent and model
  - [ ] Test model switching in chat view
  - [ ] Test provider refresh

**Phase 4 Complete**: All web UI pages work, API endpoints functional

---

### Phase 5: Models.dev Integration ✅ Estimated: 1-2 days

- [x] **5.1** Complete models.dev Client in `internal/impl/modelsdev/client.go`
  - [x] Test actual API connection to models.dev
  - [x] Validate response parsing
  - [x] Test error handling

- [x] **5.2** Implement Caching
  - [x] Create cache file on fetch
  - [x] Implement cache age check (24 hours)
  - [x] Implement cache refresh logic
  - [x] Test cache fallback when API unavailable

- [ ] **5.3** Integrate with Provider Repository
  - [ ] Map models.dev Provider to entities.Provider
  - [ ] Map models.dev Model to ModelPricing
  - [ ] Implement upsert logic for Providers
  - [ ] Update Provider entities with new models/pricing

- [ ] **5.4** Add CLI Command for Manual Refresh
  - [ ] Add command: `aiagent refresh providers`
  - [ ] Add command: `aiagent refresh provider <id>`
  - [ ] Add progress indicators
  - [ ] Add error handling

- [ ] **5.5** Add Web UI for Refresh
  - [ ] Add "Refresh All Providers" button to providers page
  - [ ] Add "Refresh" button to individual provider cards
  - [ ] Display last refresh time
  - [ ] Show refresh status/progress

- [ ] **5.6** Testing
  - [ ] Test manual refresh via CLI
  - [ ] Test refresh via Web UI
  - [ ] Test automatic refresh (mock time)
  - [ ] Test cache behavior
  - [ ] Verify providers updated with new models

**Phase 5 Complete**: Providers/models update from models.dev successfully

---

### Phase 6: Update Defaults ⏱️ Estimated: 1 day

- [ ] **6.1** Update `defaults/defaults.go`
  - [ ] Clean up `DefaultAgents()`:
    - [ ] Remove ProviderID, ProviderType, Endpoint, Model, APIKey fields
    - [ ] Keep only Name, SystemPrompt, Tools
  - [ ] Create `DefaultModels()` function:
    - [ ] Create 3-4 model presets per provider
    - [ ] Example: "GPT-4o Standard", "GPT-4o Creative", "Claude 3.5 Balanced"
    - [ ] Use reasonable default parameters

- [ ] **6.2** Update `initializeDefaults()` in `main.go`
  - [ ] Add Model initialization
  - [ ] Create default models on first run
  - [ ] Verify no conflicts with existing data

- [ ] **6.3** Test Default Data
  - [ ] Fresh install test: Verify defaults created correctly
  - [ ] Test agent creation uses defaults
  - [ ] Test model creation uses templates

**Phase 6 Complete**: Default data restructured and functional

---

### Phase 7: Testing & Documentation ⏱️ Estimated: 2-3 days

- [ ] **7.1** Write Integration Tests
  - [ ] Test model switching flow (chat + model → switch model → verify history)
  - [ ] Test agent + model creation flow
  - [ ] Test provider refresh flow
  - [ ] Test chat creation with both agent and model

- [ ] **7.2** Manual Testing Checklist - TUI
  - [ ] Ctrl+M opens model list
  - [ ] Can filter/search models
  - [ ] Selecting model switches current chat's model
  - [ ] Message history preserved after switch
  - [ ] Footer shows correct model info (Agent: X | Model: Y)
  - [ ] Can create new model from templates
  - [ ] Agent creation no longer shows model fields
  - [ ] Can create chat with both agent and model
  - [ ] Help text shows Ctrl+M

- [ ] **7.3** Manual Testing Checklist - Web UI
  - [ ] Models page lists all models
  - [ ] Can create model from provider template
  - [ ] Can edit model parameters
  - [ ] Can delete model
  - [ ] Chat creation requires both agent and model
  - [ ] Switching model in chat preserves history
  - [ ] Provider refresh updates model list
  - [ ] All API endpoints return correct data

- [ ] **7.4** Update Documentation
  - [ ] Update README.md with new features:
    - [ ] Model management section
    - [ ] Model switching (Ctrl+M)
    - [ ] Provider refresh command
    - [ ] Agent vs Model distinction
  - [ ] Update AGENTS.md if needed
  - [ ] Create migration guide for users:
    - [ ] Note: No automatic migration
    - [ ] Instructions to re-create agents (behavior only)
    - [ ] Instructions to create models from templates
    - [ ] Instructions to re-create chats with both agent and model
  - [ ] Add FAQ about Agent vs Model

- [ ] **7.5** Final Verification
  - [ ] Run all unit tests: `go test ./...`
  - [ ] Run code formatting: `go fmt ./...`
  - [ ] Run code vetting: `go vet ./...`
  - [ ] Build successfully: `go build .`
  - [ ] Test TUI launch: `./aiagent`
  - [ ] Test web server: `./aiagent serve`

**Phase 7 Complete**: All tests pass, documentation complete

---

## Success Criteria

### Functional
- [ ] Agents can be created without model selection
- [ ] Models can be created/edited/deleted independently
- [ ] Chats require both agent and model
- [ ] Switching models preserves chat history
- [ ] Ctrl+M opens model switcher in TUI
- [ ] Provider refresh updates available models
- [ ] Default data created correctly on fresh install

### Quality
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Manual testing checklist complete
- [ ] Code follows project conventions (go fmt, go vet)
- [ ] Documentation updated

### Performance
- [ ] Model switch < 1 second
- [ ] Provider refresh < 30 seconds
- [ ] No significant performance regression
- [ ] Cache reduces API calls

### User Experience
- [ ] UI changes are intuitive
- [ ] Error messages are clear
- [ ] Help text updated
- [ ] Agent vs Model distinction is clear

---

## Database Schema Changes

### JSON Storage

**New file: `.aiagent/models.json`**
```json
[
  {
    "id": "uuid",
    "name": "GPT-4o High Temp",
    "provider_id": "provider-uuid",
    "provider_type": "openai",
    "model_name": "gpt-4o",
    "api_key": "#{OPENAI_API_KEY}#",
    "temperature": 1.2,
    "max_tokens": 8000,
    "context_window": 128000,
    "reasoning_effort": "medium",
    "created_at": "2026-01-11T...",
    "updated_at": "2026-01-11T..."
  }
]
```

**Modified: `.aiagent/agents.json`**
- Remove: provider_id, provider_type, endpoint, model, api_key, temperature, max_tokens, context_window, reasoning_effort
- Keep: id, name, system_prompt, tools, created_at, updated_at

**Modified: `.aiagent/chats/*.json`**
- Add: model_id field
- Keep: agent_id (both required)

### MongoDB Storage

**New collection: `models`**
```javascript
{
  _id: ObjectId,
  id: String (unique),
  name: String,
  provider_id: String (indexed),
  provider_type: String,
  model_name: String,
  api_key: String,
  temperature: Number,
  max_tokens: Number,
  context_window: Number,
  reasoning_effort: String,
  created_at: Date,
  updated_at: Date
}
```

**Modified collection: `agents`**
- Remove fields: provider_id, provider_type, endpoint, model, api_key, temperature, max_tokens, context_window, reasoning_effort

**Modified collection: `chats`**
- Add field: model_id (indexed)

---

## Testing Strategy

### Unit Tests
- **Entity tests**: Model entity creation, validation
- **Repository tests**: CRUD operations for ModelRepository (JSON + Mongo)
- **Service tests**: ModelService, updated ChatService, ModelRefreshService
- **Client tests**: models.dev API client

### Integration Tests
- **Model switching**: Create chat → switch models → verify history preserved
- **Agent + Model**: Create agent + create model → create chat with both
- **Refresh flow**: Trigger refresh → verify providers updated with new models
- **Full workflow**: Agent creation → Model creation → Chat creation → Model switch → Send message

### Manual Testing
See Phase 7.2 and 7.3 checklists

---

## Key Dependencies

- **models.dev API**: `https://models.dev/api.json`
- **Existing services**: ChatService, AgentService, ProviderService
- **New services**: ModelService, ModelRefreshService
- **Existing UI**: AgentView, ChatView
- **New UI**: ModelView

---

## Remaining Decisions

- [ ] Default Model Selection: Should new chats have a default model or require user selection?
- [ ] Model Presets: Should we auto-create common model presets on first run? (Assuming yes in Phase 6)
- [ ] Refresh Interval: Is 24 hours the right default? (Assuming yes)
- [x] Model Sharing: Models are global like Agents (confirmed)

---

## Progress Summary

- **Phase 1** (Entity & Repository): 6/7 tasks complete (86%)
- **Phase 2** (Service Layer): 4/5 tasks complete (80%)
- **Phase 3** (TUI): 0/7 tasks complete (0%)
- **Phase 4** (Web UI): 0/8 tasks complete (0%)
- **Phase 5** (Models.dev): 6/6 tasks complete (100%)
- **Phase 6** (Defaults): 0/3 tasks complete (0%)
- **Phase 7** (Testing): 0/5 tasks complete (0%)

**Overall Progress**: 16/41 tasks (39%)

---

*Plan Version: 2.0 (Simplified - No Migration)*  
*Created: January 11, 2026*  
*Status: Ready to continue implementation*
