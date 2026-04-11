package tui

import (
	"context"
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
	"github.com/drujensen/aiagent/internal/tui/commands"
	"github.com/drujensen/aiagent/internal/tui/formatters"
	"github.com/kujtimiihoxha/vimtea"
	"go.uber.org/zap"
)

type refreshMsg struct{}

type ChatView struct {
	chatService        services.ChatService
	agentService       services.AgentService
	modelService       services.ModelService
	toolService        services.ToolService
	skillService       services.SkillService
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

func NewChatView(chatService services.ChatService, agentService services.AgentService, modelService services.ModelService, toolService services.ToolService, skillService services.SkillService, logger *zap.Logger, activeChat *entities.Chat) ChatView {
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
		toolService:        toolService,
		skillService:       skillService,
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
		select {
		case cv.eventChan <- data.Event:
		default:
			// Channel full, drop event to avoid blocking
		}
	})

	// Subscribe to process finished events
	processFinishedCancel := events.SubscribeToProcessFinishedEvents(func(data events.ProcessFinishedEventData) {
		select {
		case cv.eventChan <- data.Event:
		default:
			// Channel full, drop event to avoid blocking
		}
	})

	// Subscribe to process failed events
	processFailedCancel := events.SubscribeToProcessFailedEvents(func(data events.ProcessFailedEventData) {
		select {
		case cv.eventChan <- data.Event:
		default:
			// Channel full, drop event to avoid blocking
		}
	})

	// Subscribe to chat update events
	chatUpdateCancel := events.SubscribeToChatUpdateEvents(func(data events.ChatUpdateEventData) {
		select {
		case cv.eventChan <- data.Event:
		default:
			// Channel full, drop event to avoid blocking
		}
	})

	// Combine cancel functions
	cv.eventCancel = func() {
		toolCancel()
		processFinishedCancel()
		processFailedCancel()
		chatUpdateCancel()
	}

	return cv
}

func (c *ChatView) getToolByName(name string) entities.Tool {
	tools, err := c.toolService.ListTools()
	if err != nil {
		c.logger.Error("Failed to list tools", zap.Error(err))
		return nil
	}
	for _, tool := range tools {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
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
	if c.activeChat == nil || (len(c.activeChat.Messages) == 0 && len(c.tempMessages) == 0) {
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
		c.setEditorSize()
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
				tool := c.getToolByName(event.ToolName)
				var formattedResult, name, suffix string
				if tool != nil {
					formattedResult = tool.FormatResult("tui", event.Result, event.Diff, event.Arguments)
					name, suffix = tool.DisplayName("tui", event.Arguments)
				} else {
					formattedResult = event.Result
					name = event.ToolName
					suffix = ""
				}
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

	// Add temporary messages for real-time updates
	if len(c.tempMessages) > 0 {
		for _, tempMsg := range c.tempMessages {
			if tempMsg.Role == "user" {
				sb.WriteString("\n" + c.userStyle.Render("User: ") + tempMsg.Content + "\n\n")
			} else if tempMsg.Role == "tool" {
				sb.WriteString(c.systemStyle.Render("Tool: ") + "\n")
				// Display tool call events
				for _, event := range tempMsg.ToolCallEvents {
					tool := c.getToolByName(event.ToolName)
					var formattedResult, name, suffix string
					if tool != nil {
						formattedResult = tool.FormatResult("tui", event.Result, event.Diff, event.Arguments)
						name, suffix = tool.DisplayName("tui", event.Arguments)
					} else {
						formattedResult = event.Result
						name = event.ToolName
						suffix = ""
					}
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

	// Set editor size: textarea (dynamic) + footer (1) + separators (2) + header (1)
	if c.width > 0 && c.height > 0 {
		c.setEditorSize()
	}

	// Ensure editor maintains current focus state
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
		case *entities.ProcessFinishedEvent:
			return processFinishedEventMsg(e)
		case *entities.ProcessFailedEvent:
			return processFailedEventMsg(e)
		case *entities.ChatUpdateEvent:
			return chatUpdateEventMsg(e)
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
				mouseMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
				newModel, _ := c.editor.Update(mouseMsg)
				if editor, ok := newModel.(vimtea.Editor); ok {
					c.editor = editor
				}
			case tea.MouseWheelDown:
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
		case "ctrl+h":
			return c, func() tea.Msg { return startHistoryMsg{} }
		case "ctrl+n":
			return c, func() tea.Msg { return startAutoCreateChatMsg{} }
		case "ctrl+l":
			// Toggle line numbers
			c.lineNumbersEnabled = !c.lineNumbersEnabled
			c.updateEditorContent()
			return c, nil
		case "ctrl+s":
			return c, func() tea.Msg { return startSkillsMsg{} }
		case "ctrl+u":
			return c, func() tea.Msg { return startUsageMsg{} }
		case "ctrl+t":
			return c, func() tea.Msg { return startToolsMsg{} }
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
				message := entities.NewMessage("user", input)
				c.textarea.Reset()
				c.textarea.SetHeight(2)
				c.setEditorSize()
				c.tempMessages = append(c.tempMessages, *message)
				// Initialize tool call status tracking for this message
				c.toolCallStatus = make(map[string]bool)
				c.err = nil
				ctx, cancel := context.WithCancel(context.Background())
				c.cancel = cancel
				c.isProcessing = true
				c.startTime = time.Now()
				return c, tea.Batch(commands.SendMessageCmd(c.chatService, c.activeChat.ID, message, ctx), c.spinner.Tick, tea.Cmd(func() tea.Msg { return refreshMsg{} }))
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
					c.setEditorSize()
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

	case startSkillExecutionMsg:
		// Get skill content synchronously (fast operation)
		ctx := context.Background()
		content, err := c.skillService.GetSkillContent(ctx, m.skillName)
		if err != nil {
			c.err = err
			return c, nil
		}

		// Create and add message locally (same as user input)
		message := entities.NewMessage("user", content)
		c.tempMessages = append(c.tempMessages, *message)

		// Initialize processing state
		c.toolCallStatus = make(map[string]bool)
		c.updateEditorContent()

		// Scroll to bottom to show new message
		bottomMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		newModel, _ := c.editor.Update(bottomMsg)
		if editor, ok := newModel.(vimtea.Editor); ok {
			c.editor = editor
		}

		// Start async processing
		c.err = nil
		ctx, cancel := context.WithCancel(context.Background())
		c.cancel = cancel
		c.isProcessing = true
		c.startTime = time.Now()
		return c, tea.Batch(commands.SendMessageCmd(c.chatService, c.activeChat.ID, message, ctx), c.spinner.Tick)

	case toolCallEventMsg:
		// Handle real-time tool call event — skip events belonging to a
		// different chat (e.g. a sub-agent launched by the Agent tool).
		if m.ChatID != "" && c.activeChat != nil && m.ChatID != c.activeChat.ID {
			return c, c.listenForEvents()
		}
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

	case processFinishedEventMsg:
		// Handle process finished event
		if c.isProcessing && c.activeChat != nil && m.ChatID == c.activeChat.ID {
			c.isProcessing = false
			c.tempMessages = nil
			c.toolCallStatus = make(map[string]bool)

			// Fetch updated chat with all messages including AI responses
			ctx := context.Background()
			updatedChat, err := c.chatService.GetChat(ctx, m.ChatID)
			if err != nil {
				c.err = err
			} else {
				c.activeChat = updatedChat
			}

			c.updateEditorContent()
		}
		return c, c.listenForEvents()

	case processFailedEventMsg:
		// Handle process failed event
		if c.isProcessing && c.activeChat != nil && m.ChatID == c.activeChat.ID {
			c.isProcessing = false
			c.tempMessages = nil
			c.toolCallStatus = make(map[string]bool)

			// Fetch updated chat with any partially saved messages
			ctx := context.Background()
			updatedChat, err := c.chatService.GetChat(ctx, m.ChatID)
			if err != nil {
				c.err = err
			} else {
				c.activeChat = updatedChat
			}

			// Add system error message
			errorMsg := &entities.Message{
				Content: fmt.Sprintf("System Error: %s", m.Error),
				Role:    "system",
			}
			c.activeChat.Messages = append(c.activeChat.Messages, *errorMsg)
			c.updateEditorContent()
		}
		return c, c.listenForEvents()

	case chatUpdateEventMsg:
		// Handle chat update event (e.g., usage updates)
		if c.activeChat != nil && m.ChatID == c.activeChat.ID {
			if m.UpdateType == "usage" {
				// Usage has been updated in the service, refresh display
				// Since activeChat is a pointer to the same object, it should be updated
				// But to be safe, we can fetch the latest chat
				ctx := context.Background()
				updatedChat, err := c.chatService.GetChat(ctx, m.ChatID)
				if err == nil {
					c.activeChat = updatedChat
				}
			}
		}
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

	case refreshMsg:
		c.updateEditorContent()
		// Scroll to bottom to show the new user message
		bottomMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		newModel, _ := c.editor.Update(bottomMsg)
		if editor, ok := newModel.(vimtea.Editor); ok {
			c.editor = editor
		}
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
			c.updateEditorContent()
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
		c.setEditorSize()

		c.textarea.SetWidth(c.width)

		if c.activeChat != nil {
			c.updateEditorContent()
		}
		return c, nil
	}

	return c, tea.Batch(cmds...)
}

// setEditorSize calculates and sets the editor size based on available screen space
func (c *ChatView) setEditorSize() {
	// Set editor size to fit screen minus textarea, footer, separators, and header
	if c.width > 0 && c.height > 0 {
		editorWidth := c.width
		editorHeight := c.height - (c.textarea.Height() + 1 + 2 + 2) // textarea (dynamic) + footer (1) + separators (2) + header (1)
		if editorHeight < 1 {
			editorHeight = 1
		}
		c.editor.SetSize(editorWidth, editorHeight)
	}
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
		// Use accumulated totals for display
		tokenCountStr := formatters.FormatTokenCount(c.activeChat.Usage.TotalTokens)
		tokenInfo = fmt.Sprintf("Total Tokens: %s ($%.2f)", tokenCountStr, c.activeChat.Usage.TotalCost)

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
		modelInfo = fmt.Sprintf("Model: %s - %s", c.currentModel.ProviderType, c.currentModel.Name)
	}

	footerInfo := agentInfo + " | " + modelInfo

	footerStyle := lipgloss.NewStyle().Width(c.width)
	leftStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Inline(true)
	rightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Align(lipgloss.Right).Inline(true).Width(c.width - lipgloss.Width(instructions))
	footerPart := footerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, leftStyle.Render(instructions), rightStyle.Render(footerInfo)))

	return lipgloss.JoinVertical(lipgloss.Top, editorPart, header, separator, textareaPart, separator, footerPart)
}
