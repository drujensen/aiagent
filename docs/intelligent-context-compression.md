# Intelligent Context Compression for LLM Workflow Management

## Overview
This feature introduces intelligent context compression to improve LLM performance in multi-step software engineering tasks. Instead of automatic compression at 90% context window usage, the LLM gains control over compression timing and scope to maintain optimal context for iterative development workflows.

## Current Problem
- Automatic compression at 90% context window removes ~40% of older messages indiscriminately
- Important architectural context (interfaces, contracts, design decisions) can be lost
- LLM has no control over when/what to compress, leading to suboptimal context management
- No workflow awareness - compression doesn't understand task boundaries

## Proposed Solution

### 1. Hybrid Compression System
- **Primary**: LLM-driven intelligent compression via CompressionTool
- **Fallback**: Automatic compression at 85% context window (safety net)

### 2. Workflow-Aware Task Management
Extend TodoTool with basic workflow grouping:
```go
type TodoItem struct {
    Content    string `json:"content"`
    Status     string `json:"status"`   // pending, in_progress, completed, cancelled
    ID         string `json:"id"`
    WorkflowID string `json:"workflow_id,omitempty"` // Optional grouping
}
```

### 3. Intelligent CompressionTool
**Actions:**
- `compress_range`: Compress specific message ranges with smart summarization

**Parameters:**
- `start_message_index`: Starting message index
- `end_message_index`: Ending message index  
- `summary_type`: Type of compression (task_cleanup, plan_update, context_preservation, full_reset)
- `description`: Human-readable description of what's being compressed

**Compression Types:**
1. **`task_implementation_cleanup`**: Remove implementation details but keep interfaces/contracts
2. **`plan_update`**: Replace outdated project info/plans with current versions
3. **`context_preservation`**: Keep important context, remove debugging/chat
4. **`full_reset`**: Major changes, keep only system prompt + current overview

### 4. Smart Summarization Prompts

**Task Implementation Cleanup:**
```
Summarize this task implementation while preserving:
- Interface definitions and contracts created
- Design decisions that affect future tasks
- Error handling patterns established
- Configuration or setup changes made

Remove:
- Step-by-step implementation details
- Debugging messages and fixes
- Intermediate code iterations
- Tool execution logs
```

**Plan Update:**
```
Replace the previous project overview/plan with the current version. Keep:
- Current architectural decisions
- Active interfaces and contracts
- Established patterns and standards

Remove:
- Previous/outdated plans
- Abandoned approaches
- Historical context no longer relevant
```

### 5. LLM Workflow Guidance

LLMs should maintain context hierarchy:
1. **System prompt** (agent behavior)
2. **Project overview** (from ProjectTool)
3. **Technical design** (architecture, patterns, standards)
4. **Current feature/story** (what we're building)
5. **Active plan** (workflow with tasks)
6. **Task implementations** (compressed after completion)

## Implementation Benefits

- **Better Context Management**: LLM controls compression timing and scope
- **Preserved Architecture**: Critical interfaces and contracts maintained
- **Workflow Awareness**: Compression aligned with task completion
- **Plan Evolution**: Outdated information can be replaced
- **Safety Net**: Automatic compression prevents context window overflow

## Usage Example

```javascript
// After completing user authentication API design
CompressionTool.compress_range({
  "start_message_index": 15,
  "end_message_index": 45,
  "summary_type": "task_implementation_cleanup",
  "description": "Completed API design and user model for authentication feature"
})

// When project scope changes
CompressionTool.compress_range({
  "start_message_index": 5,
  "end_message_index": 25,
  "summary_type": "plan_update", 
  "description": "Updated project scope to include mobile app support"
})
```

## Implementation Plan

### Phase 1: Enhanced TodoTool
- Add `WorkflowID` field to TodoItem
- Update schema and actions
- Test basic workflow grouping

### Phase 2: CompressionTool Core
- Create CompressionTool with range compression
- Implement basic summarization using existing logic
- Add compression instruction handling in chat service

### Phase 3: Intelligent Summarization
- Add multiple summary types with specific prompts
- Implement content analysis for context preservation
- Test compression effectiveness

### Phase 4: Integration & Testing
- Register new tools in factory
- Update tool documentation
- Test end-to-end workflow compression
- Validate context preservation

## Concerns Addressed

- **Context Loss**: Smart compression preserves architectural elements
- **Future Task Awareness**: Interfaces and contracts remain accessible
- **Plan Evolution**: Compression can replace outdated information
- **Safety**: Automatic compression as fallback prevents overflow
- **Complexity**: Simplified range-based approach vs. complex task detection</content>
<parameter name="filePath">docs/intelligent-context-compression.md