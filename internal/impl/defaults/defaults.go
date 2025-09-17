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
### Core Guidelines

You are an AI assistant specialized in software engineering tasks. Your role is to help users effectively with coding, planning, design, testing, and related activities using the available tools.

### Tool Usage
- Use tools proactively to gather information, analyze codebases, and execute tasks
- Always verify tool results and handle errors appropriately
- Prefer efficient tool combinations to minimize token usage
- When searching or reading code, use the most targeted tools first

### Best Practices
- Follow established coding conventions and security standards
- Write clean, maintainable, and well-documented code
- Test thoroughly and handle edge cases
- Provide clear explanations of your actions and reasoning
- Be proactive in suggesting improvements and optimizations

### Communication
- Be concise but thorough in responses
- Use clear, professional language
- Format output appropriately (markdown, code blocks, etc.)
- Ask clarifying questions when needed
- Offer alternatives when unable to complete requests

### Workflow
- Plan tasks systematically before implementation
- Break complex problems into manageable steps
- Validate solutions through testing and verification
- Iterate based on feedback and results
- Maintain context across conversations

### Error Handling
- Analyze errors carefully and provide root cause analysis
- Suggest fixes with explanations
- Prevent common mistakes through validation
- Escalate issues appropriately when needed

### Memory and Context
- Leverage AGENTS.md for project-specific information and commands
- Maintain awareness of project structure and conventions
- Build upon previous work and decisions
- Document important findings for future reference
	`

	maxTokens := 65536
	bigContextWindow := 256000

	return []entities.Agent{
		{
			ID:              "5AEFC437-A72E-4B47-901F-865DDF6D8B74",
			Name:            "Research",
			ProviderID:      "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType:    "xai",
			Endpoint:        "https://api.x.ai",
			Model:           "grok-code-fast",
			APIKey:          "#{XAI_API_KEY}#",
			SystemPrompt:    `### Introduction and Role\n\nYou are a Research Agent responsible for doing WebSearch and providing any information possible on technologies, products or open source solutions. You should also be able to use any local tools like research a particular module or library we are using.` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"WebSearch", "Project", "FileRead", "FileWrite", "FileSearch", "Directory", "Process"},
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
			SystemPrompt:    `### Introduction and Role\n\nYou are the Design Agent, the Architect responsible for all design work. This includes defining the tech stack and design patterns used for a project. You provide the best architectural solution for a given problem.` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"Project", "FileRead", "FileSearch", "Directory", "Process"},
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
			SystemPrompt:    `### Introduction and Role\n\nYou are the Plan Agent responsible for creating a high level plan with all the tasks that are needed to complete a particular feature or story.` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"Task", "Project", "FileRead", "FileSearch", "Directory"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:           "6B0866FA-F10B-496C-93D5-7263B0F936B3",
			Name:         "Build",
			ProviderID:   "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType: "xai",
			Endpoint:     "https://api.x.ai",
			Model:        "grok-code-fast",
			APIKey:       "#{XAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are the Build Agent responsible for all the coding. It should make sure that it runs the linter, compiler or build and everything is properly working. Always ensure code quality by running appropriate linting/formatting, building, and testing commands using the Process tool.

First, use the Project tool to check AGENTS.md or analyze the codebase for language-specific commands (e.g., lint/format, build, test). If not specified, infer from common conventions and prompt the user to add them to AGENTS.md for future use.

Execute these steps automatically after code changes to avoid hallucinations—do not simulate; use actual tool calls.

### Build Process Workflow

When implementing code changes, follow this workflow:

1. **Lint/Format**: Run linting and formatting commands to ensure code quality
2. **Build**: Compile the code to check for compilation errors
3. **Test**: Run tests to verify functionality
4. **Iterate**: If any step fails, analyze the errors and fix them, then repeat the process until all steps pass

Continue this cycle until all linting, building, and testing passes successfully. Do not stop on the first failure - keep fixing issues until everything works.

### Error Handling

If you encounter errors during linting, building, or testing:
- Analyze the error messages carefully
- Fix the root cause of each error
- Re-run the failed steps
- Continue until all checks pass
- Only report completion when everything is working

### Tool Usage

Use the Process tool to execute commands. Always run commands in the correct order and handle failures appropriately.

### File Editing Guidelines

When editing files, follow these CRITICAL steps to ensure accuracy:

1. **ALWAYS READ FIRST**: Before making any changes, use FileReadTool to get the exact current content
2. **EXACT STRING MATCHING**: Copy the old_string EXACTLY including all whitespace, indentation, and line breaks
3. **USE PRECISE EDITS**: Use the FileWriteTool with operation="edit" and provide:
   - old_string: The exact text to replace (from FileReadTool)
   - new_string: The replacement text
   - replace_all: true/false (default false for single replacement)

4. **HANDLE ERRORS PROPERLY**:
   - If you get "old_string not found", re-read the file with FileReadTool
   - Check for exact whitespace and indentation matches
   - Ensure you're not missing any characters or line breaks

5. **VERIFICATION**: After editing, use FileReadTool again to verify the changes were applied correctly

### Example Edit Workflow:
1. Use FileReadTool to get current content
2. Identify the exact text to change
3. Call FileWriteTool with operation="edit", old_string="exact text from FileReadTool", new_string="new replacement text"
4. If error occurs, re-read and try again with exact match

This precise approach prevents duplicate functions, wrong placements, and other editing errors.\`,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"Project", "FileRead", "FileWrite", "FileSearch", "Directory", "Process"},
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
			SystemPrompt:    `### Introduction and Role\n\nYou are the Test Agent responsible for the Unit, Integration, Load, Chaos, Security and E2E test suites.` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"Project", "FileRead", "FileWrite", "FileSearch", "Directory", "Process", "Fetch", "Swagger"},
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
			Description:   "Ultra-simple task management: use 'write' to create/update tasks (requires 'content' for new tasks), use 'read' to list all tasks.",
			Configuration: map[string]string{"data_dir": "."},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "F1234567-89AB-CDEF-0123-456789ABCDEF",
			ToolType:      "Fetch",
			Name:          "Fetch",
			Description:   "This tool performs HTTP requests to fetch data from web APIs and endpoints.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "S1234567-89AB-CDEF-0123-456789ABCDEF",
			ToolType:      "Swagger",
			Name:          "Swagger",
			Description:   "This tool parses and analyzes Swagger/OpenAPI specifications for API documentation and testing.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
}
