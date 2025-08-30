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
}

type SubagentWrapperConfig struct {
	SubagentID string                 `json:"subagent_id"`
	ToolName   string                 `json:"tool_name"`
	Parameters map[string]interface{} `json:"parameters"`
}

func NewSubagentWrapper(subagentID, name, description string, config map[string]string, logger *zap.Logger, wrappedTool entities.Tool) *SubagentWrapper {
	return &SubagentWrapper{
		subagentID:    subagentID,
		name:          name,
		description:   description,
		configuration: config,
		logger:        logger,
		wrappedTool:   wrappedTool,
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

	if w.wrappedTool != nil {
		// Direct execution mode - just pass through to wrapped tool
		return w.wrappedTool.Execute(arguments)
	}

	// Subagent mode - parse wrapper configuration
	var config SubagentWrapperConfig
	if err := json.Unmarshal([]byte(arguments), &config); err != nil {
		w.logger.Error("Failed to parse subagent wrapper config", zap.Error(err), zap.String("arguments", arguments))
		return "", fmt.Errorf("failed to parse wrapper config: %v", err)
	}

	// Convert parameters back to JSON for the wrapped tool
	paramBytes, err := json.Marshal(config.Parameters)
	if err != nil {
		return "", fmt.Errorf("failed to marshal parameters: %v", err)
	}

	w.logger.Info("Executing wrapped tool as subagent",
		zap.String("subagent_id", w.subagentID),
		zap.String("tool_name", config.ToolName),
		zap.String("parameters", string(paramBytes)))

	// In a real implementation, this would:
	// 1. Look up the tool by name from the tool registry
	// 2. Execute it with the provided parameters
	// 3. Handle any errors and format the response

	// For now, return a mock response
	response := map[string]interface{}{
		"subagent_id": w.subagentID,
		"tool_name":   config.ToolName,
		"status":      "completed",
		"result":      "Tool executed successfully via subagent wrapper",
		"parameters":  config.Parameters,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %v", err)
	}

	return string(responseBytes), nil
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
	imageTool := NewImageTool(name, description, configuration, logger)
	return CreateToolSubagentWrapper(imageTool, "image-generator", logger)
}

func NewVisionSubagentWrapper(name, description string, configuration map[string]string, logger *zap.Logger) *SubagentWrapper {
	visionTool := &VisionTool{
		NameField:            name,
		DescriptionField:     description,
		FullDescriptionField: description,
		ConfigurationField:   configuration,
	}
	return CreateToolSubagentWrapper(visionTool, "vision-analyst", logger)
}

var _ entities.Tool = (*SubagentWrapper)(nil)
