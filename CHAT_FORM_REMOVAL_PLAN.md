# Plan: Remove New Chat Form and Add Automatic Chat Creation

## Overview
Remove the New Chat Form entirely and implement automatic chat creation with background title generation.

## Current State
- New Chat Form is triggered by `ctrl+n` or "new" command
- Shows in `"chat/create"` state with manual agent/model/name selection
- Creates chat via `ChatService.CreateChat(agentID, modelID, name)`

## Implementation Plan

### Phase 1: Remove New Chat Form (COMPLETED)
- ✅ Remove `internal/tui/chat_form.go` (350+ lines)
- ✅ Remove ChatForm references from `tui.go`
- ✅ Remove `"chat/create"` state handling
- ✅ Clean up message types (`startCreateChatMsg`, `canceledCreateChatMsg`)

### Phase 2: Implement Automatic Chat Creation
**Goal**: Auto-create chats with last used preferences and temp titles

#### 2.1 Add Helper Function
Create `autoCreateChatCmd()` in `tui.go`:
```go
func (t *TUI) autoCreateChatCmd() tea.Cmd {
    return func() tea.Msg {
        // Get last used or default agent
        agentID := t.globalConfig.LastUsedAgent
        if agentID == "" {
            agents, _ := t.agentService.ListAgents(context.Background())
            if len(agents) > 0 {
                agentID = agents[0].ID
            }
        }
        
        // Get last used or default model
        modelID := t.globalConfig.LastUsedModel
        if modelID == "" {
            models, _ := t.modelService.ListModels(context.Background())
            if len(models) > 0 {
                modelID = models[0].ID
            }
        }
        
        // Generate temp title
        tempTitle := fmt.Sprintf("New Chat - %s", time.Now().Format("2006-01-02 15:04"))
        
        // Create and return chat
        chat, err := t.chatService.CreateChat(context.Background(), agentID, modelID, tempTitle)
        if err != nil {
            return errMsg(err)
        }
        return chat
    }
}
```

#### 2.2 Update Key/Command Handlers
- **ctrl+n** in `chat_view.go`: Replace `startCreateChatMsg("")` with `c.autoCreateChatCmd()`
- **"new" command** in `tui.go`: Replace form trigger with `t.autoCreateChatCmd()`

#### 2.3 Handle Initial App State
- When no active chats exist on startup, auto-create one instead of showing form
- Remove the `initialState = "chat/create"` logic

### Phase 3: Add Background Title Generation
**Goal**: Generate contextual titles after first message exchange

#### 3.1 Add Title Generation Method to ChatService
```go
func (s *chatService) GenerateAndUpdateTitle(ctx context.Context, chatID string) error {
    chat, err := s.chatRepo.GetChat(ctx, chatID)
    if err != nil || len(chat.Messages) < 2 {
        return err
    }
    
    // Get first user message and assistant response
    var firstUser, firstAssistant string
    for _, msg := range chat.Messages {
        if msg.Role == "user" && firstUser == "" {
            firstUser = msg.Content
        } else if msg.Role == "assistant" && firstAssistant == "" {
            firstAssistant = msg.Content
            break
        }
    }
    
    // Generate title using AI (same model as chat)
    prompt := fmt.Sprintf("Generate a short title (max 60 chars) summarizing this conversation: User: %s, Assistant: %s", 
                        firstUser, firstAssistant)
    
    title, err := s.generateTitleWithAI(ctx, chat.ModelID, prompt)
    if err != nil {
        return err
    }
    
    // Update chat title
    return s.UpdateChat(ctx, chatID, chat.AgentID, chat.ModelID, title)
}
```

#### 3.2 Integrate into SendMessage Flow
In `ChatService.SendMessage()`, after successful response generation:
```go
if len(chat.Messages) == 2 { // First user + first assistant message
    go func() {
        ctx := context.Background()
        if err := s.GenerateAndUpdateTitle(ctx, chat.ID); err != nil {
            s.logger.Warn("Failed to generate title", zap.Error(err))
        }
    }()
}
```

#### 3.3 Handle TUI Updates
- `updatedChatMsg` will automatically refresh the title display
- No loading indicators needed - happens seamlessly in background

### Phase 4: Fix Supporting Issues

#### 4.1 ModelWithPricing Type
Move `ModelWithPricing` from `chat_form.go` to `model_view.go`:
```go
type ModelWithPricing struct {
    *entities.Model
    Pricing *entities.ModelPricing
}

func (m ModelWithPricing) FilterValue() string { return m.ModelName }
func (m ModelWithPricing) Title() string { return m.ModelName }
func (m ModelWithPricing) Description() string {
    if m.Pricing != nil {
        return fmt.Sprintf("$%.2f/$%.2f per 1M tokens", 
                          m.Pricing.InputPricePerMille*1000, 
                          m.Pricing.OutputPricePerMille*1000)
    }
    return fmt.Sprintf("Provider: %s", m.ProviderID)
}
```

#### 4.2 Message Types
Add missing message type:
```go
chatCreatedMsg *entities.Chat
```

#### 4.3 TUI Method Receivers
Ensure all TUI methods have consistent receivers (all pointer receivers for tea.Model compliance):
- `func (t *TUI) Init() tea.Cmd`
- `func (t *TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd)`
- `func (t *TUI) View() string`

### Phase 5: Testing and QA

#### 5.1 Test Scenarios
- Fresh app start → auto-create chat
- ctrl+n during chat → create new chat
- Send first message → title updates in background
- Verify last-used preferences are respected
- Check title generation uses correct model

#### 5.2 QA Checklist
- ✅ `go build .` succeeds
- ✅ `go test ./...` passes
- ✅ No form-related code remains
- ✅ Chat creation works from all entry points
- ✅ Title generation happens after first message
- ✅ Titles appear in chat header automatically

## Benefits
- **Faster workflow**: Instant chat creation, no form friction
- **Better UX**: Contextual titles generated automatically
- **Cleaner code**: Removed 350+ lines of form UI
- **Backwards compatible**: Existing chats and preferences preserved

## Files to Modify
- `internal/tui/tui.go` - Main TUI logic, auto-creation, method receivers
- `internal/tui/chat_view.go` - ctrl+n handler
- `internal/tui/model_view.go` - ModelWithPricing type
- `internal/tui/messages.go` - Message types
- `internal/domain/services/chat_service.go` - Title generation logic
- `internal/impl/integrations/` - Add title generation to AI integrations

## Risk Mitigation
- Title generation failures → Keep temp title, log error
- Missing agents/models → Fallback to first available
- AI model errors → Graceful degradation
- Concurrent title updates → Use proper context cancellation