package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/events"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GoogleIntegration implements the Google Gemini API
// For now, we'll use Base implementation as a temporary measure,
// but in a real implementation this would use the Gemini API
type GoogleIntegration struct {
	*AIModelIntegration
}

// NewGoogleIntegration creates a new Google integration
func NewGoogleIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*GoogleIntegration, error) {
	// Use native Gemini API endpoint
	nativeURL := baseURL + "/v1beta"
	if baseURL == "" {
		nativeURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	// Create a custom HTTP client for Gemini
	httpClient := &http.Client{Timeout: 30 * time.Minute}

	return &GoogleIntegration{
		AIModelIntegration: &AIModelIntegration{
			baseURL:    nativeURL,
			apiKey:     apiKey,
			httpClient: httpClient,
			model:      model,
			toolRepo:   toolRepo,
			logger:     logger,
			lastUsage:  &entities.Usage{},
		},
	}, nil
}

// convertMessagesToGeminiContents converts entities.Message to Gemini contents format
// Note: Tool messages are converted to functionResponse in user messages
func (g *GoogleIntegration) convertMessagesToGeminiContents(messages []*entities.Message) []map[string]any {
	contents := make([]map[string]any, 0)

	for _, msg := range messages {
		if msg.Role == "tool" {
			// Tool responses become functionResponse parts in user messages
			// Find the tool name from the preceding assistant message
			toolName := g.getToolNameFromID(msg.ToolCallID, messages)
			if toolName != "" {
				userContent := map[string]any{
					"role": "user",
					"parts": []map[string]any{
						{
							"functionResponse": map[string]any{
								"name": toolName,
								"response": map[string]any{
									"result": msg.Content,
								},
							},
						},
					},
				}
				contents = append(contents, userContent)
			}
		} else if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			// Assistant message with tool calls
			parts := []map[string]any{}
			if msg.Content != "" {
				parts = append(parts, map[string]any{"text": msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				part := map[string]any{
					"functionCall": map[string]any{
						"name": tc.Function.Name,
						"args": json.RawMessage(tc.Function.Arguments),
					},
				}
				// Include thoughtSignature as sibling field if present
				if tc.ThoughtSignature != "" {
					part["thoughtSignature"] = tc.ThoughtSignature
				}
				parts = append(parts, part)
			}
			contents = append(contents, map[string]any{
				"role":  msg.Role,
				"parts": parts,
			})
		} else {
			// Regular message
			contents = append(contents, map[string]any{
				"role":  msg.Role,
				"parts": []map[string]any{{"text": msg.Content}},
			})
		}
	}
	return contents
}

// getToolNameFromID finds the tool name from tool_call_id by looking back in messages
func (g *GoogleIntegration) getToolNameFromID(toolCallID string, messages []*entities.Message) string {
	for _, msg := range messages {
		if msg.Role == "assistant" {
			for _, tc := range msg.ToolCalls {
				if tc.ID == toolCallID {
					return tc.Function.Name
				}
			}
		}
	}
	return ""
}

// cleanSchemaForGemini removes fields that Gemini doesn't support
func (g *GoogleIntegration) cleanSchemaForGemini(schema map[string]any) map[string]any {
	cleaned := make(map[string]any)

	// Copy all fields except problematic ones
	for k, v := range schema {
		switch k {
		case "additionalProperties":
			// Skip additionalProperties as Gemini doesn't support it
			continue
		case "properties":
			// Recursively clean properties
			if props, ok := v.(map[string]any); ok {
				cleanedProps := make(map[string]any)
				for propName, propSchema := range props {
					if propSchemaMap, ok := propSchema.(map[string]any); ok {
						cleanedProps[propName] = g.cleanSchemaForGemini(propSchemaMap)
					} else {
						cleanedProps[propName] = propSchema
					}
				}
				cleaned[k] = cleanedProps
			} else {
				cleaned[k] = v
			}
		case "items":
			// Recursively clean items in arrays
			if itemsMap, ok := v.(map[string]any); ok {
				cleaned[k] = g.cleanSchemaForGemini(itemsMap)
			} else {
				cleaned[k] = v
			}
		default:
			cleaned[k] = v
		}
	}

	return cleaned
}

// convertToolsToGeminiFormat converts entities.Tool to Gemini tools format
func (g *GoogleIntegration) convertToolsToGeminiFormat(toolList []*entities.Tool) []map[string]any {
	if len(toolList) == 0 {
		return nil
	}

	functionDeclarations := make([]map[string]any, len(toolList))
	for i, tool := range toolList {
		cleanedSchema := g.cleanSchemaForGemini((*tool).Schema())
		schemaBytes, _ := json.Marshal(cleanedSchema)
		functionDeclarations[i] = map[string]any{
			"name":        (*tool).Name(),
			"description": (*tool).Description(),
			"parameters":  json.RawMessage(schemaBytes),
		}
	}

	return []map[string]any{
		{
			"functionDeclarations": functionDeclarations,
		},
	}
}

// GenerateResponse implements native Gemini API with tool call handling
func (g *GoogleIntegration) GenerateResponse(ctx context.Context, messages []*entities.Message, toolList []*entities.Tool, options map[string]any, callback interfaces.MessageCallback) ([]*entities.Message, error) {
	var newMessages []*entities.Message

	// Tool call handling loop (similar to OpenAI implementation)
	for {
		// Check for cancellation
		if ctx.Err() != nil {
			return nil, fmt.Errorf("operation canceled by user")
		}

		// Convert current messages to Gemini contents
		contents := g.convertMessagesToGeminiContents(messages)
		tools := g.convertToolsToGeminiFormat(toolList)

		// Build request body
		reqBody := map[string]any{
			"contents": contents,
		}
		if len(tools) > 0 {
			reqBody["tools"] = tools
		}
		if temp, ok := options["temperature"]; ok {
			reqBody["generationConfig"] = map[string]any{
				"temperature": temp,
			}
		}
		if maxTokens, ok := options["max_tokens"]; ok {
			if genConfig, ok := reqBody["generationConfig"].(map[string]any); ok {
				genConfig["maxOutputTokens"] = maxTokens
			} else {
				reqBody["generationConfig"] = map[string]any{
					"maxOutputTokens": maxTokens,
				}
			}
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request: %v", err)
		}
		g.logger.Info("Sending Gemini request", zap.String("body", string(jsonBody)))

		// Build URL
		url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", g.baseURL, g.model, g.apiKey)

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := g.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making request: %v", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return nil, fmt.Errorf("error reading response: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			g.logger.Error("Gemini API error",
				zap.Int("status_code", resp.StatusCode),
				zap.String("body", string(respBody)))
			return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
		}

		g.logger.Info("Gemini response", zap.String("body", string(respBody)))

		// Parse response
		var responseBody struct {
			Candidates []struct {
				Content struct {
					Role  string           `json:"role"`
					Parts []map[string]any `json:"parts"` // Use map to access thoughtSignature
				} `json:"content"`
				FinishReason string `json:"finishReason"`
			} `json:"candidates"`
			UsageMetadata struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				TotalTokenCount      int `json:"totalTokenCount"`
			} `json:"usageMetadata"`
		}

		if err := json.Unmarshal(respBody, &responseBody); err != nil {
			return nil, fmt.Errorf("error decoding response: %v", err)
		}

		if len(responseBody.Candidates) == 0 {
			return nil, fmt.Errorf("no candidates in response")
		}

		candidate := responseBody.Candidates[0]

		// Parse the response content
		content := ""
		var toolCalls []entities.ToolCall

		for _, part := range candidate.Content.Parts {
			if text, ok := part["text"].(string); ok && text != "" {
				content += text
			}
			if functionCall, ok := part["functionCall"].(map[string]any); ok {
				var tc entities.ToolCall
				tc.ID = uuid.New().String() // Gemini doesn't provide IDs, so we generate them
				tc.Type = "function"
				if name, ok := functionCall["name"].(string); ok {
					tc.Function.Name = name
				}
				if args, ok := functionCall["args"]; ok {
					if argsBytes, err := json.Marshal(args); err == nil {
						tc.Function.Arguments = string(argsBytes)
					}
				}
				// Extract thoughtSignature if present
				if sig, ok := part["thoughtSignature"].(string); ok {
					tc.ThoughtSignature = sig
				}
				toolCalls = append(toolCalls, tc)
			}
		}

		// Update usage
		g.lastUsage.PromptTokens = responseBody.UsageMetadata.PromptTokenCount
		g.lastUsage.CompletionTokens = responseBody.UsageMetadata.CandidatesTokenCount
		g.lastUsage.TotalTokens = responseBody.UsageMetadata.TotalTokenCount

		// Log tool calls
		if len(toolCalls) > 0 {
			g.logger.Info("Tool calls generated", zap.Any("toolCalls", toolCalls))
		} else {
			g.logger.Info("No tool calls generated")
		}

		// Create assistant message
		assistantMessage := &entities.Message{
			ID:        uuid.New().String(),
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
			Timestamp: time.Now(),
		}
		newMessages = append(newMessages, assistantMessage)

		if callback != nil {
			if err := callback([]*entities.Message{assistantMessage}); err != nil {
				g.logger.Error("Failed to save assistant message incrementally", zap.Error(err))
			}
		}

		// If no tool calls, we're done
		if len(toolCalls) == 0 {
			break
		}

		// Execute tools and create tool response messages
		for _, toolCall := range toolCalls {
			toolName := toolCall.Function.Name
			tool, err := g.toolRepo.GetToolByName(toolName)

			var toolResult string
			var toolError string
			if err != nil {
				toolResult = fmt.Sprintf("Tool %s could not be retrieved: %v", toolName, err)
				toolError = err.Error()
				g.logger.Warn("Failed to get tool", zap.String("toolName", toolName), zap.Error(err))
			} else if tool != nil {
				// Inject session_id into todowrite tool arguments
				args := toolCall.Function.Arguments
				if toolName == "todowrite" {
					if sessionID, ok := options["session_id"].(string); ok && sessionID != "" {
						var argsMap map[string]any
						if err := json.Unmarshal([]byte(args), &argsMap); err == nil {
							if _, exists := argsMap["session_id"]; !exists {
								argsMap["session_id"] = sessionID
								if newArgs, err := json.Marshal(argsMap); err == nil {
									args = string(newArgs)
								}
							}
						}
					}
				}
				result, err := (*tool).Execute(args)
				if err != nil {
					toolResult = fmt.Sprintf("Tool %s execution failed: %v", toolName, err)
					toolError = err.Error()
					g.logger.Warn("Tool execution failed", zap.String("toolName", toolName), zap.Error(err))
				} else {
					toolResult = result
				}
			} else {
				toolResult = fmt.Sprintf("Tool %s not found", toolName)
				toolError = "Tool not found"
				g.logger.Warn("Tool not found", zap.String("toolName", toolName))
			}

			// Create tool response message
			var displayContent string
			if toolError != "" {
				displayContent = fmt.Sprintf("Tool %s failed with error: %s", toolName, toolError)
			} else {
				displayContent = toolResult
			}

			toolEvent := entities.NewToolCallEvent(toolCall.ID, toolName, toolCall.Function.Arguments, displayContent, toolError, "", nil)

			// Publish event
			events.PublishToolCallEvent(toolEvent)

			toolResponseMessage := &entities.Message{
				ID:             uuid.New().String(),
				Role:           "tool",
				Content:        displayContent,
				ToolCallID:     toolCall.ID,
				ToolCallEvents: []entities.ToolCallEvent{*toolEvent},
				Timestamp:      time.Now(),
			}
			newMessages = append(newMessages, toolResponseMessage)

			if callback != nil {
				if err := callback([]*entities.Message{toolResponseMessage}); err != nil {
					g.logger.Error("Failed to save tool response message incrementally", zap.Error(err))
				}
			}

			// Append to messages for next iteration
			messages = append(messages, assistantMessage, toolResponseMessage)
		}
	}

	return newMessages, nil
}

// ProviderType returns the type of provider
func (m *GoogleIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderGoogle
}

// Ensure GoogleIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*GoogleIntegration)(nil)
