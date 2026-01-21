package service

import (
	"context"
	"fmt"
	"task-management-api/models"
	"task-management-api/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TaskService struct {
	taskRepo *repository.TaskRepository
}

func NewTaskService(taskRepo *repository.TaskRepository) *TaskService {
	return &TaskService{
		taskRepo: taskRepo,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, userID primitive.ObjectID, req *models.CreateTaskRequest) (*models.Task, error) {
	// Validate input
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	// Set default status if not provided
	status := req.Status
	if status == "" {
		status = models.TaskStatusPending
	}

	// Validate status
	if !IsValidStatus(status) {
		return nil, fmt.Errorf("invalid status, must be one of: pending, in_progress, completed")
	}

	// Create task
	task := models.NewTask(userID, req.Title, req.Description, status)
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

func (s *TaskService) GetTask(ctx context.Context, taskID primitive.ObjectID, user *models.User) (*models.Task, error) {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Authorization check: users can only access their own tasks, admins can access all
	if user.Role != models.UserRoleAdmin && task.UserID != user.ID {
		return nil, fmt.Errorf("unauthorized access to task")
	}

	return task, nil
}

func (s *TaskService) ListTasks(ctx context.Context, user *models.User, filter repository.TaskFilter) (*models.TaskListResponse, error) {
	var tasks []*models.Task
	var totalCount int64
	var err error

	// Admins can see all tasks, regular users can only see their own
	if user.Role == models.UserRoleAdmin {
		tasks, totalCount, err = s.taskRepo.FindAll(ctx, filter)
	} else {
		tasks, totalCount, err = s.taskRepo.FindByUserID(ctx, user.ID, filter)
	}

	if err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(totalCount) / filter.Limit
	if int(totalCount)%filter.Limit > 0 {
		totalPages++
	}

	return &models.TaskListResponse{
		Tasks:      tasks,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, taskID primitive.ObjectID, user *models.User) error {
	// Check if task exists and user has permission
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return err
	}

	// Authorization check: users can only delete their own tasks, admins can delete any task
	if user.Role != models.UserRoleAdmin && task.UserID != user.ID {
		return fmt.Errorf("unauthorized to delete this task")
	}

	return s.taskRepo.Delete(ctx, taskID)
}

func IsValidStatus(status models.TaskStatus) bool {
	return status == models.TaskStatusPending || status == models.TaskStatusInProgress || status == models.TaskStatusCompleted
}
