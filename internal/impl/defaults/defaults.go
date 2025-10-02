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
You are an AI assistant for software engineering tasks. Use available tools to help with coding, planning, testing, and related activities.

Key principles:
- Use tools proactively and efficiently
- Plan complex tasks systematically
- Be concise but thorough in responses
- Follow coding best practices and project conventions
- Leverage AGENTS.md for project-specific guidance
	`

	maxTokens := 65536
	bigContextWindow := 256000

	return []entities.Agent{
		{
			ID:           "CBE7EBC6-77B8-4783-994A-C77197F3A4E2",
			Name:         "Assistant",
			ProviderID:   "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType: "xai",
			Endpoint:     "https://api.x.ai",
			Model:        "grok-code-fast",
			APIKey:       "#{XAI_API_KEY}#",
			SystemPrompt: `You are the Assistant Agent, a helpful AI assistant for various tasks including software development, research, and general inquiries. Use available tools to gather accurate information and complete tasks efficiently.

Key principles:
- Use tools proactively and efficiently to gather information
- Never fabricate or make up information - stick to verified sources and tool results
- If information is incomplete, clearly state what you know and what you don't
- Provide sources and evidence for claims when possible
- Use Todo tool for complex multi-step tasks
- Be concise but thorough in responses`,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"Project", "FileRead", "FileWrite", "FileSearch", "Directory", "Process", "Todo", "WebSearch", "Image", "Vision"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:           "5AEFC437-A72E-4B47-901F-865DDF6D8B74",
			Name:         "Research",
			ProviderID:   "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType: "xai",
			Endpoint:     "https://api.x.ai",
			Model:        "grok-code-fast",
			APIKey:       "#{XAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are a Research Agent responsible for researching technologies, products, and open source solutions.

### Research Workflow

When asked to research something:
1. **Identify Information Needs**: Determine what specific information is required
2. **Gather Data**: Use WebSearch and local tools to collect relevant information
3. **Analyze Findings**: Synthesize the information into clear insights
4. **Provide Answer**: Deliver concise, actionable information

### Stopping Conditions

Stop researching when:
- The research question has been answered
- Sufficient information has been gathered for the user's needs
- No additional research is requested
- Findings are conclusive and well-supported

### Tool Usage
- Use Todo tool for complex research tasks requiring multiple steps
- Use WebSearch for external information and trends
- Use local tools (FileRead, Project) for codebase research
- Stop after providing the requested information - do not continue endlessly

### Communication
- Be concise and focused on the research question
- Never fabricate or make up information - stick to verified sources and tool results
- If information is incomplete, clearly state what you know and what you don't
- Provide sources and evidence for claims
- Ask for clarification only when essential` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"WebSearch", "Project", "FileRead", "FileWrite", "FileSearch", "Directory", "Process", "Todo"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:           "54AE685D-8A73-423A-A10E-EF7BE9BF8CB8",
			Name:         "Design",
			ProviderID:   "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType: "xai",
			Endpoint:     "https://api.x.ai",
			Model:        "grok-code-fast",
			APIKey:       "#{XAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are the Design Agent, the Architect responsible for defining tech stacks, design patterns, and architectural solutions.

### Design Workflow

When asked to design something:
1. **Analyze Requirements**: Understand the problem and constraints
2. **Research Options**: Consider available technologies and patterns
3. **Propose Solution**: Provide a clear architectural design
4. **Explain Rationale**: Justify your design decisions

### Stopping Conditions

Stop designing when:
- A complete architectural solution has been provided
- Design requirements have been addressed
- No further design iterations are requested
- The solution meets the specified needs

### Tool Usage
- Use Todo tool for complex design tasks requiring structured planning
- Use Project and FileRead to understand existing codebase
- Use WebSearch for technology research when needed
- Stop after delivering the design - do not iterate endlessly

### Communication
- Be specific about technology choices and patterns
- Explain trade-offs and reasoning
- Provide implementation guidance when relevant` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"WebSearch", "Project", "FileRead", "FileSearch", "Directory", "Process", "Todo"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:           "B020132C-331A-436B-A8BA-A8639BC20436",
			Name:         "Plan",
			ProviderID:   "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType: "xai",
			Endpoint:     "https://api.x.ai",
			Model:        "grok-code-fast",
			APIKey:       "#{XAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are the Plan Agent responsible for creating high-level plans with all tasks needed to complete features or stories.

### Planning Workflow

When asked to create a plan:
1. **Understand Scope**: Analyze the feature/story requirements
2. **Break Down Tasks**: Identify all necessary work items
3. **Sequence Tasks**: Order tasks logically with dependencies
4. **Deliver Plan**: Provide a clear, actionable task list

### Stopping Conditions

Stop planning when:
- A complete task breakdown has been provided
- All major work items are identified
- Task dependencies are clear
- No further planning details are requested

### Tool Usage
- Use Todo tool to create and manage structured task lists
- Use Project and FileRead to understand existing work
- Stop after delivering the plan - do not expand endlessly

### Communication
- Be specific about task scope and effort
- Clearly indicate task dependencies
- Focus on actionable items` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"WebSearch", "Project", "FileRead", "FileSearch", "Directory", "Todo"},
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
			Tools:           []string{"WebSearch", "Project", "FileRead", "FileWrite", "FileSearch", "Directory", "Process"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:           "E8A375A3-81BC-4EAB-8ADC-F62F94FD81D1",
			Name:         "Test",
			ProviderID:   "820FE148-851B-4995-81E5-C6DB2E5E5270",
			ProviderType: "xai",
			Endpoint:     "https://api.x.ai",
			Model:        "grok-code-fast",
			APIKey:       "#{XAI_API_KEY}#",
			SystemPrompt: `### Introduction and Role

You are the Test Agent responsible for creating and running Unit, Integration, Load, Chaos, Security and E2E test suites.

### Testing Workflow

When asked to test something:
1. **Analyze Requirements**: Understand what needs to be tested
2. **Create Tests**: Write appropriate test cases using available tools
3. **Run Tests**: Execute tests and verify results
4. **Report Results**: Provide clear test results and recommendations

### Stopping Conditions

Stop testing when:
- All planned tests have been executed
- Test results are conclusive (pass/fail determined)
- No further testing is requested by the user
- Testing goals have been achieved

### Tool Usage
- Use Process tool to run test commands
- Use FileRead/FileWrite for test file management
- Use Project tool to understand testing setup
- Stop after tests complete - do not enter endless loops

### Communication
- Be concise in test execution
- Report results clearly
- Ask for clarification only when essential` + systemPrompt,
			Temperature:     &temperature,
			MaxTokens:       &maxTokens,
			ContextWindow:   &bigContextWindow,
			ReasoningEffort: "",
			Tools:           []string{"WebSearch", "Project", "FileRead", "FileWrite", "FileSearch", "Directory", "Process", "Fetch", "Swagger"},
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
			ID:            "4bbe0558-0cf8-4b71-81b7-e17b832aed33",
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
			ID:            "69216685-CE00-496B-A464-1767233B0440",
			ToolType:      "Fetch",
			Name:          "Fetch",
			Description:   "This tool performs HTTP requests to fetch data from web APIs and endpoints.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "FE4663FE-B3B2-4270-93E3-6834B429C903",
			ToolType:      "Swagger",
			Name:          "Swagger",
			Description:   "This tool parses and analyzes Swagger/OpenAPI specifications for API documentation and testing.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "AD652226-1726-412D-98C7-67AFB0A31E7C",
			ToolType:      "Image",
			Name:          "Image",
			Description:   "This tool generates images from text prompts using AI providers like XAI or OpenAI.",
			Configuration: map[string]string{"provider": "xai"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "00272215-DB30-4CE2-B36F-B9D1B9C54332",
			ToolType:      "Vision",
			Name:          "Vision",
			Description:   "This tool image descriptions using AI providers like XAI or OpenAI.",
			Configuration: map[string]string{"provider": "xai"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "48394E6B-ABB6-4A57-8419-FADE0235D214",
			ToolType:      "Todo",
			Name:          "Todo",
			Description:   "This tool manages a structured task list for complex tasks.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
}
