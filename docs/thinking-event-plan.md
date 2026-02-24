# Plan: Add ThinkingEvent Support to AIAgent TUI

## Overview
This plan outlines adding support for displaying AI model "thinking" or reasoning responses in the TUI, providing real-time visibility into the model's thought process during conversations.

## Background
- Current aiagent does NOT support streaming responses; integrations make full HTTP requests
- Thinking responses are streamed as deltas in certain models (OpenAI o1, Claude thinking mode, Grok)
- Existing event system (ToolCallEvent) provides real-time feedback for tool usage
- ThinkingEvent would follow similar pattern for reasoning content

## Current State Analysis
- **No Streaming**: All integrations use synchronous HTTP requests
- **Callback System**: MessageCallback used for incremental saving, not real-time streaming
- **Event System**: ToolCallEvent published during tool execution for TUI updates

## Implementation Plan

### Phase 1: Implement Streaming Support
1. **Update AIModelIntegration Interface** (`internal/domain/interfaces/aimodel_integration.go`):
   - Add streaming option to `GenerateResponse` method
   - Define delta callback for processing chunks: `DeltaCallback func(delta map[string]any) error`

2. **Modify OpenAI Integration** (`internal/impl/integrations/openai.go`):
   - Enable `stream: true` in `/v1/chat/completions` requests
   - Parse SSE (Server-Sent Events) response stream
   - Process `delta.content` chunks and emit delta callbacks
   - Detect reasoning content (e.g., look for patterns like "Let me think..." or structured reasoning)

3. **Update Other Integrations** (Anthropic, Grok, etc.):
   - Add streaming support where APIs allow it
   - Handle provider-specific delta formats

4. **Update Chat Service** (`internal/domain/services/chat_service.go`):
   - Pass delta callback to integrations
   - Handle streaming errors gracefully

### Phase 2: Add Thinking Detection and Events
1. **Create ThinkingEvent Entity** (`internal/domain/entities/events.go`):
   - Add `ThinkingEvent` struct with fields: ID, ChatID, Content (reasoning text), Timestamp

2. **Extend Event System** (`internal/domain/events/events.go`):
   - Add `ThinkingEventType`, data wrappers, publish/subscribe functions

3. **Emit Thinking Events**:
   - In streaming integrations, detect reasoning content in deltas
   - Publish `ThinkingEvent` with reasoning text
   - Publish after final response to clear thinking messages

4. **TUI Integration** (`internal/tui/chat_view.go`):
   - Subscribe to `ThinkingEvent`
   - Display reasoning as temporary messages (e.g., gray/italic text)
   - Clear thinking messages when final response arrives

### Phase 3: Testing and QA
- Test with o1 models (OpenAI), Claude thinking mode, Grok
- Ensure backward compatibility with non-streaming models
- Run QA workflow after changes

## Trade-offs and Considerations
- **Complexity**: Adding streaming is a significant change requiring careful error handling for connection issues
- **Provider Support**: Not all models/providers support streaming equally (e.g., Grok may have different delta formats)
- **UI Impact**: Thinking content could be verbose; consider collapsible sections or limits
- **Fallback**: For non-streaming models, thinking won't be available

## Next Steps
- Decide on UI display approach (e.g., inline, collapsible, limited)
- Prioritize which providers to implement streaming for first
- Consider if thinking should be saved permanently or only shown temporarily