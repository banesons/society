package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"chs/backend/internal/auth"
	"chs/backend/internal/config"
	"chs/backend/internal/database"
	"chs/backend/internal/middleware"
	"chs/backend/internal/response"
)

// SetupRouter initializes the Chi router with required middlewares and routes.
func SetupRouter(cfg *config.Config, logger *slog.Logger, db *database.DB, authService *auth.Service, tokenManager *auth.TokenManager) http.Handler {
	r := chi.NewRouter()

	// Base middleware stack
	r.Use(middleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.StructuredLogger(logger))
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Custom 404 handler with standard error envelope
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		reqID := middleware.GetRequestID(req.Context())
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "The requested resource was not found", reqID)
	})

	// Custom 405 handler with standard error envelope
	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		reqID := middleware.GetRequestID(req.Context())
		response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed for this resource", reqID)
	})

	// Mount versioned API routes
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", HealthHandler(db))

		if authService != nil {
			api.Route("/auth", func(authRouter chi.Router) {
				// Public OTP flows
				authRouter.Post("/otp/send", SendOTPHandler(authService))
				authRouter.Post("/otp/verify", VerifyOTPHandler(authService))
				authRouter.Post("/refresh", RefreshTokenHandler(authService))

				// Authenticated endpoints
				if tokenManager != nil {
					authRouter.Group(func(protected chi.Router) {
						protected.Use(middleware.Authenticate(tokenManager))
						protected.Get("/me", MeHandler(authService))
						protected.Post("/logout", LogoutHandler(authService))
						protected.Post("/logout-all", LogoutAllHandler(authService))
					})
				}
			})
		}
	})

	return r
}
