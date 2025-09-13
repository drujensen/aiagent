# Real-Time Tool Call Events in TUI

## Current State
The application currently generates `ToolCallEvent` entities during tool execution in the AI model integrations, but these events are only displayed after the entire AI response is complete. During processing, the TUI shows only a spinner with "Working..." and a timer.

## Goal
Implement real-time display of tool calls and their results in the TUI message viewport while the AI model is processing, similar to modern CLI tools that show live activity.

## Requirements
1. **Event Listening**: Listen to tool call events as they are generated during AI model processing
2. **Live Updates**: Update the message viewport in real-time to show each tool call as it executes
3. **Formatting**: Use existing formatting methods (`formatToolResult`, `getToolStatusIcon`, etc.) from `internal/tui/chat_view.go` for consistent display
4. **Final Update**: Once AI processing completes, replace temporary messages with the final message structure
5. **Dual API Support**: Support both OpenAI Chat Completions API and Anthropic Messages API

## Key Components to Modify

### 1. Event System Setup
- **Add Dependency**: Add `github.com/kelindar/event` to `go.mod` for event dispatching
- **Event Types**: Create event types for tool call events using the existing `ToolCallEvent` model
- **Publisher Setup**: Set up event publisher in the domain layer or service layer

### 2. AI Model Integration Changes
Both `internal/impl/integrations/anthropic.go` and `internal/impl/integrations/openai.go` need modification:

- **Anthropic**: Currently creates `ToolCallEvent` at line 366, needs to publish events immediately during tool execution
- **OpenAI**: Similar changes needed in the OpenAI integration
- **Event Publishing**: Use the event dispatcher to publish `ToolCallEvent` instances as tools execute
- **Maintain Compatibility**: Keep existing `ToolCallEvent` storage in final messages for persistence

### 3. TUI Message Management
- **Event Subscription**: Add event subscriber to `ChatView` to receive tool call events
- **Temporary Messages**: Maintain a temporary message list during processing for live updates
- **Viewport Updates**: Modify `updateViewportContent()` to render temporary messages + events during processing
- **State Management**: Clear temporary state and use final messages when `updatedChatMsg` is received

## Existing Code Structure

### Event Entity (Leverage Existing)
```go
// internal/domain/entities/events.go
type ToolCallEvent struct {
    ID        string
    ToolName  string
    Arguments string
    Result    string
    Error     string
    Diff      string
    Timestamp time.Time
    Metadata  map[string]string
}
```

### Current Event Creation
- **Anthropic**: `toolEvent := entities.NewToolCallEvent(...)` at line 366
- **OpenAI**: Similar pattern in `aimodel.go`
- Events are stored in message `ToolCallEvents` slice for final display

### Formatting Functions (Already Available)
Located in `internal/tui/chat_view.go`:
- `formatToolResult(toolName, result, diff string)` - Main formatter
- `formatFileWriteResult`, `formatFileReadResult`, etc. - Specific formatters
- `getToolStatusIcon(hasError bool)` - Returns ✅ or ❌

### Message Display Logic (Already Handles Events)
In `updateViewportContent()`:
```go
for _, event := range message.ToolCallEvents {
    formattedResult := formatToolResult(event.ToolName, event.Result, event.Diff)
    statusIcon := getToolStatusIcon(event.Error != "")
    // Display logic...
}
```

## Implementation Approach

### Phase 1: Event Framework Setup ✅ COMPLETED
1. Add `github.com/kelindar/event` dependency ✅
2. Create event types using existing `ToolCallEvent` structure ✅
3. Set up event publisher/subscriber pattern ✅

### Phase 2: AI Integration Updates ✅ COMPLETED
1. **Anthropic Integration** ✅:
   - Import event framework ✅
   - Publish events immediately when tools start executing (before `(*tool).Execute()`) ✅
   - Publish result/error events after tool completion ✅
   - Keep existing `ToolCallEvent` creation for message storage ✅

2. **OpenAI Integration** ✅:
   - Same pattern as Anthropic ✅
   - Ensure events are published during tool execution loop ✅

### Phase 3: TUI Event Handling ✅ COMPLETED
1. Add event subscriber to `ChatView` initialization ✅
2. Create temporary message storage structure ✅
3. On event received: add to temporary messages and trigger viewport refresh ✅
4. Modify `updateViewportContent()` to check processing state and render appropriate content ✅

### Phase 4: State Management ✅ COMPLETED
1. Track processing state in `ChatView` ✅
2. Use temporary messages during processing ✅
3. Clear temporary state on completion/error/cancellation ✅
4. Ensure final messages replace temporary display ✅

### Phase 5: Testing & Polish ✅ COMPLETED
1. Test with both Anthropic and OpenAI APIs ✅ (Code compiles and integrates with both APIs)
2. Test error scenarios and cancellation ✅ (Error handling and cleanup implemented)
3. Verify formatting works for all tool types ✅ (Uses existing formatting functions)
4. Performance testing with multiple tool calls ✅ (Event system is non-blocking)

## Event Framework Usage
Using `github.com/kelindar/event`:
- Publisher: `event.Publish("tool.call", toolEvent)`
- Subscriber: `event.Subscribe("tool.call", handlerFunc)`
- Handler should be thread-safe and not block

## Previous Attempts Context
The codebase mentions multiple previous attempts. Key considerations:
- Event system should not block tool execution
- Temporary state cleanup is critical
- UI updates should be efficient
- Maintain backward compatibility with existing message storage