package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

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
	if err != nil || len(chats) == 0 {
		c.logger.Fatal("No chats available", zap.Error(err))
	}
	chat := chats[0]
	for _, activeChat := range chats {
		if activeChat.Active {
			chat = activeChat
			break
		}
	}

	// Display existing messages
	for _, msg := range chat.Messages {
		c.displayMessage(msg)
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
				fmt.Println("/new - Start a new chat")
				fmt.Println("/agents - List available agents")
				fmt.Println("/tools - List available tools")
				fmt.Println("/usage - Show usage information")
				fmt.Println("/exit - Exit the application")
				fmt.Println("/help - Show this help message")
				continue
			}

			if strings.HasPrefix(userInput, "/new") {
				fmt.Println("Starting a new chat...")

				// Get list of agents
				agents, err := c.agentService.ListAgents(ctx)
				if err != nil {
					c.logger.Error("Failed to list agents", zap.Error(err))
					fmt.Println("Error listing agents:", err)
					continue
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
						continue
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
						continue
					}
				} else if len(agents) == 1 {
					selectedAgent = agents[0]
					fmt.Printf("Automatically selected agent: %s\n", selectedAgent.Name)
				} else {
					fmt.Println("No agents available.")
					continue
				}

				// Get the chat name from input
				chatName, foundName := strings.CutPrefix(userInput, "/new ")
				if !foundName || chatName == "" {
					chatName = "New Chat"
				}

				// Create new chat with selected agent's ID
				newChat, err := c.chatService.CreateChat(ctx, selectedAgent.ID, chatName)
				if err != nil {
					c.logger.Error("Failed to create new chat", zap.Error(err))
					fmt.Println("Error creating new chat:", err)
					continue
				}
				chat = newChat // Update the current chat to the new one
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

			// Generate assistant response
			response, err := c.chatService.SendMessage(ctx, chat.ID, message)
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

// displayMessage prints a message with role prefix and formatted content.
func (c *CLI) displayMessage(msg entities.Message) {
	switch msg.Role {
	case "assistant":
		fmt.Printf("Assistant: %s\n", msg.Content)
	case "user":
		fmt.Printf("User: %s\n", msg.Content)
	case "tool":
		fmt.Printf("Tool Called.\n")
	}
}
