package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"chs/backend/internal/auth"
	"chs/backend/internal/config"
	"chs/backend/internal/database"
	"chs/backend/internal/platform"
	"chs/backend/internal/society"
)

// Server wraps the HTTP server instance.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
	db         *database.DB
}

// New creates a new Server instance.
func New(cfg *config.Config, logger *slog.Logger, db *database.DB, authService *auth.Service, tokenManager *auth.TokenManager, platformRepo *platform.Repository, societyRepo *society.Repository) *Server {
	router := SetupRouter(cfg, logger, db, authService, tokenManager, platformRepo, societyRepo)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
		logger:     logger,
		db:         db,
	}
}

// Start launches the HTTP server listening on the configured port.
func (s *Server) Start() error {
	s.logger.Info("starting backend server", slog.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failure: %w", err)
	}
	return nil
}

// Shutdown initiates graceful shutdown of the HTTP server and database pool.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down backend server...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	if s.db != nil {
		s.db.Close()
	}

	s.logger.Info("server shutdown complete")
	return nil
}
