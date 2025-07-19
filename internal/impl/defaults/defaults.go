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
				{Name: "mstral-small-latest", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 128000},
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
You are an AI assistant that helps software developers accomplish tasks on a Linux command line. You have access to a Bash shell and common command-line tools, including ls, rg (ripgrep), tree, sed, find, git, as well as compiler tools like gcc, go, rustc, javac, and dotnet. The user will provide instructions in natural language related to software development, and your goal is to translate those instructions into a sequence of Bash commands that achieve the desired outcome.

**Important Guidelines:**

- **Focus on the developer's intent:** Understand what the developer _wants_ to achieve in their development workflow, not just what they _say_ to do literally. This could involve code generation, compilation, testing, debugging, deployment, or file editing.
- **Use the available tools effectively:** Choose the right tools for the job and combine them creatively to solve complex development problems. Prioritize using standard Bash tools for file manipulation.
- **File Editing Strategy:** For file editing tasks, prioritize sed in conjunction with rg to locate the correct lines. Use rg to provide context and identify line numbers before using sed to modify the content. Consider using sed's -n (or --dry-run) option to preview changes before applying them.
- **Version Control:** You have access to git. Before making any significant changes to a file, use git to create a commit or a temporary branch. This allows you to easily rollback the changes if something goes wrong.
- **Break down complex tasks:** Divide the user's request into smaller, manageable steps. Consider common development workflows (e.g., edit -> compile -> test -> debug).
- **Prioritize correctness:** Ensure that the generated commands are syntactically correct and logically sound. Pay close attention to compiler flags and language-specific syntax.
- **Handle errors gracefully:** Anticipate potential errors during compilation, execution, or deployment and include error handling in your commands (e.g., using || and && to handle errors and check exit codes).
- **Be concise:** Use the most efficient commands possible to achieve the desired result. Avoid unnecessary steps.
- **Assume the current directory is the developer's workspace:** They can create, modify, and delete files and directories within this workspace.
- **Ask clarifying questions:** If the user's request is ambiguous or incomplete, ask questions to gather more information before generating commands. For example, ask about the programming language, target platform, or specific dependencies.
- **Explain your reasoning:** Before providing the commands, briefly explain your plan and the logic behind your approach. This helps the developer understand why you're suggesting those specific commands.
- **Output format:** Present your response in a clear and organized manner, including both the explanation and the generated commands. Use Markdown formatting for readability. Use code blocks for commands.

Example:

User: "Compile the main.c file using gcc and create an executable named my_program."

Assistant: "Okay, I will use gcc to compile main.c and create an executable named my_program. I will also include the -Wall flag to enable all warnings.

gcc -Wall main.c -o my_program

User: "Run all the go tests in the current directory"

Assistant: "Okay, I will use go test to run all tests in the current directory

go test ./...

User: "Change the variable name 'old_name' to 'new_name' in file 'my_file.txt'"

Assistant: "Okay, first I will create a git commit to allow for easy rollback. Then I will use sed with rg to replace all occurrences of 'old_name' with 'new_name' in 'my_file.txt'.

git commit -am "Backup before renaming variable"
rg 'old_name' my_file.txt # Find the line numbers (optional)
sed 's/old_name/new_name/g' my_file.txt

Now, respond to the user's requests by generating the appropriate Bash commands."
`
	maxTokens := 8192
	contextWindow := 131072
	localSystemPrompt := `
You are a helpful AI assistant for software developers using a Linux command line. Use Bash tools like ls, rg, sed, and git to accomplish tasks.

Instructions:

Understand the developer's goal.
Use Bash tools efficiently. Prefer sed with rg for file editing.
Use git to create commits before significant changes.
Provide concise, correct Bash commands.
Ask questions if unclear.
Example:

User: "List files." 
Assistant:

ls -l

Now, respond to the user's requests.

`
	localMaxTokens := 4096
	localContextWindow := 8192

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
			Tools:           []string{"WebSearch", "Project", "Bash", "Rg", "Ls", "Cat", "Sed", "Find", "Tree", "Git", "Go", "Python", "Node", "Dotnet"},
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
			Tools:           []string{"WebSearch", "Project", "Bash", "Rg", "Ls", "Cat", "Sed", "Find", "Tree", "Git", "Go", "Python", "Node", "Dotnet"},
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
			Tools:           []string{"WebSearch", "Project", "Bash", "Rg", "Ls", "Cat", "Sed", "Find", "Tree", "Git", "Go", "Python", "Node", "Dotnet"},
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
			Tools:           []string{"WebSearch", "Project", "Bash", "Rg", "Ls", "Cat", "Sed", "Find", "Tree", "Git", "Go", "Python", "Node", "Dotnet"},
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
			Tools:           []string{"WebSearch", "Project", "Bash", "Rg", "Ls", "Cat", "Sed", "Find", "Tree", "Git", "Go", "Python", "Node", "Dotnet"},
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
			ContextWindow:   &contextWindow,
			ReasoningEffort: "",
			Tools:           []string{"WebSearch", "Project", "Bash", "Rg", "Ls", "Cat", "Sed", "Find", "Tree", "Git", "Go", "Python", "Node", "Dotnet"},
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
			Tools:           []string{"WebSearch", "Project", "Bash", "Rg", "Ls", "Cat", "Sed", "Find", "Tree", "Git", "Go", "Python", "Node", "Dotnet"},
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
			Tools:           []string{"WebSearch", "Project", "Bash", "Rg", "Ls", "Cat", "Sed", "Find", "Tree", "Git", "Go", "Python", "Node", "Dotnet"},
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
			Tools:           []string{"WebSearch", "Project", "Bash", "Rg", "Ls", "Cat", "Sed", "Find", "Tree", "Git", "Go", "Python", "Node", "Dotnet"},
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
			Description:   "This tool reads project details from a configurable markdown file to provide context for AI agents. The AIAGENT.md file should contain a project description.",
			Configuration: map[string]string{"project_file": "AIAGENT.md"},
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
			ID:            "7A37A146-1B79-481F-B84A-154875A3407C",
			ToolType:      "Process",
			Name:          "Ls",
			Description:   "Lists directory contents (files and directories). The path to list is provided as the 'arguments' argument. If no path is provided, the current directory is listed. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "ls"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "8B998247-D72A-4D57-B1C3-4E39277F6719",
			ToolType:      "Process",
			Name:          "Cat",
			Description:   "Displays the contents of a file. Provide the full path to the file as the 'arguments' argument. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "cat"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "36589423-7144-4D9C-9A06-A4179DD94849",
			ToolType:      "Process",
			Name:          "Sed",
			Description:   "Stream editor for text transformations. Use to substitute text, delete lines, and perform other editing operations on files. Provide the full command, including the filename, as the 'arguments' argument. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "sed"},
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
			Description:   "Executes Python commands. Use this to run Python scripts or execute Python code. Provide the python command and arguments as a single string. The command is executed in the workspace directory.",
			Configuration: map[string]string{"command": "python"},
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
			Configuration: map[string]string{"command": "node"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "A8370999-C6F2-4D4E-9C57-CBD5056F85E3",
			ToolType:      "Image",
			Name:          "Image",
			Description:   "Generates images using AI providers like XAI or OpenAI. Provide a detailed text prompt describing the desired image.",
			Configuration: map[string]string{"provider": "xai", "api_key": "#{XAI_API_KEY}#"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "70523B6A-18CD-4EE2-8661-4782EBD34A0F",
			ToolType:      "Vision",
			Name:          "Vision",
			Description:   "Provides image understanding capabilities using providers like XAI or OpenAI, allowing processing of images via base64 or URLs combined with text prompts. Provide a detailed text prompt and either the image path or URL.",
			Configuration: map[string]string{"provider": "xai", "api_key": "#{XAI_API_KEY}#"},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
}
