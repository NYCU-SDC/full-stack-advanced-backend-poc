package user

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type Service struct {
	logger  *zap.Logger
	queries *Queries
}

func NewService(logger *zap.Logger, db DBTX) *Service {
	return &Service{
		logger:  logger,
		queries: New(db),
	}
}

func (s *Service) FindOrCreate(ctx context.Context, email, username, avatarURL string) (User, error) {
	exists, err := s.queries.ExistsByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to check user existence by email", zap.Error(err))
		return User{}, err
	}

	if !exists {
		user, err := s.queries.Create(ctx, CreateParams{
			Email:     email,
			Username:  username,
			AvatarUrl: pgtype.Text{String: avatarURL, Valid: avatarURL != ""},
		})
		if err != nil {
			s.logger.Error("Failed to create user", zap.Error(err))
			return User{}, err
		}

		s.logger.Info("Created user", zap.String("user_id", user.ID.String()), zap.String("email", user.Email))
		return user, nil
	}

	user, err := s.queries.GetByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to get user by email", zap.Error(err))
		return User{}, err
	}

	s.logger.Info("Found existing user", zap.String("user_id", user.ID.String()), zap.String("email", user.Email))
	return user, nil
}

func (s *Service) Create(ctx context.Context, email, username, avatarURL string) (User, error) {
	newUser, err := s.queries.Create(ctx, CreateParams{
		Email:     email,
		Username:  username,
		AvatarUrl: pgtype.Text{String: avatarURL, Valid: avatarURL != ""},
	})
	if err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		return User{}, err
	}

	s.logger.Info("Created user", zap.String("user_id", newUser.ID.String()), zap.String("email", newUser.Email))

	return newUser, nil
}

func (s *Service) GetByID(ctx context.Context, userID uuid.UUID) (User, error) {
	user, err := s.queries.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user by ID", zap.Error(err))
		return User{}, err
	}

	s.logger.Info("Retrieved user by ID", zap.String("user_id", user.ID.String()), zap.String("email", user.Email))
	return user, nil
}

func (s *Service) Exists(ctx context.Context, userID uuid.UUID) (bool, error) {
	exists, err := s.queries.Exist(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to check user existence", zap.Error(err))
		return false, err
	}

	return exists, nil
}

func (s *Service) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	exists, err := s.queries.ExistsByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to check user existence by email", zap.Error(err))
		return false, err
	}

	return exists, nil
}

func (s *Service) Update(ctx context.Context, userID uuid.UUID, about string) (User, error) {
	updatedUser, err := s.queries.UpdateAbout(ctx, UpdateAboutParams{
		ID:      userID,
		AboutMe: pgtype.Text{String: about, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to update user about", zap.Error(err))
		return User{}, err
	}

	s.logger.Info("Updated user about", zap.String("user_id", updatedUser.ID.String()), zap.String("about", updatedUser.AboutMe.String))
	return updatedUser, nil
}
