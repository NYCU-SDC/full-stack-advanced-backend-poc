package main

import (
	"advanced-backend/databaseutil"
	"advanced-backend/internal/task"
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	logger.Info("Starting backend service")

	err = databaseutil.MigrationUp("file://internal/database/migrations", "postgresql://postgres:password@localhost:5432/postgres?sslmode=disable", logger)
	if err != nil {
		logger.Fatal("Failed to run database migration", zap.Error(err))
	}

	poolConfig, err := pgxpool.ParseConfig("postgresql://postgres:password@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		logger.Fatal("Failed to parse database URL", zap.Error(err))
	}

	dbPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Fatal("Failed to create database connection pool", zap.Error(err))
	}
	defer dbPool.Close()

	validator := validator.New()

	taskService := task.NewService(logger, dbPool)

	taskHandler := task.NewHandler(logger, validator, taskService)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/task", taskHandler.GetAll)
	mux.HandleFunc("GET /api/task/{id}", taskHandler.GetByID)
	mux.HandleFunc("POST /api/task", taskHandler.Create)
	mux.HandleFunc("PUT /api/task/{id}", taskHandler.Update)
	mux.HandleFunc("DELETE /api/task/{id}", taskHandler.Delete)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	logger.Info("Backend started on :8080")

	err = server.ListenAndServe()
	if err != nil {
		logger.Fatal("Failed to start HTTP server", zap.Error(err))
	}
}
