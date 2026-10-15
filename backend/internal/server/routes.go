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
	"chs/backend/internal/notification"
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
	r.Use(chimiddleware.RequestSize(12 << 20)) // 12MB global body size limit (allows 10MB payload + multipart overhead)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	if societyRepo != nil && cfg != nil {
		societyRepo.SetStorageConfig(cfg.SupabaseURL, cfg.SupabaseAnonKey)
	}

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

	// Top-level health and liveness probes for container platforms and load balancers
	r.Get("/health", HealthHandler(db))
	r.Get("/healthz", HealthHandler(db))

	// Mount versioned API routes
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", HealthHandler(db))

		if authService != nil {
			api.Route("/auth", func(authRouter chi.Router) {
				// Public OTP flows
				authRouter.Post("/otp/send", SendOTPHandler(authService, cfg))
				authRouter.Post("/otp/verify", VerifyOTPHandler(authService, platformRepo, societyRepo))
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

		var notifRepo *notification.Repository
		if db != nil && db.Pool() != nil {
			notifRepo = notification.NewRepository(db.Pool())
		}

		if notifRepo != nil && tokenManager != nil {
			api.Route("/notifications", func(notifRouter chi.Router) {
				notifRouter.Use(middleware.Authenticate(tokenManager, authService))

				notifRouter.Get("/", ListNotificationsHandler(notifRepo))
				notifRouter.Get("/unread-count", GetUnreadNotificationCountHandler(notifRepo))
				notifRouter.Patch("/{id}/read", MarkNotificationReadHandler(notifRepo))
				notifRouter.Post("/mark-all-read", MarkAllNotificationsReadHandler(notifRepo))
				notifRouter.Post("/devices", RegisterDeviceHandler(notifRepo))
				notifRouter.Delete("/devices/{device_id}", DeactivateDeviceHandler(notifRepo))
				notifRouter.Get("/devices", ListUserDevicesHandler(notifRepo))
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
				platRouter.Get("/audit-logs/{id}", GetPlatformAuditLogDetailHandler(platformRepo))

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

					// S16 Platform Dashboard
					platRouter.Get("/dashboard/summary", GetPlatformDashboardHandler(societyRepo))
				}
			})
		}

		// S04, S05, S06, S07, S08 Society-scoped endpoints
		if societyRepo != nil && tokenManager != nil {
			api.Route("/societies/{id}", func(socRouter chi.Router) {
				socRouter.Use(middleware.Authenticate(tokenManager, authService))

				// Operational routes accessible to both SOC_ADMIN and GATEKEEPER
				socRouter.Group(func(opRouter chi.Router) {
					opRouter.Use(middleware.RequireAnySocietyRoleAccess(societyRepo, platformRepo, society.RoleSocietyAdmin, society.RoleGatekeeper))

					opRouter.Get("/gates", ListGatesHandler(societyRepo))
					opRouter.Get("/shifts", ListGateShiftsHandler(societyRepo))
					opRouter.Get("/duty-sessions", ListDutySessionsHandler(societyRepo))
					opRouter.Post("/gates/{gateId}/sessions/check-in", CheckInDutyHandler(societyRepo))
					opRouter.Post("/gates/{gateId}/sessions/check-out", CheckOutDutyHandler(societyRepo))

					// S09 Gate Visitor Operations
					opRouter.Get("/visitors", ListVisitsHandler(societyRepo))
					opRouter.Get("/visitors/dashboard", GetVisitorDashboardHandler(societyRepo))
					opRouter.Post("/visitors/walk-in", RegisterWalkInVisitHandler(societyRepo))
					opRouter.Post("/visitors/passes/verify", VerifyVisitorPassHandler(societyRepo))
					opRouter.Get("/visitors/{visitId}", GetVisitDetailsHandler(societyRepo))
					opRouter.Post("/visitors/{visitId}/check-in", CheckInVisitorHandler(societyRepo))
					opRouter.Post("/visitors/{visitId}/check-out", CheckOutVisitorHandler(societyRepo))

					// S16 Gate Shift Dashboard
					opRouter.Get("/gate/dashboard", GetGateDashboardHandler(societyRepo))
				})

				// S09 Resident and Admin Visitor Operations (Expected visits, approvals, passes, unit policy)
				socRouter.Group(func(residentRouter chi.Router) {
					residentRouter.Use(middleware.RequireSocietyMemberOrAdminAccess(societyRepo, platformRepo))

					residentRouter.Get("/profile", GetSocietyProfileHandler(societyRepo))
					residentRouter.Get("/units/my", GetResidentUnitsHandler(societyRepo))
					residentRouter.Patch("/units/{unitId}/visitor-policy", UpdateUnitVisitorPolicyHandler(societyRepo))
					residentRouter.Get("/visitors/pending", ListPendingVisitsHandler(societyRepo))

					residentRouter.Post("/visitors/expected", RegisterExpectedVisitHandler(societyRepo, platformRepo))
					residentRouter.Post("/visitors/{visitId}/approve", ApproveVisitHandler(societyRepo))
					residentRouter.Post("/visitors/{visitId}/reject", RejectVisitHandler(societyRepo))
					residentRouter.Post("/visitors/{visitId}/cancel", CancelVisitHandler(societyRepo))
					residentRouter.Post("/visitors/{visitId}/passes", IssueVisitorPassHandler(societyRepo))

					// S10 Society Communication - Resident Feed & Read Tracking
					residentRouter.Get("/announcements/feed", ListResidentAnnouncementFeedHandler(societyRepo))
					residentRouter.Get("/announcements/unread-count", GetUnreadAnnouncementCountHandler(societyRepo))
					residentRouter.Get("/announcements/{announcementId}", GetAnnouncementDetailsHandler(societyRepo, platformRepo))
					residentRouter.Post("/announcements/{announcementId}/read", MarkAnnouncementReadHandler(societyRepo))

					// S11 Maintenance Foundation - Resident My Bills
					residentRouter.Get("/maintenance/my-bills", ListResidentBillsHandler(societyRepo))
					residentRouter.Get("/maintenance/my-bills/{billId}", GetResidentBillDetailsHandler(societyRepo))
					// S12 Maintenance Operations & Collections - Resident Payments & Receipts
					residentRouter.Get("/maintenance/my-payments", ListResidentPaymentsHandler(societyRepo))
					residentRouter.Get("/maintenance/my-receipts/{receiptId}", GetResidentReceiptHandler(societyRepo))
				})

				// Administrative routes requiring SOC_ADMIN or SUPER_ADMIN
				socRouter.Group(func(adminRouter chi.Router) {
					adminRouter.Use(middleware.RequireSocietyAccess(society.RoleSocietyAdmin, societyRepo, platformRepo))

					// S16 Society Dashboard
					adminRouter.Get("/dashboard/summary", GetSocietyDashboardHandler(societyRepo))

					// S05 Society Structure & Formation
					adminRouter.Get("/buildings", ListBuildingsHandler(societyRepo))
					adminRouter.Post("/buildings", CreateBuildingHandler(societyRepo))
					adminRouter.Post("/buildings/{building_id}/wings", CreateWingHandler(societyRepo))
					adminRouter.Post("/units/preview", PreviewUnitsHandler())
					adminRouter.Post("/units/generate", GenerateUnitsHandler(societyRepo))
					adminRouter.Get("/units", ListUnitsHandler(societyRepo))
					adminRouter.Post("/gates", CreateGateHandler(societyRepo))
					adminRouter.Get("/staffing", GetStaffingHandler(societyRepo))
					adminRouter.Put("/staffing", SetStaffingHandler(societyRepo))
					adminRouter.Get("/formation-summary", GetFormationSummaryHandler(societyRepo))

					// S06 Members & Relationships
					adminRouter.Get("/members", ListMembersHandler(societyRepo))
					adminRouter.Post("/members", RegisterMemberHandler(societyRepo))
					adminRouter.Get("/members/{memberId}", GetMemberHandler(societyRepo))
					adminRouter.Patch("/members/{memberId}", UpdateMemberHandler(societyRepo))
					adminRouter.Post("/members/{memberId}/departure/preview", PreviewMemberDepartureHandler(societyRepo))
					adminRouter.Post("/members/{memberId}/departure/confirm", ConfirmMemberDepartureHandler(societyRepo))

					// S10 Society Communication - Admin Management
					adminRouter.Get("/announcements/admin", ListAdminAnnouncementsHandler(societyRepo))
					adminRouter.Post("/announcements", CreateAnnouncementHandler(societyRepo))
					adminRouter.Put("/announcements/{announcementId}", UpdateAnnouncementHandler(societyRepo))
					adminRouter.Post("/announcements/{announcementId}/publish", PublishAnnouncementHandler(societyRepo))
					adminRouter.Post("/announcements/{announcementId}/archive", ArchiveAnnouncementHandler(societyRepo))

					// S11 Maintenance Foundation - Admin Management
					adminRouter.Get("/maintenance/categories", ListChargeCategoriesHandler(societyRepo))
					adminRouter.Post("/maintenance/categories", CreateChargeCategoryHandler(societyRepo))
					adminRouter.Put("/maintenance/categories/{categoryId}", UpdateChargeCategoryHandler(societyRepo))
					adminRouter.Get("/maintenance/configs", ListChargeConfigsHandler(societyRepo))
					adminRouter.Post("/maintenance/configs", CreateChargeConfigHandler(societyRepo))
					adminRouter.Post("/maintenance/configs/{configId}/deactivate", DeactivateChargeConfigHandler(societyRepo))
					adminRouter.Get("/maintenance/periods", ListBillingPeriodsHandler(societyRepo))
					adminRouter.Post("/maintenance/periods", CreateBillingPeriodHandler(societyRepo))
					adminRouter.Get("/maintenance/periods/{periodId}/preview", PreviewBillGenerationHandler(societyRepo))
					adminRouter.Post("/maintenance/periods/{periodId}/generate", GenerateBillsHandler(societyRepo))
					adminRouter.Get("/maintenance/bills", ListBillsHandler(societyRepo))
					adminRouter.Get("/maintenance/bills/{billId}", GetBillDetailsHandler(societyRepo))
					adminRouter.Post("/maintenance/bills/{billId}/void", VoidBillHandler(societyRepo))

					// S12 Maintenance Operations & Collections - Admin Payments & Collections
					adminRouter.Post("/maintenance/payments", RecordPaymentHandler(societyRepo))
					adminRouter.Post("/maintenance/payments/{paymentId}/reverse", ReversePaymentHandler(societyRepo))
					adminRouter.Get("/maintenance/collections/register", ListCollectionRegisterHandler(societyRepo))
					adminRouter.Get("/maintenance/collections/summary", GetCollectionSummaryHandler(societyRepo))
					adminRouter.Get("/maintenance/receipts/{receiptId}", GetReceiptDetailsHandler(societyRepo))

					adminRouter.Get("/units/{unitId}/relationships", ListUnitRelationshipsHandler(societyRepo))
					adminRouter.Post("/units/{unitId}/relationships", CreateUnitRelationshipHandler(societyRepo))
					adminRouter.Post("/units/{unitId}/relationships/{relationshipId}/end", EndUnitRelationshipHandler(societyRepo))

					// S07 Society Roles & Designations
					adminRouter.Get("/designations", ListDesignationsHandler(societyRepo))
					adminRouter.Post("/designations", CreateDesignationHandler(societyRepo))
					adminRouter.Patch("/designations/{designationId}", UpdateDesignationHandler(societyRepo))
					adminRouter.Patch("/designations/{designationId}/status", SetDesignationStatusHandler(societyRepo))

					adminRouter.Get("/appointments", ListAppointmentsHandler(societyRepo))
					adminRouter.Post("/appointments", CreateAppointmentHandler(societyRepo))
					adminRouter.Post("/appointments/{appointmentId}/end", EndAppointmentHandler(societyRepo))
					adminRouter.Get("/committee/overview", GetCommitteeOverviewHandler(societyRepo))

					// S08 Gate Foundation Administration
					adminRouter.Patch("/gates/{gateId}", UpdateGateHandler(societyRepo))
					adminRouter.Get("/gatekeepers", ListGatekeepersHandler(societyRepo))
					adminRouter.Post("/gatekeepers", AssignGatekeeperHandler(societyRepo))
					adminRouter.Delete("/gatekeepers/{assignmentId}", RevokeGatekeeperHandler(societyRepo))
					adminRouter.Post("/shifts", CreateGateShiftHandler(societyRepo))

					// S14 Society Audit & Activity History
					adminRouter.Get("/audit-logs", ListSocietyAuditLogsHandler(platformRepo, societyRepo))
					adminRouter.Get("/audit-logs/{event_id}", GetSocietyAuditLogDetailHandler(platformRepo, societyRepo))
				})

				// S13 Documents & Records Endpoints
				socRouter.Route("/documents", func(docRouter chi.Router) {
					docRouter.Use(middleware.RequireSocietyMemberOrAdminAccess(societyRepo, platformRepo))

					docRouter.Get("/categories", ListDocumentCategoriesHandler(societyRepo))
					docRouter.Post("/categories", CreateDocumentCategoryHandler(societyRepo))
					docRouter.Get("/metrics", GetDocumentStorageMetricsHandler(societyRepo))

					docRouter.Get("/", ListDocumentsHandler(societyRepo, platformRepo))
					docRouter.Post("/", CreateDocumentHandler(societyRepo, platformRepo))

					docRouter.Get("/{documentId}", GetDocumentDetailsHandler(societyRepo, platformRepo))
					docRouter.Put("/{documentId}", UpdateDocumentHandler(societyRepo))

					docRouter.Get("/{documentId}/versions", ListDocumentVersionsHandler(societyRepo, platformRepo))
					docRouter.Post("/{documentId}/versions", UploadDocumentVersionHandler(societyRepo, platformRepo))
					docRouter.Get("/{documentId}/versions/{versionId}/download", DownloadDocumentVersionHandler(societyRepo, platformRepo))

					docRouter.Post("/{documentId}/archive", ArchiveDocumentHandler(societyRepo))
					docRouter.Post("/{documentId}/restore", RestoreDocumentHandler(societyRepo))
				})
			})
		}
	})

	return r
}
