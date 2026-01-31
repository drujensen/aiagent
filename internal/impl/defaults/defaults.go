package defaults

import (
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

// DefaultProviders returns the default list of providers.
func DefaultProviders() []*entities.Provider {
	return []*entities.Provider{
		{
			ID:         "FD3C37A7-C9C0-4AA9-A4B7-C43D52806A98",
			Name:       "X.AI",
			Type:       "xai",
			BaseURL:    "https://api.x.ai",
			APIKeyName: "XAI_API_KEY",
			Models:     []entities.ModelPricing{},
		},
		{
			ID:         "1E4697B3-233F-4004-B513-692E5F6EABE6",
			Name:       "OpenAI",
			Type:       "openai",
			BaseURL:    "https://api.openai.com",
			APIKeyName: "OPENAI_API_KEY",
			Models:     []entities.ModelPricing{},
		},
		{
			ID:         "B978105A-802B-480B-BF79-D50EB8FB21B0",
			Name:       "Anthropic",
			Type:       "anthropic",
			BaseURL:    "https://api.anthropic.com",
			APIKeyName: "ANTHROPIC_API_KEY",
			Models:     []entities.ModelPricing{},
		},
		{
			ID:         "E384327C-337D-4EA5-88D5-B1FC4147CD6D",
			Name:       "Google",
			Type:       "google",
			BaseURL:    "https://generativelanguage.googleapis.com",
			APIKeyName: "GEMINI_API_KEY",
			Models:     []entities.ModelPricing{},
		},
		{
			ID:         "ADEAC984-EBB4-491F-B041-38966A15DE83",
			Name:       "DeepSeek",
			Type:       "deepseek",
			BaseURL:    "https://api.deepseek.com",
			APIKeyName: "DEEPSEEK_API_KEY",
			Models:     []entities.ModelPricing{},
		},
		{
			ID:         "76EFB2E1-AAD9-43CC-8719-1B166F1404F1",
			Name:       "Groq",
			Type:       "groq",
			BaseURL:    "https://api.groq.com",
			APIKeyName: "GROQ_API_KEY",
			Models:     []entities.ModelPricing{},
		},
		{
			ID:         "875102A8-F3B3-40EE-BDA4-19201C5CFEF8",
			Name:       "Mistral",
			Type:       "mistral",
			BaseURL:    "https://api.mistral.ai",
			APIKeyName: "MISTRAL_API_KEY",
			Models:     []entities.ModelPricing{},
		},
		{
			ID:         "659D1EAD-AA0A-45CD-BE28-5472F419B0DB",
			Name:       "Drujensen",
			Type:       "drujensen",
			BaseURL:    "https://ai.drujensen.com",
			APIKeyName: "DRUJENSEN_API_KEY",
			Models:     []entities.ModelPricing{},
		},
		{
			ID:         "41A83584-ABEB-4490-921A-D778A296862D",
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
	systemPrompt := `
You are an AI assistant for software engineering tasks. Use available tools to help with coding, planning, testing, and related activities.

Key principles:
- Use tools proactively and efficiently
- Plan complex tasks systematically
- Be concise but thorough in responses
- Follow coding best practices and project conventions
- Leverage AGENTS.md for project-specific guidance
	`

	return []entities.Agent{
		{
			ID:   "1B2F3DCE-03C5-4376-964F-73649450AC30",
			Name: "Research",
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
			Tools:     []string{"WebSearch", "Project", "FileRead", "FileWrite", "FileSearch", "Directory", "Process", "Todo"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:   "39FDB435-37F4-4A4D-9DE6-C36243ECEE8B",
			Name: "Plan",
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
			Tools:     []string{"WebSearch", "Project", "FileRead", "FileSearch", "Directory", "Todo"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:   "830EF402-4F03-40BA-B403-25A9D732D82F",
			Name: "Build",
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
			Tools:     []string{"WebSearch", "Project", "FileRead", "FileWrite", "FileSearch", "Directory", "Process"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

// DefaultTools returns the default list of tools.
func DefaultTools() []*entities.ToolData {
	now := time.Now()

	return []*entities.ToolData{
		{
			ID:            "A121CC4A-A5CE-4054-AB8D-8486863DC7EA",
			ToolType:      "WebSearch",
			Name:          "WebSearch",
			Description:   "This tool searches the web using the Tavily API.",
			Configuration: map[string]string{"tavily_api_key": "#{TAVILY_API_KEY}#"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "575B620F-8E41-4294-ADF7-B04B8ACB8F0D",
			ToolType:      "Project",
			Name:          "Project",
			Description:   "This tool reads project details from a project file to provide context for the agent.",
			Configuration: map[string]string{"project_file": "AGENTS.md"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "7A6E93D7-7A8A-4AAE-8EFF-E87976B52C27",
			ToolType:      "FileRead",
			Name:          "FileRead",
			Description:   "This tool reads lines from a file.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "1A0CC8D3-69C0-4F2D-9BCD-B678BC412DD5",
			ToolType:      "FileWrite",
			Name:          "FileWrite",
			Description:   "This tool writes lines to a file.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "2FA32039-1596-4FD1-AAFF-46F2F17FBD61",
			ToolType:      "FileSearch",
			Name:          "FileSearch",
			Description:   "This tool searches for content in a file.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "996F432D-7505-4519-A18D-02BD4E7DCC7F",
			ToolType:      "Directory",
			Name:          "Directory",
			Description:   "This tool supports directory management",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},

		{
			ID:            "ED25354E-F10A-4D6F-979F-339E1CC74B55",
			ToolType:      "Process",
			Name:          "Process",
			Description:   "Executes any command (e.g., python, ruby, node, git) with support for background processes, stdin/stdout/stderr interaction, timeouts, and full output. Can launch interactive environments like Python REPL or Ruby IRB by running in background and using write/read actions. The command is executed in the workspace directory.",
			Configuration: map[string]string{"workspace": ""},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "7E29B4E6-3147-4826-939A-ABA82562A27B",
			ToolType:      "Fetch",
			Name:          "Fetch",
			Description:   "This tool performs HTTP requests to fetch data from web APIs and endpoints.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "44BF67C9-45DC-4A0C-947E-58604D1F37B9",
			ToolType:      "Swagger",
			Name:          "Swagger",
			Description:   "This tool parses and analyzes Swagger/OpenAPI specifications for API documentation and testing.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "C30A3419-5F10-4169-AAEB-6D606FE492C8",
			ToolType:      "Image",
			Name:          "Image",
			Description:   "This tool generates images from text prompts using AI providers like XAI or OpenAI.",
			Configuration: map[string]string{"provider": "xai"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "4DD0A108-710E-4878-8F1F-389DBDEA978F",
			ToolType:      "Vision",
			Name:          "Vision",
			Description:   "This tool image descriptions using AI providers like XAI or OpenAI.",
			Configuration: map[string]string{"provider": "xai"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "2B6CE553-B7A9-4A05-A7AF-A2EC34AA9490",
			ToolType:      "Todo",
			Name:          "Todo",
			Description:   "This tool manages a structured task list for complex tasks.",
			Configuration: map[string]string{},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
}

// DefaultModels returns the default list of models.
func DefaultModels() []*entities.Model {
	return []*entities.Model{
		// OpenAI Models
		{
			ID:              "A69AFBC4-1BB0-4D16-8FC6-3CB8E6603A68",
			Name:            "GPT-5.2 Best Performance",
			ProviderID:      "1E4697B3-233F-4004-B513-692E5F6EABE6",
			ProviderType:    entities.ProviderOpenAI,
			ModelName:       "gpt-5.2",
			APIKey:          "#{OPENAI_API_KEY}#",
			Temperature:     &[]float64{0.7}[0],
			MaxTokens:       &[]int{200000}[0],
			ContextWindow:   &[]int{400000}[0],
			ReasoningEffort: "",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "3CA94BFA-5CD5-44FD-9EBC-3F780105B821",
			Name:            "GPT-5.1 Codex Mini",
			ProviderID:      "1E4697B3-233F-4004-B513-692E5F6EABE6",
			ProviderType:    entities.ProviderOpenAI,
			ModelName:       "gpt-5.1-codex-mini",
			APIKey:          "#{OPENAI_API_KEY}#",
			Temperature:     &[]float64{0.3}[0],
			MaxTokens:       &[]int{200000}[0],
			ContextWindow:   &[]int{400000}[0],
			ReasoningEffort: "",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},

		// Anthropic Models
		{
			ID:              "5632519A-1495-46CA-BCF9-274307477894",
			Name:            "Claude Opus 4.5 Best Performance",
			ProviderID:      "B978105A-802B-480B-BF79-D50EB8FB21B0",
			ProviderType:    entities.ProviderAnthropic,
			ModelName:       "claude-opus-4.5",
			APIKey:          "#{ANTHROPIC_API_KEY}#",
			Temperature:     &[]float64{0.7}[0],
			MaxTokens:       &[]int{100000}[0],
			ContextWindow:   &[]int{200000}[0],
			ReasoningEffort: "",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "17403C9E-A943-4CF1-AC39-7DCF5B513135",
			Name:            "Claude Haiku 4.5",
			ProviderID:      "B978105A-802B-480B-BF79-D50EB8FB21B0",
			ProviderType:    entities.ProviderAnthropic,
			ModelName:       "claude-haiku-4.5",
			APIKey:          "#{ANTHROPIC_API_KEY}#",
			Temperature:     &[]float64{0.3}[0],
			MaxTokens:       &[]int{100000}[0],
			ContextWindow:   &[]int{200000}[0],
			ReasoningEffort: "",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},

		// Google Models
		{
			ID:              "FA6C6ED3-D14D-450D-9487-D10632215D1E",
			Name:            "Gemini 3 Pro Best Performance",
			ProviderID:      "E384327C-337D-4EA5-88D5-B1FC4147CD6D",
			ProviderType:    entities.ProviderGoogle,
			ModelName:       "gemini-3-pro-preview",
			APIKey:          "#{GEMINI_API_KEY}#",
			Temperature:     &[]float64{0.7}[0],
			MaxTokens:       &[]int{500000}[0],
			ContextWindow:   &[]int{1000000}[0],
			ReasoningEffort: "",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "2EEEFC70-E455-4B64-8951-038564AB9B46",
			Name:            "Gemini 2.5 Flash Lite",
			ProviderID:      "E384327C-337D-4EA5-88D5-B1FC4147CD6D",
			ProviderType:    entities.ProviderGoogle,
			ModelName:       "gemini-2.5-flash-lite",
			APIKey:          "#{GEMINI_API_KEY}#",
			Temperature:     &[]float64{0.3}[0],
			MaxTokens:       &[]int{500000}[0],
			ContextWindow:   &[]int{1000000}[0],
			ReasoningEffort: "",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},

		// X.AI Models
		{
			ID:              "4C1054B1-A3EA-45DC-9BE1-09073D74CC09",
			Name:            "Grok 4 Best Performance",
			ProviderID:      "FD3C37A7-C9C0-4AA9-A4B7-C43D52806A98",
			ProviderType:    entities.ProviderXAI,
			ModelName:       "grok-4",
			APIKey:          "#{XAI_API_KEY}#",
			Temperature:     &[]float64{0.7}[0],
			MaxTokens:       &[]int{1000000}[0],
			ContextWindow:   &[]int{2000000}[0],
			ReasoningEffort: "",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "659D1EAD-AA0A-45CD-BE28-5472F419B0DB",
			Name:            "Grok Code Fast",
			ProviderID:      "FD3C37A7-C9C0-4AA9-A4B7-C43D52806A98",
			ProviderType:    entities.ProviderXAI,
			ModelName:       "grok-code-fast-1",
			APIKey:          "#{XAI_API_KEY}#",
			Temperature:     &[]float64{0.3}[0],
			MaxTokens:       &[]int{1000000}[0],
			ContextWindow:   &[]int{2000000}[0],
			ReasoningEffort: "",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},

		// Drujensen Models
		{
			ID:              "NEW-DJ-MODEL-ID-12345",
			Name:            "Qwen3 Coder Latest",
			ProviderID:      "659D1EAD-AA0A-45CD-BE28-5472F419B0DB",
			ProviderType:    entities.ProviderDrujensen,
			ModelName:       "qwen3-coder:latest",
			APIKey:          "#{DRUJENSEN_API_KEY}#",
			Temperature:     &[]float64{0.7}[0],
			MaxTokens:       &[]int{32000}[0],
			ContextWindow:   &[]int{64000}[0],
			ReasoningEffort: "",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}
}
