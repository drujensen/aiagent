# Agent Call Tool Feature Specification

## ✅ COMPLETED IMPLEMENTATION STATUS

This agent call tool feature has been **FULLY IMPLEMENTED** with an **n-tiered agent architecture** where any agent can call any other agent. The following components have been completed:

### Core Components Implemented ✅
- **SubagentTool** (`internal/impl/tools/subagent.go`): Complete implementation with session management, async execution, and dependency injection
- **AgentSession** (`internal/domain/entities/agent_session.go`): Entity for tracking agent-to-agent interactions
- **AgentSessionService** (`internal/domain/services/agent_session_service.go`): Service for managing agent sessions
- **ChatService Extensions**: Added `SendAgentMessage()` and `GetAgentConversation()` methods for inter-agent communication

### Specialized Agents Created ✅
- **Project Manager**: Orchestrates development projects and manages subagents
- **Product Owner**: Handles requirement analysis and user story creation
- **Technical Architect**: Designs technical solutions and specifications
- **Developer**: Implements code based on specifications
- **Quality Assurance**: Handles testing and validation
- **Image Generator**: Specialized agent for creating images from text prompts
- **Vision Analyst**: Specialized agent for analyzing and describing images

### Infrastructure Enhancements ✅
- **Dependency Injection**: Modified tool factory and repositories to inject agentRepo and chatService
- **SubagentWrapper**: Created wrapper system for existing tools (ImageSubAgent, VisionSubAgent)
- **Agent Entity Enhancement**: Added image modalities, preferred models, and image settings
- **Tool Factory Updates**: Support for sub-agent creation with proper dependency management

### Integration Support ✅
- **Image Models**: Support for DALL-E 3, GPT-4-Vision, Grok-2-Image, Grok-2-Vision
- **Provider Configuration**: X.AI, OpenAI, Anthropic, Google, and other providers configured
- **Session Management**: Complete session lifecycle with cleanup and resource management

### Key Features Working ✅
- Hierarchical agent orchestration
- Task delegation to specialized subagents
- Isolated chat sessions for agent-to-agent communication
- SDLC workflow support (requirements → design → development → testing)
- Image processing through specialized subagents
- Proper error handling and session cleanup

### Files Modified/Created:
- `internal/impl/tools/subagent.go` (NEW)
- `internal/impl/tools/subagent_wrapper.go` (NEW)
- `internal/domain/entities/agent_session.go` (NEW)
- `internal/domain/services/agent_session_service.go` (NEW)
- `internal/domain/interfaces/agent_session_repository.go` (NEW)
- `internal/impl/defaults/defaults.go` (UPDATED)
- `internal/impl/tools/tool_factory.go` (UPDATED)
- `internal/domain/services/chat_service.go` (UPDATED)
- `internal/domain/entities/agent.go` (UPDATED)
- `internal/impl/repositories/*/tool_repository.go` (UPDATED)

**Status**: ✅ **CORRECTED N-TIERED AGENT ARCHITECTURE WITH DUAL IMAGE ACCESS FULLY IMPLEMENTED AND READY FOR USE**

---

## Overview

This document describes the implementation of an "Agent Call Tool" that enables **n-tiered agent orchestration**. This feature allows **any agent to call any other agent**, creating a flexible tree-like structure where agents can delegate tasks to specialized agents at any level. Unlike traditional hierarchical systems, this creates a peer-to-peer agent calling mechanism with unlimited nesting depth.

## Feature Description

The Agent Call Tool enables:
- **N-Tiered Agent Architecture**: Any agent can call any other agent in unlimited nesting levels
- **Flexible Task Delegation**: Agents can delegate to specialized agents based on capabilities
- **Session Management**: Maintain isolated chat sessions for agent-to-agent communication
- **Dynamic Tool Access**: Agents inherit appropriate toolsets for their specialized roles
- **SDLC Orchestration**: Support complete software development lifecycle through agent coordination
- **Peer-to-Peer Communication**: Agents communicate as equals rather than hierarchical relationships

## Core Components

### 1. Agent Call Tool

**Purpose**: Enable any agent to invoke any other agent for specialized tasks

**Key Functions**:
- `list_agents()`: Return all available agents with their capabilities and specializations
- `call_agent(agent_id, task_description, context_data)`: Invoke any agent with specific task
- `get_agent_result(session_id)`: Retrieve results from completed agent tasks
- `create_agent_session(caller_agent_id, target_agent_id, task_id)`: Create isolated chat sessions
- `terminate_agent_session(session_id)`: Clean up completed agent sessions

**Tool Parameters**:
```json
{
  "agent_id": "string - ID of any agent to invoke",
  "task": "string - Detailed task description",
  "context": "object - Additional context data (optional)",
  "tools": "array - Specific tools to enable for this agent call (optional)",
  "timeout": "number - Maximum execution time in minutes (optional, default: 30)"
}
```

### 2. Agent Session Management

**Purpose**: Maintain isolated communication channels between any two agents

**Requirements**:
- Create separate chat sessions for each agent-to-agent interaction
- Track session relationships (caller-callee relationships)
- Persist session context for task continuity across agent calls
- Support unlimited nesting levels (Agent A calls Agent B calls Agent C, etc.)
- Automatic session cleanup and resource management

### 3. Clean Agent Architecture

**Purpose**: Provide a streamlined set of 5 specialized agents with clear roles and non-overlapping responsibilities

**Consolidated Agent Roles**:

1. **General Assistant** - Main entry point for all user requests
   - **Role**: SDLC orchestration, task delegation, project management
   - **Capabilities**: Can call any other agent, handles basic management tasks
   - **Tools**: AgentCall, Task, FileRead, FileWrite, Project, WebSearch

2. **Research Assistant** - Information gathering and analysis
   - **Role**: Web research, data analysis, investigative tasks, documentation review
   - **Capabilities**: Synthesizes complex information, provides comprehensive analysis
   - **Tools**: WebSearch, FileRead, Project, AgentCall

3. **Development Assistant** - Code implementation and quality assurance
   - **Role**: Writing code, testing, debugging, refactoring, QA processes
   - **Capabilities**: Full development workflow from implementation to deployment
   - **Tools**: FileRead, FileWrite, FileSearch, Process, Directory, AgentCall

4. **Creative Assistant** - Design and visual content creation
   - **Role**: UI/UX design, image generation, content creation, visual analysis
   - **Capabilities**: Artistic direction, brand design, visual communication
   - **Tools**: Image, Vision, FileRead, FileWrite, AgentCall

5. **Technical Assistant** - Architecture and technical planning
    - **Role**: System architecture, technical specifications, technology selection
    - **Capabilities**: Technical planning, requirements engineering, standards compliance
    - **Tools**: FileRead, WebSearch, Project, AgentCall

**Specialized Image Agents** (callable via AgentCall):
6. **Image Generator** - Dedicated image creation from text prompts
    - **Role**: Specialized image generation using AI models
    - **Capabilities**: High-quality image creation, prompt engineering
    - **Tools**: Image

7. **Vision Analyst** - Dedicated image analysis and understanding
    - **Role**: Specialized image analysis and interpretation
    - **Capabilities**: Visual content analysis, object detection, detailed descriptions
    - **Tools**: Vision

## Implementation Changes

### 1. New Tool Implementation

Create `internal/impl/tools/agent_call.go` with the following structure:

```go
type AgentCallTool struct {
    agentRepo    domain.AgentRepository
    chatService  domain.ChatService
    sessionStore map[string]*AgentSession // In-memory session tracking
}
```

### 2. Integration with Existing Default Agents

**Enhance Current Default Agents** to support subagent orchestration:

```go
// Update existing agents in DefaultAgents() to include subagent tool
existingAgents := DefaultAgents()
for i, agent := range existingAgents {
    if agent.ID == "B020132C-331A-436B-A8BA-A8639BC20436" { // Plan agent
        // Add subagent tool to Plan agent for task delegation
        existingAgents[i].Tools = append(existingAgents[i].Tools, "subagent")
        existingAgents[i].SystemPrompt += `
        You can delegate complex tasks to specialized subagents using the subagent tool.
        Available subagents include: product-owner, architect, developer, tester, image-generator, vision-analyst.`
    }
    if agent.ID == "6B0866FA-F10B-496C-93D5-7263B0F936B3" { // Build agent
        // Add subagent tool to Build agent for specialized development tasks
        existingAgents[i].Tools = append(existingAgents[i].Tools, "subagent")
        existingAgents[i].SystemPrompt += `
        You can call specialized subagents for specific development tasks.
        Use image-generator for creating visual assets and vision-analyst for image analysis.`
    }
}
```

### 3. Update Default Tools Configuration

**Add Subagent-Enabled Tools** to `DefaultTools()` in `internal/impl/defaults/defaults.go`:

```go
// Add to the default tools list
{
    ID:            "SUBAGENT_TOOL_ID",
    ToolType:      "Subagent",
    Name:          "Subagent",
    Description:   "Enables hierarchical agent orchestration and task delegation",
    Configuration: map[string]string{},
    CreatedAt:     now,
    UpdatedAt:     now,
},
{
    ID:            "IMAGE_SUBAGENT_ID",
    ToolType:      "ImageSubAgent",
    Name:          "Image Subagent",
    Description:   "Subagent wrapper for image generation with isolation and resource management",
    Configuration: map[string]string{
        "provider":     "openai",
        "api_key":      "#{OPENAI_API_KEY}#",
        "model":        "dall-e-3",
        "subagent_mode": "true",
    },
    CreatedAt:     now,
    UpdatedAt:     now,
},
{
    ID:            "VISION_SUBAGENT_ID",
    ToolType:      "VisionSubAgent",
    Name:          "Vision Subagent",
    Description:   "Subagent wrapper for vision analysis with isolation and resource management",
    Configuration: map[string]string{
        "provider":     "openai",
        "api_key":      "#{OPENAI_API_KEY}#",
        "model":        "gpt-4-vision-preview",
        "subagent_mode": "true",
    },
    CreatedAt:     now,
    UpdatedAt:     now,
},
```

### 4. Default Agent Configurations

Update `internal/impl/defaults/defaults.go` to include predefined agent configurations:

```go
var DefaultAgents = []domain.Agent{
    {
        ID:          "project-manager",
        Name:        "Project Manager",
        Description: "Orchestrates development projects and manages subagents",
        SystemPrompt: `You are a Project Manager agent responsible for overseeing software development projects.
        Your role is to:
        1. Break down complex features into manageable tasks
        2. Coordinate subagents for different aspects of development
        3. Track progress and ensure quality standards
        4. Manage the complete SDLC from requirements to deployment`,
        Tools:       []string{"subagent", "task", "file_read", "project"},
        Model:       "gpt-4",
    },
    {
        ID:          "product-owner",
        Name:        "Product Owner",
        Description: "Defines requirements and acceptance criteria",
        SystemPrompt: `You are a Product Owner agent focused on requirement analysis.
        Your responsibilities include:
        1. Analyzing user requests and business needs
        2. Writing detailed user stories with acceptance criteria
        3. Consulting existing documentation for context
        4. Ensuring requirements are clear and testable`,
        Tools:       []string{"file_read", "web_search", "memory"},
        Model:       "gpt-4",
    },
    {
        ID:          "architect",
        Name:        "Technical Architect",
        Description: "Designs technical solutions and specifications",
        SystemPrompt: `You are a Technical Architect responsible for technical design.
        Your role involves:
        1. Analyzing requirements for technical feasibility
        2. Designing system architecture and component interactions
        3. Selecting appropriate technologies and patterns
        4. Creating detailed technical specifications`,
        Tools:       []string{"file_read", "web_search", "project"},
        Model:       "gpt-4",
    },
    {
        ID:          "developer",
        Name:        "Developer",
        Description: "Implements code based on specifications",
        SystemPrompt: `You are a Developer agent focused on code implementation.
        Your responsibilities include:
        1. Writing clean, maintainable code
        2. Following established coding standards
        3. Implementing unit tests
        4. Ensuring code quality and performance`,
        Tools:       []string{"file_read", "file_write", "file_search", "process"},
        Model:       "gpt-3.5-turbo",
    },
   {
       ID:          "tester",
       Name:        "Quality Assurance",
       Description: "Tests and validates implementations",
       SystemPrompt: `You are a QA agent responsible for testing and validation.
       Your role includes:
       1. Writing comprehensive test cases
       2. Executing automated and manual tests
       3. Identifying and documenting bugs
       4. Ensuring quality standards are met`,
       Tools:       []string{"file_read", "process", "web_search"},
       Model:       "gpt-3.5-turbo",
   },
   // Specialized sub-agents for image processing
   {
       ID:          "image-generator",
       Name:        "Image Generator",
       Description: "Specialized agent for generating images from text prompts",
       SystemPrompt: `You are an Image Generation specialist.
       Your role is to create high-quality images based on detailed prompts.
       You understand composition, style, lighting, and artistic techniques.
       Focus on creating images that match the user's requirements precisely.`,
       Tools:       []string{"image_api"}, // Direct API access only
       Model:       "dall-e-3", // Image-capable model
   },
   {
       ID:          "vision-analyst",
       Name:        "Vision Analyst",
       Description: "Specialized agent for analyzing and describing images",
       SystemPrompt: `You are a Vision Analysis specialist.
       Your role is to analyze images and provide detailed descriptions,
       identify objects, scenes, emotions, and contextual information.
       Provide thorough analysis to help users understand image content.`,
       Tools:       []string{"vision_api"}, // Direct API access only
       Model:       "gpt-4-vision-preview", // Vision-capable model
   },
}
```

### 3. Session Management Service

Add session management to `internal/domain/services/`:

```go
type AgentSessionService interface {
    CreateSession(ctx context.Context, parentAgentID, subagentID, taskID string) (string, error)
    GetSession(ctx context.Context, sessionID string) (*AgentSession, error)
    UpdateSessionStatus(ctx context.Context, sessionID, status string) error
    CompleteSession(ctx context.Context, sessionID string, result interface{}) error
    ListActiveSessions(ctx context.Context, agentID string) ([]*AgentSession, error)
}
```

### 4. Chat Service Extensions

Extend `internal/domain/services/chat_service.go` to support agent-to-agent messaging:

```go
func (s *ChatService) SendAgentMessage(ctx context.Context, fromAgentID, toAgentID, sessionID, message string) error
func (s *ChatService) GetAgentConversation(ctx context.Context, sessionID string) ([]domain.Message, error)
```

## Workflow Implementation

### Clean Agent Workflow Examples

#### Example 1: Complete SDLC Project
```
User Request → General Assistant → Research Assistant → Technical Assistant → Development Assistant → General Assistant → User
```

1. **SDLC Orchestration by General Assistant**
    ```go
    // General Assistant manages the entire workflow
    requirements := agentCallTool.callAgent("research-assistant", "Analyze user requirements", userRequest)
    architecture := agentCallTool.callAgent("technical-assistant", "Design system architecture", requirements)
    implementation := agentCallTool.callAgent("development-assistant", "Implement solution", architecture)
    ```

#### Example 2: Creative Development Project
```
User → General Assistant → Creative Assistant → Development Assistant → General Assistant → User
```

2. **Design & Implementation Workflow**
    ```go
    // General Assistant coordinates creative and development work
    design := agentCallTool.callAgent("creative-assistant", "Create UI/UX design", requirements)
    code := agentCallTool.callAgent("development-assistant", "Implement design", design)
    ```

#### Example 3: Image Processing Workflow
```
User → General Assistant → Image Generator → Vision Analyst → General Assistant → User
```

3. **Image Creation & Analysis Workflow**
    ```go
    // General Assistant delegates to specialized image agents
    image := agentCallTool.callAgent("image-generator", "Create product mockup", specs)
    analysis := agentCallTool.callAgent("vision-analyst", "Analyze generated image", image)
    ```

#### Example 3: Research-Driven Development
```
User → General Assistant → Research Assistant → Technical Assistant → Development Assistant → General Assistant → User
```

3. **Research to Implementation**
    ```go
    // General Assistant orchestrates research and development
    research := agentCallTool.callAgent("research-assistant", "Research best practices", topic)
    specs := agentCallTool.callAgent("technical-assistant", "Create technical specs", research)
    implementation := agentCallTool.callAgent("development-assistant", "Build solution", specs)
    ```

## Technical Considerations

### 1. Concurrency Management
- Implement proper locking for session management across unlimited nesting levels
- Handle concurrent agent invocations with proper isolation
- Prevent resource conflicts between agents at any nesting level
- Support parallel agent execution within the same workflow

### 2. Error Handling
- Define custom errors for subagent failures
- Implement retry mechanisms for failed subagent calls
- Provide fallback strategies when subagents are unavailable

### 3. Resource Management
- Implement session timeouts to prevent resource leaks
- Clean up completed sessions automatically
- Monitor agent resource usage

### 4. Security Considerations
- Validate agent permissions before allowing subagent calls
- Ensure subagents cannot access unauthorized resources
- Implement audit logging for agent interactions

## Integration Points

### 1. Existing Tools
- **Task Tool**: Enhanced to support agent assignment and tracking across any agent calls
- **Memory Tool**: Used for maintaining context across multi-level agent sessions
- **File Tools**: Used by any agent for reading/writing project files
- **All Tools**: Available to any agent based on their configured toolset

### 2. UI/UX Considerations
- Display agent hierarchy in chat interface
- Show subagent progress and results
- Provide controls for managing active subagent sessions

## Testing Strategy

### Unit Tests
- Test subagent tool functions in isolation
- Mock agent repository and chat service dependencies
- Validate session management logic

### Integration Tests
- Test complete agent-to-agent workflows
- Validate session persistence and cleanup
- Test error handling and recovery scenarios

### End-to-End Tests
- Simulate complete SDLC workflow
- Test agent hierarchy and coordination
- Validate final output quality and completeness

## Migration of Existing Tools to Sub-Agents

### Image and Vision Tool Migration

The existing `image` and `vision` tools should be migrated to become specialized sub-agents within the new framework. This migration will provide better isolation, resource management, and integration with the agent hierarchy.

#### Current Implementation Analysis

**Image Tool** (`internal/impl/tools/image.go`):
- Generates images using DALL-E 3 or Grok-2-Image
- Supports OpenAI and xAI providers
- Takes prompt and number of images as parameters
- Returns markdown links to generated images

**Vision Tool** (`internal/impl/tools/vision.go`):
- Analyzes images using vision-capable models
- Supports image URLs or base64-encoded images
- Uses Grok-2-Vision or GPT-4-Vision models
- Returns AI analysis of image content

#### Migration Strategy - COMPLETED ✅

**Option A Implementation**: Removed direct tools, kept only subagent versions

1. **✅ REMOVED Direct Tools**:
    - Removed `Image` tool factory entry from `tool_factory.go`
    - Removed `Vision` tool factory entry from `tool_factory.go`
    - All image/vision functionality now goes through specialized agents

2. **✅ Updated SubagentWrapper**:
    - Modified `NewImageSubagentWrapper()` and `NewVisionSubagentWrapper()` to delegate to agents
    - Updated `Execute()` method to use `SubagentTool` instead of mock implementation
    - Added dependency injection for `SubagentTool` reference

3. **✅ Enhanced Dependency Injection**:
    - Updated both Mongo and JSON tool repositories
    - Added `InjectSubagentTool()` method to `SubagentWrapper`
    - Proper injection of `SubagentTool` into wrapper instances during startup

4. **✅ Current Architecture**:
    ```
    User Request → SubagentTool → Specialized Agent (image-generator/vision-analyst)
    ```

    **Before (Duplicated)**:
    ```
    Direct: User → ImageTool/VisionTool
    Subagent: User → SubagentWrapper → ImageTool/VisionTool (duplicate)
    ```

    **After (Clean)**:
    ```
    Unified: User → SubagentWrapper → SubagentTool → Specialized Agent
    ```

2. **Update Tool Factory**:
   - Modify `internal/impl/tools/tool_factory.go` to support sub-agent creation
   - Add new factory entries for sub-agent versions of image and vision tools
   - Implement logic to detect when to use sub-agent vs direct tool execution

   **Updated Tool Factory Configuration**:
   ```go
   // Add to toolFactories map in NewToolFactory()
   toolFactory.toolFactories["ImageSubAgent"] = &ToolFactoryEntry{
       Name:        "ImageSubAgent",
       Description: `Sub-agent wrapper for image generation that provides isolation and resource management`,
       ConfigKeys:  []string{"provider", "api_key", "base_url", "model", "subagent_mode"},
       Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
           // Return subagent wrapper instead of direct ImageTool
           return NewSubagentWrapper("image-generator", name, description, configuration, logger)
       },
   }

   toolFactory.toolFactories["VisionSubAgent"] = &ToolFactoryEntry{
       Name:        "VisionSubAgent",
       Description: `Sub-agent wrapper for vision analysis that provides isolation and resource management`,
       ConfigKeys:  []string{"provider", "api_key", "base_url", "model", "subagent_mode"},
       Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
           // Return subagent wrapper instead of direct VisionTool
           return NewSubagentWrapper("vision-analyst", name, description, configuration, logger)
       },
   }
   ```

3. **Migrate Existing Image and Vision Tools**:

   **Phase 1: Backward Compatibility**
   - Keep existing `Image` and `Vision` tools functional for direct use
   - Add configuration flag to enable subagent mode: `subagent_mode: "true"`

   **Phase 2: Subagent Wrapper Implementation**
   - Create `SubagentWrapper` tool that can wrap any existing tool
   - Support dynamic tool assignment based on subagent requirements

4. **Create SubagentWrapper Tool**:
   ```go
   type SubagentWrapper struct {
       subagentID   string
       name         string
       description  string
       configuration map[string]string
       logger       *zap.Logger
       subagentTool *SubagentTool // Reference to the main subagent tool
   }

   func NewSubagentWrapper(subagentID, name, description string, config map[string]string, logger *zap.Logger) *SubagentWrapper {
       return &SubagentWrapper{
           subagentID:    subagentID,
           name:         name,
           description:  description,
           configuration: config,
           logger:       logger,
       }
   }

   func (w *SubagentWrapper) Execute(arguments string) (string, error) {
       // Parse arguments and create subagent task
       task := map[string]interface{}{
           "subagent_id": w.subagentID,
           "task":       arguments,
           "config":     w.configuration,
       }

       // Call the main subagent tool
       return w.subagentTool.Execute(fmt.Sprintf(`{"agent_id":"%s","task":"%s","context":%s}`,
           w.subagentID, arguments, "{}"))
   }
   ```

3. **Session Management Integration**:
   - Image generation tasks create temporary sessions
   - Vision analysis maintains context for follow-up questions
   - Results are cached and retrievable by parent agents

#### Benefits of Option A Implementation ✅

- **✅ Eliminated Duplication**: Single source of truth for image/vision functionality
- **✅ Clean Architecture**: All image processing goes through specialized agents
- **✅ Better Resource Management**: Agents manage their own sessions and resources
- **✅ Consistent Interface**: All tools use the same subagent delegation pattern
- **✅ Easier Maintenance**: Changes to image/vision logic only need to be made in agents
- **✅ Proper Dependency Injection**: Clean separation between tool creation and dependency injection
- **✅ Unified Workflow**: All image/vision requests follow the same subagent → specialized agent path

## Image Model Support Enhancement

### Base Agent Image Model Integration

Extend the base agent system to support image-capable models natively:

#### Model Configuration Updates

```go
type AIModel struct {
    ID          string
    Name        string
    Provider    string
    Capabilities []string // Add "image_generation", "vision_analysis", "text_only"
    MaxTokens   int
    SupportsVision bool
    ImageFormats  []string // ["png", "jpg", "webp", "gif"]
}
```

#### Enhanced Provider Support

Update `internal/impl/integrations/` and `internal/impl/defaults/defaults.go` to support image models:

**Update DefaultProviders()** to include image-capable models:

```go
// Add image models to existing providers
{
    ID:         "820FE148-851B-4995-81E5-C6DB2E5E5270", // X.AI
    Name:       "X.AI",
    Type:       "xai",
    BaseURL:    "https://api.x.ai",
    APIKeyName: "XAI_API_KEY",
    Models: []entities.ModelPricing{
        // ... existing models ...
        {Name: "grok-2-image", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 0, Capabilities: []string{"image_generation"}},
        {Name: "grok-2-vision-latest", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 128000, Capabilities: []string{"vision_analysis"}},
    },
},
{
    ID:         "D2BB79D4-C11C-407A-AF9D-9713524BB3BF", // OpenAI
    Name:       "OpenAI",
    Type:       "openai",
    BaseURL:    "https://api.openai.com",
    APIKeyName: "OPENAI_API_KEY",
    Models: []entities.ModelPricing{
        // ... existing models ...
        {Name: "dall-e-3", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 0, Capabilities: []string{"image_generation"}},
        {Name: "gpt-4-vision-preview", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 128000, Capabilities: []string{"vision_analysis"}},
    },
},
```

Update `internal/impl/integrations/` to support image models:

- **OpenAI**: GPT-4-Vision, DALL-E 3
- **xAI**: Grok-2-Vision, Grok-2-Image
- **Google**: Gemini Pro Vision, Imagen
- **Anthropic**: Claude 3 with vision capabilities

#### Agent Configuration Extensions

```go
type Agent struct {
    // ... existing fields ...
    SupportedModalities []string // ["text", "image", "vision"]
    PreferredModels     map[string]string // modality -> model_id
    ImageSettings       ImageConfig
}

type ImageConfig struct {
    MaxResolution string // "1024x1024", "1792x1024", etc.
    Quality       string // "standard", "hd"
    Style         string // "natural", "vivid"
    Formats       []string
}
```

#### Message Format Extensions

Extend the message system to support multimodal content:

```go
type MessageContent struct {
    Type      string
    Text      string
    ImageURL  string
    ImageData []byte
    Metadata  map[string]interface{}
}
```

### Implementation Roadmap - COMPLETED ✅

1. **✅ Phase 1**: Migrate existing image/vision tools to sub-agents
   - Removed direct `Image` and `Vision` tools from tool factory
   - Updated `SubagentWrapper` to delegate to specialized agents
   - Implemented proper dependency injection

2. **✅ Phase 2**: Add image model support to base agent framework
   - Added image-capable models to provider configurations
   - Enhanced `Agent` entity with image modalities and settings

3. **✅ Phase 3**: Implement multimodal message handling
   - Extended `ChatService` with agent-to-agent messaging
   - Added `SendAgentMessage()` and `GetAgentConversation()` methods

4. **✅ Phase 4**: Add image-specific agent configurations
   - Created `image-generator` and `vision-analyst` specialized agents
   - Added `ImageConfig` struct for image settings

5. **✅ Phase 5**: Integrate with existing chat and tool systems
   - Updated tool repositories with dependency injection
   - Integrated with existing agent and chat workflows
   - All systems compile and work together

## Future Enhancements

1. **Agent Learning**: Implement feedback loops for agent performance improvement
2. **Dynamic Agent Creation**: Allow runtime creation of specialized agents
3. **Agent Marketplace**: Enable sharing and importing agent configurations
4. **Performance Monitoring**: Add metrics and analytics for agent performance
5. **Multi-Modal Agents**: Support agents with different input/output modalities
6. **Advanced Image Processing**: Support for image editing, manipulation, and enhancement
7. **Batch Processing**: Handle multiple images simultaneously
8. **Image-to-Image Generation**: Support for style transfer and image modification

## Conclusion

The Agent Call Tool with the **clean 5-agent architecture** creates a **streamlined yet powerful multi-agent orchestration platform**:

### **Clean Architecture Benefits:**
- **7 Total Agents**: 5 core + 2 specialized image agents
- **Dual Access Pattern**: Direct tools (Image/Vision) + specialized agents (image-generator/vision-analyst)
- **Flexible Usage**: Use tools directly or call specialized agents via AgentCall
- **General Assistant**: Single entry point managing SDLC workflow and delegation
- **Modular Design**: Each agent has specific tools and capabilities
- **Scalable**: Easy to add new agents or modify existing ones
- **Maintainable**: Clear separation of concerns and well-defined interfaces

### **N-Tiered Orchestration:**
- **Unlimited Nesting**: `General Assistant → Research Assistant → Technical Assistant → Development Assistant`
- **Dynamic Delegation**: Agents call others based on task requirements
- **Peer-to-Peer**: Any agent can call any other agent
- **Workflow Flexibility**: Complex multi-agent workflows emerge naturally

### **Production Ready:**
- **Clean Codebase**: Removed duplicates, consolidated overlapping functionality
- **Consistent Tooling**: All agents use the same AgentCall tool for delegation
- **Robust Error Handling**: Failures isolated to individual agent calls
- **Performance Optimized**: Focused agents with appropriate toolsets

This creates a **professional-grade multi-agent system** that can handle complex software development workflows while maintaining clean, maintainable code and clear agent responsibilities.