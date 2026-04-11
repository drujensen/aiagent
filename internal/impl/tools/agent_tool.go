package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

// AgentTool launches a sub-agent to complete a specific task. Services are
// accessed lazily through the ToolFactory so that this tool can be
// instantiated before the services are wired up in main.
type AgentTool struct {
	name          string
	description   string
	configuration map[string]string
	factory       *ToolFactory
	logger        *zap.Logger
}

func NewAgentTool(name, description string, configuration map[string]string, factory *ToolFactory, logger *zap.Logger) *AgentTool {
	return &AgentTool{
		name:          name,
		description:   description,
		configuration: configuration,
		factory:       factory,
		logger:        logger,
	}
}

func (t *AgentTool) Name() string        { return t.name }
func (t *AgentTool) Description() string { return t.description }

func (t *AgentTool) FullDescription() string {
	var b strings.Builder
	b.WriteString(t.description)
	b.WriteString("\n\nConfiguration for this tool:\n")
	b.WriteString("| Key | Value |\n")
	b.WriteString("|-----|-------|\n")
	for k, v := range t.configuration {
		b.WriteString(fmt.Sprintf("| %-20s | %-20s |\n", k, v))
	}
	return b.String()
}

func (t *AgentTool) Configuration() map[string]string { return t.configuration }

func (t *AgentTool) UpdateConfiguration(config map[string]string) { t.configuration = config }

func (t *AgentTool) Schema() map[string]any {
	agentNameDesc := "Name of the agent to launch."

	// Dynamically enumerate available agents so the LLM always sees the
	// current list. Schema() is called at request time, after services are
	// wired, so it is safe to query the agent service here.
	if svc := t.factory.GetAgentService(); svc != nil {
		if agents, err := svc.ListAgents(context.Background()); err == nil && len(agents) > 0 {
			names := make([]string, len(agents))
			for i, a := range agents {
				names[i] = a.Name
			}
			agentNameDesc = "Name of the agent to launch. Available agents: " + strings.Join(names, ", ") + "."
		}
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"agent_name": map[string]any{
				"type":        "string",
				"description": agentNameDesc,
			},
			"task": map[string]any{
				"type":        "string",
				"description": "The complete task description or prompt to send to the sub-agent. Be specific and include all relevant context.",
			},
			"model_name": map[string]any{
				"type":        "string",
				"description": "Optional: name of a specific model to use for the sub-agent. If omitted the first available model is used.",
			},
		},
		"required": []string{"agent_name", "task"},
	}
}

func (t *AgentTool) Execute(arguments string) (string, error) {
	chatService := t.factory.GetChatService()
	agentService := t.factory.GetAgentService()
	modelService := t.factory.GetModelService()

	if chatService == nil || agentService == nil || modelService == nil {
		return "", fmt.Errorf("agent tool not ready: services not yet initialized")
	}

	var args struct {
		AgentName string `json:"agent_name"`
		Task      string `json:"task"`
		ModelName string `json:"model_name"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}
	if args.AgentName == "" {
		return "", fmt.Errorf("agent_name is required")
	}
	if args.Task == "" {
		return "", fmt.Errorf("task is required")
	}

	ctx := context.Background()

	// Find agent by name (case-insensitive)
	agents, err := agentService.ListAgents(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list agents: %w", err)
	}
	var targetAgent *entities.Agent
	for _, a := range agents {
		if strings.EqualFold(a.Name, args.AgentName) {
			targetAgent = a
			break
		}
	}
	if targetAgent == nil {
		names := make([]string, len(agents))
		for i, a := range agents {
			names[i] = a.Name
		}
		return "", fmt.Errorf("agent %q not found; available agents: %s", args.AgentName, strings.Join(names, ", "))
	}

	// Resolve the model to use for the sub-agent.
	// Priority: explicit model_name arg > parent chat's model > first available.
	// We read the active (parent) chat's model BEFORE calling CreateChat, because
	// CreateChat will make the new sub-agent chat active.
	var targetModel *entities.Model
	if args.ModelName != "" {
		models, err := modelService.ListModels(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to list models: %w", err)
		}
		for _, m := range models {
			if strings.EqualFold(m.Name, args.ModelName) || strings.EqualFold(m.ModelName, args.ModelName) {
				targetModel = m
				break
			}
		}
		if targetModel == nil {
			return "", fmt.Errorf("model %q not found", args.ModelName)
		}
	} else {
		// Inherit the parent chat's model so sub-agents use the same model
		// the user already configured for the Orchestrator.
		if parentChat, err := chatService.GetActiveChat(ctx); err == nil && parentChat.ModelID != "" {
			targetModel, _ = modelService.GetModel(ctx, parentChat.ModelID)
		}
		if targetModel == nil {
			// Fallback: use first available model
			models, err := modelService.ListModels(ctx)
			if err != nil {
				return "", fmt.Errorf("failed to list models: %w", err)
			}
			if len(models) == 0 {
				return "", fmt.Errorf("no models available")
			}
			targetModel = models[0]
		}
	}

	// Create a new chat for the sub-agent
	chatTitle := fmt.Sprintf("%s: %s", targetAgent.Name, truncateStr(args.Task, 50))
	chat, err := chatService.CreateChat(ctx, targetAgent.ID, targetModel.ID, chatTitle)
	if err != nil {
		return "", fmt.Errorf("failed to create sub-agent chat: %w", err)
	}

	t.logger.Info("Launching sub-agent",
		zap.String("agent", targetAgent.Name),
		zap.String("model", targetModel.Name),
		zap.String("chat_id", chat.ID),
	)

	// Send the task and wait for the response
	msg := entities.NewMessage("user", args.Task)
	response, err := chatService.SendMessage(ctx, chat.ID, msg)
	if err != nil {
		return "", fmt.Errorf("sub-agent %q failed: %w", args.AgentName, err)
	}

	return response.Content, nil
}

func (t *AgentTool) DisplayName(ui string, arguments string) (string, string) {
	var args struct {
		AgentName string `json:"agent_name"`
		Task      string `json:"task"`
	}
	json.Unmarshal([]byte(arguments), &args) //nolint:errcheck
	label := fmt.Sprintf("Launch %s agent", args.AgentName)
	detail := truncateStr(args.Task, 80)
	return label, detail
}

func (t *AgentTool) FormatResult(ui string, result, diff, arguments string) string {
	var args struct {
		AgentName string `json:"agent_name"`
	}
	json.Unmarshal([]byte(arguments), &args) //nolint:errcheck

	// The result IS the sub-agent's full response — show a meaningful preview.
	const previewLines = 10
	lines := strings.SplitN(result, "\n", previewLines+1)
	var preview string
	if len(lines) > previewLines {
		preview = strings.Join(lines[:previewLines], "\n") + fmt.Sprintf("\n... (%d+ lines)", previewLines)
	} else {
		preview = result
	}

	switch ui {
	case "webui":
		// webui rendering is handled by formatters.go; this path is unused
		// but keep it consistent in case it is ever called directly.
		return fmt.Sprintf(
			`<details class="agent-result"><summary>Sub-agent <strong>%s</strong> response</summary><pre>%s</pre></details>`,
			html.EscapeString(args.AgentName),
			html.EscapeString(preview),
		)
	default:
		return fmt.Sprintf("[%s response]\n%s", args.AgentName, preview)
	}
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
