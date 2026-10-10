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
	"chs/backend/internal/platform"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

// SetupRouter initializes the Chi router with required middlewares and routes.
func SetupRouter(cfg *config.Config, logger *slog.Logger, db *database.DB, authService *auth.Service, tokenManager *auth.TokenManager, platformRepo *platform.Repository, societyRepo *society.Repository) http.Handler {
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
				authRouter.Post("/otp/verify", VerifyOTPHandler(authService, platformRepo))
				authRouter.Post("/refresh", RefreshTokenHandler(authService))

				// Authenticated endpoints
				if tokenManager != nil {
					authRouter.Group(func(protected chi.Router) {
						protected.Use(middleware.Authenticate(tokenManager, authService))
						protected.Get("/me", MeHandler(authService, platformRepo, societyRepo))
						protected.Post("/logout", LogoutHandler(authService))
						protected.Post("/logout-all", LogoutAllHandler(authService))
					})
				}
			})
		}

		if platformRepo != nil && tokenManager != nil {
			api.Route("/platform", func(platRouter chi.Router) {
				platRouter.Use(middleware.Authenticate(tokenManager, authService))
				platRouter.Use(middleware.RequireSuperAdmin(platformRepo))

				platRouter.Get("/admins", ListAdminsHandler(platformRepo))
				platRouter.Post("/admins", CreateAdminHandler(platformRepo))
				platRouter.Patch("/admins/{id}/status", UpdateAdminStatusHandler(platformRepo))
				platRouter.Get("/audit-logs", ListAuditLogsHandler(platformRepo))

				if societyRepo != nil {
					platRouter.Get("/societies", ListSocietiesHandler(societyRepo))
					platRouter.Post("/societies", CreateSocietyHandler(societyRepo))
					platRouter.Get("/societies/{id}", GetSocietyHandler(societyRepo))
					platRouter.Patch("/societies/{id}", UpdateSocietyHandler(societyRepo))
					platRouter.Post("/societies/{id}/activate", ActivateSocietyHandler(societyRepo))
					platRouter.Post("/societies/{id}/suspend", SuspendSocietyHandler(societyRepo))
					platRouter.Post("/societies/{id}/reactivate", ReactivateSocietyHandler(societyRepo))
					platRouter.Get("/society-suspension-reasons", ListSuspensionReasonsHandler(societyRepo))

					// S04 Society Administration platform endpoints (SUPER_ADMIN only)
					platRouter.Post("/societies/{id}/admins", AssignSocietyAdminHandler(societyRepo))
					platRouter.Get("/societies/{id}/admins", ListSocietyAdminsHandler(societyRepo))
					platRouter.Delete("/societies/{id}/admins/{assignment_id}", RevokeSocietyAdminHandler(societyRepo))
				}
			})
		}

		// S04 & S05 Society-scoped endpoints (RequireSocietyAccess: SUPER_ADMIN or SOC_ADMIN)
		if societyRepo != nil && tokenManager != nil {
			api.Route("/societies/{id}", func(socRouter chi.Router) {
				socRouter.Use(middleware.Authenticate(tokenManager, authService))
				socRouter.Use(middleware.RequireSocietyAccess(society.RoleSocietyAdmin, societyRepo, platformRepo))

				socRouter.Get("/profile", GetSocietyProfileHandler(societyRepo))

				// S05 Society Structure & Formation
				socRouter.Get("/buildings", ListBuildingsHandler(societyRepo))
				socRouter.Post("/buildings", CreateBuildingHandler(societyRepo))
				socRouter.Post("/buildings/{building_id}/wings", CreateWingHandler(societyRepo))
				socRouter.Post("/units/preview", PreviewUnitsHandler())
				socRouter.Post("/units/generate", GenerateUnitsHandler(societyRepo))
				socRouter.Get("/units", ListUnitsHandler(societyRepo))
				socRouter.Get("/gates", ListGatesHandler(societyRepo))
				socRouter.Post("/gates", CreateGateHandler(societyRepo))
				socRouter.Get("/staffing", GetStaffingHandler(societyRepo))
				socRouter.Put("/staffing", SetStaffingHandler(societyRepo))
				socRouter.Get("/formation-summary", GetFormationSummaryHandler(societyRepo))
			})
		}
	})

	return r
}
