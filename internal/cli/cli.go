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
	chatService services.ChatService
	chatID      string
	logger      *zap.Logger
}

// NewCLI creates a new CLI instance.
func NewCLI(chatService services.ChatService, chatID string, logger *zap.Logger) *CLI {
	return &CLI{
		chatService: chatService,
		chatID:      chatID,
		logger:      logger,
	}
}

// Run starts the CLI interface, displaying chat history and handling user input.
func (c *CLI) Run(ctx context.Context) error {
	fmt.Println("AI Agent Console. Type 'exit' to quit.")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load initial chat history
	chat, err := c.chatService.GetChat(ctx, c.chatID)
	if err != nil {
		c.logger.Error("Failed to load chat", zap.Error(err))
		return fmt.Errorf("failed to load chat: %v", err)
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
					fmt.Println("\nShutting down...")
					return nil
				}
				c.logger.Error("Prompt error", zap.Error(err))
				continue
			}

			userInput = strings.TrimSpace(userInput)
			if userInput == "exit" {
				fmt.Println("Shutting down...")
				return nil
			}

			// Create and save user message
			message := entities.NewMessage("user", userInput)
			c.displayMessage(*message)

			// Generate assistant response
			response, err := c.chatService.SendMessage(ctx, c.chatID, message)
			if err != nil {
				c.logger.Error("Failed to generate response", zap.Error(err))
				fmt.Println("Error generating response:", err)
				continue
			}
			c.displayMessage(*response)
		}
	}
}

// displayMessage prints a message with role prefix and formatted content.
func (c *CLI) displayMessage(msg entities.Message) {
	prefix := "Uknown: "
	switch msg.Role {
	case "system":
		prefix = "System: "
	case "assistant":
		prefix = "Assistant: "
	case "user":
		prefix = "User: "
	case "tool":
		prefix = "Tool: "
	}
	fmt.Printf("%s%s\n", prefix, msg.Content)
}
