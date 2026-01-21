package service

import (
	"context"
	"log"
	"task-management-api/models"
	"task-management-api/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TaskWorker struct {
	taskRepo            *repository.TaskRepository
	autoCompleteMinutes int
	taskChannel         chan primitive.ObjectID
}

func NewTaskWorker(taskRepo *repository.TaskRepository, autoCompleteMinutes int) *TaskWorker {
	return &TaskWorker{
		taskRepo:            taskRepo,
		autoCompleteMinutes: autoCompleteMinutes,
		taskChannel:         make(chan primitive.ObjectID, 100),
	}
}

func (w *TaskWorker) Start(ctx context.Context) {
	log.Printf("Starting background worker - auto-complete after %d minutes", w.autoCompleteMinutes)

	// Start worker goroutines to process tasks from the channel
	for i := 0; i < 3; i++ {
		go w.processTasksFromChannel(ctx)
	}

	// Periodically check for tasks that need auto-completion
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Background worker stopped")
			close(w.taskChannel)
			return
		case <-ticker.C:
			w.checkAndQueueTasks(ctx)
		}
	}
}

func (w *TaskWorker) checkAndQueueTasks(ctx context.Context) {
	// Find tasks that are older than the auto-complete threshold
	threshold := time.Now().Add(-time.Duration(w.autoCompleteMinutes) * time.Minute)

	tasks, err := w.taskRepo.FindPendingTasks(ctx, threshold)
	if err != nil {
		log.Printf("Error finding pending tasks: %v", err)
		return
	}

	// Queue tasks for auto-completion
	for _, task := range tasks {
		select {
		case w.taskChannel <- task.ID:
			log.Printf("Queued task %s for auto-completion", task.ID.Hex())
		default:
			log.Printf("Task channel full, skipping task %s", task.ID.Hex())
		}
	}
}

func (w *TaskWorker) processTasksFromChannel(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case taskID, ok := <-w.taskChannel:
			if !ok {
				return
			}
			w.autoCompleteTask(ctx, taskID)
		}
	}
}

func (w *TaskWorker) autoCompleteTask(ctx context.Context, taskID primitive.ObjectID) {
	// Verify the task still exists and is in a valid state
	task, err := w.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		log.Printf("Task %s not found or already deleted, skipping auto-completion", taskID.Hex())
		return
	}

	// Only auto-complete if still in pending or in_progress status
	if task.Status == models.TaskStatusPending || task.Status == models.TaskStatusInProgress {
		// Check if task is old enough
		threshold := time.Now().Add(-time.Duration(w.autoCompleteMinutes) * time.Minute)
		if task.CreatedAt.Before(threshold) {
			err := w.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusCompleted)
			if err != nil {
				log.Printf("Failed to auto-complete task %s: %v", taskID.Hex(), err)
				return
			}
			log.Printf("Auto-completed task %s", taskID.Hex())
		}
	}
}
