package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"chs/backend/internal/middleware"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

func parseDateFlexible(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("date is empty")
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339Nano, s)
}

// ListChargeCategoriesHandler handles GET /api/v1/societies/{id}/maintenance/categories.
func ListChargeCategoriesHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		status := r.URL.Query().Get("status")
		categories, err := repo.ListChargeCategories(r.Context(), societyID, status)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list charge categories", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"categories": categories,
		}, nil)
	}
}

// CreateChargeCategoryHandler handles POST /api/v1/societies/{id}/maintenance/categories.
func CreateChargeCategoryHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var params society.CreateChargeCategoryParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed JSON body", reqID)
			return
		}

		cat, err := repo.CreateChargeCategory(r.Context(), societyID, params, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrChargeCategoryNameExists) {
				response.Error(w, http.StatusConflict, "CATEGORY_EXISTS", "A charge category with this name already exists", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"category": cat,
		}, nil)
	}
}

// UpdateChargeCategoryHandler handles PUT /api/v1/societies/{id}/maintenance/categories/{categoryId}.
func UpdateChargeCategoryHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		categoryID, err := uuid.Parse(chi.URLParam(r, "categoryId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid category ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var params society.UpdateChargeCategoryParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed JSON body", reqID)
			return
		}

		cat, err := repo.UpdateChargeCategory(r.Context(), societyID, categoryID, params, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, society.ErrChargeCategoryNotFound):
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Charge category not found", reqID)
			case errors.Is(err, society.ErrChargeCategoryNameExists):
				response.Error(w, http.StatusConflict, "CATEGORY_EXISTS", "A charge category with this name already exists", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"category": cat,
		}, nil)
	}
}

// ListChargeConfigsHandler handles GET /api/v1/societies/{id}/maintenance/configs.
func ListChargeConfigsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		status := r.URL.Query().Get("status")
		configs, err := repo.ListChargeConfigs(r.Context(), societyID, status)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list charge configs", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"configs": configs,
		}, nil)
	}
}

// CreateChargeConfigHandler handles POST /api/v1/societies/{id}/maintenance/configs.
func CreateChargeConfigHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var req struct {
			CategoryID         uuid.UUID `json:"category_id"`
			Name               string    `json:"name"`
			ChargeType         string    `json:"charge_type"`
			ApplicableUnitType *string   `json:"applicable_unit_type,omitempty"`
			Amount             float64   `json:"amount"`
			BillingFrequency   string    `json:"billing_frequency"`
			EffectiveFrom      string    `json:"effective_from"`
			EffectiveUntil     *string   `json:"effective_until,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed JSON body", reqID)
			return
		}

		effFrom, err := parseDateFlexible(req.EffectiveFrom)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", "effective_from must be YYYY-MM-DD or RFC3339", reqID)
			return
		}

		var effUntil *time.Time
		if req.EffectiveUntil != nil && strings.TrimSpace(*req.EffectiveUntil) != "" {
			u, err := parseDateFlexible(*req.EffectiveUntil)
			if err != nil {
				response.Error(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", "effective_until must be YYYY-MM-DD or RFC3339", reqID)
				return
			}
			effUntil = &u
		}

		params := society.CreateChargeConfigParams{
			CategoryID:         req.CategoryID,
			Name:               req.Name,
			ChargeType:         req.ChargeType,
			ApplicableUnitType: req.ApplicableUnitType,
			Amount:             req.Amount,
			BillingFrequency:   req.BillingFrequency,
			EffectiveFrom:      effFrom,
			EffectiveUntil:     effUntil,
		}

		cfg, err := repo.CreateChargeConfig(r.Context(), societyID, params, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, society.ErrChargeCategoryNotFound):
				response.Error(w, http.StatusNotFound, "CATEGORY_NOT_FOUND", "Referenced charge category does not exist", reqID)
			case errors.Is(err, society.ErrInvalidAmount):
				response.Error(w, http.StatusBadRequest, "INVALID_AMOUNT", "Amount must be strictly greater than zero", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"config": cfg,
		}, nil)
	}
}

// DeactivateChargeConfigHandler handles POST /api/v1/societies/{id}/maintenance/configs/{configId}/deactivate.
func DeactivateChargeConfigHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		configID, err := uuid.Parse(chi.URLParam(r, "configId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid config ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		err = repo.DeactivateChargeConfig(r.Context(), societyID, configID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrChargeConfigNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Charge configuration not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to deactivate charge config", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"deactivated": true,
		}, nil)
	}
}

// ListBillingPeriodsHandler handles GET /api/v1/societies/{id}/maintenance/periods.
func ListBillingPeriodsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		periods, err := repo.ListBillingPeriods(r.Context(), societyID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list billing periods", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"periods": periods,
		}, nil)
	}
}

// CreateBillingPeriodHandler handles POST /api/v1/societies/{id}/maintenance/periods.
func CreateBillingPeriodHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var req struct {
			PeriodName  string `json:"period_name"`
			PeriodStart string `json:"period_start"`
			PeriodEnd   string `json:"period_end"`
			DueDate     string `json:"due_date"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed JSON body", reqID)
			return
		}

		pStart, err := parseDateFlexible(req.PeriodStart)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", "period_start must be YYYY-MM-DD or RFC3339", reqID)
			return
		}
		pEnd, err := parseDateFlexible(req.PeriodEnd)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", "period_end must be YYYY-MM-DD or RFC3339", reqID)
			return
		}
		dDate, err := parseDateFlexible(req.DueDate)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", "due_date must be YYYY-MM-DD or RFC3339", reqID)
			return
		}

		params := society.CreateBillingPeriodParams{
			PeriodName:  req.PeriodName,
			PeriodStart: pStart,
			PeriodEnd:   pEnd,
			DueDate:     dDate,
		}

		period, err := repo.CreateBillingPeriod(r.Context(), societyID, params, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, society.ErrBillingPeriodOverlap):
				response.Error(w, http.StatusConflict, "PERIOD_EXISTS", "A billing period with these exact dates already exists", reqID)
			case errors.Is(err, society.ErrInvalidBillingDates):
				response.Error(w, http.StatusBadRequest, "INVALID_DATES", "Period end date and due date must be on or after start date", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"period": period,
		}, nil)
	}
}

// PreviewBillGenerationHandler handles GET /api/v1/societies/{id}/maintenance/periods/{periodId}/preview.
func PreviewBillGenerationHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		periodID, err := uuid.Parse(chi.URLParam(r, "periodId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid period ID", reqID)
			return
		}

		preview, err := repo.PreviewBillGeneration(r.Context(), societyID, periodID)
		if err != nil {
			if errors.Is(err, society.ErrBillingPeriodNotFound) {
				response.Error(w, http.StatusNotFound, "PERIOD_NOT_FOUND", "Billing period not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"preview": preview,
		}, nil)
	}
}

// GenerateBillsHandler handles POST /api/v1/societies/{id}/maintenance/periods/{periodId}/generate.
func GenerateBillsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		periodID, err := uuid.Parse(chi.URLParam(r, "periodId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid period ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		result, err := repo.GenerateBills(r.Context(), societyID, periodID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, society.ErrBillingPeriodNotFound):
				response.Error(w, http.StatusNotFound, "PERIOD_NOT_FOUND", "Billing period not found", reqID)
			case errors.Is(err, society.ErrNoEligibleCharges):
				response.Error(w, http.StatusBadRequest, "NO_ELIGIBLE_CHARGES", "No active charge configurations overlap this period", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "GENERATION_FAILED", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"result": result,
		}, nil)
	}
}

// ListBillsHandler handles GET /api/v1/societies/{id}/maintenance/bills (Admin register).
func ListBillsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		filter := society.ListBillsFilter{
			Status: strings.TrimSpace(r.URL.Query().Get("status")),
			Search: strings.TrimSpace(r.URL.Query().Get("search")),
		}

		if p := r.URL.Query().Get("period_id"); p != "" {
			if pID, err := uuid.Parse(p); err == nil {
				filter.PeriodID = &pID
			}
		}

		if u := r.URL.Query().Get("unit_id"); u != "" {
			if uID, err := uuid.Parse(u); err == nil {
				filter.UnitID = &uID
			}
		}

		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				filter.Limit = parsed
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				filter.Offset = parsed
			}
		}

		result, err := repo.ListBills(r.Context(), societyID, filter)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list maintenance bills", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"bills":  result.Items,
			"items":  result.Items,
			"total":  result.Total,
			"limit":  result.Limit,
			"offset": result.Offset,
		}, nil)
	}
}

// GetBillDetailsHandler handles GET /api/v1/societies/{id}/maintenance/bills/{billId}.
func GetBillDetailsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		billID, err := uuid.Parse(chi.URLParam(r, "billId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid bill ID", reqID)
			return
		}

		bill, err := repo.GetBillDetails(r.Context(), societyID, billID)
		if err != nil {
			if errors.Is(err, society.ErrBillNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Maintenance bill not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get bill details", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"bill": bill,
		}, nil)
	}
}

// VoidBillHandler handles POST /api/v1/societies/{id}/maintenance/bills/{billId}/void.
func VoidBillHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		billID, err := uuid.Parse(chi.URLParam(r, "billId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid bill ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var payload struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed JSON body", reqID)
			return
		}

		bill, err := repo.VoidBill(r.Context(), societyID, billID, payload.Reason, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, society.ErrBillNotFound):
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Maintenance bill not found", reqID)
			case errors.Is(err, society.ErrBillAlreadyVoid):
				response.Error(w, http.StatusBadRequest, "ALREADY_VOID", "Bill is already voided", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"bill": bill,
		}, nil)
	}
}

// ListResidentBillsHandler handles GET /api/v1/societies/{id}/maintenance/my-bills.
func ListResidentBillsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		limit := 20
		offset := 0
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		result, err := repo.ListResidentBills(r.Context(), societyID, userID, limit, offset)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve resident maintenance bills", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"bills":  result.Items,
			"total":  result.Total,
			"limit":  result.Limit,
			"offset": result.Offset,
		}, nil)
	}
}

// GetResidentBillDetailsHandler handles GET /api/v1/societies/{id}/maintenance/my-bills/{billId}.
func GetResidentBillDetailsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		billID, err := uuid.Parse(chi.URLParam(r, "billId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid bill ID", reqID)
			return
		}

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		bill, err := repo.GetResidentBillDetails(r.Context(), societyID, billID, userID)
		if err != nil {
			if errors.Is(err, society.ErrBillNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Maintenance bill not found", reqID)
				return
			}
			response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"bill": bill,
		}, nil)
	}
}

// RecordPaymentHandler handles POST /api/v1/societies/{id}/maintenance/payments.
func RecordPaymentHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var req struct {
			BillID            string  `json:"bill_id"`
			Amount            float64 `json:"amount"`
			PaymentMethod     string  `json:"payment_method"`
			PaymentDate       string  `json:"payment_date"`
			ExternalReference *string `json:"external_reference"`
			PayerName         *string `json:"payer_name"`
			Notes             *string `json:"notes"`
			IdempotencyKey    *string `json:"idempotency_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed JSON body", reqID)
			return
		}

		billID, err := uuid.Parse(req.BillID)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid bill ID", reqID)
			return
		}

		var payDate time.Time
		if req.PaymentDate != "" {
			parsed, err := parseDateFlexible(req.PaymentDate)
			if err != nil {
				response.Error(w, http.StatusBadRequest, "INVALID_DATE", "Invalid payment date format", reqID)
				return
			}
			payDate = parsed
		} else {
			payDate = time.Now()
		}

		if idemHeader := r.Header.Get("Idempotency-Key"); idemHeader != "" && (req.IdempotencyKey == nil || *req.IdempotencyKey == "") {
			req.IdempotencyKey = &idemHeader
		}

		params := society.RecordPaymentParams{
			BillID:            billID,
			Amount:            req.Amount,
			PaymentMethod:     strings.ToUpper(strings.TrimSpace(req.PaymentMethod)),
			PaymentDate:       payDate,
			ExternalReference: req.ExternalReference,
			PayerName:         req.PayerName,
			Notes:             req.Notes,
			IdempotencyKey:    req.IdempotencyKey,
		}

		result, err := repo.RecordPayment(r.Context(), societyID, params, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, society.ErrBillNotFound):
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Maintenance bill not found", reqID)
			case errors.Is(err, society.ErrBillCannotBePaid):
				response.Error(w, http.StatusBadRequest, "BILL_VOIDED", "Cannot record payment for voided bill", reqID)
			case errors.Is(err, society.ErrInvalidAmount):
				response.Error(w, http.StatusBadRequest, "INVALID_AMOUNT", "Payment amount must be greater than zero", reqID)
			case errors.Is(err, society.ErrAllocationExceedsBalance):
				response.Error(w, http.StatusBadRequest, "EXCEEDS_BALANCE", "Payment amount exceeds outstanding bill balance", reqID)
			case errors.Is(err, society.ErrInvalidPaymentMethod):
				response.Error(w, http.StatusBadRequest, "INVALID_METHOD", "Invalid payment method", reqID)
			case errors.Is(err, society.ErrIdempotencyConflict):
				response.Error(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency key reused with differing payment parameters", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "RECORD_PAYMENT_FAILED", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, result, nil)
	}
}

// ReversePaymentHandler handles POST /api/v1/societies/{id}/maintenance/payments/{paymentId}/reverse.
func ReversePaymentHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		paymentID, err := uuid.Parse(chi.URLParam(r, "paymentId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid payment ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var req struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed JSON body", reqID)
			return
		}

		if strings.TrimSpace(req.Reason) == "" {
			response.Error(w, http.StatusBadRequest, "REASON_REQUIRED", "Reversal reason cannot be empty", reqID)
			return
		}

		if err := repo.ReversePayment(r.Context(), societyID, paymentID, req.Reason, actorID, r.RemoteAddr, r.UserAgent()); err != nil {
			switch {
			case errors.Is(err, society.ErrPaymentNotFound):
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Payment not found", reqID)
			case errors.Is(err, society.ErrPaymentAlreadyReversed):
				response.Error(w, http.StatusBadRequest, "ALREADY_REVERSED", "Payment has already been reversed", reqID)
			case errors.Is(err, society.ErrReversalReasonRequired):
				response.Error(w, http.StatusBadRequest, "REASON_REQUIRED", "Reversal reason cannot be empty", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reverse payment", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"message": "Payment reversed successfully",
		}, nil)
	}
}

// ListCollectionRegisterHandler handles GET /api/v1/societies/{id}/maintenance/collections/register.
func ListCollectionRegisterHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		filter := society.ListCollectionRegisterFilter{
			Limit:  20,
			Offset: 0,
		}

		if periodIDStr := r.URL.Query().Get("period_id"); periodIDStr != "" {
			if pID, err := uuid.Parse(periodIDStr); err == nil {
				filter.PeriodID = &pID
			}
		}

		if status := r.URL.Query().Get("status"); status != "" {
			filter.Status = strings.ToUpper(strings.TrimSpace(status))
		}

		if search := r.URL.Query().Get("search"); search != "" {
			filter.Search = strings.TrimSpace(search)
		}

		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				if parsed > 100 {
					parsed = 100
				}
				filter.Limit = parsed
			}
		}

		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				filter.Offset = parsed
			}
		}

		result, err := repo.ListCollectionRegister(r.Context(), societyID, filter)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list collection register", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"items":  result.Items,
			"total":  result.Total,
			"limit":  result.Limit,
			"offset": result.Offset,
		}, nil)
	}
}

// GetCollectionSummaryHandler handles GET /api/v1/societies/{id}/maintenance/collections/summary.
func GetCollectionSummaryHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		var periodID *uuid.UUID
		if periodIDStr := r.URL.Query().Get("period_id"); periodIDStr != "" {
			if pID, err := uuid.Parse(periodIDStr); err == nil {
				periodID = &pID
			}
		}

		summary, err := repo.GetCollectionSummary(r.Context(), societyID, periodID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get collection summary", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"summary": summary,
		}, nil)
	}
}

// GetReceiptDetailsHandler handles GET /api/v1/societies/{id}/maintenance/receipts/{receiptId}.
func GetReceiptDetailsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		receiptID, err := uuid.Parse(chi.URLParam(r, "receiptId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid receipt ID", reqID)
			return
		}

		receipt, err := repo.GetReceiptDetails(r.Context(), societyID, receiptID)
		if err != nil {
			if errors.Is(err, society.ErrReceiptNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Receipt not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get receipt details", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"receipt": receipt,
		}, nil)
	}
}

// ListResidentPaymentsHandler handles GET /api/v1/societies/{id}/maintenance/my-payments.
func ListResidentPaymentsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		limit := 20
		offset := 0
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		items, total, err := repo.ListResidentPayments(r.Context(), societyID, userID, limit, offset)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve resident payment history", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"payments": items,
			"total":    total,
			"limit":    limit,
			"offset":   offset,
		}, nil)
	}
}

// GetResidentReceiptHandler handles GET /api/v1/societies/{id}/maintenance/my-receipts/{receiptId}.
func GetResidentReceiptHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		receiptID, err := uuid.Parse(chi.URLParam(r, "receiptId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid receipt ID", reqID)
			return
		}

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		receipt, err := repo.GetResidentReceipt(r.Context(), societyID, receiptID, userID)
		if err != nil {
			if errors.Is(err, society.ErrReceiptNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Receipt not found", reqID)
				return
			}
			response.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"receipt": receipt,
		}, nil)
	}
}
