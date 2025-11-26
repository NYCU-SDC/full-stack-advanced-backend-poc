package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
)

type jwtService interface {
	New(ctx context.Context, userID uuid.UUID, email string) (string, error)
	ValidateRefreshToken(ctx context.Context, refreshToken uuid.UUID) (User, error)
	CreateRefreshToken(ctx context.Context, userID uuid.UUID) (RefreshToken, error)
}

type Response struct {
	AccessToken    string `json:"access_token"`
	ExpirationTime int64  `json:"expiration"`
	RefreshToken   string `json:"refresh_token"`
}

type Handler struct {
	logger    *zap.Logger
	jwtIssuer jwtService
}

func NewHandler(logger *zap.Logger, jwtIssuer jwtService) *Handler {
	return &Handler{
		logger:    logger,
		jwtIssuer: jwtIssuer,
	}
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validate the request and extract the refresh token
	pathRefreshToken := r.PathValue("refreshToken")
	if pathRefreshToken == "" {
		http.Error(w, "Refresh token is required", http.StatusBadRequest)
		return
	}
	refreshTokenID, err := uuid.Parse(pathRefreshToken)
	if err != nil {
		http.Error(w, "Invalid refresh token format", http.StatusBadRequest)
		return
	}

	// Get the user associated with the refresh token
	jwtUser, err := h.jwtIssuer.ValidateRefreshToken(ctx, refreshTokenID)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			http.Error(w, "Invalid refresh token", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to get user by refresh token", http.StatusInternalServerError)
		return
	}

	// Generate a new JWT and refresh token
	jwtToken, err := h.jwtIssuer.New(ctx, jwtUser.ID, jwtUser.Email)
	if err != nil {
		http.Error(w, "Failed to generate new JWT", http.StatusInternalServerError)
		return
	}

	newRefreshToken, err := h.jwtIssuer.CreateRefreshToken(ctx, jwtUser.ID)
	if err != nil {
		http.Error(w, "Failed to generate new refresh token", http.StatusInternalServerError)
		return
	}

	response := Response{
		AccessToken:    jwtToken,
		ExpirationTime: newRefreshToken.ExpirationDate.Time.Unix(),
		RefreshToken:   newRefreshToken.ID.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
