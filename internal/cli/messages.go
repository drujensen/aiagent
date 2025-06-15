package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// MessageType represents the type of message
type MessageType int

const (
	UserMessage MessageType = iota
	AssistantMessage
	ToolMessage
	ToolCallMessage
	SystemMessage
	ErrorMessage
)

// UIMessage represents a rendered message for display
type UIMessage struct {
	ID        string
	Type      MessageType
	Position  int
	Height    int
	Content   string
	Timestamp time.Time
}

// Color constants
var (
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#06B6D4") // Cyan
	systemColor    = lipgloss.Color("#10B981") // Green
	textColor      = lipgloss.Color("#FFFFFF") // White
	mutedColor     = lipgloss.Color("#6B7280") // Gray
	errorColor     = lipgloss.Color("#EF4444") // Red
	toolColor      = lipgloss.Color("#F59E0B") // Orange/Amber
)

// MessageRenderer handles rendering of messages with proper styling
type MessageRenderer struct {
	width int
	debug bool
}

// NewMessageRenderer creates a new message renderer
func NewMessageRenderer(width int, debug bool) *MessageRenderer {
	return &MessageRenderer{
		width: width,
		debug: debug,
	}
}

// SetWidth updates the renderer width
func (r *MessageRenderer) SetWidth(width int) {
	r.width = width
}

// RenderUserMessage renders a user message
func (r *MessageRenderer) RenderUserMessage(content string, timestamp time.Time) UIMessage {
	baseStyle := lipgloss.NewStyle()

	style := baseStyle.
		Width(r.width - 1).
		BorderLeft(true).
		Foreground(mutedColor).
		BorderForeground(secondaryColor).
		BorderStyle(lipgloss.ThickBorder()).
		PaddingLeft(1)

	timeStr := timestamp.Local().Format("02 Jan 2006 03:04 PM")
	username := "You"

	info := baseStyle.
		Width(r.width - 1).
		Foreground(mutedColor).
		Render(fmt.Sprintf(" %s (%s)", username, timeStr))

	messageContent := r.renderMarkdown(content, r.width-2)

	parts := []string{
		strings.TrimSuffix(messageContent, "\n"),
		info,
	}

	rendered := style.Render(
		lipgloss.JoinVertical(lipgloss.Left, parts...),
	)

	return UIMessage{
		Type:      UserMessage,
		Content:   rendered,
		Height:    lipgloss.Height(rendered),
		Timestamp: timestamp,
	}
}

// RenderAssistantMessage renders an assistant message
func (r *MessageRenderer) RenderAssistantMessage(content string, timestamp time.Time, modelName string) UIMessage {
	baseStyle := lipgloss.NewStyle()

	style := baseStyle.
		Width(r.width - 1).
		BorderLeft(true).
		Foreground(mutedColor).
		BorderForeground(primaryColor).
		BorderStyle(lipgloss.ThickBorder()).
		PaddingLeft(1)

	timeStr := timestamp.Local().Format("02 Jan 2006 03:04 PM")
	if modelName == "" {
		modelName = "Assistant"
	}

	info := baseStyle.
		Width(r.width - 1).
		Foreground(mutedColor).
		Render(fmt.Sprintf(" %s (%s)", modelName, timeStr))

	messageContent := r.renderMarkdown(content, r.width-2)

	if strings.TrimSpace(content) == "" {
		messageContent = baseStyle.
			Italic(true).
			Foreground(mutedColor).
			Render("*Finished without output*")
	}

	parts := []string{
		strings.TrimSuffix(messageContent, "\n"),
		info,
	}

	rendered := style.Render(
		lipgloss.JoinVertical(lipgloss.Left, parts...),
	)

	return UIMessage{
		Type:      AssistantMessage,
		Content:   rendered,
		Height:    lipgloss.Height(rendered),
		Timestamp: timestamp,
	}
}

// RenderSystemMessage renders a system message
func (r *MessageRenderer) RenderSystemMessage(content string, timestamp time.Time) UIMessage {
	baseStyle := lipgloss.NewStyle()

	style := baseStyle.
		Width(r.width - 1).
		BorderLeft(true).
		Foreground(mutedColor).
		BorderForeground(systemColor).
		BorderStyle(lipgloss.ThickBorder()).
		PaddingLeft(1)

	timeStr := timestamp.Local().Format("02 Jan 2006 03:04 PM")

	info := baseStyle.
		Width(r.width - 1).
		Foreground(mutedColor).
		Render(fmt.Sprintf(" AI Agent (%s)", timeStr))

	messageContent := r.renderMarkdown(content, r.width-2)

	if strings.TrimSpace(content) == "" {
		messageContent = baseStyle.
			Italic(true).
			Foreground(mutedColor).
			Render("*No content*")
	}

	parts := []string{
		strings.TrimSuffix(messageContent, "\n"),
		info,
	}

	rendered := style.Render(
		lipgloss.JoinVertical(lipgloss.Left, parts...),
	)

	return UIMessage{
		Type:      SystemMessage,
		Content:   rendered,
		Height:    lipgloss.Height(rendered),
		Timestamp: timestamp,
	}
}

// RenderErrorMessage renders an error message
func (r *MessageRenderer) RenderErrorMessage(errorMsg string, timestamp time.Time) UIMessage {
	baseStyle := lipgloss.NewStyle()

	style := baseStyle.
		Width(r.width - 1).
		BorderLeft(true).
		Foreground(mutedColor).
		BorderForeground(errorColor).
		BorderStyle(lipgloss.ThickBorder()).
		PaddingLeft(1)

	timeStr := timestamp.Local().Format("02 Jan 2006 03:04 PM")

	info := baseStyle.
		Width(r.width - 1).
		Foreground(mutedColor).
		Render(fmt.Sprintf(" Error (%s)", timeStr))

	errorContent := baseStyle.
		Foreground(errorColor).
		Bold(true).
		Render(fmt.Sprintf("❌ %s", errorMsg))

	parts := []string{
		errorContent,
		info,
	}

	rendered := style.Render(
		lipgloss.JoinVertical(lipgloss.Left, parts...),
	)

	return UIMessage{
		Type:      ErrorMessage,
		Content:   rendered,
		Height:    lipgloss.Height(rendered),
		Timestamp: timestamp,
	}
}

// RenderToolCallMessage renders a tool call in progress
func (r *MessageRenderer) RenderToolCallMessage(toolName, toolArgs string, timestamp time.Time) UIMessage {
	baseStyle := lipgloss.NewStyle()

	style := baseStyle.
		Width(r.width - 1).
		BorderLeft(true).
		Foreground(mutedColor).
		BorderForeground(toolColor).
		BorderStyle(lipgloss.ThickBorder()).
		PaddingLeft(1)

	timeStr := timestamp.Local().Format("02 Jan 2006 03:04 PM")

	toolIcon := "🔧"
	header := baseStyle.
		Foreground(toolColor).
		Bold(true).
		Render(fmt.Sprintf("%s Calling %s", toolIcon, toolName))

	var argsContent string
	if toolArgs != "" && toolArgs != "{}" {
		argsContent = baseStyle.
			Foreground(mutedColor).
			Render(fmt.Sprintf("Arguments: %s", r.formatToolArgs(toolArgs)))
	}

	info := baseStyle.
		Width(r.width - 1).
		Foreground(mutedColor).
		Render(fmt.Sprintf(" Tool Call (%s)", timeStr))

	parts := []string{header}
	if argsContent != "" {
		parts = append(parts, argsContent)
	}
	parts = append(parts, info)

	rendered := style.Render(
		lipgloss.JoinVertical(lipgloss.Left, parts...),
	)

	return UIMessage{
		Type:      ToolCallMessage,
		Content:   rendered,
		Height:    lipgloss.Height(rendered),
		Timestamp: timestamp,
	}
}

// RenderToolMessage renders a tool call result
func (r *MessageRenderer) RenderToolMessage(toolName, toolArgs, toolResult string, isError bool) UIMessage {
	baseStyle := lipgloss.NewStyle()

	style := baseStyle.
		Width(r.width - 1).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		PaddingLeft(1).
		BorderForeground(mutedColor)

	toolNameText := baseStyle.
		Foreground(mutedColor).
		Render(fmt.Sprintf("%s: ", toolName))

	argsText := baseStyle.
		Width(r.width - 2 - lipgloss.Width(toolNameText)).
		Foreground(mutedColor).
		Render(r.truncateText(toolArgs, r.width-2-lipgloss.Width(toolNameText)))

	var resultContent string
	if isError {
		resultContent = baseStyle.
			Width(r.width - 2).
			Foreground(errorColor).
			Render(fmt.Sprintf("Error: %s", toolResult))
	} else {
		resultContent = r.formatToolResult(toolName, toolResult, r.width-2)
	}

	headerLine := lipgloss.JoinHorizontal(lipgloss.Left, toolNameText, argsText)
	parts := []string{headerLine}

	if resultContent != "" {
		parts = append(parts, strings.TrimSuffix(resultContent, "\n"))
	}

	rendered := style.Render(
		lipgloss.JoinVertical(lipgloss.Left, parts...),
	)

	return UIMessage{
		Type:    ToolMessage,
		Content: rendered,
		Height:  lipgloss.Height(rendered),
	}
}

// formatToolArgs formats tool arguments for display
func (r *MessageRenderer) formatToolArgs(args string) string {
	args = strings.TrimSpace(args)
	if strings.HasPrefix(args, "{") && strings.HasSuffix(args, "}") {
		args = strings.TrimPrefix(args, "{")
		args = strings.TrimSuffix(args, "}")
		args = strings.TrimSpace(args)
	}

	if args == "" {
		return "(no arguments)"
	}

	if !r.debug {
		maxLen := 100
		if len(args) > maxLen {
			return args[:maxLen] + "..."
		}
	}

	return args
}

// formatToolResult formats tool results based on tool type
func (r *MessageRenderer) formatToolResult(toolName, result string, width int) string {
	baseStyle := lipgloss.NewStyle()

	if !r.debug {
		maxLines := 10
		lines := strings.Split(result, "\n")
		if len(lines) > maxLines {
			result = strings.Join(lines[:maxLines], "\n") + "\n... (truncated)"
		}
	}

	if strings.Contains(strings.ToLower(toolName), "bash") || strings.Contains(strings.ToLower(toolName), "command") {
		formatted := fmt.Sprintf("```bash\n%s\n```", result)
		return r.renderMarkdown(formatted, width)
	}

	return baseStyle.
		Width(width).
		Foreground(mutedColor).
		Render(result)
}

// truncateText truncates text to fit within the specified width
func (r *MessageRenderer) truncateText(text string, maxWidth int) string {
	if r.debug {
		return strings.ReplaceAll(text, "\n", " ")
	}

	text = strings.ReplaceAll(text, "\n", " ")

	if lipgloss.Width(text) <= maxWidth {
		return text
	}

	for i := len(text) - 1; i >= 0; i-- {
		truncated := text[:i] + "..."
		if lipgloss.Width(truncated) <= maxWidth {
			return truncated
		}
	}

	return "..."
}

// renderMarkdown renders markdown content using glamour
func (r *MessageRenderer) renderMarkdown(content string, width int) string {
	rendered := toMarkdown(content, width)
	return strings.TrimSuffix(rendered, "\n")
}

// MessageContainer wraps multiple messages in a container
type MessageContainer struct {
	messages []UIMessage
	width    int
	height   int
}

// NewMessageContainer creates a new message container
func NewMessageContainer(width, height int) *MessageContainer {
	return &MessageContainer{
		messages: make([]UIMessage, 0),
		width:    width,
		height:   height,
	}
}

// AddMessage adds a message to the container
func (c *MessageContainer) AddMessage(msg UIMessage) {
	c.messages = append(c.messages, msg)
}

// Clear clears all messages from the container
func (c *MessageContainer) Clear() {
	c.messages = make([]UIMessage, 0)
}

// SetSize updates the container size
func (c *MessageContainer) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// Render renders all messages in the container
func (c *MessageContainer) Render() string {
	if len(c.messages) == 0 {
		return c.renderEmptyState()
	}

	baseStyle := lipgloss.NewStyle()
	var parts []string

	for _, msg := range c.messages {
		parts = append(parts, msg.Content)
		parts = append(parts, baseStyle.Width(c.width).Render(""))
	}

	return baseStyle.
		Width(c.width).
		PaddingBottom(1).
		Render(
			lipgloss.JoinVertical(lipgloss.Top, parts...),
		)
}

// renderEmptyState renders the initial empty state
func (c *MessageContainer) renderEmptyState() string {
	baseStyle := lipgloss.NewStyle()

	header := baseStyle.
		Width(c.width).
		Align(lipgloss.Center).
		Foreground(systemColor).
		Bold(true).
		Render("AI Agent Console")

	subtitle := baseStyle.
		Width(c.width).
		Align(lipgloss.Center).
		Foreground(mutedColor).
		Render("Start a conversation by typing your message below")

	return baseStyle.
		Width(c.width).
		Height(c.height).
		PaddingBottom(1).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				"",
				header,
				"",
				subtitle,
				"",
			),
		)
}
