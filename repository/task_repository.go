package repository

import (
	"context"
	"fmt"
	"sync"
	"task-management-api/database"
	"task-management-api/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TaskRepository struct {
	collection *mongo.Collection
	mu         sync.RWMutex
}

type TaskFilter struct {
	Status *models.TaskStatus
	Page   int
	Limit  int
}

func NewTaskRepository(db *database.MongoDB) *TaskRepository {
	return &TaskRepository{
		collection: db.Database.Collection("tasks"),
	}
}

func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.collection.InsertOne(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	task.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *TaskRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var task models.Task
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	return &task, nil
}

func (r *TaskRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID, filter TaskFilter) ([]*models.Task, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Build query
	query := bson.M{"user_id": userID}
	if filter.Status != nil {
		query["status"] = *filter.Status
	}

	// Count total documents
	totalCount, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Set pagination defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}

	// Calculate skip
	skip := (filter.Page - 1) * filter.Limit

	// Find options with pagination and sorting
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(filter.Limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []*models.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, 0, fmt.Errorf("failed to decode tasks: %w", err)
	}

	return tasks, totalCount, nil
}

func (r *TaskRepository) FindAll(ctx context.Context, filter TaskFilter) ([]*models.Task, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Build query
	query := bson.M{}
	if filter.Status != nil {
		query["status"] = *filter.Status
	}

	// Count total documents
	totalCount, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Set pagination defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}

	// Calculate skip
	skip := (filter.Page - 1) * filter.Limit

	// Find options with pagination and sorting
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(filter.Limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []*models.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, 0, fmt.Errorf("failed to decode tasks: %w", err)
	}

	return tasks, totalCount, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.TaskStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

func (r *TaskRepository) FindPendingTasks(ctx context.Context, olderThan time.Time) ([]*models.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	query := bson.M{
		"status": bson.M{
			"$in": []models.TaskStatus{models.TaskStatusPending, models.TaskStatusInProgress},
		},
		"created_at": bson.M{"$lt": olderThan},
	}

	cursor, err := r.collection.Find(ctx, query, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find pending tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []*models.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode tasks: %w", err)
	}

	return tasks, nil
}
