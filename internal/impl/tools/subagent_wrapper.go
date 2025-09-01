package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"go.uber.org/zap"
)

type SubagentWrapper struct {
	subagentID    string
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
	wrappedTool   entities.Tool
	subagentTool  *AgentCallTool // Reference to the main agent call tool
}

type SubagentWrapperConfig struct {
	SubagentID string         `json:"subagent_id"`
	ToolName   string         `json:"tool_name"`
	Parameters map[string]any `json:"parameters"`
}

func NewSubagentWrapper(subagentID, name, description string, config map[string]string, logger *zap.Logger, wrappedTool entities.Tool) *SubagentWrapper {
	return &SubagentWrapper{
		subagentID:    subagentID,
		name:          name,
		description:   description,
		configuration: config,
		logger:        logger,
		wrappedTool:   wrappedTool,
		subagentTool:  nil, // Will be injected later
	}
}

// InjectAgentCallTool allows injecting the AgentCallTool dependency after creation
func (w *SubagentWrapper) InjectAgentCallTool(agentCallTool any) {
	if act, ok := agentCallTool.(*AgentCallTool); ok {
		w.subagentTool = act
	}
}

func (w *SubagentWrapper) Name() string {
	return w.name
}

func (w *SubagentWrapper) Description() string {
	return w.description
}

func (w *SubagentWrapper) Configuration() map[string]string {
	return w.configuration
}

func (w *SubagentWrapper) UpdateConfiguration(config map[string]string) {
	w.configuration = config
}

func (w *SubagentWrapper) FullDescription() string {
	var b strings.Builder
	b.WriteString(w.Description() + "\n\n")
	b.WriteString("Configuration for this tool:\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range w.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (w *SubagentWrapper) Parameters() []entities.Parameter {
	if w.wrappedTool != nil {
		return w.wrappedTool.Parameters()
	}

	return []entities.Parameter{
		{
			Name:        "tool_name",
			Type:        "string",
			Description: "Name of the tool to execute as subagent",
			Required:    true,
		},
		{
			Name:        "parameters",
			Type:        "object",
			Description: "Parameters to pass to the wrapped tool",
			Required:    true,
		},
	}
}

func (w *SubagentWrapper) Execute(arguments string) (string, error) {
	w.logger.Debug("Executing subagent wrapper", zap.String("arguments", arguments))

	// Check if we have a SubagentTool to delegate to
	if w.subagentTool == nil {
		return "", fmt.Errorf("subagent tool not injected - cannot execute subagent wrapper")
	}

	// Create subagent task request
	taskRequest := map[string]any{
		"agent_id": w.subagentID,
		"task":     arguments,
		"context": map[string]any{
			"tool_name":  w.name,
			"parameters": arguments,
		},
	}

	// Convert to JSON for SubagentTool
	taskBytes, err := json.Marshal(taskRequest)
	if err != nil {
		w.logger.Error("Failed to marshal subagent task", zap.Error(err))
		return "", fmt.Errorf("failed to marshal subagent task: %v", err)
	}

	w.logger.Info("Delegating to subagent",
		zap.String("subagent_id", w.subagentID),
		zap.String("task", arguments))

	// Execute via SubagentTool
	return w.subagentTool.Execute(string(taskBytes))
}

// Helper function to create a subagent wrapper for existing tools
func CreateToolSubagentWrapper(tool entities.Tool, subagentID string, logger *zap.Logger) *SubagentWrapper {
	wrapperName := fmt.Sprintf("%s Subagent", tool.Name())
	wrapperDescription := fmt.Sprintf("Subagent wrapper for %s tool providing isolation and resource management", tool.Name())

	return NewSubagentWrapper(
		subagentID,
		wrapperName,
		wrapperDescription,
		tool.Configuration(),
		logger,
		tool,
	)
}

// Factory function to create subagent wrappers for image and vision tools
func NewImageSubagentWrapper(name, description string, configuration map[string]string, logger *zap.Logger) *SubagentWrapper {
	return NewSubagentWrapper("image-generator", name, description, configuration, logger, nil)
}

func NewVisionSubagentWrapper(name, description string, configuration map[string]string, logger *zap.Logger) *SubagentWrapper {
	return NewSubagentWrapper("vision-analyst", name, description, configuration, logger, nil)
}

var _ entities.Tool = (*SubagentWrapper)(nil)
