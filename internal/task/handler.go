package task

import (
	"advanced-backend/internal"
	"context"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

type Response struct {
	ID          int32      `json:"id"`
	Labels      []string   `json:"labels"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	DueDate     time.Time  `json:"due_date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateRequest struct {
	Title string `json:"title" validate:"required"`
}

type UpdateRequest struct {
	Labels      []string   `json:"labels" validate:"omitempty"`
	Title       string     `json:"title" validate:"required"`
	Description string     `json:"description" validate:"omitempty"`
	Status      TaskStatus `json:"status" validate:"required,oneof=INBOX TO_DO IN_PROGRESS DONE"`
	DueDate     time.Time  `json:"due_date" validate:"omitempty"`
}

type Store interface {
	GetAll(ctx context.Context) ([]Task, error)
	GetByID(ctx context.Context, id int32) (Task, error)
	Create(ctx context.Context, title string) (Task, error)
	Update(ctx context.Context, id int32, labels []string, title, description string, status TaskStatus, dueDate time.Time) (Task, error)
	Delete(ctx context.Context, id int32) error
}
type Handler struct {
	logger    *zap.Logger
	validator *validator.Validate
	store     Store
}

func NewHandler(logger *zap.Logger, validator *validator.Validate, store Store) *Handler {
	return &Handler{
		logger:    logger,
		validator: validator,
		store:     store,
	}
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tasks, err := h.store.GetAll(ctx)
	if err != nil {
		h.logger.Error("Failed to get all tasks", zap.Error(err))
		http.Error(w, "Failed to get tasks", http.StatusInternalServerError)
		return
	}

	var resp = make([]Response, len(tasks))
	for i, task := range tasks {
		resp[i] = Response{
			ID:          task.ID,
			Labels:      task.Labels,
			Title:       task.Title,
			Description: task.Description.String,
			Status:      task.Status,
			DueDate:     task.DueDate.Time,
			CreatedAt:   task.CreatedAt.Time,
			UpdatedAt:   task.UpdatedAt.Time,
		}
	}

	// Write response
	internal.WriteJSONResponse(w, http.StatusOK, resp)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract task ID from URL
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := h.store.GetByID(ctx, int32(id))
	if err != nil {
		h.logger.Error("Failed to get task by ID", zap.Error(err))
		http.Error(w, "Failed to get task", http.StatusInternalServerError)
		return
	}

	resp := Response{
		ID:          task.ID,
		Labels:      task.Labels,
		Title:       task.Title,
		Description: task.Description.String,
		Status:      task.Status,
		DueDate:     task.DueDate.Time,
		CreatedAt:   task.CreatedAt.Time,
		UpdatedAt:   task.UpdatedAt.Time,
	}
	// Write response
	internal.WriteJSONResponse(w, http.StatusOK, resp)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRequest
	err := internal.ParseRequestBody(h.validator, r, &req)
	if err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	newTask, err := h.store.Create(ctx, req.Title)
	if err != nil {
		h.logger.Error("Failed to create task", zap.Error(err))
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	resp := Response{
		ID:          newTask.ID,
		Labels:      newTask.Labels,
		Title:       newTask.Title,
		Description: newTask.Description.String,
		Status:      newTask.Status,
		DueDate:     newTask.DueDate.Time,
		CreatedAt:   newTask.CreatedAt.Time,
		UpdatedAt:   newTask.UpdatedAt.Time,
	}
	// Write response
	internal.WriteJSONResponse(w, http.StatusCreated, resp)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract task ID from URL
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var req UpdateRequest
	err = internal.ParseRequestBody(h.validator, r, &req)
	if err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedTask, err := h.store.Update(ctx, int32(id), req.Labels, req.Title, req.Description, req.Status, req.DueDate)
	if err != nil {
		h.logger.Error("Failed to update task", zap.Error(err))
		http.Error(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	resp := Response{
		ID:          updatedTask.ID,
		Labels:      updatedTask.Labels,
		Title:       updatedTask.Title,
		Description: updatedTask.Description.String,
		Status:      updatedTask.Status,
		DueDate:     updatedTask.DueDate.Time,
		CreatedAt:   updatedTask.CreatedAt.Time,
		UpdatedAt:   updatedTask.UpdatedAt.Time,
	}
	// Write response
	internal.WriteJSONResponse(w, http.StatusOK, resp)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract task ID from URL
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	err = h.store.Delete(ctx, int32(id))
	if err != nil {
		h.logger.Error("Failed to delete task", zap.Error(err))
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
