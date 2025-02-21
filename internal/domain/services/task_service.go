package services

import (
	"context"
	"fmt"
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
 * - Task Processing: Processes tasks, including tool execution and subtask delegation.
 * - Human Interaction: Notifies for human input when required.
 * - Concurrency: Uses goroutines for concurrent task execution with synchronization.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides Task and AIAgent entity definitions.
 * - aiagent/internal/domain/repositories: Provides TaskRepository and AgentRepository interfaces.
 *
 * @notes
 * - AI model integration is stubbed out, awaiting implementation in Step 19.
 * - ChatService integration is stubbed out, awaiting implementation in Step 8.
 * - Validation occurs before repository operations to prevent invalid data persistence.
 * - Edge cases include invalid agent IDs, missing parent tasks, and concurrent task conflicts.
 * - Assumption: Task IDs are strings matching MongoDB ObjectID hex format.
 * - Limitation: AI response parsing for tool calls and subtasks is placeholder; will evolve with AI integration.
 */

// TaskService defines the interface for managing tasks in the domain layer.
// It provides methods for starting workflows, processing tasks, delegating subtasks,
// and notifying for human interaction, abstracting the underlying implementation.
type TaskService interface {
	// StartWorkflow assigns an initial task to a Manager Agent and triggers processing.
	// The task must have a valid AssignedTo referencing an existing AIAgent.
	// Returns an error if the agent doesn't exist or task creation fails.
	StartWorkflow(ctx context.Context, task *entities.Task) error

	// ProcessTask processes a task by generating an AI response and parsing for actions.
	// Actions may include tool execution, subtask delegation, or direct result updates.
	// Handles human interaction requirements by notifying via ChatService if needed.
	// Returns an error if processing fails (e.g., AI model error, repository failure).
	ProcessTask(ctx context.Context, task *entities.Task) error

	// DelegateSubtasks creates subtasks and assigns them to child agents.
	// Subtasks inherit the parent task's context and are processed concurrently.
	// Returns an error if subtask creation or assignment fails.
	DelegateSubtasks(ctx context.Context, parentTask *entities.Task, subtasks []*entities.Task) error

	// NotifyHumanInteraction creates a conversation for tasks requiring human input.
	// This method prepares for integration with ChatService to notify users.
	// Returns an error if conversation creation fails (stubbed for now).
	NotifyHumanInteraction(ctx context.Context, task *entities.Task) error
}

// taskService implements the TaskService interface.
// It uses repositories for persistence and prepares for AI and chat integrations.
type taskService struct {
	taskRepo  repositories.TaskRepository
	agentRepo repositories.AgentRepository
	// aiModel    AIModel // Stubbed for future AI provider integration
	// chatService ChatService // Stubbed for future chat service integration
}

// NewTaskService creates a new instance of taskService with the given repositories.
//
// Parameters:
// - taskRepo: The repository for managing Task entities.
// - agentRepo: The repository for managing AIAgent entities.
//
// Returns:
// - *taskService: A new instance implementing TaskService.
func NewTaskService(taskRepo repositories.TaskRepository, agentRepo repositories.AgentRepository) *taskService {
	return &taskService{
		taskRepo:  taskRepo,
		agentRepo: agentRepo,
	}
}

// StartWorkflow assigns an initial task to a Manager Agent and triggers processing.
// It validates the AssignedTo agent exists and creates the task in the repository.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - task: Pointer to the Task entity to start; must have valid AssignedTo.
//
// Returns:
// - error: Nil on success, or an error if validation or creation fails.
func (s *taskService) StartWorkflow(ctx context.Context, task *entities.Task) error {
	// Validate required fields
	if task.Description == "" {
		return fmt.Errorf("task description is required")
	}
	if task.AssignedTo == "" {
		return fmt.Errorf("task must be assigned to an agent")
	}

	// Verify the assigned agent exists
	_, err := s.agentRepo.GetAgent(ctx, task.AssignedTo)
	if err != nil {
		if err == repositories.ErrNotFound {
			return fmt.Errorf("assigned agent not found: %s", task.AssignedTo)
		}
		return fmt.Errorf("failed to verify agent: %w", err)
	}

	// Set initial status
	task.Status = entities.TaskPending

	// Create the task in the repository
	err = s.taskRepo.CreateTask(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Trigger task processing in a goroutine for non-blocking execution
	go func() {
		processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if procErr := s.ProcessTask(processCtx, task); procErr != nil {
			// Log error; cannot return to caller as we're in a goroutine
			// In future, integrate with logging service (Step 20)
			fmt.Printf("Task processing error for task %s: %v\n", task.ID, procErr)
			// Update task status to failed
			task.Status = entities.TaskFailed
			task.Result = procErr.Error()
			if updateErr := s.taskRepo.UpdateTask(processCtx, task); updateErr != nil {
				fmt.Printf("Failed to update task status to failed: %v\n", updateErr)
			}
		}
	}()

	return nil
}

// ProcessTask processes a task by generating an AI response and parsing for actions.
// It updates the task status and handles human interaction if required.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - task: Pointer to the Task entity to process; must exist in repository.
//
// Returns:
// - error: Nil on success, or an error if processing fails.
func (s *taskService) ProcessTask(ctx context.Context, task *entities.Task) error {
	// Retrieve the latest task state
	latestTask, err := s.taskRepo.GetTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve task: %w", err)
	}
	task = latestTask

	// Update status to in_progress
	task.Status = entities.TaskInProgress
	if err := s.taskRepo.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status to in_progress: %w", err)
	}

	// Stubbed AI model integration: Generate response based on prompt
	// In future, integrate with AIModel (Step 19)
	agent, err := s.agentRepo.GetAgent(ctx, task.AssignedTo)
	if err != nil {
		task.Status = entities.TaskFailed
		task.Result = fmt.Sprintf("Agent retrieval failed: %v", err)
		s.taskRepo.UpdateTask(ctx, task)
		return fmt.Errorf("failed to retrieve agent for processing: %w", err)
	}

	// Placeholder AI response generation
	aiResponse := generatePlaceholderAIResponse(agent, task)

	// Parse AI response for actions (stubbed for now)
	// In future, implement parsing for tool calls and subtasks
	if task.RequiresHumanInteraction {
		if err := s.NotifyHumanInteraction(ctx, task); err != nil {
			task.Status = entities.TaskFailed
			task.Result = err.Error()
			s.taskRepo.UpdateTask(ctx, task)
			return fmt.Errorf("failed to notify for human interaction: %w", err)
		}
		// Task remains in_progress awaiting human input
		return nil
	}

	// Placeholder for tool execution and subtask delegation
	// Example parsed actions (to be replaced with AI parsing)
	parsedSubtasks := parsePlaceholderSubtasks(aiResponse)
	if len(parsedSubtasks) > 0 {
		if err := s.DelegateSubtasks(ctx, task, parsedSubtasks); err != nil {
			task.Status = entities.TaskFailed
			task.Result = fmt.Sprintf("Subtask delegation failed: %v", err)
			s.taskRepo.UpdateTask(ctx, task)
			return fmt.Errorf("failed to delegate subtasks: %w", err)
		}
	}

	// Update task status to completed with placeholder result
	task.Status = entities.TaskCompleted
	task.Result = fmt.Sprintf("Processed with AI response: %s", aiResponse)
	if err := s.taskRepo.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status to completed: %w", err)
	}

	return nil
}

// DelegateSubtasks creates subtasks and assigns them to child agents.
// It processes subtasks concurrently and waits for completion using sync.WaitGroup.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - parentTask: Pointer to the parent Task entity.
// - subtasks: Slice of subtasks to delegate; must reference valid child agents.
//
// Returns:
// - error: Nil on success, or an error if subtask creation or processing fails.
func (s *taskService) DelegateSubtasks(ctx context.Context, parentTask *entities.Task, subtasks []*entities.Task) error {
	// Retrieve parent agent to access child agents
	parentAgent, err := s.agentRepo.GetAgent(ctx, parentTask.AssignedTo)
	if err != nil {
		return fmt.Errorf("failed to retrieve parent agent: %w", err)
	}

	// Validate subtasks and assign to child agents
	childAgentIDs := parentAgent.ChildrenIDs
	if len(childAgentIDs) < len(subtasks) {
		return fmt.Errorf("not enough child agents (%d) for %d subtasks", len(childAgentIDs), len(subtasks))
	}

	for i, subtask := range subtasks {
		if subtask.Description == "" {
			return fmt.Errorf("subtask %d description is required", i+1)
		}
		subtask.AssignedTo = childAgentIDs[i]
		subtask.ParentTaskID = parentTask.ID
		subtask.Status = entities.TaskPending
		_, err := s.agentRepo.GetAgent(ctx, subtask.AssignedTo)
		if err != nil {
			return fmt.Errorf("invalid child agent for subtask %d: %w", i+1, err)
		}
		if err := s.taskRepo.CreateTask(ctx, subtask); err != nil {
			return fmt.Errorf("failed to create subtask %d: %w", i+1, err)
		}
	}

	// Process subtasks concurrently
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

	// Wait for all subtasks to complete
	wg.Wait()
	close(errChan)

	// Collect and aggregate errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return fmt.Errorf("subtask processing errors: %v", errors)
	}

	return nil
}

// NotifyHumanInteraction creates a conversation for tasks requiring human input.
// This method is a stub awaiting ChatService implementation in Step 8.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - task: Pointer to the Task entity requiring human interaction.
//
// Returns:
// - error: Nil on success (stubbed), or an error if conversation creation fails.
func (s *taskService) NotifyHumanInteraction(ctx context.Context, task *entities.Task) error {
	// Placeholder for ChatService integration
	// In future, create a Conversation and notify user via WebSocket
	// For now, simulate delay and return success
	time.Sleep(1 * time.Second)
	fmt.Printf("Stub: Would notify human for task %s\n", task.ID)
	return nil
}

// generatePlaceholderAIResponse generates a placeholder AI response for a task.
// This is a temporary function awaiting AI model integration in Step 19.
//
// Parameters:
// - agent: Pointer to the AIAgent processing the task.
// - task: Pointer to the Task entity being processed.
//
// Returns:
// - string: Placeholder AI response.
func generatePlaceholderAIResponse(agent *entities.AIAgent, task *entities.Task) string {
	return fmt.Sprintf("AI response for task '%s' by agent '%s': Process with tools or delegate", task.Description, agent.Name)
}

// parsePlaceholderSubtasks parses the AI response for subtasks (placeholder).
// This is a temporary function awaiting AI model parsing implementation.
//
// Parameters:
// - aiResponse: The AI-generated response to parse.
//
// Returns:
// - []*entities.Task: Slice of parsed subtasks (placeholder implementation).
func parsePlaceholderSubtasks(aiResponse string) []*entities.Task {
	// Placeholder: Simulate subtask detection
	// In future, parse AI response for "create subtask: description" patterns
	if aiResponse == "AI response for task 'Test workflow' by agent 'Manager': Process with tools or delegate" {
		return []*entities.Task{
			{
				Description: "Subtask 1: Perform step A",
			},
			{
				Description: "Subtask 2: Perform step B",
			},
		}
	}
	return nil
}
