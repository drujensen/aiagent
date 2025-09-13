package tui

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type TUI struct {
	chatService  services.ChatService
	agentService services.AgentService
	toolService  services.ToolService
	activeChat   *entities.Chat

	chatView    ChatView
	chatForm    ChatForm
	historyView HistoryView
	usageView   UsageView
	helpView    HelpView
	agentView   AgentView
	toolView    ToolView
	commandMenu CommandMenu

	state string
	err   error
}

func NewTUI(chatService services.ChatService, agentService services.AgentService, toolService services.ToolService) TUI {
	ctx := context.Background()

	activeChat, err := chatService.GetActiveChat(ctx)
	if err != nil {
		activeChat = nil
	}

	initialState := "chat/view"
	if activeChat == nil {
		initialState = "chat/create"
	}

	return TUI{
		chatService:  chatService,
		agentService: agentService,
		toolService:  toolService,
		activeChat:   activeChat,

		chatView:    NewChatView(chatService, agentService, activeChat),
		chatForm:    NewChatForm(chatService, agentService),
		historyView: NewHistoryView(chatService),
		usageView:   NewUsageView(chatService, agentService),
		helpView:    NewHelpView(),
		agentView:   NewAgentView(agentService),
		toolView:    NewToolView(toolService),
		commandMenu: NewCommandMenu(),

		state: initialState,
		err:   nil,
	}
}

func (t TUI) Init() tea.Cmd {
	return tea.Batch(
		t.chatForm.Init(),
		t.chatView.Init(),
		t.historyView.Init(),
		t.usageView.Init(),
		t.helpView.Init(),
		t.agentView.Init(),
		t.toolView.Init(),
		t.commandMenu.Init(),
	)
}

func (t TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle chat view messages
	case startCreateChatMsg:
		t.state = "chat/create"
		t.chatForm.SetChatName(string(msg))
		return t, t.chatForm.Init()
	case updatedChatMsg:
		t.activeChat = msg
		t.state = "chat/view"
		var cmd tea.Cmd
		t.chatView, cmd = t.chatView.Update(msg)
		return t, cmd
	case canceledCreateChatMsg:
		t.state = "chat/view"
		t.chatView.err = errors.New("New chat canceled")
		if t.activeChat != nil {
			t.chatView.updateViewportContent()
		} else {
			return t, tea.Quit
		}
		return t, t.chatView.Init()
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
		t.chatView.updateViewportContent()
		t.state = "chat/view"
		return t, nil
	case historyCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.updateViewportContent()
		}
		return t, nil

	// Handle usage view messages
	case startUsageMsg:
		t.state = "chat/usage"
		return t, t.usageView.Init()
	case usageCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.updateViewportContent()
		}
		return t, nil

	// Handle help view messages
	case startHelpMsg:
		t.state = "chat/help"
		return t, t.helpView.Init()
	case helpCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.SetActiveChat(t.activeChat)
		}
		return t, nil

	// Handle agents view messages
	case startAgentsMsg:
		t.state = "agents/list"
		return t, t.agentView.Init()
	case agentsCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.updateViewportContent()
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
		updatedChat, err := t.chatService.UpdateChat(ctx, t.activeChat.ID, msg.agentID, t.activeChat.Name)
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
		t.chatView.updateViewportContent()
		t.state = "chat/view"
		return t, t.chatView.Init()

	// Handle tools view messages
	case startToolsMsg:
		t.state = "tools/list"
		return t, t.toolView.Init()
	case toolsCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.updateViewportContent()
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
			t.chatView.updateViewportContent()
		}

		switch msg.command {
		case "new":
			t.state = "chat/create"
			t.chatForm.SetChatName("") // No name provided
			return t, t.chatForm.Init()
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
		case "help":
			t.state = "chat/help"
			return t, t.helpView.Init()
		case "exit":
			return t, tea.Quit
		}
		return t, nil

	case commandsCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.updateViewportContent()
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

		t.chatForm, cmd = t.chatForm.Update(msg)
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

		t.helpView, cmd = t.helpView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		t.agentView, cmd = t.agentView.Update(msg)
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
	case "chat/create":
		t.chatForm, cmd = t.chatForm.Update(msg)
	case "chat/history":
		t.historyView, cmd = t.historyView.Update(msg)
	case "chat/usage":
		t.usageView, cmd = t.usageView.Update(msg)
	case "chat/help":
		t.helpView, cmd = t.helpView.Update(msg)
	case "agents/list":
		t.agentView, cmd = t.agentView.Update(msg)
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
	case "chat/create":
		return t.chatForm.View()
	case "chat/history":
		return t.historyView.View()
	case "chat/usage":
		return t.usageView.View()
	case "chat/help":
		return t.helpView.View()
	case "agents/list":
		return t.agentView.View()
	case "tools/list":
		return t.toolView.View()
	case "chat/commands":
		return t.commandMenu.View()
	}

	return "Error: Invalid state"
}
