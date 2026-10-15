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
	"chs/backend/internal/notification"
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
	if err := cfg.Validate(); err != nil {
		logger.Error("configuration validation failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("configuration loaded and validated",
		slog.String("port", cfg.Port),
		slog.String("env", cfg.Env),
	)

	// Initialize database connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		if cfg.IsProduction() {
			logger.Error("database connection failed in production mode", slog.String("error", err.Error()))
			os.Exit(1)
		}
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
	var smsProvider sms.Provider
	if cfg.IsProduction() {
		if cfg.SMSProvider == "msg91" {
			smsProvider = sms.NewMSG91Provider(sms.MSG91Config{
				BaseURL:    cfg.MSG91BaseURL,
				AuthKey:    cfg.MSG91AuthKey,
				TemplateID: cfg.MSG91TemplateID,
				SenderID:   cfg.MSG91SenderID,
				EntityID:   cfg.MSG91EntityID,
				Timeout:    5 * time.Second,
			}, logger)
			logger.Info("production MSG91 SMS provider configured",
				slog.String("template_id", cfg.MSG91TemplateID),
				slog.String("sender_id", cfg.MSG91SenderID),
			)
		} else {
			logger.Error("unsupported or unconfigured SMS_PROVIDER in production mode", slog.String("provider", cfg.SMSProvider))
			os.Exit(1)
		}
	} else {
		if cfg.SMSProvider == "msg91" && cfg.MSG91AuthKey != "" {
			smsProvider = sms.NewMSG91Provider(sms.MSG91Config{
				BaseURL:    cfg.MSG91BaseURL,
				AuthKey:    cfg.MSG91AuthKey,
				TemplateID: cfg.MSG91TemplateID,
				SenderID:   cfg.MSG91SenderID,
				EntityID:   cfg.MSG91EntityID,
				Timeout:    5 * time.Second,
			}, logger)
			logger.Info("MSG91 SMS provider configured in dev mode")
		} else {
			smsProvider = sms.NewMockProvider(logger)
			logger.Info("using Mock SMS Provider in non-production mode")
		}
	}
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
		var emailProvider notification.EmailProvider
		if cfg.ResendAPIKey != "" {
			emailProvider = notification.NewResendEmailProvider(cfg.ResendAPIKey, cfg.ResendFromEmail)
			logger.Info("resend email provider configured", slog.String("from_email", cfg.ResendFromEmail))
		} else {
			logger.Warn("resend email provider not configured (missing RESEND_API_KEY)")
		}

		var pushProvider notification.PushProvider
		if cfg.FirebaseProjectID != "" {
			fcmProv, err := notification.NewFCMPushProvider(context.Background(), cfg.FirebaseProjectID, cfg.FirebaseCredentialsFile)
			if err != nil {
				logger.Warn("firebase fcm push provider unavailable", slog.String("error", err.Error()))
			} else {
				pushProvider = fcmProv
				logger.Info("firebase fcm push provider configured", slog.String("project_id", cfg.FirebaseProjectID))
			}
		}

		notifRepo := notification.NewRepository(db.Pool())
		notifWorker := notification.NewWorker(notifRepo, emailProvider, pushProvider, logger)
		workerCtx, workerCancel := context.WithCancel(context.Background())
		defer workerCancel()

		go notifWorker.Start(workerCtx, 2*time.Second)
		logger.Info("notification delivery worker started")
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
