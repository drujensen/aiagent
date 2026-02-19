package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/events"
	"github.com/drujensen/aiagent/internal/domain/services"
	"github.com/kujtimiihoxha/vimtea"
	"go.uber.org/zap"
)

// formatToolName formats tool names with relevant arguments for display
func formatToolName(toolName, arguments string) (string, string) {
	switch toolName {
	case "FileRead":
		// Filename will be shown in the result, so return empty suffix
		return toolName, ""
	case "FileWrite", "Directory":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err == nil && args.Path != "" {
			return toolName, args.Path
		}
	case "FileSearch":
		var args struct {
			Pattern   string `json:"pattern"`
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err == nil {
			detail := args.Pattern
			if args.Directory != "" && args.Directory != "." {
				detail += " in " + args.Directory
			}
			return toolName, detail
		}
	case "Process":
		var args struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err == nil && args.Command != "" {
			// Truncate long commands
			if len(args.Command) > 50 {
				args.Command = args.Command[:47] + "..."
			}
			return toolName, args.Command
		}
	}
	return toolName, ""
}

// formatToolResult formats tool execution results for display
func formatToolResult(toolName, result string, diff string, arguments string) string {
	switch toolName {
	case "FileRead":
		return formatFileReadResult(result, arguments)
	case "FileWrite":
		return formatFileWriteResult(result, diff)
	case "FileSearch":
		return formatFileSearchResult(result)
	case "Memory":
		return formatMemoryResult(result)
	case "Process":
		return formatProcessResult(result)
	case "WebSearch":
		return formatWebSearchResult(result, arguments)
	default:
		// Try to extract summary from JSON responses
		var jsonResponse struct {
			Summary string `json:"summary"`
		}
		if err := json.Unmarshal([]byte(result), &jsonResponse); err == nil && jsonResponse.Summary != "" {
			// Return only the summary for TUI display
			return jsonResponse.Summary
		}
		// For non-JSON results or JSON without summary, return as-is
		return result
	}
}

// formatProcessResult formats Process tool results
func formatProcessResult(result string) string {
	// First, try to parse as summary format (if tool returns {"summary": "..."})
	var summaryResponse struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(result), &summaryResponse); err == nil && summaryResponse.Summary != "" {
		// Fix incomplete summaries
		if summaryResponse.Summary == "Executed: " {
			return "Executed successfully"
		}
		return summaryResponse.Summary
	}

	// Otherwise, parse as output format
	var response struct {
		Output   string `json:"output"`
		ExitCode int    `json:"exit_code"`
		Error    string `json:"error"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		// If parsing fails, return the original result
		return result
	}

	if response.Error != "" {
		return fmt.Sprintf("Failed: %s", response.Error)
	}

	if response.ExitCode != 0 {
		return fmt.Sprintf("Failed with exit code %d", response.ExitCode)
	}

	return "Executed successfully"
}

// formatWebSearchResult formats WebSearch tool results
func formatWebSearchResult(result string, arguments string) string {
	var args struct {
		Query string `json:"query"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err == nil && args.Query != "" {
		return fmt.Sprintf("Searched for: %s", args.Query)
	}

	// Fallback
	var jsonResponse struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(result), &jsonResponse); err == nil && jsonResponse.Summary != "" {
		return jsonResponse.Summary
	}
	return "Web search completed"
}

// getToolStatusIcon returns an appropriate icon based on tool execution status
func getToolStatusIcon(hasError bool) string {
	if hasError {
		return "❌"
	}
	return "✅"
}

// formatFileWriteResult formats FileWrite tool results
func formatFileWriteResult(result string, diff string) string {
	var resultData struct {
		Summary     string `json:"summary"`
		Success     bool   `json:"success"`
		Path        string `json:"path"`
		Occurrences int    `json:"occurrences"`
		ReplacedAll bool   `json:"replaced_all"`
		Diff        string `json:"diff"`
	}

	// First, try to use the diff parameter if available
	if diff != "" {
		// Try to parse JSON to get summary
		if err := json.Unmarshal([]byte(result), &resultData); err == nil && resultData.Summary != "" {
			var output strings.Builder
			output.WriteString(resultData.Summary)
			output.WriteString("\n\n" + formatDiff(diff))
			return output.String()
		} else {
			// If JSON parsing fails, create a simple summary
			var output strings.Builder
			output.WriteString("File modified successfully\n\n")
			output.WriteString(formatDiff(diff))
			return output.String()
		}
	}

	// If no diff parameter, try to parse the full JSON result
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		// If parsing fails, try to extract summary from JSON
		var jsonResponse struct {
			Summary string `json:"summary"`
		}
		if err2 := json.Unmarshal([]byte(result), &jsonResponse); err2 == nil && jsonResponse.Summary != "" {
			return jsonResponse.Summary
		}
		return result // Return raw if parsing fails
	}

	var output strings.Builder

	// Use the summary from the JSON response
	output.WriteString(resultData.Summary)

	// Add the diff from JSON if available
	if resultData.Diff != "" {
		output.WriteString("\n\n" + formatDiff(resultData.Diff))
	}

	return output.String()
}

// formatFileSearchResult formats FileSearch tool results
func formatFileSearchResult(result string) string {
	var response struct {
		Summary string `json:"summary"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		// If parsing fails, return the original result
		return result
	}

	// Return only the summary for TUI display
	return response.Summary
}

// formatMemoryResult formats Memory tool results
func formatMemoryResult(result string) string {
	// Try to parse as different memory result types
	var output strings.Builder

	// Try parsing as entities array
	var entities []interface{}
	if err := json.Unmarshal([]byte(result), &entities); err == nil && len(entities) > 0 {
		output.WriteString(fmt.Sprintf("Memory Entities (%d created):\n", len(entities)))

		// Show first 5 entities
		maxEntities := 5
		for i, entity := range entities {
			if i >= maxEntities {
				break
			}

			if entityMap, ok := entity.(map[string]interface{}); ok {
				name := entityMap["name"]
				entityType := entityMap["type"]
				output.WriteString(fmt.Sprintf("  • %s (%s)\n", name, entityType))
			}
		}

		if len(entities) > maxEntities {
			output.WriteString(fmt.Sprintf("  ... and %d more entities\n", len(entities)-maxEntities))
		}

		return output.String()
	}

	// Try parsing as relations array
	var relations []interface{}
	if err := json.Unmarshal([]byte(result), &relations); err == nil && len(relations) > 0 {
		output.WriteString(fmt.Sprintf("Memory Relations (%d created):\n", len(relations)))

		// Show first 5 relations
		maxRelations := 5
		for i, relation := range relations {
			if i >= maxRelations {
				break
			}

			if relationMap, ok := relation.(map[string]interface{}); ok {
				source := relationMap["source"]
				relationType := relationMap["type"]
				target := relationMap["target"]
				output.WriteString(fmt.Sprintf("  • %s --%s--> %s\n", source, relationType, target))
			}
		}

		if len(relations) > maxRelations {
			output.WriteString(fmt.Sprintf("  ... and %d more relations\n", len(relations)-maxRelations))
		}

		return output.String()
	}

	// Try parsing as graph structure
	var graph map[string]interface{}
	if err := json.Unmarshal([]byte(result), &graph); err == nil {
		if entities, ok := graph["entities"].([]interface{}); ok {
			output.WriteString(fmt.Sprintf("Knowledge Graph - Entities (%d):\n", len(entities)))

			// Show first 5 entities
			maxEntities := 5
			for i, entity := range entities {
				if i >= maxEntities {
					break
				}

				if entityMap, ok := entity.(map[string]interface{}); ok {
					name := entityMap["name"]
					entityType := entityMap["type"]
					output.WriteString(fmt.Sprintf("  • %s (%s)\n", name, entityType))
				}
			}

			if len(entities) > maxEntities {
				output.WriteString(fmt.Sprintf("  ... and %d more entities\n", len(entities)-maxEntities))
			}
		}

		if relations, ok := graph["relations"].([]interface{}); ok {
			output.WriteString(fmt.Sprintf("\nRelations (%d):\n", len(relations)))

			// Show first 5 relations
			maxRelations := 5
			for i, relation := range relations {
				if i >= maxRelations {
					break
				}

				if relationMap, ok := relation.(map[string]interface{}); ok {
					source := relationMap["source"]
					relationType := relationMap["type"]
					target := relationMap["target"]
					output.WriteString(fmt.Sprintf("  • %s --%s--> %s\n", source, relationType, target))
				}
			}

			if len(relations) > maxRelations {
				output.WriteString(fmt.Sprintf("  ... and %d more relations\n", len(relations)-maxRelations))
			}
		}

		if output.Len() > 0 {
			return output.String()
		}
	}

	// Fallback to generic formatting
	return formatGenericResult(result)
}

// formatDiff formats diff content with colors and proper formatting
func formatDiff(diff string) string {
	var diffContent string

	if strings.Contains(diff, "```diff") {
		// Extract diff content from markdown code block
		start := strings.Index(diff, "```diff\n")
		if start == -1 {
			diffContent = diff
		} else {
			start += 8 // Length of "```diff\n"
			end := strings.Index(diff[start:], "\n```")
			if end == -1 {
				diffContent = diff[start:]
			} else {
				// Extract the actual diff content (without the closing ```)
				diffContent = diff[start : start+end]
			}
		}
	} else {
		// Raw diff content
		diffContent = diff
	}

	// Check if this looks like a unified diff
	hasDiffMarkers := strings.Contains(diffContent, "---") ||
		strings.Contains(diffContent, "+++") ||
		strings.Contains(diffContent, "@@")

	if !hasDiffMarkers {
		// If it doesn't look like a diff, just return the content
		return diffContent
	}

	var output strings.Builder
	output.WriteString("Changes:\n")

	// Define styles for diff elements
	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))     // Green
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))     // Red
	hunkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))    // Cyan
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))    // Blue

	lines := strings.Split(diffContent, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			output.WriteString(addStyle.Render(line) + "\n")
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			output.WriteString(delStyle.Render(line) + "\n")
		} else if strings.HasPrefix(line, "@@") {
			output.WriteString(hunkStyle.Render(line) + "\n")
		} else if strings.HasPrefix(line, " ") {
			output.WriteString(contextStyle.Render(line) + "\n")
		} else if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			output.WriteString(fileStyle.Render(line) + "\n")
		} else if line != "" {
			output.WriteString(line + "\n")
		}
	}

	return output.String()
}

// formatFileReadResult formats FileRead tool results
func formatFileReadResult(result string, arguments string) string {
	var response struct {
		Content string `json:"content"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		// If parsing fails, return the original result
		return result
	}

	if response.Error != "" {
		return fmt.Sprintf("Error reading file: %s", response.Error)
	}

	// Extract filename from arguments
	var args struct {
		FilePath string `json:"filePath"`
	}
	var filename string
	if err := json.Unmarshal([]byte(arguments), &args); err == nil && args.FilePath != "" {
		filename = args.FilePath
	}

	// Generate TUI-friendly summary with filename and line numbers
	lines := strings.Split(response.Content, "\n")
	var summary strings.Builder

	if filename != "" {
		summary.WriteString(fmt.Sprintf("FileRead: %s\n", filename))
	}
	summary.WriteString(fmt.Sprintf("📄 File content (%d lines)\n\n", len(lines)))

	previewCount := 20
	if len(lines) < previewCount {
		previewCount = len(lines)
	}

	for i := 0; i < previewCount; i++ {
		summary.WriteString(fmt.Sprintf("%4d: %s\n", i+1, lines[i]))
	}

	if len(lines) > 20 {
		summary.WriteString(fmt.Sprintf("\n... and %d more lines\n", len(lines)-20))
	}

	return summary.String()
}

// formatDirectoryResult formats Directory tool results
func formatDirectoryResult(result string) string {
	var response struct {
		Summary string `json:"summary"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		// If parsing fails, return the original result
		return result
	}

	// Return only the summary for TUI display
	return response.Summary
}

// formatGenericResult tries to parse generic JSON results
func formatGenericResult(result string) string {
	var jsonData map[string]any
	if err := json.Unmarshal([]byte(result), &jsonData); err != nil {
		// If not JSON, check if it's a long text and truncate if needed
		if len(result) > 500 {
			lines := strings.Split(result, "\n")
			if len(lines) > 8 {
				var output strings.Builder
				for i := 0; i < 8; i++ {
					output.WriteString(lines[i] + "\n")
				}
				output.WriteString(fmt.Sprintf("... and %d more lines", len(lines)-8))
				return output.String()
			}
		}
		return result // Return raw if not JSON and not too long
	}

	var output strings.Builder
	for key, value := range jsonData {
		// Handle long string values
		if str, ok := value.(string); ok && len(str) > 200 {
			lines := strings.Split(str, "\n")
			if len(lines) > 8 {
				var truncated strings.Builder
				for i := 0; i < 8; i++ {
					truncated.WriteString(lines[i] + "\n")
				}
				truncated.WriteString(fmt.Sprintf("... and %d more lines", len(lines)-8))
				value = truncated.String()
			}
		}
		output.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}

	return output.String()
}

// formatTokenCount formats a token count with comma separators for thousands
func formatTokenCount(count int) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}

	// Convert to string and add commas
	str := fmt.Sprintf("%d", count)
	var result strings.Builder

	// Process from right to left, adding commas every 3 digits
	for i, digit := range reversedString(str) {
		if i > 0 && i%3 == 0 {
			result.WriteByte(',')
		}
		result.WriteByte(byte(digit))
	}

	return reversedString(result.String())
}

// reversedString returns the reverse of a string
func reversedString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// getLastRequestUsage returns the token count and cost from the most recent API request
func getLastRequestUsage(chat *entities.Chat) (int, float64) {
	if chat == nil || len(chat.Messages) == 0 {
		return 0, 0.0
	}

	// Find the most recent assistant message (which has the API usage)
	for i := len(chat.Messages) - 1; i >= 0; i-- {
		msg := chat.Messages[i]
		if msg.Role == "assistant" && msg.Usage != nil {
			return msg.Usage.TotalTokens, msg.Usage.Cost
		}
	}

	// If no assistant message with usage found, return 0
	return 0, 0.0
}

type ChatView struct {
	chatService        services.ChatService
	agentService       services.AgentService
	modelService       services.ModelService
	logger             *zap.Logger
	activeChat         *entities.Chat
	editor             vimtea.Editor
	textarea           textarea.Model
	spinner            spinner.Model
	userStyle          lipgloss.Style
	asstStyle          lipgloss.Style
	systemStyle        lipgloss.Style
	err                error
	cancel             context.CancelFunc
	isProcessing       bool
	startTime          time.Time
	focused            string // "textarea" or "editor"
	width              int
	height             int
	currentAgent       *entities.Agent
	currentModel       *entities.Model
	previousAgentID    string             // Track previous agent ID to detect changes
	tempMessages       []entities.Message // Temporary messages for real-time tool events
	eventCancel        func()             // Event subscription cancel function
	eventChan          chan interface{}   // Channel for receiving events (ToolCallEvent or MessageHistoryEvent)
	lineNumbersEnabled bool               // Track whether line numbers are enabled
	toolCallStatus     map[string]bool    // Track completion status of tool calls (toolCallID -> completed)
}

func NewChatView(chatService services.ChatService, agentService services.AgentService, modelService services.ModelService, logger *zap.Logger, activeChat *entities.Chat) ChatView {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.SetWidth(30)
	ta.SetHeight(2)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("ctrl+j"))

	us := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	as := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	ss := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	cv := ChatView{
		chatService:        chatService,
		agentService:       agentService,
		modelService:       modelService,
		logger:             logger,
		activeChat:         activeChat,
		textarea:           ta,
		spinner:            s,
		userStyle:          us,
		asstStyle:          as,
		systemStyle:        ss,
		err:                nil,
		focused:            "textarea",
		width:              30,
		height:             5,
		lineNumbersEnabled: false, // Start with line numbers disabled
		toolCallStatus:     make(map[string]bool),
	}

	// Initialize the vimtea editor
	cv.editor = vimtea.NewEditor(
		vimtea.WithEnableModeCommand(true), // Enable command mode for :set commands
		vimtea.WithEnableStatusBar(false),
		vimtea.WithShowLineNumbers(cv.lineNumbersEnabled),
		vimtea.WithReadOnly(true),
		vimtea.WithSelectedStyle(lipgloss.NewStyle().Background(lipgloss.Color("#586e75"))),
	)

	if activeChat != nil {
		cv.activeChat = activeChat
		ctx := context.Background()
		agent, err := agentService.GetAgent(ctx, activeChat.AgentID)
		if err != nil {
			cv.err = err
			cv.currentAgent = nil
		} else {
			cv.currentAgent = agent
		}
		model, err := modelService.GetModel(ctx, activeChat.ModelID)
		if err != nil {
			cv.err = err
			cv.currentModel = nil
		} else {
			cv.currentModel = model
		}
	}
	cv.updateEditorContent()

	// Set up event channel and subscriptions for real-time updates
	cv.eventChan = make(chan interface{}, 50) // Buffered channel - increased buffer to prevent event drops

	// Subscribe to tool call events
	toolCancel := events.SubscribeToToolCallEvents(func(data events.ToolCallEventData) {
		// Send event to channel (non-blocking)
		select {
		case cv.eventChan <- data.Event:
		default:
			// Channel full, drop event to avoid blocking
		}
	})

	// Subscribe to message history events
	messageCancel := events.SubscribeToMessageHistoryEvents(func(data events.MessageHistoryEventData) {
		// Send event to channel (non-blocking)
		select {
		case cv.eventChan <- data:
		default:
			// Channel full, drop event to avoid blocking
		}
	})

	// Combine cancel functions
	cv.eventCancel = func() {
		toolCancel()
		messageCancel()
	}

	return cv
}

func (c *ChatView) SetActiveChat(chat *entities.Chat) {
	// Check if agent is changing
	agentChanged := c.activeChat == nil || c.activeChat.AgentID != chat.AgentID
	c.previousAgentID = ""
	if c.activeChat != nil {
		c.previousAgentID = c.activeChat.AgentID
	}

	c.activeChat = chat
	ctx := context.Background()
	agent, err := c.agentService.GetAgent(ctx, chat.AgentID)
	if err != nil {
		c.err = err
		c.currentAgent = nil
	} else {
		c.currentAgent = agent
	}
	model, err := c.modelService.GetModel(ctx, chat.ModelID)
	if err != nil {
		c.err = err
		c.currentModel = nil
	} else {
		c.currentModel = model
	}

	// Add system message if agent changed
	if agentChanged && c.activeChat != nil && len(c.activeChat.Messages) > 0 {
		systemMsg := &entities.Message{
			Content: "Switched to new agent",
			Role:    "system",
		}
		c.activeChat.Messages = append(c.activeChat.Messages, *systemMsg)
	}

	c.updateEditorContent()

	// Self-heal: Regenerate title if it's still default and has conversation
	if strings.HasPrefix(chat.Name, "New Chat") && len(chat.Messages) >= 2 {
		// Trigger title regeneration asynchronously
		go func() {
			ctx := context.Background()
			if updatedChat, err := c.chatService.GenerateAndUpdateTitle(ctx, chat.ID); err != nil {
				c.logger.Warn("Failed to regenerate title for old chat", zap.Error(err))
			} else {
				c.logger.Info("Regenerated title for old chat", zap.String("chat_id", chat.ID), zap.String("new_title", updatedChat.Name))
				// Update the active chat reference if it matches
				if c.activeChat != nil && c.activeChat.ID == updatedChat.ID {
					c.activeChat = updatedChat
				}
			}
		}()
	}
}

func (c *ChatView) updateEditorContent() {
	if c.activeChat == nil || len(c.activeChat.Messages) == 0 {
		c.editor = vimtea.NewEditor(
			vimtea.WithContent("How can I help you today?\n"),
			vimtea.WithEnableModeCommand(true), // Enable command mode for :set commands
			vimtea.WithEnableStatusBar(false),
			vimtea.WithShowLineNumbers(c.lineNumbersEnabled),
			vimtea.WithReadOnly(true),
			vimtea.WithSelectedStyle(lipgloss.NewStyle().Background(lipgloss.Color("#586e75"))),
		)
		// Ensure editor is not focused initially
		c.editor.SetFocus(false)
		// Set editor size to fit screen minus textarea, footer, separators, and header
		if c.width > 0 && c.height > 0 {
			editorWidth := c.width
			editorHeight := c.height - (c.textarea.Height() + 1 + 2 + 1) // textarea (dynamic) + footer (1) + separators (2) + header (1)
			if editorHeight < 1 {
				editorHeight = 1
			}
			c.editor.SetSize(editorWidth, editorHeight)
		}
		return
	}

	var sb strings.Builder
	for _, message := range c.activeChat.Messages {
		if message.Role == "user" {
			sb.WriteString("\n" + c.userStyle.Render("User: ") + message.Content + "\n\n")
		} else if message.Role == "assistant" {
			// Skip displaying tool execution announcements in TUI
			if len(message.ToolCalls) == 0 {
				sb.WriteString(c.asstStyle.Render("Assistant: ") + message.Content + "\n")
			}
		} else if message.Role == "tool" {
			sb.WriteString(c.systemStyle.Render("Tool: ") + "\n")
			// Display tool call events
			for _, event := range message.ToolCallEvents {
				formattedResult := formatToolResult(event.ToolName, event.Result, event.Diff, event.Arguments)
				name, suffix := formatToolName(event.ToolName, event.Arguments)
				statusIcon := getToolStatusIcon(event.Error != "")
				displayName := name + ":"
				if suffix != "" {
					displayName += " " + suffix
				}
				sb.WriteString(c.systemStyle.Render("  ↳ ") + statusIcon + " " + displayName + "\n")
				sb.WriteString(c.systemStyle.Render("    ") + strings.ReplaceAll(formattedResult, "\n", "\n    ") + "\n")
				if event.Error != "" {
					errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true) // Red and bold
					sb.WriteString(errorStyle.Render("    ✗ Error: ") + event.Error + "\n")
				}
			}
		} else {
			sb.WriteString(c.systemStyle.Render("System: ") + message.Content + "\n")
		}
	}

	// Add temporary tool call messages for real-time updates during processing
	if c.isProcessing && len(c.tempMessages) > 0 {
		for _, tempMsg := range c.tempMessages {
			sb.WriteString(c.systemStyle.Render("Tool: ") + "\n")
			// Display tool call events
			for _, event := range tempMsg.ToolCallEvents {
				formattedResult := formatToolResult(event.ToolName, event.Result, event.Diff, event.Arguments)
				name, suffix := formatToolName(event.ToolName, event.Arguments)
				statusIcon := getToolStatusIcon(event.Error != "")
				displayName := name + ":"
				if suffix != "" {
					displayName += " " + suffix
				}
				sb.WriteString(c.systemStyle.Render("  ↳ ") + statusIcon + " " + displayName + "\n")
				sb.WriteString(c.systemStyle.Render("    ") + strings.ReplaceAll(formattedResult, "\n", "\n    ") + "\n")
				if event.Error != "" {
					errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true) // Red and bold
					sb.WriteString(errorStyle.Render("    ✗ Error: ") + event.Error + "\n")
				}
			}
		}
	}

	// Add current tool calls being executed if processing
	if c.isProcessing && len(c.activeChat.Messages) > 0 {
		lastMsg := c.activeChat.Messages[len(c.activeChat.Messages)-1]
		if lastMsg.Role == "assistant" && len(lastMsg.ToolCalls) > 0 {
			sb.WriteString("\n" + c.systemStyle.Render("Executing tools:") + "\n\n")
			for _, toolCall := range lastMsg.ToolCalls {
				sb.WriteString(c.systemStyle.Render("  ↳ ") + "🔄 " + toolCall.Function.Name + "\n")
			}
		}
	}

	// Add error as temporary system message if present
	if c.err != nil {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(c.systemStyle.Render("System: Error - "+c.err.Error()) + "\n\n")
		// Clear the error after displaying it once
		c.err = nil
	}

	content := sb.String()
	// Recreate editor with new content
	c.editor = vimtea.NewEditor(
		vimtea.WithContent(content),
		vimtea.WithEnableModeCommand(true),               // Enable command mode for :set commands
		vimtea.WithEnableStatusBar(false),                // Disable status bar
		vimtea.WithShowLineNumbers(c.lineNumbersEnabled), // Use current line numbers setting
		vimtea.WithReadOnly(true),
		vimtea.WithSelectedStyle(lipgloss.NewStyle().Background(lipgloss.Color("#586e75"))),
	)
	// Ensure editor maintains current focus state
	c.editor.SetFocus(c.focused == "editor")

	// Set editor size: textarea (dynamic) + footer (1) + separators (2) + header (1)
	if c.width > 0 && c.height > 0 {
		editorWidth := c.width
		editorHeight := c.height - (c.textarea.Height() + 1 + 2 + 1)
		if editorHeight < 1 {
			editorHeight = 1
		}
		c.editor.SetSize(editorWidth, editorHeight)
	}

	// Ensure editor is in normal mode and scroll to bottom to show latest messages
	c.editor.SetMode(vimtea.ModeNormal)
	c.editor.SetFocus(true)
	_, _ = c.editor.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	c.editor.SetFocus(false)

}

func (c ChatView) Init() tea.Cmd {
	c.textarea.Focus()
	c.focused = "textarea"
	// Initialize editor with proper options
	c.editor = vimtea.NewEditor(
		vimtea.WithEnableModeCommand(true), // Enable command mode for :set commands
		vimtea.WithEnableStatusBar(false),
		vimtea.WithShowLineNumbers(c.lineNumbersEnabled),
		vimtea.WithReadOnly(true),
		vimtea.WithSelectedStyle(lipgloss.NewStyle().Background(lipgloss.Color("#586e75"))),
	)

	// Ensure editor starts in normal mode and is not focused
	c.editor.SetMode(vimtea.ModeNormal)
	c.editor.SetFocus(false)
	return tea.Batch(textarea.Blink, c.listenForEvents())
}

// listenForEvents returns a command that listens for events
func (c *ChatView) listenForEvents() tea.Cmd {
	return func() tea.Msg {
		event := <-c.eventChan
		switch e := event.(type) {
		case *entities.ToolCallEvent:
			return toolCallEventMsg(e)
		case events.MessageHistoryEventData:
			return messageHistoryEventMsg(e)
		default:
			return nil
		}
	}
}

func (c ChatView) Update(msg tea.Msg) (ChatView, tea.Cmd) {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case tea.MouseMsg:
		// Handle mouse events for focus switching and editor interaction
		// Calculate approximate textarea position (bottom of screen)
		textareaHeight := c.textarea.Height() + 2 + 1 // height + borders + footer line
		editorBottom := c.height - textareaHeight

		if m.Y < editorBottom {
			// Mouse event in editor area - always pass to vimtea editor
			if c.focused != "editor" {
				c.focused = "editor"
				c.textarea.Blur()
				if c.editor != nil {
					c.editor.SetMode(vimtea.ModeNormal)
					c.editor.SetFocus(true)
				}
			}

			// Handle mouse wheel events for scrolling the editor
			switch m.Type {
			case tea.MouseWheelUp:
				// Send mouse wheel up to vimtea (moves cursor up)
				mouseMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
				newModel, _ := c.editor.Update(mouseMsg)
				if editor, ok := newModel.(vimtea.Editor); ok {
					c.editor = editor
				}
			case tea.MouseWheelDown:
				// Send mouse wheel down to vimtea (moves cursor down)
				mouseMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
				newModel, _ := c.editor.Update(mouseMsg)
				if editor, ok := newModel.(vimtea.Editor); ok {
					c.editor = editor
				}
			default:
				// Adjust mouse coordinates for borders before passing to vimtea
				adjustedMsg := m
				if m.X > 0 {
					adjustedMsg.X = m.X - 1 // Adjust for left border
				}
				if m.Y > 0 {
					adjustedMsg.Y = m.Y - 1 // Adjust for top border
				}

				// Pass adjusted mouse event to vimtea editor
				if c.editor != nil {
					newModel, cmd := c.editor.Update(adjustedMsg)
					if editor, ok := newModel.(vimtea.Editor); ok {
						c.editor = editor
					}
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
			return c, tea.Batch(cmds...)
		} else {
			// Mouse event in textarea area
			if c.focused != "textarea" {
				c.focused = "textarea"
				c.textarea.Focus()
				if c.editor != nil {
					c.editor.SetFocus(false)
				}
				cmds = append(cmds, textarea.Blink)
			}
		}
		return c, tea.Batch(cmds...)

	case tea.KeyMsg:
		if c.isProcessing {
			if m.Type == tea.KeyEsc {
				if c.cancel != nil {
					c.cancel()
					c.isProcessing = false
					// Don't set error immediately - let the sendMessageCmd complete with partial results
					// Send 'G' key to vimtea to go to bottom
					bottomMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
					newModel, _ := c.editor.Update(bottomMsg)
					if editor, ok := newModel.(vimtea.Editor); ok {
						c.editor = editor
					}
				}
				return c, nil
			}
			return c, nil
		}

		switch m.String() {
		case "ctrl+c":
			return c, tea.Quit
		case "esc":
			return c, nil
		case "ctrl+p":
			if c.focused == "textarea" {
				return c, func() tea.Msg { return startCommandsMsg{} }
			}
		case "ctrl+a":
			return c, func() tea.Msg { return startAgentSwitchMsg{} }
		case "ctrl+o":
			return c, func() tea.Msg { return startModelSwitchMsg{} }
		case "ctrl+n":
			return c, func() tea.Msg { return startAutoCreateChatMsg{} }
		case "ctrl+l":
			// Toggle line numbers
			c.lineNumbersEnabled = !c.lineNumbersEnabled
			c.updateEditorContent()
			return c, nil
		case "enter":
			if c.focused == "textarea" {
				input := c.textarea.Value()
				if input == "" {
					c.err = fmt.Errorf("message cannot be empty")
					return c, nil
				}
				if c.activeChat == nil {
					c.err = fmt.Errorf("no active chat")
					return c, nil
				}
				message := &entities.Message{
					Content: input,
					Role:    "user",
				}
				c.textarea.Reset()
				c.textarea.SetHeight(2)
				editorHeight := c.height - (2 + 1 + 2 + 1)
				if editorHeight < 1 {
					editorHeight = 1
				}
				c.editor.SetSize(c.width, editorHeight)
				c.activeChat.Messages = append(c.activeChat.Messages, *message)
				// Initialize tool call status tracking for this message
				c.toolCallStatus = make(map[string]bool)
				c.updateEditorContent()
				// Scroll to bottom to show the new user message
				bottomMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
				newModel, _ := c.editor.Update(bottomMsg)
				if editor, ok := newModel.(vimtea.Editor); ok {
					c.editor = editor
				}
				c.err = nil
				ctx, cancel := context.WithCancel(context.Background())
				c.cancel = cancel
				c.isProcessing = true
				c.startTime = time.Now()
				return c, tea.Batch(sendMessageCmd(c.chatService, c.activeChat.ID, message, ctx), c.spinner.Tick)
			}
		case "tab", "shift+tab":
			if c.focused == "textarea" {
				c.focused = "editor"
				c.textarea.Blur()
				// Focus the editor
				if c.editor != nil {
					c.editor.SetMode(vimtea.ModeNormal)
					c.editor.SetFocus(true)
				}
			} else {
				c.focused = "textarea"
				c.textarea.Focus()
				// Unfocus the editor
				if c.editor != nil {
					c.editor.SetFocus(false)
				}
				cmd := textarea.Blink
				cmds = append(cmds, cmd)
			}
			return c, tea.Batch(cmds...)
		default:
			if c.focused == "textarea" {
				var cmd tea.Cmd
				c.textarea, cmd = c.textarea.Update(m)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				logicalLines := c.textarea.LineCount()
				currentLineHeight := c.textarea.LineInfo().Height
				effectiveLines := max(logicalLines, currentLineHeight)
				currentHeight := c.textarea.Height()
				if effectiveLines != currentHeight {
					newHeight := min(5, max(2, effectiveLines))
					c.textarea.SetHeight(newHeight)
					editorHeight := c.height - (newHeight + 1 + 2 + 1)
					if editorHeight < 1 {
						editorHeight = 1
					}
					c.editor.SetSize(c.width, editorHeight)
				}
			} else if c.focused == "editor" {
				// Pass all keystrokes to vimtea editor when focused
				var cmd tea.Cmd
				newModel, cmd := c.editor.Update(m)
				if editor, ok := newModel.(vimtea.Editor); ok {
					c.editor = editor
				}
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}

	case spinner.TickMsg:
		if c.isProcessing {
			var cmd tea.Cmd
			c.spinner, cmd = c.spinner.Update(m)
			return c, cmd
		}

	case toolCallEventMsg:
		// Handle real-time tool call event
		if c.isProcessing && c.activeChat != nil {
			// Mark this tool call as completed
			if m.ToolCallID != "" {
				c.toolCallStatus[m.ToolCallID] = true
			}

			// Create a temporary message for the tool call event
			tempMsg := entities.Message{
				ID:             m.ID,
				Role:           "tool",
				Content:        m.Result,
				ToolCallID:     "", // Will be set when final message arrives
				ToolCallEvents: []entities.ToolCallEvent{*m},
				Timestamp:      m.Timestamp,
			}
			c.tempMessages = append(c.tempMessages, tempMsg)
			// Update editor content with new tool event
			c.updateEditorContent()
		}
		// Continue listening for more events
		return c, c.listenForEvents()

	case messageHistoryEventMsg:
		// Handle message history change event
		if c.isProcessing && c.activeChat != nil && m.ChatID == c.activeChat.ID {
			// Sync active chat messages to prevent stale data display
			messages := make([]entities.Message, len(m.Messages))
			for i, msg := range m.Messages {
				messages[i] = *msg
			}
			c.activeChat.Messages = messages

			// Only clear temp messages if this is the final message (no tool calls pending)
			// This prevents tool call results from disappearing during live updates
			if len(m.Messages) > 0 {
				lastMsg := m.Messages[len(m.Messages)-1]
				if lastMsg.Role == "assistant" && len(lastMsg.ToolCalls) == 0 {
					// This is the final assistant response, clear temp messages
					c.tempMessages = nil
				}
			}
			// Refresh the editor content with updated message history
			c.updateEditorContent()
		}
		// Continue listening for more events
		return c, c.listenForEvents()

	case updatedChatMsg:
		c.textarea.Reset()
		// Check if agent is changing
		agentChanged := c.activeChat == nil || c.activeChat.AgentID != m.AgentID
		c.previousAgentID = ""
		if c.activeChat != nil {
			c.previousAgentID = c.activeChat.AgentID
		}

		c.activeChat = m
		ctx := context.Background()
		agent, err := c.agentService.GetAgent(ctx, m.AgentID)
		if err != nil {
			c.err = err
			c.currentAgent = nil
		} else {
			c.currentAgent = agent
		}
		model, err := c.modelService.GetModel(ctx, m.ModelID)
		if err != nil {
			c.err = err
			c.currentModel = nil
		} else {
			c.currentModel = model
		}

		// Add system message if agent changed
		if agentChanged && c.activeChat != nil && len(c.activeChat.Messages) > 0 {
			systemMsg := &entities.Message{
				Content: "Switched to new agent",
				Role:    "system",
			}
			c.activeChat.Messages = append(c.activeChat.Messages, *systemMsg)
		}

		// Clear temporary messages and tool call status since we now have the final messages
		c.tempMessages = nil
		c.toolCallStatus = make(map[string]bool)

		c.updateEditorContent()
		c.isProcessing = false
		c.cancel = nil
		return c, nil

	case errMsg:
		c.isProcessing = false
		c.cancel = nil
		// Don't remove user's message if the error is due to cancellation
		// The server should have saved the user's message and any partial results
		if c.activeChat != nil && len(c.activeChat.Messages) > 0 && !strings.Contains(m.Error(), "canceled") {
			lastIdx := len(c.activeChat.Messages) - 1
			if c.activeChat.Messages[lastIdx].Role == "user" {
				c.activeChat.Messages = c.activeChat.Messages[:lastIdx]
			}
		}
		// Add error message to chat history only if we have an active chat
		if c.activeChat != nil {
			errorMsg := &entities.Message{
				Content: "Error: " + m.Error(),
				Role:    "system",
			}
			c.activeChat.Messages = append(c.activeChat.Messages, *errorMsg)
		}
		// Set error for immediate display
		c.err = m
		// Clear temporary messages and tool call status on error
		c.tempMessages = nil
		c.toolCallStatus = make(map[string]bool)
		c.updateEditorContent()
		return c, nil

	case tea.WindowSizeMsg:
		c.width = m.Width
		c.height = m.Height

		// Set editor size to fit screen minus textarea, footer, separators, and header
		editorWidth := c.width
		editorHeight := c.height - (c.textarea.Height() + 1 + 2 + 1) // textarea (dynamic) + footer (1) + separators (2) + header (1)
		if editorHeight < 1 {
			editorHeight = 1
		}
		c.editor.SetSize(editorWidth, editorHeight)

		c.textarea.SetWidth(c.width)

		if c.activeChat != nil {
			c.updateEditorContent()
		}
		return c, nil
	}

	return c, tea.Batch(cmds...)
}

func (c ChatView) View() string {
	// Define styles
	style := lipgloss.NewStyle().Width(c.width)

	// Separator
	separator := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render(strings.Repeat("\u2500", c.width))

	// Editor - needs to account for header taking space
	editorHeight := c.height - (c.textarea.Height() + 1 + 2 + 1) // textarea (dynamic) + footer (1) + separators (2) + header (1)
	if editorHeight < 1 {
		editorHeight = 1
	}
	editorStyle := lipgloss.NewStyle().Width(c.width).Height(editorHeight)
	editorPart := editorStyle.Render(c.editor.View())

	// Header - title left (bold bright white), token info right
	header := ""
	if c.activeChat != nil && c.width > 0 {
		titleStyle := lipgloss.NewStyle().PaddingLeft(1).Bold(true).Foreground(lipgloss.Color("15"))

		var tokenInfo string
		// Use per-request usage instead of accumulated totals
		requestTokens, requestCost := getLastRequestUsage(c.activeChat)
		// Format token count with commas
		tokenCountStr := formatTokenCount(requestTokens)
		if c.currentModel != nil && c.currentModel.ContextWindow != nil && *c.currentModel.ContextWindow > 0 && requestTokens > 0 {
			percentage := int(float64(requestTokens) / float64(*c.currentModel.ContextWindow) * 100)
			tokenInfo = fmt.Sprintf("%s %d%% ($%.2f)", tokenCountStr, percentage, requestCost)
		} else {
			tokenInfo = fmt.Sprintf("%s ($%.2f)", tokenCountStr, requestCost)
		}

		chatName := "New Chat"
		if c.activeChat != nil {
			chatName = c.activeChat.Name
		}
		headerLine := lipgloss.JoinHorizontal(
			lipgloss.Top,
			titleStyle.Render(chatName),
			lipgloss.NewStyle().Width(c.width-lipgloss.Width(chatName)-2).Align(lipgloss.Right).Render(tokenInfo),
		)
		header = headerLine
	}

	// Textarea
	taStyle := style.Height(c.textarea.Height())
	textareaPart := taStyle.Render(c.textarea.View())

	instructions := "Ctrl+P: menu | Tab: focus | Ctrl+C: exit"
	if c.isProcessing {
		elapsed := time.Since(c.startTime).Round(time.Second)
		instructions = c.spinner.View() + fmt.Sprintf(" Working... (%ds) esc to interrupt", int(elapsed.Seconds()))
	}

	agentInfo := "No agent selected"
	if c.currentAgent != nil {
		agentInfo = fmt.Sprintf("Agent: %s", c.currentAgent.Name)
	}

	modelInfo := "No model selected"
	if c.currentModel != nil {
		modelInfo = fmt.Sprintf("Model: %s", c.currentModel.Name)
	}

	footerInfo := agentInfo + " | " + modelInfo

	footerStyle := lipgloss.NewStyle().Width(c.width)
	leftStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Inline(true)
	rightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Align(lipgloss.Right).Inline(true).Width(c.width - lipgloss.Width(instructions))
	footerPart := footerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, leftStyle.Render(instructions), rightStyle.Render(footerInfo)))

	return lipgloss.JoinVertical(lipgloss.Top, editorPart, header, separator, textareaPart, separator, footerPart)
}

func sendMessageCmd(cs services.ChatService, chatID string, msg *entities.Message, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		// Add a very long timeout to allow for complex operations (1 hour)
		cmdCtx, cmdCancel := context.WithTimeout(ctx, 1*time.Hour)
		defer cmdCancel()

		_, err := cs.SendMessage(cmdCtx, chatID, msg)
		if err != nil {
			if cmdCtx.Err() == context.Canceled {
				// User cancelled - try to get partial results
				getChatCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				updatedChat, getChatErr := cs.GetChat(getChatCtx, chatID)
				if getChatErr == nil && len(updatedChat.Messages) > 0 {
					noticeMsg := &entities.Message{
						Content: "⚠️  Operation cancelled by user. Showing partial results.",
						Role:    "system",
					}
					updatedChat.Messages = append(updatedChat.Messages, *noticeMsg)
					return updatedChatMsg(updatedChat)
				}
				return errMsg(fmt.Errorf("operation cancelled by user - no results available"))
			} else if cmdCtx.Err() == context.DeadlineExceeded {
				// Timeout - try to get partial results
				getChatCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				updatedChat, getChatErr := cs.GetChat(getChatCtx, chatID)
				if getChatErr == nil && len(updatedChat.Messages) > 0 {
					noticeMsg := &entities.Message{
						Content: "⚠️  Operation timed out after 1 hour. Showing partial results.",
						Role:    "system",
					}
					updatedChat.Messages = append(updatedChat.Messages, *noticeMsg)
					return updatedChatMsg(updatedChat)
				}
				return errMsg(fmt.Errorf("operation timed out after 1 hour - no results available"))
			} else {
				// Other error - get updated chat state and add error message
				getChatCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				updatedChat, getChatErr := cs.GetChat(getChatCtx, chatID)
				if getChatErr == nil {
					// Add error message to chat
					errorMsg := &entities.Message{
						Content: "Error: " + err.Error(),
						Role:    "system",
					}
					updatedChat.Messages = append(updatedChat.Messages, *errorMsg)
					return updatedChatMsg(updatedChat)
				}
				return errMsg(err)
			}
		}
		// Use a new context for GetChat in case the original context was cancelled
		getChatCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		updatedChat, err := cs.GetChat(getChatCtx, chatID)
		if err != nil {
			return errMsg(err)
		}
		return updatedChatMsg(updatedChat)
	}
}
