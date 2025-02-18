package models

// AIAgent represents an AI agent that can be configured
// with a prompt and a list of available tools

type AIAgent struct {
    Prompt string
    Tools  []Tool
}

// NewAIAgent creates a new AI agent with the given prompt and tools
func NewAIAgent(prompt string, tools []Tool) *AIAgent {
    return &AIAgent{
        Prompt: prompt,
        Tools:  tools,
    }
}

// AddTool adds a new tool to the AI agent's list of tools
func (agent *AIAgent) AddTool(tool Tool) {
    agent.Tools = append(agent.Tools, tool)
}
