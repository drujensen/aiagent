package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/events"
	"github.com/drujensen/aiagent/internal/domain/services"
	"github.com/kujtimiihoxha/vimtea"
)

// formatToolResult formats tool execution results for display
func formatToolResult(toolName, result string, diff string) string {
	switch toolName {
	case "FileWrite":
		return formatFileWriteResult(result, diff)
	case "FileSearch":
		return formatFileSearchResult(result)
	case "Memory":
		return formatMemoryResult(result)
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

	// Add the diff if available
	if diff != "" {
		output.WriteString("\n\n" + formatDiff(diff))
	} else if resultData.Diff != "" {
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

// formatTaskResult formats Task tool results
func formatTaskResult(result string) string {
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

// formatDiff formats diff content with colors and proper formatting
func formatDiff(diff string) string {
	var diffContent string

	if strings.Contains(diff, "```diff") {
		// Extract diff content from markdown code block
		start := strings.Index(diff, "```diff\n")
		if start == -1 {
			return diff
		}
		start += 8 // Length of "```diff\n"

		end := strings.Index(diff[start:], "\n```")
		if end == -1 {
			diffContent = diff[start:]
		} else {
			// Extract the actual diff content (without the closing ```)
			diffContent = diff[start : start+end]
		}
	} else {
		// Raw diff content
		diffContent = diff
	}

	var output strings.Builder
	output.WriteString("Changes:\n")

	// Define styles for diff elements
	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))     // Green
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))     // Red
	hunkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))    // Cyan
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray

	lines := strings.Split(diffContent, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") {
			output.WriteString(addStyle.Render(line) + "\n")
		} else if strings.HasPrefix(line, "-") {
			output.WriteString(delStyle.Render(line) + "\n")
		} else if strings.HasPrefix(line, "@@") {
			output.WriteString(hunkStyle.Render(line) + "\n")
		} else if strings.HasPrefix(line, " ") {
			output.WriteString(contextStyle.Render(line) + "\n")
		} else {
			output.WriteString(line + "\n")
		}
	}

	return output.String()
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

type ChatView struct {
	chatService        services.ChatService
	agentService       services.AgentService
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
	previousAgentID    string                       // Track previous agent ID to detect changes
	tempMessages       []entities.Message           // Temporary messages for real-time tool events
	eventCancel        func()                       // Event subscription cancel function
	eventChan          chan *entities.ToolCallEvent // Channel for receiving tool call events
	lineNumbersEnabled bool                         // Track whether line numbers are enabled
}

func NewChatView(chatService services.ChatService, agentService services.AgentService, activeChat *entities.Chat) ChatView {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.SetWidth(30)
	ta.SetHeight(2)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	us := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	as := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	ss := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	cv := ChatView{
		chatService:        chatService,
		agentService:       agentService,
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
	}
	cv.updateEditorContent()

	// Set up event channel and subscription for real-time tool call updates
	cv.eventChan = make(chan *entities.ToolCallEvent, 10) // Buffered channel
	cv.eventCancel = events.SubscribeToToolCallEvents(func(data events.ToolCallEventData) {
		// Send event to channel (non-blocking)
		select {
		case cv.eventChan <- data.Event:
		default:
			// Channel full, drop event to avoid blocking
		}
	})

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

	// Add system message if agent changed
	if agentChanged && c.activeChat != nil && len(c.activeChat.Messages) > 0 {
		systemMsg := &entities.Message{
			Content: "Switched to new agent",
			Role:    "system",
		}
		c.activeChat.Messages = append(c.activeChat.Messages, *systemMsg)
	}

	c.updateEditorContent()
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
		// Set editor size - outer border (2) + footer (1) + textarea (2) + inner borders (4) + text wrapping adjustment = 10 total
		if c.width > 0 && c.height > 0 {
			editorWidth := c.width - 4
			editorHeight := c.height - 10
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
			sb.WriteString(c.userStyle.Render("User: ") + message.Content + "\n")
		} else if message.Role == "assistant" {
			sb.WriteString(c.asstStyle.Render("Assistant: ") + message.Content + "\n")
		} else if message.Role == "tool" {
			sb.WriteString(c.systemStyle.Render("Tool: ") + "\n")
			// Display tool call events
			for _, event := range message.ToolCallEvents {
				formattedResult := formatToolResult(event.ToolName, event.Result, event.Diff)
				statusIcon := getToolStatusIcon(event.Error != "")
				sb.WriteString(c.systemStyle.Render("  ↳ ") + statusIcon + " " + event.ToolName + ":\n")
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
				formattedResult := formatToolResult(event.ToolName, event.Result, event.Diff)
				statusIcon := getToolStatusIcon(event.Error != "")
				sb.WriteString(c.systemStyle.Render("  ↳ ") + statusIcon + " " + event.ToolName + ":\n")
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
			sb.WriteString("\n" + c.systemStyle.Render("Executing tools:") + "\n")
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
		sb.WriteString(c.systemStyle.Render("System: Error - ") + c.err.Error() + "\n")
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

	// Set editor size - outer border (2) + footer (1) + textarea (2) + inner borders (4) + text wrapping adjustment = 10 total
	if c.width > 0 && c.height > 0 {
		editorWidth := c.width - 4
		editorHeight := c.height - 10
		if editorHeight < 1 {
			editorHeight = 1
		}
		c.editor.SetSize(editorWidth, editorHeight)
	}
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

// listenForEvents returns a command that listens for tool call events
func (c *ChatView) listenForEvents() tea.Cmd {
	return func() tea.Msg {
		event := <-c.eventChan
		return toolCallEventMsg(event)
	}
}

func (c ChatView) Update(msg tea.Msg) (ChatView, tea.Cmd) {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case tea.KeyMsg:
		if c.isProcessing {
			if m.Type == tea.KeyEsc {
				if c.cancel != nil {
					c.cancel()
					c.isProcessing = false
					c.err = fmt.Errorf("request cancelled")
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
		case "ctrl+n":
			return c, func() tea.Msg { return startCreateChatMsg("") }
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
				c.activeChat.Messages = append(c.activeChat.Messages, *message)
				c.updateEditorContent()
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

		// Add system message if agent changed
		if agentChanged && c.activeChat != nil && len(c.activeChat.Messages) > 0 {
			systemMsg := &entities.Message{
				Content: "Switched to new agent",
				Role:    "system",
			}
			c.activeChat.Messages = append(c.activeChat.Messages, *systemMsg)
		}

		// Clear temporary messages since we now have the final messages
		c.tempMessages = nil

		c.updateEditorContent()
		c.isProcessing = false
		c.cancel = nil
		return c, nil

	case errMsg:
		c.isProcessing = false
		c.cancel = nil
		c.err = m
		if len(c.activeChat.Messages) > 0 {
			lastIdx := len(c.activeChat.Messages) - 1
			if c.activeChat.Messages[lastIdx].Role == "user" {
				c.activeChat.Messages = c.activeChat.Messages[:lastIdx]
			}
		}
		// Clear temporary messages on error
		c.tempMessages = nil
		c.updateEditorContent()
		return c, nil

	case tea.WindowSizeMsg:
		c.width = m.Width
		c.height = m.Height

		// Set editor size - outer border (2) + footer (1) + textarea (2) + inner borders (4) + text wrapping adjustment = 10 total
		editorWidth := c.width - 4
		editorHeight := c.height - 10
		if editorHeight < 1 {
			editorHeight = 1
		}
		c.editor.SetSize(editorWidth, editorHeight)

		c.textarea.SetWidth(c.width - 4)

		if c.activeChat != nil {
			c.updateEditorContent()
		}
		return c, nil
	case tea.MouseMsg:
		editorYStart := 1
		editorBlockHeight := c.height - 4 + 2
		editorYEnd := editorYStart + editorBlockHeight
		if m.Y >= editorYStart && m.Y < editorYEnd {
			switch m.Type {
			case tea.MouseWheelUp:
				// Send mouse wheel up to vimtea
				mouseMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
				newModel, _ := c.editor.Update(mouseMsg)
				if editor, ok := newModel.(vimtea.Editor); ok {
					c.editor = editor
				}
			case tea.MouseWheelDown:
				// Send mouse wheel down to vimtea
				mouseMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
				newModel, _ := c.editor.Update(mouseMsg)
				if editor, ok := newModel.(vimtea.Editor); ok {
					c.editor = editor
				}
			}
		}
		return c, nil
	}

	return c, tea.Batch(cmds...)
}

func (c ChatView) View() string {
	// Define border styles
	focusedBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")) // Bright cyan for focused

	unfocusedBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")) // Dim gray for unfocused

	// Outer container style (Vim-like overall border)
	outerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("4")). // Blue for outer border
		Width(c.width - 2).
		Height(c.height - 2)

	var sb strings.Builder

	// Style editor - let outer container limit height to handle text wrapping
	editorStyle := unfocusedBorder.Width(c.width - 4)
	if c.focused == "editor" {
		editorStyle = focusedBorder.Width(c.width - 4)
	}

	sb.WriteString(editorStyle.Render(c.editor.View()))

	// Style textarea
	taStyle := unfocusedBorder.Width(c.width - 4).Height(c.textarea.Height())
	if c.focused == "textarea" {
		taStyle = focusedBorder.Width(c.width - 4).Height(c.textarea.Height())
	}
	sb.WriteString(taStyle.Render(c.textarea.View()))

	instructions := "Ctrl+P: menu | Tab: focus | Ctrl+C: exit"
	if c.isProcessing {
		elapsed := time.Since(c.startTime).Round(time.Second)
		instructions = c.spinner.View() + fmt.Sprintf(" Working... (%ds) esc to interrupt", int(elapsed.Seconds()))
	}

	agentInfo := "No agent selected"
	if c.currentAgent != nil {
		agentInfo = fmt.Sprintf("%s (%s: %s)", c.currentAgent.Name, c.currentAgent.ProviderType, c.currentAgent.Model)
	}

	footerStyle := lipgloss.NewStyle().Width(c.width - 4)
	leftStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Inline(true)
	rightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Align(lipgloss.Right).Inline(true).Width(c.width - 4 - len(instructions))
	footer := footerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, leftStyle.Render(instructions), rightStyle.Render(agentInfo)))
	sb.WriteString("\n" + footer)

	// Wrap everything in the outer border
	return outerStyle.Render(sb.String())
}

func sendMessageCmd(cs services.ChatService, chatID string, msg *entities.Message, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		_, err := cs.SendMessage(ctx, chatID, msg)
		if err != nil {
			return errMsg(err)
		}
		updatedChat, err := cs.GetChat(ctx, chatID)
		if err != nil {
			return errMsg(err)
		}
		return updatedChatMsg(updatedChat)
	}
}
