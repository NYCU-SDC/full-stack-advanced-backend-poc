package auth

import (
	"advanced-backend/internal/auth/oauthprovider"
	"advanced-backend/internal/jwt"
	"advanced-backend/internal/user"
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"net/http"
)

type OAuthProvider interface {
	Name() string
	Config() *oauth2.Config
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (oauthprovider.UserInfo, error)
}

type jwtService interface {
	New(ctx context.Context, userID uuid.UUID, email string) (string, error)
	CreateRefreshToken(ctx context.Context, userID uuid.UUID) (jwt.RefreshToken, error)
	InactivateRefreshTokenByUserID(ctx context.Context, userID uuid.UUID) error
}

type userStore interface {
	FindOrCreate(ctx context.Context, email, username, avatarURL string) (user.User, error)
}

type Handler struct {
	logger             *zap.Logger
	baseURL            string
	googleClientID     string
	googleClientSecret string
	jwtService         jwtService
	userStore          userStore
	provider           map[string]OAuthProvider
}

func NewHandler(logger *zap.Logger, baseURL, googleClientID, googleClientSecret string, jwtService jwtService, userStore userStore) *Handler {
	return &Handler{
		logger:     logger,
		jwtService: jwtService,
		baseURL:    baseURL,
		userStore:  userStore,
		provider: map[string]OAuthProvider{
			"google": oauthprovider.NewGoogleConfig(
				googleClientID,
				googleClientSecret,
				fmt.Sprintf("%s/api/oauth/google/callback", baseURL)),
		},
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	providerName := "google"
	provider := h.provider[providerName]
	if provider == nil {
		h.logger.Warn("No such provider", zap.String("provider", providerName))
		http.Error(w, "Unsupported OAuth2 provider", http.StatusBadRequest)
		return
	}

	redirectTo := r.URL.Query().Get("c")
	frontendRedirectTo := r.URL.Query().Get("r")
	if redirectTo == "" {
		redirectTo = fmt.Sprintf("%s/api/oauth/debug/token", h.baseURL)
	}
	if frontendRedirectTo != "" {
		redirectTo = fmt.Sprintf("%s?r=%s", redirectTo, frontendRedirectTo)
	}

	authURL := provider.Config().AuthCodeURL(redirectTo, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	h.logger.Info("Redirecting to Google OAuth2", zap.String("url", authURL))
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := "google"
	provider := h.provider[providerName]
	if provider == nil {
		h.logger.Warn("No such provider", zap.String("provider", providerName))
		http.Error(w, "Unsupported OAuth2 provider", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	redirectTo := state
	if redirectTo == "" {
		redirectTo = fmt.Sprintf("%s/api/oauth/debug/token", h.baseURL)
	}

	authError := r.URL.Query().Get("error")
	if authError != "" {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, authError)
		h.logger.Warn("OAuth2 callback returned error", zap.String("error", authError))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, "missing_code")
		h.logger.Warn("Missing code in callback")
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}
	token, err := provider.Exchange(r.Context(), code)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to exchange code for token", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	userInfo, err := provider.GetUserInfo(r.Context(), token)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to get user info", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	user, err := h.userStore.FindOrCreate(r.Context(), userInfo.Email, userInfo.Name, userInfo.Picture)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to find or create user", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	jwtToken, err := h.jwtService.New(r.Context(), user.ID, user.Email)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to create JWT token", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	refreshToken, err := h.jwtService.CreateRefreshToken(r.Context(), user.ID)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to create refresh token", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	redirectTo = fmt.Sprintf("%s?access_token=%s&refresh_token=%s", redirectTo, jwtToken, refreshToken.ID.String())

	http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
	h.logger.Info("OAuth2 callback successful", zap.String("user_email", userInfo.Email))
}

func (h *Handler) DebugToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, err := w.Write([]byte(`{"message":"Login successful"}`))
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(jwt.UserContextKey).(uuid.UUID)
	if !ok {
		h.logger.Warn("No user in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.jwtService.InactivateRefreshTokenByUserID(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to inactivate refresh tokens", zap.Error(err))
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	h.logger.Info("User logged out successfully", zap.String("user_id", userID.String()))
}
