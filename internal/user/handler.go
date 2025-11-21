package user

import (
	"advanced-backend/internal/jwt"
	"context"
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
)

type Store interface {
	Create(ctx context.Context, email, username, avatarURL string) (User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	Update(ctx context.Context, id uuid.UUID, about string) (User, error)
}

type Request struct {
	About string `json:"about" validate:"max=500"`
}

type Response struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	About     string `json:"about"`
	AvatarURL string `json:"avatarUrl"`
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

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := ctx.Value(jwt.UserContextKey).(uuid.UUID)

	user, err := h.store.GetByID(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get user by ID", zap.Error(err))
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	resp := Response{
		ID:        user.ID.String(),
		Email:     user.Email,
		Username:  user.Username,
		About:     user.AboutMe.String,
		AvatarURL: user.AvatarUrl.String,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	userID := ctx.Value(jwt.UserContextKey).(uuid.UUID)

	user, err := h.store.Update(ctx, userID, req.About)
	if err != nil {
		h.logger.Error("Failed to update user", zap.Error(err))
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	resp := Response{
		ID:        user.ID.String(),
		Email:     user.Email,
		Username:  user.Username,
		About:     user.AboutMe.String,
		AvatarURL: user.AvatarUrl.String,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
