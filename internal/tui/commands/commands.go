package commands

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type updatedChatMsg *entities.Chat
type errMsg error

func SendMessageCmd(cs services.ChatService, chatID string, msg *entities.Message, ctx context.Context) tea.Cmd {
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
