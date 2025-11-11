package main

import (
	"advanced-backend/databaseutil"
	"advanced-backend/internal/config"
	"advanced-backend/internal/cors"
	"advanced-backend/internal/task"
	"context"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"log"
	"net/http"
)

func main() {
	cfg, cfgLog := config.Load()
	err := cfg.Validate()
	if err != nil {
		if errors.Is(err, config.ErrDatabaseURLRequired) {
			message := "Please set the DATABASE_URL environment variable or provide a config file with the database_url key."
			log.Fatal(message)
		} else {
			log.Fatalf("Failed to validate config: %v, exiting...", err)
		}
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	cfgLog.FlushToZap(logger)

	logger.Info("Starting backend service")

	err = databaseutil.MigrationUp(cfg.MigrationSource, cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatal("Failed to run database migration", zap.Error(err))
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
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

	corsMiddleware := cors.NewMiddleware(logger, cfg.AllowOrigins)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/task", taskHandler.GetAll)
	mux.HandleFunc("GET /api/task/{id}", taskHandler.GetByID)
	mux.HandleFunc("POST /api/task", taskHandler.Create)
	mux.HandleFunc("PUT /api/task/{id}", taskHandler.Update)
	mux.HandleFunc("DELETE /api/task/{id}", taskHandler.Delete)

	server := &http.Server{
		Addr:    ":8080",
		Handler: corsMiddleware.HandlerFunc(mux.ServeHTTP),
	}

	logger.Info("Backend started on :8080")

	err = server.ListenAndServe()
	if err != nil {
		logger.Fatal("Failed to start HTTP server", zap.Error(err))
	}
}
