package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
	"github.com/drujensen/aiagent/internal/impl/config"
	"go.uber.org/zap"
)

type TUI struct {
	chatService     services.ChatService
	agentService    services.AgentService
	modelService    services.ModelService
	providerService services.ProviderService
	toolService     services.ToolService
	globalConfig    *config.GlobalConfig
	logger          *zap.Logger
	activeChat      *entities.Chat

	chatView    ChatView
	historyView HistoryView
	usageView   UsageView
	agentView   AgentView
	modelView   ModelView
	toolView    ToolView
	commandMenu CommandMenu

	state string
	err   error
}

func NewTUI(chatService services.ChatService, agentService services.AgentService, modelService services.ModelService, providerService services.ProviderService, toolService services.ToolService, globalConfig *config.GlobalConfig, logger *zap.Logger) TUI {
	ctx := context.Background()

	activeChat, err := chatService.GetActiveChat(ctx)
	if err != nil {
		activeChat = nil
	}

	initialState := "chat/view"

	return TUI{
		chatService:     chatService,
		agentService:    agentService,
		modelService:    modelService,
		providerService: providerService,
		toolService:     toolService,
		globalConfig:    globalConfig,
		logger:          logger,
		activeChat:      activeChat,

		chatView:    NewChatView(chatService, agentService, modelService, logger, activeChat),
		historyView: NewHistoryView(chatService),
		usageView:   NewUsageView(chatService, agentService, modelService),
		agentView:   NewAgentView(agentService),
		modelView:   NewModelViewWithMode(modelService, providerService, "view"),
		toolView:    NewToolView(toolService),
		commandMenu: NewCommandMenu(),

		state: initialState,
		err:   nil,
	}
}

func (t TUI) Init() tea.Cmd {
	cmds := []tea.Cmd{
		t.chatView.Init(),
		t.historyView.Init(),
		t.usageView.Init(),
		t.agentView.Init(),
		t.modelView.Init(),
		t.toolView.Init(),
		t.commandMenu.Init(),
	}

	if t.activeChat == nil {
		cmds = append(cmds, t.autoCreateChatCmd())
	}

	return tea.Batch(cmds...)
}

func (t TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle chat view messages
	case startAutoCreateChatMsg:
		return t, t.autoCreateChatCmd()
	case updatedChatMsg:
		t.activeChat = msg
		t.state = "chat/view"
		var cmd tea.Cmd
		t.chatView, cmd = t.chatView.Update(msg)
		return t, cmd
	case chatCreatedMsg:
		ctx := context.Background()
		err := t.chatService.SetActiveChat(ctx, msg.ID)
		if err != nil {
			return t, func() tea.Msg { return errMsg(err) }
		}
		t.activeChat = msg
		t.chatView.SetActiveChat(msg)
		t.state = "chat/view"
		return t, nil
	// Handle history view messages
	case startHistoryMsg:
		t.state = "chat/history"
		return t, t.historyView.Init()
	case historySelectedMsg:
		ctx := context.Background()
		err := t.chatService.SetActiveChat(ctx, msg.chatID)
		if err != nil {
			return t, func() tea.Msg { return errMsg(err) }
		}
		chat, err := t.chatService.GetChat(ctx, msg.chatID)
		if err != nil {
			return t, func() tea.Msg { return errMsg(err) }
		}
		t.activeChat = chat
		t.chatView.activeChat = chat
		ctx2 := context.Background()
		agent, err := t.agentService.GetAgent(ctx2, chat.AgentID)
		if err != nil {
			t.chatView.err = err
			t.chatView.currentAgent = nil
		} else {
			t.chatView.currentAgent = agent
		}
		t.chatView.updateEditorContent()
		t.state = "chat/view"
		return t, nil
	case historyCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.updateEditorContent()
		}
		return t, nil

	// Handle usage view messages
	case startUsageMsg:
		t.state = "chat/usage"
		return t, t.usageView.Init()
	case usageCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.updateEditorContent()
		}
		return t, nil

	// Handle agents view messages
	case startAgentsMsg:
		t.state = "agents/list"
		return t, t.agentView.Init()
	case agentsCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.updateEditorContent()
		}
		return t, nil

	case startAgentSwitchMsg:
		if t.activeChat == nil {
			return t, nil
		}
		t.state = "agents/list"
		t.agentView.mode = "switch"
		return t, t.agentView.Init()
	case agentSelectedMsg:
		ctx := context.Background()
		updatedChat, err := t.chatService.UpdateChat(ctx, t.activeChat.ID, msg.agentID, t.activeChat.ModelID, t.activeChat.Name)
		if err != nil {
			return t, func() tea.Msg { return errMsg(err) }
		}
		t.activeChat = updatedChat
		t.chatView.activeChat = updatedChat
		ctx2 := context.Background()
		agent, err := t.agentService.GetAgent(ctx2, updatedChat.AgentID)
		if err != nil {
			t.chatView.err = err
			t.chatView.currentAgent = nil
		} else {
			t.chatView.currentAgent = agent
		}
		t.chatView.updateEditorContent()
		t.state = "chat/view"
		// Save last used agent to global config
		t.globalConfig.LastUsedAgent = msg.agentID
		if err := config.SaveGlobalConfig(t.globalConfig, t.logger); err != nil {
			t.logger.Error("Failed to save global config", zap.Error(err))
		}
		return t, t.chatView.Init()

	case startModelSwitchMsg:
		if t.activeChat == nil {
			return t, nil
		}
		t.state = "models/list"
		t.modelView.mode = "switch"
		return t, t.modelView.Init()
	case modelSelectedMsg:
		ctx := context.Background()
		updatedChat, err := t.chatService.UpdateChat(ctx, t.activeChat.ID, t.activeChat.AgentID, msg.modelID, t.activeChat.Name)
		if err != nil {
			return t, func() tea.Msg { return errMsg(err) }
		}
		t.activeChat = updatedChat
		t.chatView.activeChat = updatedChat
		ctx2 := context.Background()
		model, err := t.modelService.GetModel(ctx2, updatedChat.ModelID)
		if err != nil {
			t.chatView.err = err
			t.chatView.currentModel = nil
		} else {
			t.chatView.currentModel = model
		}
		t.chatView.updateEditorContent()
		t.state = "chat/view"
		// Save last used model to global config
		t.globalConfig.LastUsedModel = msg.modelID
		if err := config.SaveGlobalConfig(t.globalConfig, t.logger); err != nil {
			t.logger.Error("Failed to save global config", zap.Error(err))
		}

		// If the chat title is still the default and has messages, try to regenerate it with the new model
		if strings.HasPrefix(updatedChat.Name, "New Chat") && len(updatedChat.Messages) >= 2 {
			ctx := context.Background()
			if regeneratedChat, err := t.chatService.GenerateAndUpdateTitle(ctx, updatedChat.ID); err != nil {
				t.logger.Warn("Failed to regenerate title after model change", zap.Error(err))
			} else {
				// Update the active chat with the new title
				t.activeChat = regeneratedChat
				t.chatView.SetActiveChat(regeneratedChat)
				t.logger.Info("Regenerated chat title after model change", zap.String("title", regeneratedChat.Name))
			}
		}

		return t, nil

	// Handle tools view messages
	case startToolsMsg:
		t.state = "tools/list"
		return t, t.toolView.Init()
	case toolsCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.updateEditorContent()
		}
		return t, nil

	// Handle command menu messages
	case startCommandsMsg:
		t.state = "chat/commands"
		t.commandMenu.list.ResetFilter()
		return t, t.commandMenu.Init()

	case executeCommandMsg:
		// Default back to chat view
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.updateEditorContent()
		}

		switch msg.command {
		case "new":
			return t, t.autoCreateChatCmd()
		case "history":
			t.state = "chat/history"
			return t, t.historyView.Init()
		case "agents":
			t.state = "agents/list"
			return t, t.agentView.Init()
		case "tools":
			t.state = "tools/list"
			return t, t.toolView.Init()
		case "usage":
			t.state = "chat/usage"
			return t, t.usageView.Init()
		case "models":
			t.state = "models/list"
			t.modelView.SetMode("switch")
			return t, t.modelView.Init()
		case "exit":
			return t, tea.Quit
		}
		return t, nil

	case commandsCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.updateEditorContent()
		}
		return t, nil

	// Handle global key messages and errors
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
		case tea.KeyCtrlH:
			if t.state == "chat/view" {
				t.state = "chat/history"
				return t, t.historyView.Init()
			}
			return t, tea.Quit
		}

	// case errMsg:
	//   t.err = msg
	//   return t, nil

	case tea.WindowSizeMsg:
		var (
			cmd  tea.Cmd
			cmds []tea.Cmd
		)

		t.chatView, cmd = t.chatView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		t.historyView, cmd = t.historyView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		t.usageView, cmd = t.usageView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		t.agentView, cmd = t.agentView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		t.modelView, cmd = t.modelView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		t.toolView, cmd = t.toolView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		t.commandMenu, cmd = t.commandMenu.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		return t, tea.Batch(cmds...)
	}

	var cmd tea.Cmd
	switch t.state {
	case "chat/view":
		t.chatView, cmd = t.chatView.Update(msg)
	case "chat/history":
		t.historyView, cmd = t.historyView.Update(msg)
	case "chat/usage":
		t.usageView, cmd = t.usageView.Update(msg)
	case "agents/list":
		t.agentView, cmd = t.agentView.Update(msg)
	case "models/list":
		t.modelView, cmd = t.modelView.Update(msg)
	case "tools/list":
		t.toolView, cmd = t.toolView.Update(msg)
	case "chat/commands":
		t.commandMenu, cmd = t.commandMenu.Update(msg)
	}
	return t, cmd
}

func (t TUI) View() string {
	switch t.state {
	case "chat/view":
		return t.chatView.View()
	case "chat/history":
		return t.historyView.View()
	case "chat/usage":
		return t.usageView.View()
	case "agents/list":
		return t.agentView.View()
	case "models/list":
		return t.modelView.View()
	case "tools/list":
		return t.toolView.View()
	case "chat/commands":
		return t.commandMenu.View()
	}

	return "Error: Invalid state"
}

func (t *TUI) autoCreateChatCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Get last used or default agent
		agentID := t.globalConfig.LastUsedAgent
		if agentID != "" {
			// Validate that the agent exists
			if _, err := t.agentService.GetAgent(ctx, agentID); err != nil {
				agentID = "" // Reset if not found
			}
		}
		if agentID == "" {
			agents, _ := t.agentService.ListAgents(ctx)
			if len(agents) > 0 {
				agentID = agents[0].ID
			}
		}

		// Get last used or default model
		modelID := t.globalConfig.LastUsedModel
		if modelID != "" {
			// Validate that the model exists
			if _, err := t.modelService.GetModel(ctx, modelID); err != nil {
				modelID = "" // Reset if not found
			}
		}
		if modelID == "" {
			models, _ := t.modelService.ListModels(ctx)
			if len(models) > 0 {
				modelID = models[0].ID
			}
		}

		// Update global config with last used agent and model
		if agentID != "" {
			t.globalConfig.LastUsedAgent = agentID
			if err := config.SaveGlobalConfig(t.globalConfig, t.logger); err != nil {
				t.logger.Warn("Failed to save global config", zap.Error(err))
			}
		}
		if modelID != "" {
			t.globalConfig.LastUsedModel = modelID
			if err := config.SaveGlobalConfig(t.globalConfig, t.logger); err != nil {
				t.logger.Warn("Failed to save global config", zap.Error(err))
			}
		}

		// Generate temp title
		tempTitle := fmt.Sprintf("New Chat - %s", time.Now().Format("2006-01-02 15:04"))

		// Create and return chat
		chat, err := t.chatService.CreateChat(ctx, agentID, modelID, tempTitle)
		if err != nil {
			return errMsg(err)
		}
		return chatCreatedMsg(chat)
	}
}
