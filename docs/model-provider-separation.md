# Model-Provider Separation Feature

## Overview
Currently, agents in the AIAgent framework are tightly coupled to specific AI providers and models. This makes it difficult to switch providers or models without creating entirely new agents. We need to decouple agents from providers/models to allow dynamic switching via a new keyboard shortcut (CTRL+M), similar to how CTRL+A switches agents.

**Note**: This is a breaking change. We do not need to maintain backward compatibility with existing agents or APIs. We can remove or refactor any code that is no longer needed to achieve a cleaner architecture.

## Goals
- Allow users to change provider and model on-the-fly without changing agents
- Implement CTRL+M shortcut for model/provider selection
- Move context window to the model entity (since it's model-specific)
- Keep temperature with the agent (as it defines agent personality/creativity)
- Account for model capabilities (tool support, image support, etc.)

## Current Structure Analysis
- Agents currently include provider, model, temperature, and context window
- Providers are separate entities with API keys and base URLs
- Models are part of providers but not independently selectable

## Proposed Changes

### New Entities
1. **Model Entity**: Independent from agents, containing:
   - Provider reference
   - Model name/ID
   - Context window size
   - Capabilities (tools, images, vision, etc.)
   - Cost information (if available)

2. **Agent Updates**:
   - Remove provider and model fields
   - Remove context window (moves to model)
   - Keep temperature
   - Add default model reference

### UI Changes
- Add model selection interface (similar to agent selection)
- CTRL+M shortcut to open model selector
- Display current model in status/UI
- Update chat interface to show selected model

### Backend Changes
- Update domain entities (agent.go, add model.go)
- Modify services to handle model selection
- Update repositories for model storage
- Update chat service to use selected model per message/chat
- Ensure backward compatibility with existing agents

### Storage Changes
- Add model repository (JSON and MongoDB implementations)
- Migrate existing agent data (extract models from agents)
- Update default data initialization

### API Changes
- Update chat endpoints to accept model selection
- Add model management endpoints
- Maintain backward compatibility

## Implementation Steps
1. Define new Model entity and interfaces
2. Update Agent entity (remove provider/model/context window)
3. Create model repositories (JSON/MongoDB)
4. Update services (agent, chat, provider)
5. Implement model selection in TUI (CTRL+M)
6. Update web UI for model selection
7. Add model capabilities checking
8. Update default data and migrations
9. Add tests for new functionality
10. Update documentation

## Challenges
- Handling model capabilities (some models don't support tools/images)
- Backward compatibility with existing data
- UI/UX for model selection (similar to agent selection)
- Ensuring chat history works with model changes
- Cost tracking per model usage

## Acceptance Criteria
- Users can press CTRL+M to select different models/providers
- Model selection persists per chat or globally
- Context window adjusts automatically based on selected model
- Temperature remains agent-specific
- Tool usage respects model capabilities
- Existing agents continue to work (backward compatibility)
- Both TUI and web UI support model selection