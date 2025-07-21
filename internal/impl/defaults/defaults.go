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
				{Name: "grok-3-fast", InputPricePerMille: 5.00, OutputPricePerMille: 25.00, ContextWindow: 131072},
				{Name: "grok-3", InputPricePerMille: 3.00, OutputPricePerMille: 15.00, ContextWindow: 131072},
				{Name: "grok-3-mini-fast", InputPricePerMille: 0.60, OutputPricePerMille: 4.00, ContextWindow: 131072},
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
				{Name: "llama4", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "llama3.3", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "llama3.2", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "llama3.1", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "qwen3", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "qwen2.5", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "qwen2.5-coder", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "mistral", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "mistral-nemo", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "devstral", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "mixtral", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "nemotron", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "cogito", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "granite3.3", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
				{Name: "granite3.2", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
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
### Introduction and Role

You are an interactive CLI tool that helps users with software engineering tasks. Use the instructions below and the tools available to you to assist the user.

### Memory

If the current working directory contains a AIAGENT.md file, it is added to project->read tool for:

1. Storing bash commands (e.g., build, test).
2. Recording code style preferences.
3. Maintaining codebase information.

Proactively ask users to add commands or preferences to AIAGENT.md for future reference.

### Tone and Style

- Be concise, direct, and to the point.
- Use GitHub-flavored Markdown for formatting.
- Output text for user communication; use tools only for tasks.
- If unable to help, offer alternatives.
- Minimize tokens: Respond in 1-3 sentences or a short paragraph if possible.

### Following Conventions
- Mimic existing code styles, libraries, and patterns.
- Verify library availability before use.
- Follow security best practices (e.g., never commit secrets).
- Do not add comments to code unless requested.

### Doing Tasks

For software engineering tasks (e.g., bugs, features):

1. Use the project -> get_structure or get_source tool to understand the codebase.
2. Implement changes using available tools.
3. Verify with tests; check for testing commands.
4. Run lint and typecheck commands if available; suggest adding to AIAGENT.md.

- Never commit changes unless explicitly asked.
`
	maxTokens := 65536
	contextWindow := 131072
	bigContextWindow := 256000

	// Local system prompt for Ollama models
	localSystemPrompt := `
Help users with coding, debugging, and enhancing projects leveraging the tools provided. Be concise, proactive, and persistent: analyze tasks quickly, use tools to edit files, run commands, and iterate until success. Keep responses short, directly addressing queries without preamble.
`
	localMaxTokens := 4096
	localContextWindow := 8192
	tools := []string{"WebSearch", "Project", "FileRead", "FileWrite", "FileSearch", "Bash", "Rg", "Find", "Tree", "Git", "Go", "Python", "Node", "Ruby", "Dotnet"}

	return []entities.Agent{
		{
			ID:              "1A3F3DCB-255D-46B3-A4F4-E2E118FBA82B",
			Name:            "Grok",
			ProviderID:      "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType:    "xai",
			Endpoint:        "https://api.x.ai",
			Model:           "grok-3-mini",
			APIKey:          "#{XAI_API_KEY}#",
			SystemPrompt:    systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &contextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "7F1C8EDF-7899-4691-997C-421795719EB3",
			Name:            "GPT",
			ProviderID:      "D2BB79D4-C11C-407A-AF9D-9713524BB3BF",
			ProviderType:    "openai",
			Endpoint:        "https://api.openai.com",
			Model:           "gpt-4.1-nano",
			APIKey:          "#{OPENAI_API_KEY}#",
			SystemPrompt:    systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &contextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "65DD6A7E-992E-4603-AFB1-F6F9314DFA52",
			Name:            "Claude",
			ProviderID:      "28451B8D-1937-422A-BA93-9795204EC5A5",
			ProviderType:    "anthropic",
			Endpoint:        "https://api.anthropic.com",
			Model:           "claude-3-5-haiku-latest",
			APIKey:          "#{ANTHROPIC_API_KEY}#",
			SystemPrompt:    systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &contextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "B64CA989-A4E6-4870-9C2D-AF9848C98EF7",
			Name:            "Gemini",
			ProviderID:      "2BD2B8A5-5A2A-439B-8D02-C6BE34705011",
			ProviderType:    "google",
			Endpoint:        "https://generativelanguage.googleapis.com",
			Model:           "gemini-2.0-flash",
			APIKey:          "#{GEMINI_API_KEY}#",
			SystemPrompt:    systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &contextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "A7D02F67-283D-41A5-A6F0-41B3DE3EA454",
			Name:            "Maverick",
			ProviderID:      "CFA9E279-2CD3-4929-A92E-EC4584DC5089",
			ProviderType:    "groq",
			Endpoint:        "https://api.groq.com",
			Model:           "meta-llama/llama-4-maverick-17b-128e-instruct",
			APIKey:          "#{GROQ_API_KEY}#",
			SystemPrompt:    systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &contextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "360B443F-7689-4DDE-BBB1-D929F65D446B",
			Name:            "Codestral",
			ProviderID:      "B0A5D2E7-DC94-4028-9EAB-BD0F3FE3CD66",
			ProviderType:    "mistral",
			Endpoint:        "https://api.mistral.ai",
			Model:           "codestral-latest",
			APIKey:          "#{MISTRAL_API_KEY}#",
			SystemPrompt:    systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "F42E8A17-06D3-457D-94F1-72D0FF150865",
			Name:            "Llama",
			ProviderID:      "3B369D62-BB4E-4B4F-8C75-219796E9521A",
			ProviderType:    "ollama",
			Endpoint:        "http://localhost:11434",
			Model:           "llama3.1",
			APIKey:          "n/a",
			SystemPrompt:    localSystemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &localMaxTokens,
			ContextWindow:   &localContextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "B9A9C0F4-52F4-4458-9E69-6C7C16F1648B",
			Name:            "Qwen",
			ProviderID:      "3B369D62-BB4E-4B4F-8C75-219796E9521A",
			ProviderType:    "ollama",
			Endpoint:        "http://localhost:11434",
			Model:           "qwen3",
			APIKey:          "n/a",
			SystemPrompt:    localSystemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &localMaxTokens,
			ContextWindow:   &localContextWindow,
			ReasoningEffort: "",
			Tools:           tools,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "A0DB85D1-69A3-4D2B-94C5-907709F2D360",
			Name:            "Cogito",
			ProviderID:      "3B369D62-BB4E-4B4F-8C75-219796E9521A",
			ProviderType:    "ollama",
			Endpoint:        "http://localhost:11434",
			Model:           "cogito",
			APIKey:          "n/a",
			SystemPrompt:    localSystemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &localMaxTokens,
			ContextWindow:   &localContextWindow,
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
			Configuration: map[string]string{"project_file": "AIAGENT.md"},
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
			ID:            "AE3E4944-253D-4188-BEB0-F370A6F9DC6F",
			ToolType:      "Bash",
			Name:          "Bash",
			Description:   "This tool executes a bash command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "2D22D358-51A7-432D-851F-4E7198084BB5",
			ToolType:      "Process",
			Name:          "Rg",
			Description:   "Searches files for a regex pattern. Very fast, respects .gitignore, ignores hidden files. The command is executed in the workspace directory. The 'arguments' argument should be provided.",
			Configuration: map[string]string{"command": "rg"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "4E169241-1227-47A9-909F-34D40657A6A9",
			ToolType:      "Process",
			Name:          "Find",
			Description:   "Searches for files based on criteria like name, type, size, and modification time within a directory hierarchy. Provide the full command, including the search path, as the 'arguments' argument. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "find"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "961B2F87-00B7-40B9-A70B-4327918D398A",
			ToolType:      "Process",
			Name:          "Tree",
			Description:   "Displays the directory structure in a hierarchical tree format. Provide the path to display as the 'arguments' argument. If no path is provided, the current directory is displayed. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "tree"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "4EA3F4A2-EFCD-4E9A-A5F8-4DFFAFB018E7",
			ToolType:      "Process",
			Name:          "Git",
			Description:   "Executes git commands. Use this for version control operations. Provide the git command and arguments as a single string. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "git"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "8C4E1573-59D9-463B-AF5F-1EA7620F469D",
			ToolType:      "Process",
			Name:          "Go",
			Description:   "Executes Go commands. Use this to build, test, and run Go programs. Provide the go command and arguments as a single string. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "go"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "50A77E90-D6D3-410C-A7B4-6A3E5E58253E",
			ToolType:      "Process",
			Name:          "Python",
			Description:   "Executes Python. Use this to run Python scripts or execute Python code. Provide the python command and arguments as a single string. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "python"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "5CCC1A23-081C-4DDC-BBEC-7A877C749825",
			ToolType:      "Process",
			Name:          "Ruby",
			Description:   "Executes Ruby. Use this to run Python scripts or execute Ruby code. Provide the python command and arguments as a single string. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "ruby"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "382EE72F-68A2-41C2-B2B6-1F729324BCEC",
			ToolType:      "Process",
			Name:          "Node",
			Description:   "Executes Node.js commands. Use this to run JavaScript files or execute Node.js code. Provide the node command and arguments as a single string. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "node"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "3D596BD2-BB1E-4D4A-8254-F08AC5D75BEA",
			ToolType:      "Process",
			Name:          "Dotnet",
			Description:   "Executes Dotnet commands. Use this to work with Dotnet projects. Provide the dotnet command and arguments as a single string. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "dotnet"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
}
