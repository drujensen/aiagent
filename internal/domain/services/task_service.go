package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/repositories"
)

/**
 * @description
 * This file implements the TaskService for the AI Workflow Automation Platform.
 * It manages task creation, processing, delegation, and human interaction notifications,
 * enabling workflow execution by AI agents. The service integrates with repositories for
 * persistence and prepares for AI model and chat service integrations.
 *
 * Key features:
 * - Workflow Management: Starts workflows by assigning tasks to agents.
 * - Task Processing: Processes tasks with a loop handling tool execution, subtask delegation, and results.
 * - Human Interaction: Notifies for human input when required, pausing processing.
 * - Concurrency: Uses goroutines for concurrent subtask execution with synchronization.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides Task and AIAgent entity definitions.
 * - aiagent/internal/domain/repositories: Provides TaskRepository and AgentRepository interfaces.
 *
 * @notes
 * - AI model integration is stubbed, awaiting Step 19 (placeholder uses task description).
 * - ChatService integration is stubbed, awaiting Step 8 (human input assumed handled externally).
 * - Task context is stored in Messages, updated after each action for persistence.
 * - Edge cases include invalid agent IDs, insufficient child agents, and database failures.
 * - Assumption: Task IDs are strings matching MongoDB ObjectID hex format.
 * - Limitation: Placeholder AI parsing is simplistic; will improve with real AI integration.
 */

type TaskService interface {
	StartWorkflow(ctx context.Context, task *entities.Task) error
	ProcessTask(ctx context.Context, task *entities.Task) error
	DelegateSubtasks(ctx context.Context, parentTask *entities.Task, subtasks []*entities.Task) error
	NotifyHumanInteraction(ctx context.Context, task *entities.Task) error
}

type taskService struct {
	taskRepo  repositories.TaskRepository
	agentRepo repositories.AgentRepository
	// aiModel    AIModel // Stubbed for future integration
	// chatService ChatService // Stubbed for future integration
}

func NewTaskService(taskRepo repositories.TaskRepository, agentRepo repositories.AgentRepository) *taskService {
	return &taskService{
		taskRepo:  taskRepo,
		agentRepo: agentRepo,
	}
}

func (s *taskService) StartWorkflow(ctx context.Context, task *entities.Task) error {
	if task.Description == "" {
		return fmt.Errorf("task description is required")
	}
	if task.AssignedTo == "" {
		return fmt.Errorf("task must be assigned to an agent")
	}

	_, err := s.agentRepo.GetAgent(ctx, task.AssignedTo)
	if err != nil {
		if err == repositories.ErrNotFound {
			return fmt.Errorf("assigned agent not found: %s", task.AssignedTo)
		}
		return fmt.Errorf("failed to verify agent: %w", err)
	}

	task.Status = entities.TaskPending
	task.Messages = []entities.Message{} // Initialize Messages for context tracking

	err = s.taskRepo.CreateTask(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	go func() {
		processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if procErr := s.ProcessTask(processCtx, task); procErr != nil {
			fmt.Printf("Task processing error for task %s: %v\n", task.ID, procErr)
			task.Status = entities.TaskFailed
			task.Result = procErr.Error()
			if updateErr := s.taskRepo.UpdateTask(processCtx, task); updateErr != nil {
				fmt.Printf("Failed to update task status to failed: %v\n", updateErr)
			}
		}
	}()

	return nil
}

func (s *taskService) ProcessTask(ctx context.Context, task *entities.Task) error {
	latestTask, err := s.taskRepo.GetTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve task: %w", err)
	}
	task = latestTask

	task.Status = entities.TaskInProgress
	if err := s.taskRepo.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status to in_progress: %w", err)
	}

	// Process until completed or failed
	for task.Status != entities.TaskCompleted && task.Status != entities.TaskFailed {
		if task.RequiresHumanInteraction {
			if err := s.NotifyHumanInteraction(ctx, task); err != nil {
				task.Status = entities.TaskFailed
				task.Result = err.Error()
				s.taskRepo.UpdateTask(ctx, task)
				return fmt.Errorf("failed to notify for human interaction: %w", err)
			}
			// Break loop; task awaits external input, reprocessed later
			break
		}

		agent, err := s.agentRepo.GetAgent(ctx, task.AssignedTo)
		if err != nil {
			task.Status = entities.TaskFailed
			task.Result = err.Error()
			s.taskRepo.UpdateTask(ctx, task)
			return fmt.Errorf("failed to retrieve agent: %w", err)
		}

		aiResponse := generatePlaceholderAIResponse(agent, task)
		aiMessage := entities.Message{
			Type:      entities.ChatMessageType,
			Content:   aiResponse,
			Sender:    agent.ID,
			Timestamp: time.Now(),
		}
		task.Messages = append(task.Messages, aiMessage)
		if err := s.taskRepo.UpdateTask(ctx, task); err != nil {
			return fmt.Errorf("failed to update task with AI message: %w", err)
		}

		action, params := parseAIResponse(aiResponse)
		switch action {
		case "use_tool":
			toolName := params["tool"]
			toolInput := params["input"]
			toolResult := executePlaceholderTool(toolName, toolInput)
			toolMessage := entities.Message{
				Type:      entities.ToolMessageType,
				ToolName:  toolName,
				Request:   toolInput,
				Result:    toolResult,
				Timestamp: time.Now(),
			}
			task.Messages = append(task.Messages, toolMessage)
			if err := s.taskRepo.UpdateTask(ctx, task); err != nil {
				return fmt.Errorf("failed to update task with tool message: %w", err)
			}

		case "delegate_subtasks":
			subtaskDescs := strings.Split(params["subtasks"], "|")
			parentAgent := agent
			if len(parentAgent.ChildrenIDs) < len(subtaskDescs) {
				task.Status = entities.TaskFailed
				task.Result = "not enough child agents for subtasks"
				s.taskRepo.UpdateTask(ctx, task)
				return fmt.Errorf("not enough child agents: %d required, %d available", len(subtaskDescs), len(parentAgent.ChildrenIDs))
			}

			var subtasks []*entities.Task
			for i, desc := range subtaskDescs {
				subtask := &entities.Task{
					Description:  desc,
					AssignedTo:   parentAgent.ChildrenIDs[i],
					ParentTaskID: task.ID,
					Status:       entities.TaskPending,
					Messages:     []entities.Message{},
				}
				subtasks = append(subtasks, subtask)
			}

			if err := s.DelegateSubtasks(ctx, task, subtasks); err != nil {
				task.Status = entities.TaskFailed
				task.Result = err.Error()
				s.taskRepo.UpdateTask(ctx, task)
				return fmt.Errorf("failed to delegate subtasks: %w", err)
			}

			var subtaskResults string
			for _, subtask := range subtasks {
				subtaskResults += subtask.Result + "\n"
			}
			task.Result = "Subtasks completed:\n" + subtaskResults
			task.Status = entities.TaskCompleted
			if err := s.taskRepo.UpdateTask(ctx, task); err != nil {
				return fmt.Errorf("failed to update task with subtask results: %w", err)
			}
			return nil

		case "provide_result":
			task.Result = params["result"]
			task.Status = entities.TaskCompleted
			if err := s.taskRepo.UpdateTask(ctx, task); err != nil {
				return fmt.Errorf("failed to update task with result: %w", err)
			}
			return nil

		default:
			task.Status = entities.TaskFailed
			task.Result = fmt.Sprintf("invalid AI response action: %s", action)
			s.taskRepo.UpdateTask(ctx, task)
			return fmt.Errorf("invalid AI response action: %s", action)
		}
	}
	return nil
}

func (s *taskService) DelegateSubtasks(ctx context.Context, parentTask *entities.Task, subtasks []*entities.Task) error {
	for i, subtask := range subtasks {
		if subtask.Description == "" {
			return fmt.Errorf("subtask %d description is required", i+1)
		}
		if subtask.AssignedTo == "" {
			return fmt.Errorf("subtask %d must be assigned to an agent", i+1)
		}

		_, err := s.agentRepo.GetAgent(ctx, subtask.AssignedTo)
		if err != nil {
			return fmt.Errorf("invalid child agent for subtask %d: %w", i+1, err)
		}

		subtask.ParentTaskID = parentTask.ID
		subtask.Status = entities.TaskPending
		subtask.Messages = []entities.Message{}
		if err := s.taskRepo.CreateTask(ctx, subtask); err != nil {
			return fmt.Errorf("failed to create subtask %d: %w", i+1, err)
		}
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(subtasks))
	for _, subtask := range subtasks {
		wg.Add(1)
		go func(st *entities.Task) {
			defer wg.Done()
			if procErr := s.ProcessTask(ctx, st); procErr != nil {
				errChan <- fmt.Errorf("failed to process subtask %s: %w", st.ID, procErr)
			}
		}(subtask)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return fmt.Errorf("subtask processing errors: %v", errors)
	}

	return nil
}

func (s *taskService) NotifyHumanInteraction(ctx context.Context, task *entities.Task) error {
	// Placeholder for ChatService integration (Step 8)
	fmt.Printf("Stub: Would notify human for task %s\n", task.ID)
	return nil
}

// Placeholder functions below; to be replaced in future steps

var placeholderTools = map[string]func(string) string{
	"DuckDuckGoSearch": func(query string) string {
		return fmt.Sprintf("Search results for '%s': [dummy results]", query)
	},
	"Bash": func(command string) string {
		return fmt.Sprintf("Executed command '%s' with output: [dummy output]", command)
	},
}

func executePlaceholderTool(toolName string, input string) string {
	if toolFunc, ok := placeholderTools[toolName]; ok {
		return toolFunc(input)
	}
	return fmt.Sprintf("Unknown tool: %s", toolName)
}

func generatePlaceholderAIResponse(agent *entities.AIAgent, task *entities.Task) string {
	if len(task.Messages) == 0 {
		return fmt.Sprintf("use tool: DuckDuckGoSearch with input: %s", task.Description)
	}
	lastMessage := task.Messages[len(task.Messages)-1]
	if lastMessage.Type == entities.ToolMessageType {
		return fmt.Sprintf("result: Summarized tool output: %s", lastMessage.Result)
	}
	return "result: Default result"
}

func parseAIResponse(response string) (string, map[string]string) {
	if strings.HasPrefix(response, "use tool:") {
		parts := strings.Split(response, "use tool: ")
		if len(parts) > 1 {
			toolParts := strings.Split(parts[1], " with input: ")
			if len(toolParts) == 2 {
				toolName := toolParts[0]
				input := toolParts[1]
				return "use_tool", map[string]string{"tool": toolName, "input": input}
			}
		}
	} else if strings.HasPrefix(response, "create subtask:") {
		subtaskLines := strings.Split(response, "\n")
		var subtasks []string
		for _, line := range subtaskLines {
			if strings.HasPrefix(line, "create subtask:") {
				desc := strings.TrimPrefix(line, "create subtask:")
				subtasks = append(subtasks, strings.TrimSpace(desc))
			}
		}
		if len(subtasks) > 0 {
			return "delegate_subtasks", map[string]string{"subtasks": strings.Join(subtasks, "|")}
		}
	} else if strings.HasPrefix(response, "result:") {
		parts := strings.Split(response, "result: ")
		if len(parts) > 1 {
			result := parts[1]
			return "provide_result", map[string]string{"result": result}
		}
	}
	return "invalid", nil
}
