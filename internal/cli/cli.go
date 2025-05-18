package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"

	"github.com/manifoldco/promptui"
	"go.uber.org/zap"
)

// CLI manages the text-based user interface for the AI Agent console application.

type CLI struct {
	chatService  services.ChatService
	agentService services.AgentService
	toolService  services.ToolService
	logger       *zap.Logger
}

// NewCLI creates a new CLI instance.
func NewCLI(chatService services.ChatService, agentService services.AgentService, toolService services.ToolService, logger *zap.Logger) *CLI {
	return &CLI{
		chatService:  chatService,
		agentService: agentService,
		toolService:  toolService,
		logger:       logger,
	}
}

// Run starts the CLI interface, displaying chat history and handling user input.
func (c *CLI) Run(ctx context.Context) error {
	fmt.Println("AI Agent Console. Type '/help' for list of commands.")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Select the active chat from the repository
	chats, err := c.chatService.ListChats(context.Background())
	var chat *entities.Chat
	if err != nil || len(chats) == 0 {
		userInput := "/new New Chat"
		chat, err = c.NewChatCommand(ctx, userInput)
		if err != nil {
			c.logger.Error("Failed to create new chat", zap.Error(err))
			fmt.Println("Error creating new chat:", err)
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
	for _, msg := range chat.Messages {
		c.displayMessage(msg)
	}

	// Spinner functions
	var wg sync.WaitGroup
	stopSpinner := make(chan bool)

	startSpinner := func() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			spinner := []string{"-", "\\", "|", "/"}
			idx := 0
			for {
				select {
				case <-stopSpinner:
					fmt.Print("\r") // Clear the spinner
					return
				default:
					fmt.Printf("\r%s Thinking...", spinner[idx])
					idx = (idx + 1) % len(spinner)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
	}

	// Main interaction loop
	for {
		select {
		case <-sigChan:
			fmt.Println("\nShutting down...")
			return nil
		default:
			// Prompt for user input
			prompt := promptui.Prompt{
				Label: ">",
				Validate: func(input string) error {
					if strings.TrimSpace(input) == "" {
						return fmt.Errorf("input cannot be empty")
					}
					return nil
				},
			}

			userInput, err := prompt.Run()
			if err != nil {
				if err == promptui.ErrInterrupt {
					fmt.Println("\nInterrupted. Shutting down...")
					return nil
				}
				c.logger.Error("Prompt error", zap.Error(err))
				continue
			}

			userInput = strings.TrimSpace(userInput)
			if userInput == "/help" {
				fmt.Println("Available commands:")
				fmt.Println("!<command> - Execute a shell command")
				fmt.Println("/new - Start a new chat")
				fmt.Println("/agents - List available agents")
				fmt.Println("/tools - List available tools")
				fmt.Println("/usage - Show usage information")
				fmt.Println("/exit - Exit the application")
				fmt.Println("/help - Show this help message")
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
				fmt.Printf("Chat Usage:\n- Prompt Tokens: %d\n- Completion Tokens: %d\n- Total Tokens: %d\n- Total Cost: $%.2f\n",
					chat.Usage.TotalPromptTokens, chat.Usage.TotalCompletionTokens, chat.Usage.TotalTokens, chat.Usage.TotalCost)
				continue
			}

			if userInput == "exit" || userInput == "quit" || userInput == "/exit" || userInput == "/quit" {
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
		fmt.Printf("Assistant:\n%s\n", msg.Content)
	case "user":
		fmt.Printf("User: %s\n", msg.Content)
	case "tool":
		fmt.Printf("Tool Called.\n")
	}
}
