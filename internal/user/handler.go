package user

import (
	"context"
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"net/http"
)

type Store interface {
	Create(ctx context.Context, email string) (User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type Request struct {
	Email string `json:"email" validate:"required,email"`
}

type Response struct {
	ID    string `json:"id"`
	Email string `json:"email"`
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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	h.logger.Info("CreateUser handler called", zap.Any("context", ctx))
	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = h.validator.Struct(req)
	if err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	exists, err := h.store.ExistsByEmail(ctx, req.Email)
	if err != nil {
		h.logger.Error("Failed to check if user exists", zap.Error(err))
		http.Error(w, "Failed to check user existence", http.StatusInternalServerError)
		return
	}
	if exists {
		h.logger.Warn("User with this email already exists", zap.String("email", req.Email))
		http.Error(w, "User with this email already exists", http.StatusConflict)
		return
	}

	newUser, err := h.store.Create(ctx, req.Email)
	if err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	resp := Response{
		ID:    newUser.ID.String(),
		Email: newUser.Email,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
