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
				{Name: "grok-code-fast", InputPricePerMille: 0.20, OutputPricePerMille: 1.50, ContextWindow: 256000},
				{Name: "grok-3", InputPricePerMille: 3.00, OutputPricePerMille: 15.00, ContextWindow: 131072},
				{Name: "grok-3-mini", InputPricePerMille: 0.30, OutputPricePerMille: 0.50, ContextWindow: 131072},
			},
		},
		{
			ID:         "D2BB79D4-C11C-407A-AF9D-9713524BB3BF",
			Name:       "OpenAI",
			Type:       "openai",
			BaseURL:    "https://api.openai.com",
			APIKeyName: "OPENAI_API_KEY",
			Models: []entities.ModelPricing{
				{Name: "gpt-4.1", InputPricePerMille: 2.50, OutputPricePerMille: 8.00, ContextWindow: 1000000},
				{Name: "gpt-4.1-mini", InputPricePerMille: 0.40, OutputPricePerMille: 1.60, ContextWindow: 1000000},
				{Name: "gpt-4.1-nano", InputPricePerMille: 0.10, OutputPricePerMille: 0.40, ContextWindow: 1000000},
				{Name: "o4-mini", InputPricePerMille: 1.10, OutputPricePerMille: 4.40, ContextWindow: 200000},
				{Name: "o3", InputPricePerMille: 2.50, OutputPricePerMille: 40.00, ContextWindow: 200000},
				{Name: "o3-mini", InputPricePerMille: 1.10, OutputPricePerMille: 4.40, ContextWindow: 200000},
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
				{Name: "gemini-2.5-pro-preview-03-25", InputPricePerMille: 2.50, OutputPricePerMille: 15.00, ContextWindow: 1000000},
				{Name: "gemini-2.5-flash-preview-04-17", InputPricePerMille: 0.15, OutputPricePerMille: 3.50, ContextWindow: 1000000},
				{Name: "gemini-2.0-flash", InputPricePerMille: 0.10, OutputPricePerMille: 0.40, ContextWindow: 1000000},
				{Name: "gemini-2.0-flash-lite", InputPricePerMille: 0.075, OutputPricePerMille: 0.30, ContextWindow: 1000000},
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
	systemPrompt := `
### Interface

You are built as an interactive CLI tool that helps users with software engineering tasks. Use the instructions below and the tools available to you to assist the user.

### Memory

If the current working directory contains a AGENTS.md file, it is added to the Project tool for:

1. Storing bash commands (e.g., build, test).
2. Recording code style preferences.
3. Maintaining codebase information.

Proactively ask users to add commands or preferences to AGENTS.md for future reference.

### Tone and Style

- Be concise, direct, and to the point.
- Use GitHub-flavored Markdown for formatting.
- Output text for user communication; use tools only for tasks.
- If unable to help, offer alternatives.
- Minimize tokens: Respond in 1-3 sentences or a short paragraph if possible.

### Tool Usage

You use tools via function calls to help you solve questions. Make sure to use the following format for function calls, including the <xai:function_call> and </xai:function_call tags.

Function call should follow the following XML-inspired format:
<xai:function_call name="example_tool_name">
<parameter name="arg1">value1
	`

	maxTokens := 65536
	bigContextWindow := 256000

	tools := []string{"WebSearch", "Project", "Task", "FileRead", "FileWrite", "FileSearch", "Directory", "Process"}

	return []entities.Agent{
		{
			ID:              "5AEFC437-A72E-4B47-901F-865DDF6D8B74",
			Name:            "Research",
			ProviderID:      "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType:    "xai",
			Endpoint:        "https://api.x.ai",
			Model:           "grok-code-fast",
			APIKey:          "#{XAI_API_KEY}#",
			SystemPrompt:    `### Introduction and Role\n\nYou are a Research agent, specializing in gathering information, analyzing data, and conducting thorough investigations.` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "54AE685D-8A73-423A-A10E-EF7BE9BF8CB8",
			Name:            "Design",
			ProviderID:      "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType:    "xai",
			Endpoint:        "https://api.x.ai",
			Model:           "grok-code-fast",
			APIKey:          "#{XAI_API_KEY}#",
			SystemPrompt:    `### Introduction and Role\n\nYou are a Design agent, specializing in software architecture, system design, and high-level planning. Use tools to analyze codebases and propose designs.` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "B020132C-331A-436B-A8BA-A8639BC20436",
			Name:            "Plan",
			ProviderID:      "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType:    "xai",
			Endpoint:        "https://api.x.ai",
			Model:           "grok-code-fast",
			APIKey:          "#{XAI_API_KEY}#",
			SystemPrompt:    `### Introduction and Role\n\nYou are a Plan agent, specializing in creating detailed plans, breaking down tasks, and outlining steps for implementation.` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "6B0866FA-F10B-496C-93D5-7263B0F936B3",
			Name:            "Build",
			ProviderID:      "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType:    "xai",
			Endpoint:        "https://api.x.ai",
			Model:           "grok-code-fast",
			APIKey:          "#{XAI_API_KEY}#",
			SystemPrompt:    `### Introduction and Role\n\nYou are a Build agent, specializing in writing code, implementing features, and refactoring based on plans and designs.` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "E8A375A3-81BC-4EAB-8ADC-F62F94FD81D1",
			Name:            "Test",
			ProviderID:      "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType:    "xai",
			Endpoint:        "https://api.x.ai",
			Model:           "grok-code-fast",
			APIKey:          "#{XAI_API_KEY}#",
			SystemPrompt:    `### Introduction and Role\n\nYou are a Test agent, specializing in writing tests, debugging, and ensuring code quality through verification.` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           tools,
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
	}
}
