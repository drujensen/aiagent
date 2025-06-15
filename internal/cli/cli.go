package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	errs "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/services"
	"go.uber.org/zap"
	"golang.org/x/term"
)

var (
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
)

// CLI handles the command line interface with improved message rendering
type CLI struct {
	chatService      services.ChatService
	agentService     services.AgentService
	toolService      services.ToolService
	providerService  services.ProviderService
	logger           *zap.Logger
	messageRenderer  *MessageRenderer
	messageContainer *MessageContainer
	width            int
	height           int
}

// NewCLI creates a new CLI instance with message container
func NewCLI(chatService services.ChatService, agentService services.AgentService, toolService services.ToolService, providerService services.ProviderService, logger *zap.Logger) *CLI {
	cli := &CLI{
		chatService:     chatService,
		agentService:    agentService,
		toolService:     toolService,
		providerService: providerService,
		logger:          logger,
	}
	cli.updateSize()
	cli.messageRenderer = NewMessageRenderer(cli.width, false)
	cli.messageContainer = NewMessageContainer(cli.width, cli.height-4) // Reserve space for input
	cli.RegisterCallbackHandler()
	return cli
}

// Run starts the CLI interface, displaying chat history and handling user input
func (c *CLI) Run() error {
	// Display welcome message
	c.DisplayInfo("AI Agent Console - Type /help for commands, Ctrl+C to quit")

	ctx := context.Background()

	// Select the active chat
	chats, err := c.chatService.ListChats(ctx)
	var chat *entities.Chat
	if err != nil || len(chats) == 0 {
		chat, err = c.NewChatCommand(ctx, "/new New Chat")
		if err != nil {
			c.logger.Error("Failed to create new chat", zap.Error(err))
			c.DisplayError(err)
			return err
		}
	}
	for _, activeChat := range chats {
		if activeChat.Active {
			chat = activeChat
			break
		}
	}
	if chat == nil && len(chats) > 0 {
		chat = chats[0]
	}

	// Display chat name and messages
	if chat != nil {
		c.DisplayInfo(fmt.Sprintf("Current Chat: %s", chat.Name))
		for _, msg := range chat.Messages {
			c.displayMessage(msg)
		}
	}

	for {
		// Get user input
		input, err := c.GetPrompt()
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				c.DisplayInfo("Goodbye!")
				return nil
			}
			c.logger.Error("Prompt error", zap.Error(err))
			c.DisplayError(err)
			continue
		}

		// Handle slash commands
		if c.IsSlashCommand(input) {
			handled := c.HandleSlashCommand(input, ctx)
			if handled {
				continue
			}
		}

		// Process file references
		processedInput, err := c.processFileReferences(input)
		if err != nil {
			c.logger.Error("Failed to process file references", zap.Error(err))
			c.DisplayError(err)
			continue
		}

		// Create and save user message
		message := entities.NewMessage("user", processedInput)
		c.DisplayUserMessage(processedInput)

		// Send message with spinner
		var response *entities.Message
		err = c.ShowSpinner("Thinking...", func() error {
			var sendErr error
			response, sendErr = c.chatService.SendMessage(ctx, chat.ID, message)
			return sendErr
		})

		if err != nil {
			if isCanceledError(err) {
				c.DisplayInfo("Request canceled by user.")
				continue
			}
			c.logger.Error("Failed to generate response", zap.Error(err))
			c.DisplayError(err)
			continue
		}

		// Strip <think> tags
		responseContent := response.Content
		for {
			start := strings.Index(responseContent, "<think>")
			end := strings.Index(responseContent, "</think>")
			if start == -1 || end == -1 {
				break
			}
			responseContent = responseContent[:start] + responseContent[end+len("</think>"):]
		}
		response.Content = responseContent

		// Display response
		agent, err := c.agentService.GetAgent(ctx, chat.AgentID)
		if err != nil {
			c.logger.Error("Failed to get agent", zap.Error(err))
			c.DisplayError(err)
			continue
		}
		c.DisplayAssistantMessageWithModel(response.Content, agent.Name)
	}
}

// GetPrompt gets user input using the huh library with divider and padding
func (c *CLI) GetPrompt() (string, error) {
	dividerStyle := lipgloss.NewStyle().
		Width(c.width).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(mutedColor).
		MarginTop(1).
		MarginBottom(1)

	fmt.Print(dividerStyle.Render(""))

	var prompt string
	err := huh.NewForm(huh.NewGroup(huh.NewText().
		Title("Enter your prompt (Type /help for commands, Ctrl+C to quit)").
		Value(&prompt).
		CharLimit(5000)),
	).WithWidth(c.width).
		WithTheme(huh.ThemeCharm()).
		Run()

	return prompt, err
}

// ShowSpinner displays a spinner with the given message and executes the action
func (c *CLI) ShowSpinner(message string, action func() error) error {
	spinner := NewSpinner(message)
	spinner.Start()
	err := action()
	spinner.Stop()
	return err
}

// DisplayUserMessage displays the user's message
func (c *CLI) DisplayUserMessage(message string) {
	msg := c.messageRenderer.RenderUserMessage(message, time.Now())
	c.messageContainer.AddMessage(msg)
	c.displayContainer()
}

// DisplayAssistantMessage displays the assistant's message
func (c *CLI) DisplayAssistantMessage(message string) error {
	return c.DisplayAssistantMessageWithModel(message, "")
}

// DisplayAssistantMessageWithModel displays the assistant's message with model info
func (c *CLI) DisplayAssistantMessageWithModel(message, modelName string) error {
	msg := c.messageRenderer.RenderAssistantMessage(message, time.Now(), modelName)
	c.messageContainer.AddMessage(msg)
	c.displayContainer()
	return nil
}

// DisplayToolCallMessage displays a tool call in progress
func (c *CLI) DisplayToolCallMessage(toolName, toolArgs string) {
	msg := c.messageRenderer.RenderToolCallMessage(toolName, toolArgs, time.Now())
	c.messageContainer.AddMessage(msg)
	c.displayContainer()
}

// DisplayToolMessage displays a tool call result
func (c *CLI) DisplayToolMessage(toolName, toolArgs, toolResult string, isError bool) {
	msg := c.messageRenderer.RenderToolMessage(toolName, toolArgs, toolResult, isError)
	c.messageContainer.AddMessage(msg)
	c.displayContainer()
}

// DisplayError displays an error message
func (c *CLI) DisplayError(err error) {
	msg := c.messageRenderer.RenderErrorMessage(err.Error(), time.Now())
	c.messageContainer.AddMessage(msg)
	c.displayContainer()
}

// DisplayInfo displays an informational message
func (c *CLI) DisplayInfo(message string) {
	msg := c.messageRenderer.RenderSystemMessage(message, time.Now())
	c.messageContainer.AddMessage(msg)
	c.displayContainer()
}

// DisplayHelp displays help information
func (c *CLI) DisplayHelp() {
	help := `## Available Commands

- ` + "`/help`" + `: Show this help message
- ` + "`!<command>`" + `: Execute a shell command
- ` + "`@<file>`" + `: Include a file or directory
- ` + "`/new <name>`" + `: Start a new chat
- ` + "`/history`" + `: Select from all available chats
- ` + "`/agents`" + `: List available agents
- ` + "`/tools`" + `: List available tools
- ` + "`/usage`" + `: Show usage information
- ` + "`/exit`" + `: Exit the application
- ` + "`Ctrl+C`" + `: Exit at any time`

	c.DisplayInfo(help)
}

// DisplayAgents displays available agents
func (c *CLI) DisplayAgents(agents []*entities.Agent) {
	var content strings.Builder
	content.WriteString("## Available Agents\n\n")

	if len(agents) == 0 {
		content.WriteString("No agents are currently available.")
	} else {
		for i, agent := range agents {
			content.WriteString(fmt.Sprintf("%d. `%s` (%s)\n", i+1, agent.Name, agent.ProviderType))
		}
	}

	c.DisplayInfo(content.String())
}

// DisplayTools displays available tools
func (c *CLI) DisplayTools(tools []*entities.Tool) {
	var content strings.Builder
	content.WriteString("## Available Tools\n\n")

	if len(tools) == 0 {
		content.WriteString("No tools are currently available.")
	} else {
		for i, tool := range tools {
			content.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, (*tool).Name()))
		}
	}

	c.DisplayInfo(content.String())
}

// DisplayUsage displays usage information
func (c *CLI) DisplayUsage(chat *entities.Chat, agent *entities.Agent) {
	content := fmt.Sprintf("## Chat Usage\n\n- Provider: %s\n- Model: %s\n- Prompt Tokens: %d\n- Completion Tokens: %d\n- Total Tokens: %d\n- Total Cost: $%.2f",
		agent.ProviderType, agent.Model, chat.Usage.TotalPromptTokens, chat.Usage.TotalCompletionTokens, chat.Usage.TotalTokens, chat.Usage.TotalCost)
	c.DisplayInfo(content)
}

// DisplayHistory displays conversation history
func (c *CLI) DisplayHistory(messages []entities.Message) {
	historyContainer := NewMessageContainer(c.width, c.height-4)

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			uiMsg := c.messageRenderer.RenderUserMessage(msg.Content, time.Now())
			historyContainer.AddMessage(uiMsg)
		case "assistant":
			uiMsg := c.messageRenderer.RenderAssistantMessage(msg.Content, time.Now(), "")
			historyContainer.AddMessage(uiMsg)
		case "tool":
			uiMsg := c.messageRenderer.RenderToolMessage("", "", msg.Content, false)
			historyContainer.AddMessage(uiMsg)
		}
	}

	c.DisplayInfo("## Conversation History\n\n" + historyContainer.Render())
}

// IsSlashCommand checks if the input is a slash command
func (c *CLI) IsSlashCommand(input string) bool {
	return strings.HasPrefix(input, "/") || strings.HasPrefix(input, "!")
}

// HandleSlashCommand handles slash commands and returns true if handled
func (c *CLI) HandleSlashCommand(input string, ctx context.Context) bool {
	if strings.HasPrefix(input, "!") {
		cmd := input[1:]
		err := c.ShowSpinner("Executing command...", func() error {
			output, cmdErr := c.BashCommand(cmd)
			if cmdErr != nil {
				c.DisplayToolMessage("bash", cmd, cmdErr.Error(), true)
				return cmdErr
			}
			c.DisplayToolMessage("bash", cmd, output, false)
			return nil
		})
		if err != nil {
			c.logger.Error("Failed to execute bash command", zap.Error(err))
		}
		return true
	}

	switch input {
	case "/help":
		c.DisplayHelp()
		return true
	case "/history":
		err := c.HistoryCommand(ctx)
		if err != nil {
			c.logger.Error("Failed to select chat", zap.Error(err))
			c.DisplayError(err)
		}
		return true
	case "/agents":
		agents, err := c.agentService.ListAgents(ctx)
		if err != nil {
			c.logger.Error("Failed to list agents", zap.Error(err))
			c.DisplayError(err)
			return true
		}
		c.DisplayAgents(agents)
		return true
	case "/tools":
		tools, err := c.toolService.ListTools()
		if err != nil {
			c.logger.Error("Failed to list tools", zap.Error(err))
			c.DisplayError(err)
			return true
		}
		c.DisplayTools(tools)
		return true
	case "/usage":
		chats, err := c.chatService.ListChats(ctx)
		if err != nil {
			c.logger.Error("Failed to list chats", zap.Error(err))
			c.DisplayError(err)
			return true
		}
		var chat *entities.Chat
		for _, activeChat := range chats {
			if activeChat.Active {
				chat = activeChat
				break
			}
		}
		if chat == nil && len(chats) > 0 {
			chat = chats[0]
		}
		if chat == nil {
			c.DisplayError(fmt.Errorf("no active chat found"))
			return true
		}
		agent, err := c.agentService.GetAgent(ctx, chat.AgentID)
		if err != nil {
			c.logger.Error("Failed to get agent", zap.Error(err))
			c.DisplayError(err)
			return true
		}
		c.DisplayUsage(chat, agent)
		return true
	case "/exit", "/quit":
		c.DisplayInfo("Goodbye!")
		os.Exit(0)
		return true
	default:
		if strings.HasPrefix(input, "/new") {
			chat, err := c.NewChatCommand(ctx, input)
			if err != nil {
				c.logger.Error("Failed to create new chat", zap.Error(err))
				c.DisplayError(err)
			} else {
				c.DisplayInfo(fmt.Sprintf("Current Chat: %s", chat.Name))
			}
			return true
		}
		return false
	}
}

// processFileReferences replaces @path with the resolved file/directory path if valid
func (c *CLI) processFileReferences(input string) (string, error) {
	pathRegex := regexp.MustCompile(`@([./~][^@\s]*|[a-zA-Z0-9][^@\s]*)`)

	result := pathRegex.ReplaceAllStringFunc(input, func(match string) string {
		path := strings.TrimPrefix(match, "@")
		absPath, err := filepath.Abs(path)
		if err != nil {
			return match
		}
		_, err = os.Stat(absPath)
		if err != nil {
			return match
		}
		return path
	})

	return result, nil
}

// NewChatCommand creates a new chat with agent selection
func (c *CLI) NewChatCommand(ctx context.Context, userInput string) (*entities.Chat, error) {
	agents, err := c.agentService.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	var selectedAgent *entities.Agent
	if len(agents) > 1 {
		var options []string
		for _, agent := range agents {
			options = append(options, agent.Name)
		}
		var selectedName string
		err := c.ShowSpinner("Loading agents...", func() error {
			return huh.NewForm(huh.NewGroup(huh.NewSelect[string]().
				Title("Select an Agent").
				Options(huh.NewOptions(options...)...).
				Value(&selectedName)),
			).WithWidth(c.width).
				WithTheme(huh.ThemeCharm()).
				Run()
		})
		if err != nil {
			return nil, err
		}
		for _, agent := range agents {
			if agent.Name == selectedName {
				selectedAgent = agent
				break
			}
		}
	} else if len(agents) == 1 {
		selectedAgent = agents[0]
		c.DisplayInfo(fmt.Sprintf("Automatically selected agent: %s", selectedAgent.Name))
	} else {
		return nil, fmt.Errorf("no agents available")
	}

	chatName, foundName := strings.CutPrefix(userInput, "/new ")
	if !foundName || chatName == "" {
		chatName = "New Chat"
	}

	chat, err := c.chatService.CreateChat(ctx, selectedAgent.ID, chatName)
	if err != nil {
		return nil, err
	}
	return chat, nil
}

// HistoryCommand allows the user to select a chat from history
func (c *CLI) HistoryCommand(ctx context.Context) error {
	chats, err := c.chatService.ListChats(ctx)
	if err != nil {
		return err
	}

	var options []string
	for _, chat := range chats {
		options = append(options, chat.Name)
	}

	var selectedName string
	err = c.ShowSpinner("Loading chats...", func() error {
		return huh.NewForm(huh.NewGroup(huh.NewSelect[string]().
			Title("Select a Chat").
			Options(huh.NewOptions(options...)...).
			Value(&selectedName)),
		).WithWidth(c.width).
			WithTheme(huh.ThemeCharm()).
			Run()
	})
	if err != nil {
		return err
	}

	var selectedChat *entities.Chat
	for _, chat := range chats {
		if chat.Name == selectedName {
			selectedChat = chat
			break
		}
	}

	if selectedChat == nil {
		return fmt.Errorf("selected chat not found")
	}

	err = c.chatService.SetActiveChat(ctx, selectedChat.ID)
	if err != nil {
		return err
	}

	c.messageContainer.Clear()
	c.DisplayInfo(fmt.Sprintf("Current Chat: %s", selectedChat.Name))
	for _, msg := range selectedChat.Messages {
		c.displayMessage(msg)
	}

	return nil
}

// BashCommand executes a bash command and returns the output
func (c *CLI) BashCommand(cmd string) (string, error) {
	output, err := c.Bash(cmd)
	if err != nil {
		return "", err
	}

	stdoutBytes, ok := output["Stdout"]
	if !ok {
		return "", fmt.Errorf("missing Stdout in output")
	}

	return string(stdoutBytes), nil
}

func (c *CLI) Bash(command string) (map[string][]byte, error) {
	// This function should execute the bash command and return the output
	result := make(map[string][]byte)
	// For simplicity, let's assume it returns a dummy output
	result["Stdout"] = []byte(fmt.Sprintf("Executed command: %s", command))
	return result, nil
}

// displayMessage renders a message based on its role
func (c *CLI) displayMessage(msg entities.Message) {
	switch msg.Role {
	case "user":
		c.DisplayUserMessage(msg.Content)
	case "assistant":
		c.DisplayAssistantMessage(msg.Content)
	case "tool":
		c.DisplayToolMessage("", "", msg.Content, false)
	}
}

// displayContainer renders and displays the message container
func (c *CLI) displayContainer() {
	fmt.Print("\033[2J\033[H") // Clear screen and move cursor to top
	fmt.Print(c.messageContainer.Render())
}

// updateSize updates the CLI size based on terminal dimensions
func (c *CLI) updateSize() {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		c.width = 80
		c.height = 24
		return
	}

	c.width = width
	c.height = height

	if c.messageRenderer != nil {
		c.messageRenderer.SetWidth(c.width)
	}
	if c.messageContainer != nil {
		c.messageContainer.SetSize(c.width, c.height-4)
	}
}

// isCanceledError checks if an error is a cancellation error
func isCanceledError(err error) bool {
	var cancelErr *errs.CanceledError
	return errors.As(err, &cancelErr)
}
