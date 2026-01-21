package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"task-management-api/models"
	"task-management-api/repository"
	"task-management-api/service"
	"task-management-api/utils"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TaskHandler struct {
	taskService *service.TaskService
	authService *service.AuthService
}

func NewTaskHandler(taskService *service.TaskService, authService *service.AuthService) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
		authService: authService,
	}
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	user, err := service.GetUserFromContext(r.Context())
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.taskService.CreateTask(r.Context(), user.ID, &req)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	user, err := service.GetUserFromContext(r.Context())
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	taskID, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	task, err := h.taskService.GetTask(r.Context(), taskID, user)
	if err != nil {
		if err.Error() == "task not found" {
			utils.RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		if err.Error() == "unauthorized access to task" {
			utils.RespondError(w, http.StatusForbidden, "you don't have permission to access this task")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	utils.RespondJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	user, err := service.GetUserFromContext(r.Context())
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse query parameters for pagination and filtering
	filter := repository.TaskFilter{
		Page:  1,
		Limit: 10,
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := models.TaskStatus(statusStr)
		if service.IsValidStatus(status) {
			filter.Status = &status
		} else {
			utils.RespondError(w, http.StatusBadRequest, "invalid status filter, must be one of: pending, in_progress, completed")
			return
		}
	}

	response, err := h.taskService.ListTasks(r.Context(), user, filter)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	user, err := service.GetUserFromContext(r.Context())
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	taskID, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	if err := h.taskService.DeleteTask(r.Context(), taskID, user); err != nil {
		if err.Error() == "task not found" {
			utils.RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		if err.Error() == "unauthorized to delete this task" {
			utils.RespondError(w, http.StatusForbidden, "you don't have permission to delete this task")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]string{"message": "task deleted successfully"})
}
