package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

// formatToolResult parses and formats tool execution results for display
func formatToolResult(toolName, result string, diff string) string {
	switch toolName {
	case "FileWrite":
		return formatFileWriteResult(result, diff)
	case "FileRead":
		return formatFileReadResult(result)
	case "Directory":
		return formatDirectoryResult(result)
	case "Process":
		return formatProcessResult(result)
	case "Project":
		return formatProjectResult(result)
	case "FileSearch":
		return formatFileSearchResult(result)
	case "Memory":
		return formatMemoryResult(result)
	default:
		// For other tools, try to parse as JSON and display key fields
		return formatGenericResult(result)
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
		Diff      string `json:"diff"`
		LineCount int    `json:"line_count"`
	}

	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		return result // Return raw if parsing fails
	}

	var output strings.Builder

	if resultData.LineCount > 0 {
		output.WriteString(fmt.Sprintf("File updated (%d lines)", resultData.LineCount))
	} else {
		output.WriteString("No changes made to file")
	}

	if diff != "" {
		output.WriteString("\n\n" + formatDiff(diff))
	} else if resultData.Diff != "" {
		output.WriteString("\n\n" + formatDiff(resultData.Diff))
	}

	return output.String()
}

// formatFileSearchResult formats FileSearch tool results
func formatFileSearchResult(result string) string {
	var resultData struct {
		FileResponse struct {
			Results interface{} `json:"results"` // Can be []LineResult or map[string][]LineResult
		} `json:"File_response"`
	}

	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		return result
	}

	var output strings.Builder

	// Handle single file results
	if singleResults, ok := resultData.FileResponse.Results.([]interface{}); ok {
		output.WriteString(fmt.Sprintf("Search Results (%d matches):\n", len(singleResults)))

		// Show first 10 matches
		maxMatches := 10
		for i, match := range singleResults {
			if i >= maxMatches {
				break
			}

			if matchMap, ok := match.(map[string]interface{}); ok {
				line := int(matchMap["line"].(float64))
				text := matchMap["text"].(string)
				output.WriteString(fmt.Sprintf("  %4d: %s\n", line, text))
			}
		}

		if len(singleResults) > maxMatches {
			output.WriteString(fmt.Sprintf("  ... and %d more matches\n", len(singleResults)-maxMatches))
		}

	} else if multiResults, ok := resultData.FileResponse.Results.(map[string]interface{}); ok {
		// Handle multi-file results
		totalMatches := 0
		fileCount := 0

		output.WriteString("Multi-file Search Results:\n")

		// Show first 5 files with their matches
		maxFiles := 5
		for filePath, matches := range multiResults {
			if fileCount >= maxFiles {
				break
			}

			if matchArray, ok := matches.([]interface{}); ok {
				if len(matchArray) > 0 {
					output.WriteString(fmt.Sprintf("📄 %s (%d matches):\n", filePath, len(matchArray)))
					totalMatches += len(matchArray)

					// Show first 5 matches per file
					maxMatchesPerFile := 5
					for i, match := range matchArray {
						if i >= maxMatchesPerFile {
							break
						}

						if matchMap, ok := match.(map[string]interface{}); ok {
							line := int(matchMap["line"].(float64))
							text := matchMap["text"].(string)
							output.WriteString(fmt.Sprintf("    %4d: %s\n", line, text))
						}
					}

					if len(matchArray) > maxMatchesPerFile {
						output.WriteString(fmt.Sprintf("    ... and %d more matches\n", len(matchArray)-maxMatchesPerFile))
					}

					fileCount++
				}
			}
		}

		if len(multiResults) > maxFiles {
			output.WriteString(fmt.Sprintf("... and %d more files with matches\n", len(multiResults)-maxFiles))
		}

		output.WriteString(fmt.Sprintf("\nTotal matches: %d across %d files", totalMatches, len(multiResults)))
	}

	return output.String()
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
	if !strings.Contains(diff, "```diff") {
		return diff
	}

	// Extract diff content from markdown code block
	start := strings.Index(diff, "```diff\n")
	if start == -1 {
		return diff
	}
	start += 8 // Length of "```diff\n"

	end := strings.Index(diff[start:], "\n```")
	if end == -1 {
		return diff[start:]
	}

	diffContent := diff[start : start+end]

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

// formatFileReadResult formats FileRead tool results
func formatFileReadResult(result string) string {
	var lines []struct {
		Line int    `json:"line"`
		Text string `json:"text"`
	}

	if err := json.Unmarshal([]byte(result), &lines); err != nil {
		return result
	}

	if len(lines) == 0 {
		return "No content found"
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Read %d lines:\n", len(lines)))

	// Show first 8 lines with line numbers
	maxLines := 8
	if len(lines) > maxLines {
		for i := 0; i < maxLines; i++ {
			output.WriteString(fmt.Sprintf("%4d: %s\n", lines[i].Line, lines[i].Text))
		}
		output.WriteString(fmt.Sprintf("... and %d more lines", len(lines)-maxLines))
	} else {
		for _, line := range lines {
			output.WriteString(fmt.Sprintf("%4d: %s\n", line.Line, line.Text))
		}
	}

	return output.String()
}

// formatDirectoryResult formats Directory tool results
func formatDirectoryResult(result string) string {
	var resultData struct {
		Path    string   `json:"path"`
		Entries []string `json:"entries"`
	}

	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		return result
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Directory listing: %s\n", resultData.Path))

	// Show first 8 entries
	maxEntries := 8
	if len(resultData.Entries) > maxEntries {
		for i := 0; i < maxEntries; i++ {
			output.WriteString(fmt.Sprintf("  %s\n", resultData.Entries[i]))
		}
		output.WriteString(fmt.Sprintf("  ... and %d more entries", len(resultData.Entries)-maxEntries))
	} else {
		for _, entry := range resultData.Entries {
			output.WriteString(fmt.Sprintf("  %s\n", entry))
		}
	}

	return output.String()
}

// formatProcessResult formats Process tool results
func formatProcessResult(result string) string {
	var resultData struct {
		Command string `json:"command"`
		Stdout  string `json:"stdout"`
		Stderr  string `json:"stderr"`
		Status  string `json:"status"`
	}

	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		return result
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Executed: %s\n", resultData.Command))

	if resultData.Stdout != "" {
		outputLines := strings.Split(strings.TrimSuffix(resultData.Stdout, "\n"), "\n")
		output.WriteString(fmt.Sprintf("\nOutput (%d lines):\n", len(outputLines)))

		// Show first 8 lines of output
		maxLines := 8
		if len(outputLines) > maxLines {
			for i := 0; i < maxLines; i++ {
				output.WriteString(fmt.Sprintf("  %s\n", outputLines[i]))
			}
			output.WriteString(fmt.Sprintf("  ... and %d more lines", len(outputLines)-maxLines))
		} else {
			for _, line := range outputLines {
				output.WriteString(fmt.Sprintf("  %s\n", line))
			}
		}
	}

	if resultData.Stderr != "" {
		errorLines := strings.Split(strings.TrimSuffix(resultData.Stderr, "\n"), "\n")
		output.WriteString(fmt.Sprintf("\nError (%d lines):\n", len(errorLines)))

		// Show first 8 lines of error
		maxLines := 8
		if len(errorLines) > maxLines {
			for i := 0; i < maxLines; i++ {
				output.WriteString(fmt.Sprintf("  %s\n", errorLines[i]))
			}
			output.WriteString(fmt.Sprintf("  ... and %d more lines", len(errorLines)-maxLines))
		} else {
			for _, line := range errorLines {
				output.WriteString(fmt.Sprintf("  %s\n", line))
			}
		}
	}

	if resultData.Status != "" && resultData.Status != "completed" {
		output.WriteString(fmt.Sprintf("\nStatus: %s", resultData.Status))
	}

	return output.String()
}

// formatProjectResult formats Project tool results (get_source)
func formatProjectResult(result string) string {
	var resultData struct {
		FileMap      string            `json:"file_map"`
		FileContents map[string]string `json:"file_contents"`
	}

	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		return result
	}

	var output strings.Builder

	// Show file map (directory tree) - limit to reasonable size
	if resultData.FileMap != "" {
		output.WriteString("Project Structure:\n")
		lines := strings.Split(strings.TrimSuffix(resultData.FileMap, "\n"), "\n")

		// Show first 15 lines of directory tree
		maxLines := 15
		if len(lines) > maxLines {
			for i := 0; i < maxLines; i++ {
				output.WriteString(fmt.Sprintf("  %s\n", lines[i]))
			}
			output.WriteString(fmt.Sprintf("  ... and %d more directories/files\n", len(lines)-maxLines))
		} else {
			for _, line := range lines {
				output.WriteString(fmt.Sprintf("  %s\n", line))
			}
		}
		output.WriteString("\n")
	}

	// Show file contents or structure
	if len(resultData.FileContents) > 0 {
		output.WriteString(fmt.Sprintf("Source Files (%d files):\n", len(resultData.FileContents)))

		// Show first 5 files with limited content
		maxFiles := 5
		fileCount := 0
		for path, content := range resultData.FileContents {
			if fileCount >= maxFiles {
				break
			}

			output.WriteString(fmt.Sprintf("📄 %s\n", path))

			// Show first 10 lines of each file
			lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
			maxFileLines := 10
			if len(lines) > maxFileLines {
				for i := 0; i < maxFileLines; i++ {
					output.WriteString(fmt.Sprintf("    %d: %s\n", i+1, lines[i]))
				}
				output.WriteString(fmt.Sprintf("    ... and %d more lines\n", len(lines)-maxFileLines))
			} else {
				for i, line := range lines {
					output.WriteString(fmt.Sprintf("    %d: %s\n", i+1, line))
				}
			}
			output.WriteString("\n")
			fileCount++
		}

		if len(resultData.FileContents) > maxFiles {
			output.WriteString(fmt.Sprintf("... and %d more files\n", len(resultData.FileContents)-maxFiles))
		}
	}

	return output.String()
}

// formatGenericResult tries to parse generic JSON results
func formatGenericResult(result string) string {
	var jsonData map[string]interface{}
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
	chatService  services.ChatService
	agentService services.AgentService
	activeChat   *entities.Chat
	viewport     viewport.Model
	textarea     textarea.Model
	spinner      spinner.Model
	userStyle    lipgloss.Style
	asstStyle    lipgloss.Style
	systemStyle  lipgloss.Style
	err          error
	cancel       context.CancelFunc
	isProcessing bool
	startTime    time.Time
	focused      string // "textarea" or "viewport"
	width        int
	height       int
	currentAgent *entities.Agent
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

	vp := viewport.New(30, 5)

	us := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	as := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	ss := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	cv := ChatView{
		chatService:  chatService,
		agentService: agentService,
		activeChat:   activeChat,
		textarea:     ta,
		viewport:     vp,
		spinner:      s,
		userStyle:    us,
		asstStyle:    as,
		systemStyle:  ss,
		err:          nil,
		focused:      "textarea",
		width:        30,
		height:       5,
	}

	if activeChat != nil {
		cv.SetActiveChat(activeChat)
	}

	return cv
}

func (c *ChatView) SetActiveChat(chat *entities.Chat) {
	c.activeChat = chat
	ctx := context.Background()
	agent, err := c.agentService.GetAgent(ctx, chat.AgentID)
	if err != nil {
		c.err = err
		c.currentAgent = nil
	} else {
		c.currentAgent = agent
	}
	var sb strings.Builder
	for _, message := range chat.Messages {
		if message.Role == "user" {
			sb.WriteString(c.userStyle.Render("User: ") + message.Content + "\n")
		} else if message.Role == "assistant" {
			sb.WriteString(c.asstStyle.Render("Assistant: ") + message.Content + "\n")
		} else if message.Role == "tool" {
			sb.WriteString(c.systemStyle.Render("Tool: ") + message.Content + "\n")
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
	if len(chat.Messages) > 0 && chat.Messages[len(chat.Messages)-1].Role != "system" {
		sb.WriteString(c.systemStyle.Render("System: Switched to new agent\n"))
	}

	// Add current tool calls being executed if processing
	if c.isProcessing && len(chat.Messages) > 0 {
		lastMsg := chat.Messages[len(chat.Messages)-1]
		if lastMsg.Role == "assistant" && len(lastMsg.ToolCalls) > 0 {
			sb.WriteString("\n" + c.systemStyle.Render("Executing tools:") + "\n")
			for _, toolCall := range lastMsg.ToolCalls {
				sb.WriteString(c.systemStyle.Render("  ↳ ") + "🔄 " + toolCall.Function.Name + "\n")
			}
		}
	}

	c.viewport.SetContent(lipgloss.NewStyle().Width(c.viewport.Width).Render(sb.String()))
	c.viewport.GotoBottom()
}

func (c ChatView) Init() tea.Cmd {
	c.textarea.Focus()
	c.focused = "textarea"
	return textarea.Blink
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
					c.viewport.GotoBottom()
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
				c.SetActiveChat(c.activeChat)
				c.err = nil
				ctx, cancel := context.WithCancel(context.Background())
				c.cancel = cancel
				c.isProcessing = true
				c.startTime = time.Now()
				return c, tea.Batch(sendMessageCmd(c.chatService, c.activeChat.ID, message, ctx), c.spinner.Tick)
			}
		case "tab", "shift+tab":
			if c.focused == "textarea" {
				c.focused = "viewport"
				c.textarea.Blur()
			} else {
				c.focused = "textarea"
				c.textarea.Focus()
				cmd := textarea.Blink
				cmds = append(cmds, cmd)
			}
			return c, tea.Batch(cmds...)
		case "j", "down":
			if c.focused == "viewport" {
				c.viewport.ScrollDown(1)
			} else {
				var cmd tea.Cmd
				c.textarea, cmd = c.textarea.Update(m)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		case "k", "up":
			if c.focused == "viewport" {
				c.viewport.ScrollUp(1)
			} else {
				var cmd tea.Cmd
				c.textarea, cmd = c.textarea.Update(m)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		default:
			if c.focused == "textarea" {
				var cmd tea.Cmd
				c.textarea, cmd = c.textarea.Update(m)
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

	case updatedChatMsg:
		c.textarea.Reset()
		c.SetActiveChat(m)
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
		c.SetActiveChat(c.activeChat)
		return c, nil

	case tea.WindowSizeMsg:
		c.width = m.Width
		c.height = m.Height
		innerWidth := c.width - 4
		innerHeight := c.height - 4

		c.viewport.Width = innerWidth
		// Subtract textarea height (2), instructions (1), and adjust for borders
		c.viewport.Height = innerHeight - 2 - 1 - 2
		if c.viewport.Height < 1 {
			c.viewport.Height = 1
		}

		c.textarea.SetWidth(innerWidth)

		if c.activeChat != nil {
			var sb strings.Builder
			for _, message := range c.activeChat.Messages {
				if message.Role == "user" {
					sb.WriteString(c.userStyle.Render("User: ") + message.Content + "\n")
				} else if message.Role == "assistant" {
					sb.WriteString(c.asstStyle.Render("Assistant: ") + message.Content + "\n")
				} else if message.Role == "tool" {
					sb.WriteString(c.systemStyle.Render("Tool: ") + message.Content + "\n")
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
			}
			c.viewport.SetContent(lipgloss.NewStyle().Width(c.viewport.Width).Render(sb.String()))
		}
		c.viewport.GotoBottom()
		return c, nil
	case tea.MouseMsg:
		viewportYStart := 1
		viewportBlockHeight := c.viewport.Height + 2
		viewportYEnd := viewportYStart + viewportBlockHeight
		if m.Y >= viewportYStart && m.Y < viewportYEnd {
			switch m.Type {
			case tea.MouseWheelUp:
				c.viewport.LineUp(2)
			case tea.MouseWheelDown:
				c.viewport.LineDown(2)
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

	// Style viewport
	vpStyle := unfocusedBorder.Width(c.width - 4).Height(c.viewport.Height)
	if c.focused == "viewport" {
		vpStyle = focusedBorder.Width(c.width - 4).Height(c.viewport.Height)
	}

	var content strings.Builder

	// Check if activeChat is nil OR has no messages
	if c.activeChat == nil || len(c.activeChat.Messages) == 0 {
		content.WriteString("How can I help you today?")
	} else {
		// Display chat messages
		for _, message := range c.activeChat.Messages {
			if message.Role == "user" {
				content.WriteString(c.userStyle.Render("User: ") + message.Content + "\n")
			} else if message.Role == "assistant" {
				content.WriteString(c.asstStyle.Render("Assistant: ") + message.Content + "\n")
			} else if message.Role == "tool" {
				content.WriteString(c.systemStyle.Render("Tool: ") + message.Content + "\n")
				// Display tool call events
				for _, event := range message.ToolCallEvents {
					formattedResult := formatToolResult(event.ToolName, event.Result, event.Diff)
					statusIcon := getToolStatusIcon(event.Error != "")
					content.WriteString(c.systemStyle.Render("  ↳ ") + statusIcon + " " + event.ToolName + ":\n")
					content.WriteString(c.systemStyle.Render("    ") + strings.ReplaceAll(formattedResult, "\n", "\n    ") + "\n")
					if event.Error != "" {
						errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true) // Red and bold
						content.WriteString(errorStyle.Render("    ✗ Error: ") + event.Error + "\n")
					}
				}
			} else {
				content.WriteString(c.systemStyle.Render("System: ") + message.Content + "\n")
			}
		}
	}

	// Add error as temporary system message if present
	if c.err != nil {
		if content.Len() > 0 {
			content.WriteString("\n")
		}
		content.WriteString(c.systemStyle.Render("System: Error - ") + c.err.Error() + "\n")
		c.err = nil // Clear error after displaying
	}

	// Add current tool calls being executed if processing
	if c.isProcessing && c.activeChat != nil && len(c.activeChat.Messages) > 0 {
		lastMsg := c.activeChat.Messages[len(c.activeChat.Messages)-1]
		if lastMsg.Role == "assistant" && len(lastMsg.ToolCalls) > 0 {
			content.WriteString("\n" + c.systemStyle.Render("Executing tools:") + "\n")
			for _, toolCall := range lastMsg.ToolCalls {
				content.WriteString(c.systemStyle.Render("  ↳ ") + "🔄 " + toolCall.Function.Name + "\n")
			}
		}
	}

	// Set the viewport content
	c.viewport.SetContent(lipgloss.NewStyle().Width(c.viewport.Width).Render(content.String()))

	sb.WriteString(vpStyle.Render(c.viewport.View()))

	// Style textarea
	taStyle := unfocusedBorder.Width(c.width - 4).Height(c.textarea.Height())
	if c.focused == "textarea" {
		taStyle = focusedBorder.Width(c.width - 4).Height(c.textarea.Height())
	}
	sb.WriteString(taStyle.Render(c.textarea.View()))

	if c.isProcessing {
		elapsed := time.Since(c.startTime).Round(time.Second)
		sb.WriteString("\n" + c.spinner.View() + fmt.Sprintf(" Working... (%ds) esc to interrupt", int(elapsed.Seconds())))
	} else {
		instructions := "Press Ctrl+P for menu, Tab to switch focus, j/k to navigate, Ctrl+C to exit."
		var agentInfo string
		if c.currentAgent != nil {
			agentInfo = fmt.Sprintf("%s (%s: %s)", c.currentAgent.Name, c.currentAgent.ProviderType, c.currentAgent.Model)
		} else {
			agentInfo = "No agent selected"
		}
		footerStyle := lipgloss.NewStyle().Width(c.width - 4)
		leftStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Inline(true)
		rightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Align(lipgloss.Right).Inline(true).Width(c.width - 4 - len(instructions))
		footer := footerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, leftStyle.Render(instructions), rightStyle.Render(agentInfo)))
		sb.WriteString("\n" + footer)
	}

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
