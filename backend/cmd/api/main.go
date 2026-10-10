package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chs/backend/internal/auth"
	"chs/backend/internal/config"
	"chs/backend/internal/database"
	"chs/backend/internal/platform"
	"chs/backend/internal/server"
	"chs/backend/internal/sms"
	"chs/backend/internal/society"
)

func main() {
	// Initialize structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg := config.Load()
	logger.Info("configuration loaded",
		slog.String("port", cfg.Port),
		slog.String("env", cfg.Env),
	)

	// Initialize database connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Warn("failed to connect to database at startup (will continue without database)",
			slog.String("error", err.Error()),
		)
	} else if db.IsConnected() {
		logger.Info("database connection pool initialized")
	}

	// Initialize Token Manager (Ed25519)
	tokenManager, err := auth.NewTokenManager(cfg.JWTPrivateKey)
	if err != nil {
		logger.Error("failed to initialize token manager", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize SMS Provider and Rate Limiter
	smsProvider := sms.NewMockProvider(logger)
	rateLimiter := auth.NewRateLimiter()

	// Initialize Services & Repositories
	var authService *auth.Service
	var platformRepo *platform.Repository
	var societyRepo *society.Repository

	if db != nil && db.IsConnected() {
		authService = auth.NewService(
			db.Pool(),
			tokenManager,
			smsProvider,
			rateLimiter,
			auth.ServiceConfig{
				DevTestPhones: cfg.DevTestPhones,
				DevTestOTP:    cfg.DevTestOTP,
			},
			logger,
		)
		logger.Info("authentication service initialized")

		platformRepo = platform.NewRepository(db.Pool())
		societyRepo = society.NewRepository(db.Pool())
		bootstrapCtx, bootstrapCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer bootstrapCancel()

		if role, err := platformRepo.BootstrapSuperAdmin(bootstrapCtx, cfg.SuperAdminPhone); err != nil {
			logger.Warn("super admin bootstrap skipped or failed", slog.String("error", err.Error()))
		} else {
			logger.Info("super admin bootstrap verified",
				slog.String("user_id", role.UserID.String()),
				slog.String("role", role.Role),
				slog.String("status", role.Status),
			)
		}
	}

	// Initialize HTTP server
	srv := server.New(cfg, logger, db, authService, tokenManager, platformRepo, societyRepo)

	// Start server in background
	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("server startup error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Listen for shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.String("error", err.Error()))
	}
}
