/**
 * @description
 * TaskService manages the lifecycle of tasks in the AI Workflow Automation Platform.
 * It handles task creation, processing, delegation, and retrieval, enabling workflow execution
 * by AI agents. This service integrates with repositories for persistence and prepares for
 * integrations with AI models and chat services.
 *
 * Key features:
 * - Workflow Management: Starts workflows by assigning tasks to agents.
 * - Task Processing: Processes tasks with a loop handling tool execution, subtask delegation, and results.
 * - Human Interaction: Notifies for human input when required, pausing processing.
 * - Concurrency: Uses goroutines for concurrent subtask execution with synchronization.
 * - Task Retrieval: Allows fetching tasks by ID or all tasks for status checks and monitoring.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides Task and AIAgent entity definitions.
 * - aiagent/internal/domain/repositories: Provides TaskRepository and AgentRepository interfaces.
 * - context: For managing request timeouts and cancellations.
 * - fmt: For formatting error messages.
 * - strings: For parsing AI responses.
 * - sync: For synchronizing subtask execution.
 * - time: For handling timeouts and timestamps.
 *
 * @notes
 * - AI model integration is stubbed, awaiting Step 19 (placeholder uses task description).
 * - ChatService integration is stubbed, awaiting Step 8 (human input assumed handled externally).
 * - Task context is stored in Messages, updated after each action for persistence.
 * - Edge cases include invalid agent IDs, insufficient child agents, and database failures.
 * - Assumption: Task IDs are strings matching MongoDB ObjectID hex format.
 * - Limitation: Placeholder AI parsing is simplistic; will improve with real AI integration.
 */

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

// TaskService defines the interface for managing tasks in the domain layer.
// It encapsulates workflow initiation, task processing, and retrieval logic.
type TaskService interface {
	// StartWorkflow initiates a new workflow by creating and assigning a task to an agent.
	// It validates the task and starts processing asynchronously.
	// Returns nil on success or an error if validation or creation fails.
	StartWorkflow(ctx context.Context, task *entities.Task) error

	// ProcessTask executes the logic to process a task, including tool use or subtask delegation.
	// It updates the task status and result based on processing outcomes.
	// Returns nil on success or an error if processing fails.
	ProcessTask(ctx context.Context, task *entities.Task) error

	// DelegateSubtasks assigns subtasks to child agents and processes them concurrently.
	// It updates the parent task with aggregated results.
	// Returns nil on success or an error if delegation fails.
	DelegateSubtasks(ctx context.Context, parentTask *entities.Task, subtasks []*entities.Task) error

	// NotifyHumanInteraction flags a task for human input and notifies via chat (stubbed).
	// Returns nil on success or an error if notification fails.
	NotifyHumanInteraction(ctx context.Context, task *entities.Task) error

	// GetTask retrieves a task by its ID from the repository.
	// Returns the task and nil error on success, or nil and an error if not found or retrieval fails.
	GetTask(ctx context.Context, id string) (*entities.Task, error)

	// ListTasks retrieves all tasks from the repository for monitoring purposes.
	// Returns a slice of tasks and nil error on success, or nil and an error if retrieval fails.
	ListTasks(ctx context.Context) ([]*entities.Task, error)
}

// taskService implements the TaskService interface.
// It uses repositories to persist tasks and agents, ensuring domain consistency.
type taskService struct {
	taskRepo  repositories.TaskRepository  // Repository for task persistence
	agentRepo repositories.AgentRepository // Repository for agent validation and retrieval
	// aiModel    AIModel // Stubbed for future integration
	// chatService ChatService // Stubbed for future integration
}

// NewTaskService creates a new instance of taskService with the given repositories.
//
// Parameters:
// - taskRepo: Repository for managing Task entities.
// - agentRepo: Repository for managing AIAgent entities.
//
// Returns:
// - *taskService: A new instance implementing TaskService.
func NewTaskService(taskRepo repositories.TaskRepository, agentRepo repositories.AgentRepository) *taskService {
	return &taskService{
		taskRepo:  taskRepo,
		agentRepo: agentRepo,
	}
}

// GetTask retrieves a task by its ID from the repository.
// It performs a simple validation on the ID and delegates to the repository.
//
// Parameters:
// - ctx: Context for managing request lifecycle.
// - id: The string ID of the task to retrieve.
//
// Returns:
// - *entities.Task: The retrieved task, or nil if not found.
// - error: Nil on success, repositories.ErrNotFound if task doesn't exist, or another error.
func (s *taskService) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	if id == "" {
		return nil, fmt.Errorf("task ID is required")
	}
	task, err := s.taskRepo.GetTask(ctx, id)
	if err != nil {
		return nil, err // ErrNotFound or other errors from repository
	}
	return task, nil
}

// ListTasks retrieves all tasks from the repository.
// It delegates to the TaskRepository and wraps any errors with context.
//
// Parameters:
// - ctx: Context for managing request lifecycle.
//
// Returns:
// - []*entities.Task: Slice of all tasks, empty if none exist.
// - error: Nil on success, or an error if retrieval fails.
func (s *taskService) ListTasks(ctx context.Context) ([]*entities.Task, error) {
	tasks, err := s.taskRepo.ListTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	return tasks, nil
}

// StartWorkflow initiates a new workflow by creating and assigning a task to an agent.
// It validates the task, persists it, and starts processing in a goroutine.
//
// Parameters:
// - ctx: Context for managing request lifecycle.
// - task: Pointer to the Task entity to start; its ID and status are updated.
//
// Returns:
// - error: Nil on success, or an error if validation or creation fails.
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
		// Create a new context with timeout for processing
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

// ProcessTask executes the logic to process a task, handling tool use, subtask delegation, or results.
// It updates the task status and persists changes after each action.
//
// Parameters:
// - ctx: Context for managing request lifecycle.
// - task: Pointer to the Task entity to process; its fields are updated in place.
//
// Returns:
// - error: Nil on success, or an error if processing fails.
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

// DelegateSubtasks assigns subtasks to child agents and processes them concurrently.
// It creates and persists subtasks, then aggregates their results into the parent task.
//
// Parameters:
// - ctx: Context for managing request lifecycle.
// - parentTask: Pointer to the parent Task entity; updated with results.
// - subtasks: Slice of Task pointers to delegate.
//
// Returns:
// - error: Nil on success, or an error if any subtask creation or processing fails.
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

// NotifyHumanInteraction flags a task for human input and notifies via chat (stubbed).
// This is a placeholder until ChatService integration is implemented.
//
// Parameters:
// - ctx: Context for managing request lifecycle.
// - task: Pointer to the Task entity requiring human interaction.
//
// Returns:
// - error: Nil (stubbed), or an error if notification fails in future implementation.
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

// executePlaceholderTool simulates tool execution for placeholder AI responses.
// It maps tool names to dummy functions returning mock results.
//
// Parameters:
// - toolName: Name of the tool to execute.
// - input: Input string for the tool.
//
// Returns:
// - string: Mock result of tool execution.
func executePlaceholderTool(toolName string, input string) string {
	if toolFunc, ok := placeholderTools[toolName]; ok {
		return toolFunc(input)
	}
	return fmt.Sprintf("Unknown tool: %s", toolName)
}

// generatePlaceholderAIResponse simulates AI response generation.
// It provides mock responses based on task state for testing purposes.
//
// Parameters:
// - agent: Pointer to the AIAgent processing the task.
// - task: Pointer to the Task being processed.
//
// Returns:
// - string: Mock AI response string.
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

// parseAIResponse parses placeholder AI responses into actions and parameters.
// It interprets mock responses to simulate tool use or result provision.
//
// Parameters:
// - response: String response from the placeholder AI.
//
// Returns:
// - string: Action type (e.g., "use_tool", "provide_result").
// - map[string]string: Parameters extracted from the response.
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
