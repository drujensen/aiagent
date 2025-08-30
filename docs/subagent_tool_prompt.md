# Subagent Tool Feature Specification

## Overview

This document describes the implementation of a new "Subagent Tool" that enables hierarchical agent orchestration. This feature allows agents to call other agents as subagents, creating a multi-level agent hierarchy for complex task management and execution.

## Feature Description

The Subagent Tool will enable:
- **Agent Hierarchy**: Create orchestrator/project manager agents that coordinate subagents
- **Task Delegation**: Assign specific tasks to specialized subagents
- **Session Management**: Maintain separate chat sessions for agent-to-agent communication
- **Tool Customization**: Allow subagents to have specialized toolsets rather than full access
- **SDLC Orchestration**: Support complete software development lifecycle management through agent coordination

## Core Components

### 1. Subagent Tool

**Purpose**: Enable parent agents to invoke child agents for specific tasks

**Key Functions**:
- `list_agents()`: Return available agents with their capabilities and specializations
- `call_subagent(agent_id, task_description, context_data)`: Invoke a subagent with specific task
- `get_subagent_result(session_id)`: Retrieve results from completed subagent tasks
- `create_agent_session(parent_agent_id, subagent_id, task_id)`: Create isolated chat sessions
- `terminate_subagent(session_id)`: Clean up completed subagent sessions

**Tool Parameters**:
```json
{
  "agent_id": "string - ID of the subagent to invoke",
  "task": "string - Detailed task description",
  "context": "object - Additional context data (optional)",
  "tools": "array - Specific tools to enable for this subagent (optional)",
  "timeout": "number - Maximum execution time in minutes (optional, default: 30)"
}
```

### 2. Agent Session Management

**Purpose**: Maintain isolated communication channels between agents

**Requirements**:
- Create separate chat sessions for each agent interaction
- Track session hierarchy (parent-child relationships)
- Persist session context for task continuity
- Support session cleanup and resource management

### 3. Agent Specialization Framework

**Purpose**: Define agent roles with appropriate toolsets and capabilities

**Specialized Agent Types**:
- **Project Manager**: Full tool access, coordination capabilities
- **Product Owner**: Documentation access, requirement analysis tools
- **Architect**: Technical specification tools, design pattern libraries
- **Developer**: Code writing, testing, debugging tools
- **Tester**: Testing frameworks, validation tools

## Implementation Changes

### 1. New Tool Implementation

Create `internal/impl/tools/subagent.go` with the following structure:

```go
type SubagentTool struct {
    agentRepo    domain.AgentRepository
    chatService  domain.ChatService
    sessionStore map[string]*AgentSession // In-memory session tracking
}

type AgentSession struct {
    ID           string
    ParentAgent  string
    Subagent     string
    TaskID       string
    Status       string // pending, active, completed, failed
    CreatedAt    time.Time
    CompletedAt  *time.Time
    Result       interface{}
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

### Standard SDLC Workflow

1. **Session Initialization**
   - User creates new development session
   - Project Manager agent is instantiated as orchestrator

2. **Requirements Gathering**
   ```go
   // Project Manager calls Product Owner subagent
   result := subagentTool.callSubagent("product-owner", "Analyze user requirements and create user story", userRequest)
   // Product Owner creates story document and returns to Project Manager
   ```

3. **Technical Design**
   ```go
   // Project Manager calls Architect subagent
   result := subagentTool.callSubagent("architect", "Design technical solution for story", storyData)
   // Architect creates technical specification and returns to Project Manager
   ```

4. **Project Planning**
   ```go
   // Project Manager calls planning subagent or handles internally
   tasks := subagentTool.callSubagent("planner", "Break down feature into tasks", techSpec)
   // Tasks are created and managed via Task tool
   ```

5. **Development Execution**
   ```go
   // Project Manager iterates through tasks
   for each task in tasks {
       result := subagentTool.callSubagent("developer", task.description, task.context)
       // Validate completion and update task status
   }
   ```

6. **Testing and Validation**
   ```go
   // Project Manager calls Tester subagent
   testResults := subagentTool.callSubagent("tester", "Test implemented feature", implementationData)
   // Review results and iterate if needed
   ```

## Technical Considerations

### 1. Concurrency Management
- Implement proper locking for session management
- Handle concurrent subagent invocations
- Prevent resource conflicts between agents

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
- **Task Tool**: Enhanced to support agent assignment and tracking
- **Memory Tool**: Used for maintaining context across agent sessions
- **File Tools**: Used by subagents for reading/writing project files

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

#### Migration Strategy

1. **Create Specialized Sub-Agents**:
   ```go
   // Image Generation Agent
   {
       ID:          "image-generator",
       Name:        "Image Generator",
       Description: "Specialized agent for generating images from text prompts",
       SystemPrompt: `You are an Image Generation specialist.
       Your role is to create high-quality images based on detailed prompts.
       You understand composition, style, lighting, and artistic techniques.`,
       Tools:       []string{"image_api"}, // Direct API access only
       Model:       "dall-e-3", // Image-capable model
   }

   // Vision Analysis Agent
   {
       ID:          "vision-analyst",
       Name:        "Vision Analyst",
       Description: "Specialized agent for analyzing and describing images",
       SystemPrompt: `You are a Vision Analysis specialist.
       Your role is to analyze images and provide detailed descriptions,
       identify objects, scenes, emotions, and contextual information.`,
       Tools:       []string{"vision_api"}, // Direct API access only
       Model:       "gpt-4-vision-preview", // Vision-capable model
   }
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

#### Benefits of Migration

- **Resource Isolation**: Image processing doesn't compete with general agent resources
- **Specialized Models**: Use optimal models for specific tasks (DALL-E for generation, GPT-4-Vision for analysis)
- **Better Error Handling**: Sub-agent failures don't crash the main agent
- **Scalability**: Multiple image tasks can run concurrently as separate sub-agents
- **Cost Tracking**: Better separation of costs for different types of processing

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

### Implementation Roadmap

1. **Phase 1**: Migrate existing image/vision tools to sub-agents
2. **Phase 2**: Add image model support to base agent framework
3. **Phase 3**: Implement multimodal message handling
4. **Phase 4**: Add image-specific agent configurations
5. **Phase 5**: Integrate with existing chat and tool systems

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

The Subagent Tool will transform the AI agent system from a single-agent model to a powerful multi-agent orchestration platform. By implementing hierarchical agent coordination, we enable complex task management and execution across the complete software development lifecycle, significantly enhancing the system's capabilities for handling sophisticated development workflows.