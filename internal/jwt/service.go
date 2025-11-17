package jwt

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"strings"
	"time"
)

const secret = "default_secret"

var (
	ErrInvalidRefreshToken = errors.New("invalid token")
)

type Service struct {
	logger                 *zap.Logger
	expiration             time.Duration
	refreshTokenExpiration time.Duration
	queries                *Queries
}

func NewService(logger *zap.Logger, expiration time.Duration, refreshTokenExpiration time.Duration, db DBTX) *Service {
	return &Service{
		logger:                 logger,
		expiration:             expiration,
		refreshTokenExpiration: refreshTokenExpiration,
		queries:                New(db),
	}
}

type claims struct {
	UserID uuid.UUID
	Email  string
	jwt.RegisteredClaims
}

func (s Service) New(ctx context.Context, userID uuid.UUID, email string) (string, error) {
	jwtID := uuid.New()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "Backend-Training",
			Subject:   "Backend-Training Token",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiration)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        jwtID.String(),
		},
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		s.logger.Error("Failed to sign token", zap.Error(err))
		return "", err
	}

	s.logger.Debug("Generated new JWT token")

	return tokenString, nil
}

func (s Service) Parse(ctx context.Context, tokenString string) (User, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenMalformed):
			s.logger.Warn("Failed to parse JWT token due to malformed structure, this is not a JWT token", zap.String("error", err.Error()))
			return User{}, err
		case errors.Is(err, jwt.ErrSignatureInvalid):
			s.logger.Warn("Failed to parse JWT token due to invalid signature", zap.String("error", err.Error()))
			return User{}, err
		case errors.Is(err, jwt.ErrTokenExpired):
			expiredTime, getErr := token.Claims.GetExpirationTime()
			if getErr != nil {
				s.logger.Warn("Failed to parse JWT token due to expired timestamp", zap.String("error", err.Error()))
			} else {
				s.logger.Warn("Failed to parse JWT token due to expired timestamp", zap.String("error", err.Error()), zap.Time("expired_at", expiredTime.Time))
			}

			return User{}, err
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			notBeforeTime, getErr := token.Claims.GetNotBefore()
			if getErr != nil {
				s.logger.Warn("Failed to parse JWT token due to not valid yet timestamp", zap.String("error", err.Error()))
			} else {
				s.logger.Warn("Failed to parse JWT token due to not valid yet timestamp", zap.String("error", err.Error()), zap.Time("not_valid_yet", notBeforeTime.Time))
			}

			return User{}, err
		default:
			s.logger.Error("Failed to parse or validate JWT token", zap.Error(err))
			return User{}, err
		}
	}

	c, ok := token.Claims.(*claims)
	if !ok {
		s.logger.Warn("Invalid JWT token claims")
		return User{}, errors.New("invalid token claims")
	}

	s.logger.Debug("Parsed JWT token successfully")

	return User{
		ID:    c.UserID,
		Email: c.Email,
	}, nil
}

func (s Service) CreateRefreshToken(ctx context.Context, userID uuid.UUID) (RefreshToken, error) {
	expirationDate := time.Now().Add(s.refreshTokenExpiration)

	token, err := s.queries.Create(ctx, CreateParams{
		UserID:         userID,
		ExpirationDate: pgtype.Timestamptz{Time: expirationDate, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to create refresh token", zap.Error(err))
		return RefreshToken{}, err
	}

	s.logger.Info("Created refresh token", zap.String("token_id", token.ID.String()), zap.String("user_id", userID.String()), zap.Time("expiration_date", expirationDate))

	return token, nil
}

func (s Service) ValidateRefreshToken(ctx context.Context, id uuid.UUID) (User, error) {
	refreshToken, err := s.queries.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get refresh token by ID", zap.Error(err))
		return User{}, err
	}

	// Check if the refresh token is expired
	if refreshToken.ExpirationDate.Time.Before(time.Now()) {
		err = ErrInvalidRefreshToken
		s.logger.Error("Refresh token is expired", zap.String("token_id", id.String()), zap.Time("expiration_date", refreshToken.ExpirationDate.Time))
		return User{}, err
	}

	// Check if the refresh token is active
	if !refreshToken.IsAvailable.Bool {
		err = ErrInvalidRefreshToken
		s.logger.Error("Refresh token is not active", zap.String("token_id", id.String()))
		return User{}, err
	}

	jwtUser, err := s.queries.GetUserByRefreshToken(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get user by refresh token", zap.Error(err))
		return User{}, err
	}

	_, err = s.queries.Inactivate(ctx, id)
	if err != nil {
		s.logger.Error("Failed to inactivate refresh token after use", zap.Error(err))
		return User{}, err
	}

	s.logger.Info("Validated refresh token", zap.String("token_id", id.String()), zap.String("user_id", jwtUser.ID.String()))

	return jwtUser, nil
}

func (s Service) InactivateRefreshTokenByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := s.queries.InactivateByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to inactivate refresh token", zap.Error(err))
		return err
	}

	s.logger.Info("Inactivated refresh token", zap.String("user_id", userID.String()))
	return nil
}
