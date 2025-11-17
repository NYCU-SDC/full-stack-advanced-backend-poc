package main

import (
	"advanced-backend/databaseutil"
	"advanced-backend/internal/auth"
	"advanced-backend/internal/config"
	"advanced-backend/internal/cors"
	"advanced-backend/internal/jwt"
	"advanced-backend/internal/task"
	"advanced-backend/internal/user"
	"context"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"log"
	"net/http"
	"time"
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
	userService := user.NewService(logger, dbPool)
	jwtService := jwt.NewService(logger, 15*time.Minute, 30*time.Minute, dbPool)

	taskHandler := task.NewHandler(logger, validator, taskService)
	jwtHandler := jwt.NewHandler(logger, jwtService)
	authHandler := auth.NewHandler(logger, cfg.BaseURL, cfg.GoogleClientID, cfg.GoogleClientSecret, jwtService, userService)

	jwtMiddleware := jwt.NewMiddleware(logger, jwtService)

	corsMiddleware := cors.NewMiddleware(logger, cfg.AllowOrigins)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/task", taskHandler.GetAll)
	mux.HandleFunc("GET /api/task/{id}", taskHandler.GetByID)
	mux.HandleFunc("POST /api/task", jwtMiddleware.HandlerFunc(taskHandler.Create))
	mux.HandleFunc("PUT /api/task/{id}", jwtMiddleware.HandlerFunc(taskHandler.Update))
	mux.HandleFunc("DELETE /api/task/{id}", jwtMiddleware.HandlerFunc(taskHandler.Delete))

	mux.HandleFunc("GET /api/login/google", authHandler.Login)
	mux.HandleFunc("GET /api/oauth/google/callback", authHandler.Callback)
	mux.HandleFunc("GET /api/logout", jwtMiddleware.HandlerFunc(authHandler.Logout))
	mux.HandleFunc("GET /api/refreshToken/{refreshToken}", jwtHandler.RefreshToken)

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
