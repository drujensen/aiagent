package tools

import (
	"encoding/json"
	"fmt"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

type CompressionTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

func NewCompressionTool(name, description string, configuration map[string]string, logger *zap.Logger) *CompressionTool {
	return &CompressionTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *CompressionTool) Name() string {
	return t.name
}

func (t *CompressionTool) Description() string {
	return t.description
}

func (t *CompressionTool) Configuration() map[string]string {
	return t.configuration
}

func (t *CompressionTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *CompressionTool) FullDescription() string {
	return fmt.Sprintf("%s\n\nParameters:\n- action: compress_range\n- start_message_index: starting message index (0-based)\n- end_message_index: ending message index (0-based)\n- summary_type: task_cleanup, plan_update, context_preservation, full_reset\n- description: human-readable description of compression purpose", t.Description())
}

func (t *CompressionTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform",
				"enum":        []string{"compress_range"},
			},
			"start_message_index": map[string]any{
				"type":        "integer",
				"description": "Starting message index (0-based)",
				"minimum":     0,
			},
			"end_message_index": map[string]any{
				"type":        "integer",
				"description": "Ending message index (0-based)",
				"minimum":     0,
			},
			"summary_type": map[string]any{
				"type":        "string",
				"description": "Type of compression to apply",
				"enum":        []string{"task_cleanup", "plan_update", "context_preservation", "full_reset"},
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Human-readable description of what is being compressed",
			},
		},
		"required": []string{"action", "start_message_index", "end_message_index", "summary_type"},
	}
}

func (t *CompressionTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing compression command", zap.String("arguments", arguments))

	var args struct {
		Action            string `json:"action"`
		StartMessageIndex int    `json:"start_message_index"`
		EndMessageIndex   int    `json:"end_message_index"`
		SummaryType       string `json:"summary_type"`
		Description       string `json:"description,omitempty"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Action != "compress_range" {
		return "", fmt.Errorf("unsupported action: %s", args.Action)
	}

	if args.StartMessageIndex < 0 || args.EndMessageIndex < args.StartMessageIndex {
		return "", fmt.Errorf("invalid message range: start=%d, end=%d", args.StartMessageIndex, args.EndMessageIndex)
	}

	validTypes := map[string]bool{
		"task_cleanup":         true,
		"plan_update":          true,
		"context_preservation": true,
		"full_reset":           true,
	}

	if !validTypes[args.SummaryType] {
		return "", fmt.Errorf("invalid summary_type: %s", args.SummaryType)
	}

	// Return compression instruction for chat service to execute
	result := map[string]interface{}{
		"compression_instruction": map[string]interface{}{
			"action":              "compress_range",
			"start_message_index": args.StartMessageIndex,
			"end_message_index":   args.EndMessageIndex,
			"summary_type":        args.SummaryType,
			"description":         args.Description,
		},
		"message": fmt.Sprintf("Requested compression of messages %d-%d with type '%s'",
			args.StartMessageIndex, args.EndMessageIndex, args.SummaryType),
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

var _ entities.Tool = (*CompressionTool)(nil)
