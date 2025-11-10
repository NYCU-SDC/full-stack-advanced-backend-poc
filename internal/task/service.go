package task

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"time"
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

func (s Service) GetAll(ctx context.Context) ([]Task, error) {
	tasks, err := s.queries.GetAll(ctx)
	if err != nil {
		s.logger.Error("Failed to get all tasks", zap.Error(err))
		return nil, err
	}
	return tasks, nil
}

func (s Service) GetByID(ctx context.Context, id int32) (Task, error) {
	task, err := s.queries.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get task by ID", zap.Error(err))
		return Task{}, err
	}
	return task, nil
}

func (s Service) Create(ctx context.Context, title string) (Task, error) {
	task, err := s.queries.Create(ctx, title)
	if err != nil {
		s.logger.Error("Failed to create task", zap.Error(err))
		return Task{}, err
	}
	return task, nil
}

func (s Service) Update(ctx context.Context,
	id int32,
	labels []string,
	title, description string,
	status TaskStatus,
	dueDate time.Time) (Task, error) {
	updatedTask, err := s.queries.Update(ctx, UpdateParams{
		ID:          id,
		Labels:      labels,
		Title:       title,
		Description: pgtype.Text{String: description, Valid: true},
		Status:      status,
		DueDate:     pgtype.Timestamptz{Time: dueDate, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to update task", zap.Error(err))
		return Task{}, err
	}
	return updatedTask, nil
}

func (s Service) Delete(ctx context.Context, id int32) error {
	err := s.queries.Delete(ctx, id)
	if err != nil {
		s.logger.Error("Failed to delete task", zap.Error(err))
		return err
	}
	return nil
}
