package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"

	"github.com/c-bata/go-prompt"
	"github.com/manifoldco/promptui"
	"go.uber.org/zap"
	"golang.org/x/term"
)

var termState *term.State

func saveTermState() {
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return
	}
	termState = oldState
}

func restoreTermState() {
	if termState != nil {
		term.Restore(int(os.Stdin.Fd()), termState)
	}
}

type CLI struct {
	chatService     services.ChatService
	agentService    services.AgentService
	toolService     services.ToolService
	providerService services.ProviderService
	logger          *zap.Logger
	cancel          context.CancelFunc
}

func NewCLI(chatService services.ChatService, agentService services.AgentService, toolService services.ToolService, providerService services.ProviderService, logger *zap.Logger) *CLI {
	return &CLI{
		chatService:     chatService,
		agentService:    agentService,
		toolService:     toolService,
		providerService: providerService,
		logger:          logger,
	}
}

// Run starts the CLI interface, displaying chat history and handling user input.
func (c *CLI) Run(ctx context.Context) error {
	saveTermState()
	// convert the context to a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	fmt.Println("AI Agent Console. Type '?' for help.")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	stopSpinner := make(chan bool)

	// Handle interrupt signal in a separate goroutine
	go func() {
		<-sigChan
		close(stopSpinner)
		wg.Wait()
		restoreTermState()
		fmt.Println("\nReceived interrupt signal. Shutting down...")
		cancel()
		os.Exit(0)
	}()

	// Select the active chat from the repository
	chats, err := c.chatService.ListChats(context.Background())
	var chat *entities.Chat
	if err != nil || len(chats) == 0 {
		userInput := "/new New Chat"
		chat, err = c.NewChatCommand(ctx, userInput)
		if err != nil {
			c.logger.Error("Failed to create new chat", zap.Error(err))
			fmt.Println("Error creating new chat:", err)
			restoreTermState()
			return err
		}
	}
	for _, activeChat := range chats {
		if activeChat.Active {
			chat = activeChat
			break
		}
	}
	// If no active chat is found, use the first chat. This shouldn't happen in normal operation.
	if chat == nil {
		chat = chats[0]
	}

	// Display existing messages
	fmt.Println(chat.Name)
	for _, msg := range chat.Messages {
		c.displayMessage(msg)
	}

	startSpinner := func() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			spinner := []string{"-", "\\", "|", "/"}
			idx := 0
			startTime := time.Now()

			for {
				select {
				case <-stopSpinner:
					fmt.Print("\r") // Clear the spinner
					return
				default:
					elapsed := time.Since(startTime).Seconds()
					fmt.Printf("\r%s Thinking... (%.0fs)", spinner[idx], elapsed)
					idx = (idx + 1) % len(spinner)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
	}

	// Main interaction loop
	for {
		userInput := prompt.Input(">: ", completer,
			prompt.OptionPrefixTextColor(prompt.Blue),
			prompt.OptionAddKeyBind(prompt.KeyBind{
				Key: prompt.ControlC,
				Fn: func(buf *prompt.Buffer) {
					fmt.Println("\nRecieved CTRL+C. Shutting down...")
					close(stopSpinner)
					wg.Wait()
					restoreTermState()
					c.cancel()
					os.Exit(0)
				},
			}),
		)
		userInput = strings.TrimSpace(userInput)
		if userInput == "?" {
			fmt.Println("Available commands:")
			fmt.Println("? - Show this help message")
			fmt.Println("!<command> - Execute a shell command")
			fmt.Println("@<file or dir> - Include a file or directory")
			fmt.Println("/new <name> - Start a new chat")
			fmt.Println("/history - Select from all available chats")
			fmt.Println("/agents - List available agents")
			fmt.Println("/tools - List available tools")
			fmt.Println("/usage - Show usage information")
			fmt.Println("/exit - Exit the application")
			continue
		}

		if strings.HasPrefix(userInput, "!") {
			cmd := userInput[1:]
			output, err := c.RunBashCommand(cmd)
			if err != nil {
				fmt.Printf("Error executing command: %s\n", err)
			} else {
				fmt.Println(output)
			}
			continue
		}

		if userInput == "/history" {
			err := c.HistoryCommand(ctx)
			if err != nil {
				c.logger.Error("Failed to select chat", zap.Error(err))
				fmt.Println("Error selecting chat:", err)
			}
			continue
		}

		if strings.HasPrefix(userInput, "/new") {
			newChat, err := c.NewChatCommand(ctx, userInput)
			if err != nil {
				c.logger.Error("Failed to create new chat", zap.Error(err))
				fmt.Println("Error creating new chat:", err)
			}
			chat = newChat
			continue
		}

		if userInput == "/agents" {
			agents, err := c.agentService.ListAgents(ctx)
			if err != nil {
				c.logger.Error("Failed to list agents", zap.Error(err))
				fmt.Println("Error listing agents:", err)
				continue
			}
			fmt.Println("Available agents:")
			for _, agent := range agents {
				fmt.Printf("- %s (%s)\n", agent.Name, agent.ProviderType)
			}
			continue
		}

		if userInput == "/tools" {
			tools, err := c.toolService.ListTools()
			if err != nil {
				c.logger.Error("Failed to list tools", zap.Error(err))
				fmt.Println("Error listing tools:", err)
				continue
			}
			fmt.Println("Available tools:")
			for _, tool := range tools {
				fmt.Printf("- %s\n", (*tool).Name())
			}
			continue
		}

		if userInput == "/usage" {
			agent, err := c.agentService.GetAgent(ctx, chat.AgentID)
			if err != nil {
				c.logger.Error("Failed to get agent", zap.Error(err))
				fmt.Println("Error getting agent:", err)
				continue
			}
			fmt.Printf("Chat Usage:\n- Provider: %s\n- Model: %s\n- Prompt Tokens: %d\n- Completion Tokens: %d\n- Total Tokens: %d\n- Total Cost: $%.2f\n",
				agent.ProviderType, agent.Model, chat.Usage.TotalPromptTokens, chat.Usage.TotalCompletionTokens, chat.Usage.TotalTokens, chat.Usage.TotalCost)
			continue
		}

		if userInput == "exit" || userInput == "quit" || userInput == "/exit" || userInput == "/quit" {
			restoreTermState()
			fmt.Println("Shutting down...")
			fmt.Printf("Chat Usage:\n- Prompt Tokens: %d\n- Completion Tokens: %d\n- Total Tokens: %d\n- Total Cost: $%.2f\n",
				chat.Usage.TotalPromptTokens, chat.Usage.TotalCompletionTokens, chat.Usage.TotalTokens, chat.Usage.TotalCost)
			return nil
		}

		// Create and save user message
		message := entities.NewMessage("user", userInput)

		// Start the spinner
		startSpinner()

		// Generate assistant response
		response, err := c.chatService.SendMessage(ctx, chat.ID, message)

		// Stop the spinner
		stopSpinner <- true
		wg.Wait()

		if err != nil {
			c.logger.Error("Failed to generate response", zap.Error(err))
			fmt.Println("Error generating response:", err)
			continue
		}

		// Strip any <think>*</think> tags from the response including the content
		responseContent := response.Content
		for {
			start := strings.Index(responseContent, "<think>")
			end := strings.Index(responseContent, "</think>")
			if start == -1 || end == -1 {
				break
			}

			// Remove the <think></think> section, turning the string into "before" + "after"
			responseContent = responseContent[:start] + responseContent[end+len("</think>"):]
		}

		// Update the response Content
		response.Content = responseContent

		c.displayMessage(*response)
	}
}

func completer(d prompt.Document) []prompt.Suggest {
	text := strings.TrimSpace(d.TextBeforeCursor())

	// Static command suggestions
	suggestions := []prompt.Suggest{
		{Text: "?", Description: "Show help information"},
		{Text: "!<command>", Description: "Execute a shell command"},
		{Text: "/new <name>", Description: "Start a new chat"},
		{Text: "/history", Description: "Select from all available chats"},
		{Text: "/agents", Description: "List available agents"},
		{Text: "/tools", Description: "List available tools"},
		{Text: "/usage", Description: "Show usage information"},
		{Text: "/exit", Description: "Exit the application"},
	}

	// Handle empty input or command suggestions
	if text == "" || strings.HasPrefix(text, "/") {
		return prompt.FilterHasPrefix(suggestions, text, true)
	}

	// Handle bash command completion for ! prefix (only at start)
	if strings.HasPrefix(text, "!") {
		cmdText := strings.TrimSpace(strings.TrimPrefix(text, "!"))
		return bashCompleter(cmdText)
	}

	// Handle file/directory completion for @ anywhere in the input
	lastAt := strings.LastIndex(text, "@")
	if lastAt != -1 {
		pathText := strings.TrimSpace(text[lastAt+1:])
		// Only trigger file completion if @ is followed by a path or nothing (not another command)
		if pathText == "" || !strings.HasPrefix(pathText, "/") && !strings.HasPrefix(pathText, "!") {
			return fileCompleter(pathText)
		}
	}

	return []prompt.Suggest{}
}

// bashCompleter generates bash command and path completions using compgen
func bashCompleter(input string) []prompt.Suggest {
	var suggestions []prompt.Suggest

	// Split input to separate command and arguments
	parts := strings.Fields(input)
	if len(parts) == 0 {
		// Suggest available commands
		output, err := exec.Command("bash", "-c", "compgen -c").Output()
		if err != nil {
			return suggestions
		}
		commands := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, cmd := range commands {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        "!" + cmd,
				Description: "Shell command",
			})
		}
		return suggestions
	}

	// For arguments, suggest files and directories
	lastPart := parts[len(parts)-1]
	cmd := fmt.Sprintf("compgen -f %s", lastPart)
	output, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return suggestions
	}

	paths := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, path := range paths {
		desc := "File"
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			desc = "Directory"
			path = path + "/"
		}
		suggestions = append(suggestions, prompt.Suggest{
			Text:        "!" + strings.Join(parts[:len(parts)-1], " ") + " " + path,
			Description: desc,
		})
	}
	return suggestions
}

// fileCompleter generates file and directory suggestions for @ prefix
func fileCompleter(input string) []prompt.Suggest {
	var suggestions []prompt.Suggest
	dir := "."
	base := input

	// If input contains a path, split into directory and base
	if strings.Contains(input, "/") {
		lastSlash := strings.LastIndex(input, "/")
		dir = input[:lastSlash]
		base = input[lastSlash+1:]
		if dir == "" {
			dir = "/"
		}
	}

	// Read directory contents
	entries, err := os.ReadDir(dir)
	if err != nil {
		return suggestions
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, base) {
			desc := "File"
			fullPath := filepath.Join(dir, name)
			if entry.IsDir() {
				desc = "Directory"
				fullPath = fullPath + "/"
			}
			// Only include @ and the completed path in the suggestion
			suggestions = append(suggestions, prompt.Suggest{
				Text:        "@" + fullPath,
				Description: desc,
			})
		}
	}
	return suggestions
}

func (c *CLI) NewChatCommand(ctx context.Context, userInput string) (*entities.Chat, error) {
	fmt.Println("Starting a new chat...")

	// Get list of agents
	agents, err := c.agentService.ListAgents(ctx)
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		fmt.Println("Error listing agents:", err)
		return nil, err
	}

	var selectedAgent *entities.Agent
	if len(agents) > 1 {
		// Prompt user to select agent
		agentNames := make([]string, len(agents))
		for i, agent := range agents {
			agentNames[i] = agent.Name
		}
		prompt := promptui.Select{
			Label: "Select an Agent",
			Items: agentNames,
		}
		_, selectedName, err := prompt.Run()
		if err != nil {
			c.logger.Error("Prompt error", zap.Error(err))
			fmt.Println("Error selecting agent:", err)
			return nil, err
		}
		// Find the selected agent by name
		foundAgent := false
		for _, agent := range agents {
			if agent.Name == selectedName {
				selectedAgent = agent
				foundAgent = true
				break
			}
		}
		if !foundAgent {
			fmt.Println("Selected agent not found.")
			return nil, err
		}
	} else if len(agents) == 1 {
		selectedAgent = agents[0]
		fmt.Printf("Automatically selected agent: %s\n", selectedAgent.Name)
	} else {
		fmt.Println("No agents available.")
		return nil, err
	}

	// Get the chat name from input
	chatName, foundName := strings.CutPrefix(userInput, "/new ")
	if !foundName || chatName == "" {
		chatName = "New Chat"
	}

	// Create new chat with selected agent's ID
	chat, err := c.chatService.CreateChat(ctx, selectedAgent.ID, chatName)
	if err != nil {
		c.logger.Error("Failed to create new chat", zap.Error(err))
		fmt.Println("Error creating new chat:", err)
		return nil, err
	}
	return chat, nil
}

func (c *CLI) HistoryCommand(ctx context.Context) error {
	chats, err := c.chatService.ListChats(ctx)
	if err != nil {
		c.logger.Error("Failed to list chats", zap.Error(err))
		fmt.Println("Error listing chats:", err)
		return err
	}

	chatNames := make([]string, len(chats))
	for i, chat := range chats {
		chatNames[i] = chat.Name
	}

	prompt := promptui.Select{
		Label: "Select a Chat",
		Items: chatNames,
	}

	i, _, err := prompt.Run()
	if err != nil {
		c.logger.Error("Prompt error", zap.Error(err))
		fmt.Println("Error selecting chat:", err)
		return err
	}

	selectedChat := chats[i]

	err = c.chatService.SetActiveChat(ctx, selectedChat.ID)
	if err != nil {
		c.logger.Error("Failed to set active chat", zap.Error(err))
		fmt.Println("Error setting active chat:", err)
		return err
	}

	fmt.Println(selectedChat.Name)
	for _, msg := range selectedChat.Messages {
		c.displayMessage(msg)
	}

	return nil
}

func (cli *CLI) RunBashCommand(cmd string) (string, error) {
	output, err := Bash(cmd)
	if err != nil {
		return "", err
	}

	stdoutBytes, ok := output["Stdout"]
	if !ok {
		return "", fmt.Errorf("missing Stdout in output")
	}

	return string(stdoutBytes), nil
}

func Bash(cmd string) (map[string][]byte, error) {
	var out, stderr bytes.Buffer

	// Create a new bash command
	command := exec.Command("bash", "-c", cmd)

	// Set the output destinations
	command.Stdout = &out
	command.Stderr = &stderr

	// Run the command
	err := command.Run()
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %v", err)
	}

	return map[string][]byte{
		"Stdout": out.Bytes(),
		"Stderr": stderr.Bytes(),
	}, nil
}

// displayMessage prints a message with role prefix and formatted content.
func (c *CLI) displayMessage(msg entities.Message) {
	switch msg.Role {
	case "assistant":
		fmt.Printf("\rAssistant:\n%s\n", msg.Content)
	case "user":
		fmt.Printf("User: %s\n", msg.Content)
	case "tool":
		fmt.Printf("Tool Called.\n")
	}
}
