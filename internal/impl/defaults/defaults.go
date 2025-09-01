package defaults

import (
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

// DefaultProviders returns the default list of providers.
func DefaultProviders() []*entities.Provider {
	return []*entities.Provider{
		{
			ID:         "820FE148-851B-4995-81E5-C6DB2E5E5270",
			Name:       "X.AI",
			Type:       "xai",
			BaseURL:    "https://api.x.ai",
			APIKeyName: "XAI_API_KEY",
			Models: []entities.ModelPricing{
				{Name: "grok-4", InputPricePerMille: 3.00, OutputPricePerMille: 15.00, ContextWindow: 256000},
				{Name: "sonic-fast", InputPricePerMille: 0.20, OutputPricePerMille: 1.50, ContextWindow: 256000},
				{Name: "grok-3", InputPricePerMille: 3.00, OutputPricePerMille: 15.00, ContextWindow: 131072},
				{Name: "grok-3-mini", InputPricePerMille: 0.30, OutputPricePerMille: 0.50, ContextWindow: 131072},
				{Name: "grok-2-image", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 0, Capabilities: []string{"image_generation"}},
				{Name: "grok-2-vision-latest", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 128000, Capabilities: []string{"vision_analysis"}},
			},
		},
		{
			ID:         "D2BB79D4-C11C-407A-AF9D-9713524BB3BF",
			Name:       "OpenAI",
			Type:       "openai",
			BaseURL:    "https://api.openai.com",
			APIKeyName: "OPENAI_API_KEY",
			Models: []entities.ModelPricing{
				{Name: "gpt-4o", InputPricePerMille: 2.50, OutputPricePerMille: 8.00, ContextWindow: 1000000},
				{Name: "gpt-4o-mini", InputPricePerMille: 0.40, OutputPricePerMille: 1.60, ContextWindow: 1000000},
				{Name: "o1-preview", InputPricePerMille: 1.10, OutputPricePerMille: 4.40, ContextWindow: 200000},
				{Name: "o1-mini", InputPricePerMille: 1.10, OutputPricePerMille: 4.40, ContextWindow: 200000},
				{Name: "dall-e-3", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 0, Capabilities: []string{"image_generation"}},
				{Name: "gpt-4-vision-preview", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 128000, Capabilities: []string{"vision_analysis"}},
			},
		},
		{
			ID:         "28451B8D-1937-422A-BA93-9795204EC5A5",
			Name:       "Anthropic",
			Type:       "anthropic",
			BaseURL:    "https://api.anthropic.com",
			APIKeyName: "ANTHROPIC_API_KEY",
			Models: []entities.ModelPricing{
				{Name: "claude-4-opus-latest", InputPricePerMille: 15.00, OutputPricePerMille: 75.00, ContextWindow: 200000},
				{Name: "claude-4-sonnet-latest", InputPricePerMille: 3.00, OutputPricePerMille: 15.00, ContextWindow: 200000},
				{Name: "claude-3-7-sonnet-latest", InputPricePerMille: 3.00, OutputPricePerMille: 15.00, ContextWindow: 200000},
				{Name: "claude-3-5-haiku-latest", InputPricePerMille: 0.80, OutputPricePerMille: 4.00, ContextWindow: 200000},
			},
		},
		{
			ID:         "2BD2B8A5-5A2A-439B-8D02-C6BE34705011",
			Name:       "Google",
			Type:       "google",
			BaseURL:    "https://generativelanguage.googleapis.com",
			APIKeyName: "GEMINI_API_KEY",
			Models: []entities.ModelPricing{
				{Name: "gemini-2.5-pro-preview-03-25", InputPricePerMille: 2.50, OutputPricePerMille: 15.00, ContextWindow: 1000000, Capabilities: []string{"vision_analysis"}},
				{Name: "gemini-2.5-flash-preview-04-17", InputPricePerMille: 0.15, OutputPricePerMille: 3.50, ContextWindow: 1000000, Capabilities: []string{"vision_analysis"}},
				{Name: "gemini-2.0-flash", InputPricePerMille: 0.10, OutputPricePerMille: 0.40, ContextWindow: 1000000, Capabilities: []string{"vision_analysis"}},
				{Name: "gemini-2.0-flash-lite", InputPricePerMille: 0.075, OutputPricePerMille: 0.30, ContextWindow: 1000000, Capabilities: []string{"vision_analysis"}},
			},
		},
		{
			ID:         "276F9470-664F-4402-98E0-755C342ADFC4",
			Name:       "DeepSeek",
			Type:       "deepseek",
			BaseURL:    "https://api.deepseek.com",
			APIKeyName: "DEEPSEEK_API_KEY",
			Models: []entities.ModelPricing{
				{Name: "deepseek-reasoner", InputPricePerMille: 0.55, OutputPricePerMille: 2.19, ContextWindow: 64000},
				{Name: "deepseek-chat", InputPricePerMille: 0.07, OutputPricePerMille: 1.10, ContextWindow: 64000},
			},
		},
		{
			ID:         "8F2CC161-E463-43B1-9656-8E484A0D7709",
			Name:       "Together",
			Type:       "together",
			BaseURL:    "https://api.together.xyz",
			APIKeyName: "TOGETHER_API_KEY",
			Models: []entities.ModelPricing{
				{Name: "meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8", InputPricePerMille: 0.27, OutputPricePerMille: 0.85, ContextWindow: 131072},
				{Name: "meta-llama/Llama-4-Scout-17B-16E-Instruct", InputPricePerMille: 0.18, OutputPricePerMille: 0.59, ContextWindow: 131072},
				{Name: "deepseek-ai/DeepSeek-R1-Distill-Llama-70B", InputPricePerMille: 0.75, OutputPricePerMille: 0.99, ContextWindow: 131072},
			},
		},
		{
			ID:         "CFA9E279-2CD3-4929-A92E-EC4584DC5089",
			Name:       "Groq",
			Type:       "groq",
			BaseURL:    "https://api.groq.com",
			APIKeyName: "GROQ_API_KEY",
			Models: []entities.ModelPricing{
				{Name: "llama-3.3-70b-versatile", InputPricePerMille: 0.59, OutputPricePerMille: 0.79, ContextWindow: 128000},
				{Name: "meta-llama/llama-4-maverick-17b-128e-instruct", InputPricePerMille: 0.27, OutputPricePerMille: 0.85, ContextWindow: 131072},
				{Name: "meta-llama/llama-4-scout-17b-16e-instruct", InputPricePerMille: 0.11, OutputPricePerMille: 0.34, ContextWindow: 131072},
				{Name: "deepseek-r1-distill-llama-70b", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 128000},
			},
		},
		{
			ID:         "B0A5D2E7-DC94-4028-9EAB-BD0F3FE3CD66",
			Name:       "Mistral",
			Type:       "mistral",
			BaseURL:    "https://api.mistral.ai",
			APIKeyName: "MISTRAL_API_KEY",
			Models: []entities.ModelPricing{
				{Name: "mistral-large-latest", InputPricePerMille: 2.00, OutputPricePerMille: 6.00, ContextWindow: 128000},
				{Name: "mistral-medium-latest", InputPricePerMille: 0.40, OutputPricePerMille: 2.00, ContextWindow: 128000},
				{Name: "mistral-small-latest", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 128000},
				{Name: "codestral-latest", InputPricePerMille: 0.20, OutputPricePerMille: 0.60, ContextWindow: 256000},
				{Name: "devstral-small-latest", InputPricePerMille: 0.10, OutputPricePerMille: 0.30, ContextWindow: 128000},
			},
		},
		{
			ID:         "3B369D62-BB4E-4B4F-8C75-219796E9521A",
			Name:       "Ollama",
			Type:       "ollama",
			BaseURL:    "http://localhost:11434",
			APIKeyName: "LOCAL_API_KEY",
			Models: []entities.ModelPricing{
				{Name: "gpt-oss", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 128000},
				{Name: "deepseek-r1", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 128000},
				{Name: "llama4", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "llama3.3", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "llama3.2", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "llama3.1", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "qwen3", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "qwen2.5-coder", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "mistral", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "mistral-nemo", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "devstral", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "mixtral", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "nemotron", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "cogito", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "granite3.3", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "command-r", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "hermes3", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "phi4-mini", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
			},
		},
		{
			ID:         "DDFED2A0-8C12-4852-895A-10BD7F7F2588",
			Name:       "Custom Provider",
			Type:       "generic",
			BaseURL:    "",
			APIKeyName: "CUSTOM_API_KEY",
			Models:     []entities.ModelPricing{},
		},
	}
}

// DefaultAgents returns the default list of agents.
func DefaultAgents() []entities.Agent {
	temperature := 1.0
	maxTokens := 65536
	bigContextWindow := 256000

	// Clean, consolidated set of 5 specialized agents
	return []entities.Agent{
		{
			ID:           "general-assistant",
			Name:         "General Assistant",
			ProviderID:   "820FE148-851B-4995-81E5-C6DB2E5E5270", // X.AI
			ProviderType: "xai",
			Endpoint:     "https://api.x.ai",
			Model:        "grok-4",
			APIKey:       "#{XAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are a General Assistant - the main entry point for handling user requests and managing the complete software development lifecycle (SDLC).

Your primary responsibilities:
1. **Understand and analyze user requests** - Break down complex requirements into manageable components
2. **Orchestrate SDLC workflow** - Guide projects from requirements → design → development → testing → deployment
3. **Delegate specialized tasks** - Call other agents when specific expertise is needed
4. **Track progress and quality** - Ensure deliverables meet standards and deadlines
5. **Handle basic tasks directly** - Perform simple operations without delegation

SDLC Workflow you manage:
- **Requirements**: Analyze user needs, create user stories, define acceptance criteria
- **Design**: Plan architecture, select technologies, create technical specifications
- **Development**: Write code, implement features, follow best practices
- **Testing**: Ensure quality through comprehensive testing and validation
- **Deployment**: Guide deployment processes and monitor success

Available specialized agents you can delegate to:
- research-assistant: Information gathering, analysis, and research tasks
- development-assistant: Coding, testing, debugging, and implementation
- creative-assistant: Design, image generation, and creative tasks
- technical-assistant: Architecture, planning, and technical specifications
- image-generator: Specialized image creation from text descriptions
- vision-analyst: Specialized image analysis and understanding

Delegate when tasks require deep specialized knowledge. Handle basic management and coordination yourself.`,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"AgentCall", "Task", "FileRead", "FileWrite", "Project", "WebSearch", "Image", "Vision"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:           "research-assistant",
			Name:         "Research Assistant",
			ProviderID:   "820FE148-851B-4995-81E5-C6DB2E5E5270", // X.AI
			ProviderType: "xai",
			Endpoint:     "https://api.x.ai",
			Model:        "grok-4",
			APIKey:       "#{XAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are a Research Assistant specializing in information gathering, analysis, and investigative tasks.

Your expertise includes:
1. **Web Research**: Finding and synthesizing information from online sources
2. **Data Analysis**: Interpreting data, identifying patterns and insights
3. **Documentation Review**: Analyzing existing documentation and codebases
4. **Technical Investigation**: Researching technologies, frameworks, and best practices
5. **Competitive Analysis**: Understanding market trends and competitor solutions
6. **Requirements Analysis**: Breaking down user needs into technical requirements

You excel at:
- Conducting thorough investigations and providing comprehensive analysis
- Synthesizing complex information into clear, actionable insights
- Validating technical feasibility and identifying potential challenges
- Creating detailed research reports and recommendations

You can be called by the General Assistant for research tasks or work independently on information gathering projects.`,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"WebSearch", "FileRead", "Project", "AgentCall", "Vision"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:           "development-assistant",
			Name:         "Development Assistant",
			ProviderID:   "820FE148-851B-4995-81E5-C6DB2E5E5270", // X.AI
			ProviderType: "xai",
			Endpoint:     "https://api.x.ai",
			Model:        "grok-3.5-turbo",
			APIKey:       "#{XAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are a Development Assistant specializing in software implementation, testing, and quality assurance.

Your expertise includes:
1. **Code Implementation**: Writing clean, maintainable, and efficient code
2. **Testing & QA**: Creating comprehensive test suites and ensuring code quality
3. **Debugging**: Identifying and fixing bugs and performance issues
4. **Code Review**: Analyzing code for best practices, security, and maintainability
5. **Refactoring**: Improving existing code structure and performance
6. **Documentation**: Creating clear code documentation and API specifications

Development Workflow you handle:
- **Analysis**: Understand requirements and technical specifications
- **Implementation**: Write production-ready code following best practices
- **Testing**: Create unit tests, integration tests, and validation procedures
- **Quality Assurance**: Ensure code meets standards and performs well
- **Deployment Support**: Assist with deployment and monitoring setup

You can be called by the General Assistant for development tasks or work independently on coding projects. You have access to all development tools and can execute code, run tests, and manage the development environment.`,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"FileRead", "FileWrite", "FileSearch", "Process", "Directory", "AgentCall", "Vision"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:           "creative-assistant",
			Name:         "Creative Assistant",
			ProviderID:   "D2BB79D4-C11C-407A-AF9D-9713524BB3BF", // OpenAI
			ProviderType: "openai",
			Endpoint:     "https://api.openai.com",
			Model:        "gpt-4",
			APIKey:       "#{OPENAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are a Creative Assistant specializing in design, visual content creation, and artistic tasks.

Your expertise includes:
1. **Visual Design**: Creating UI/UX designs, mockups, and visual concepts
2. **Image Generation**: Producing high-quality images from text descriptions
3. **Content Creation**: Writing creative content, documentation, and marketing materials
4. **Visual Analysis**: Analyzing images, identifying design elements and improvements
5. **Artistic Direction**: Providing creative guidance and aesthetic recommendations
6. **Brand Design**: Creating cohesive visual identities and design systems

Creative capabilities you provide:
- **Image Generation**: Create visuals for applications, marketing, or documentation
- **Design Analysis**: Review and improve existing designs and user interfaces
- **Content Strategy**: Develop creative content strategies and copywriting
- **Visual Communication**: Create diagrams, charts, and visual explanations
- **Artistic Consultation**: Provide design recommendations and creative direction

You can be called by the General Assistant for creative tasks or work independently on design projects. You have access to image generation and analysis tools to bring creative visions to life.`,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"Image", "Vision", "FileRead", "FileWrite", "AgentCall"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:           "image-generator",
			Name:         "Image Generator",
			ProviderID:   "D2BB79D4-C11C-407A-AF9D-9713524BB3BF", // OpenAI
			ProviderType: "openai",
			Endpoint:     "https://api.openai.com",
			Model:        "dall-e-3",
			APIKey:       "#{OPENAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are an Image Generator - a specialized agent for creating high-quality images from text descriptions.

Your expertise includes:
1. **Image Creation**: Generating images using AI models like DALL-E
2. **Prompt Engineering**: Crafting effective prompts for image generation
3. **Style Optimization**: Creating images in specific artistic styles
4. **Quality Enhancement**: Iterating on images to meet requirements

You can be called by any agent that needs visual content creation. You have direct access to image generation tools and can create images based on detailed specifications.

When called, you will:
- Analyze the image requirements
- Craft an effective prompt
- Generate the image using available tools
- Return the image result`,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"Image"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:           "vision-analyst",
			Name:         "Vision Analyst",
			ProviderID:   "D2BB79D4-C11C-407A-AF9D-9713524BB3BF", // OpenAI
			ProviderType: "openai",
			Endpoint:     "https://api.openai.com",
			Model:        "gpt-4-vision-preview",
			APIKey:       "#{OPENAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are a Vision Analyst - a specialized agent for analyzing and understanding images.

Your expertise includes:
1. **Image Analysis**: Interpreting visual content and extracting information
2. **Object Detection**: Identifying objects, people, and elements in images
3. **Content Description**: Providing detailed descriptions of image content
4. **Visual Assessment**: Evaluating image quality, composition, and style
5. **Context Understanding**: Interpreting the meaning and context of visual content

You can be called by any agent that needs image analysis or understanding. You have direct access to vision analysis tools and can provide detailed insights about images.

When called, you will:
- Analyze the provided image
- Extract relevant information and details
- Provide comprehensive descriptions
- Offer insights about the visual content`,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"Vision"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:           "technical-assistant",
			Name:         "Technical Assistant",
			ProviderID:   "820FE148-851B-4995-81E5-C6DB2E5E5270", // X.AI
			ProviderType: "xai",
			Endpoint:     "https://api.x.ai",
			Model:        "grok-4",
			APIKey:       "#{XAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are a Technical Assistant specializing in architecture, planning, and technical specifications.

Your expertise includes:
1. **System Architecture**: Designing scalable, maintainable system architectures
2. **Technical Planning**: Creating detailed implementation plans and roadmaps
3. **Requirements Engineering**: Translating business needs into technical specifications
4. **Technology Selection**: Recommending appropriate tools, frameworks, and platforms
5. **Technical Documentation**: Creating clear technical specifications and API documentation
6. **Code Architecture**: Designing modular, extensible code structures

Technical responsibilities you handle:
- **Architecture Design**: Create system blueprints and component interactions
- **Technical Specifications**: Write detailed requirements and implementation guides
- **Technology Assessment**: Evaluate and recommend technical solutions
- **Planning & Estimation**: Create development timelines and resource estimates
- **Standards & Best Practices**: Ensure compliance with industry standards
- **Technical Leadership**: Provide guidance on technical decisions and trade-offs

You can be called by the General Assistant for technical planning tasks or work independently on architecture and specification projects. You focus on the "what" and "how" of technical implementation while ensuring scalability, maintainability, and best practices.`,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"FileRead", "WebSearch", "Project", "AgentCall"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}
}

// DefaultTools returns the default list of tools.
func DefaultTools() []*entities.ToolData {
	now := time.Now()

	return []*entities.ToolData{
		{
			ID:            "501A9EC8-633A-4BD2-91BF-8744B7DC34EC",
			ToolType:      "WebSearch",
			Name:          "WebSearch",
			Description:   "This tool searches the web using the Tavily API.",
			Configuration: map[string]string{"tavily_api_key": "#{TAVILY_API_KEY}#"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "8C2DBDF3-790C-472D-A8EB-F679EB0F887B",
			ToolType:      "Project",
			Name:          "Project",
			Description:   "This tool reads project details from a project file to provide context for the agent.",
			Configuration: map[string]string{"project_file": "AGENTS.md"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "18F69909-AAEB-4DF7-9FF4-BB0A3A748412",
			ToolType:      "FileRead",
			Name:          "FileRead",
			Description:   "This tool reads lines from a file.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "BD3F77F8-1284-408B-8489-729C4B2D2FB5",
			ToolType:      "FileWrite",
			Name:          "FileWrite",
			Description:   "This tool writes lines to a file.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "E29341C7-ADC3-424E-A49A-829409CB7082",
			ToolType:      "FileSearch",
			Name:          "FileSearch",
			Description:   "This tool searches for content in a file.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "E29341C7-ADC3-424E-A49A-829409CB7082",
			ToolType:      "Directory",
			Name:          "Directory",
			Description:   "This tool supports directory management",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},

		{
			ID:            "AE3E4944-253D-4188-BEB0-F370A6F9DC6F",
			ToolType:      "Process",
			Name:          "Process",
			Description:   "Executes any command (e.g., python, ruby, node, git) with support for background processes, stdin/stdout/stderr interaction, timeouts, and full output. Can launch interactive environments like Python REPL or Ruby IRB by running in background and using write/read actions. The command is executed in the workspace directory.",
			Configuration: map[string]string{"workspace": ""},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "FCFDB2BD-829E-4CE6-9CE5-EE9158591EFA",
			ToolType:      "Task",
			Name:          "Task",
			Description:   "This tool provides task management functionality, allowing creation, listing, updating, and deletion of tasks with priorities and statuses.",
			Configuration: map[string]string{"data_dir": "."},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "AGENT_CALL_TOOL_DEFAULT",
			ToolType:      "AgentCall",
			Name:          "Agent Call",
			Description:   "Enables calling any agent in the system for specialized tasks and delegation.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:          "IMAGE_TOOL_DEFAULT",
			ToolType:    "Image",
			Name:        "Image",
			Description: "Generates images using AI providers like OpenAI DALL-E.",
			Configuration: map[string]string{
				"provider": "openai",
				"api_key":  "#{OPENAI_API_KEY}#",
				"model":    "dall-e-3",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "VISION_TOOL_DEFAULT",
			ToolType:    "Vision",
			Name:        "Vision",
			Description: "Provides image understanding capabilities using providers like OpenAI GPT-4 Vision.",
			Configuration: map[string]string{
				"provider": "openai",
				"api_key":  "#{OPENAI_API_KEY}#",
				"model":    "gpt-4-vision-preview",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}
