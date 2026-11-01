package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"chs/backend/internal/middleware"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

// ------------------------------------------------------------------------------
// 1. Template Management Handlers
// ------------------------------------------------------------------------------

func ListBillTemplatesHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		templates, err := repo.ListBillTemplates(r.Context(), societyID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list bill templates", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"templates": templates,
		}, nil)
	}
}

func GetBillTemplateHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		templateID, err := uuid.Parse(chi.URLParam(r, "templateId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid template ID", reqID)
			return
		}

		tpl, err := repo.GetBillTemplate(r.Context(), societyID, templateID)
		if err != nil {
			if err == society.ErrTemplateNotFound {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing template not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve template", reqID)
			return
		}

		response.Success(w, http.StatusOK, tpl, nil)
	}
}

func CreateBillTemplateHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		var req society.CreateBillTemplateParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		tpl, err := repo.CreateBillTemplate(r.Context(), societyID, req, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if err == society.ErrTemplateNameExists {
				response.Error(w, http.StatusConflict, "NAME_EXISTS", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "CREATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, tpl, nil)
	}
}

func UpdateBillTemplateHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		templateID, err := uuid.Parse(chi.URLParam(r, "templateId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid template ID", reqID)
			return
		}

		var req society.UpdateBillTemplateParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		tpl, err := repo.UpdateBillTemplate(r.Context(), societyID, templateID, req, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if err == society.ErrTemplateNotFound {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Template not found", reqID)
				return
			}
			if err == society.ErrTemplateImmutable {
				response.Error(w, http.StatusConflict, "TEMPLATE_IMMUTABLE", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, tpl, nil)
	}
}

func DuplicateBillTemplateHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		templateID, err := uuid.Parse(chi.URLParam(r, "templateId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid template ID", reqID)
			return
		}

		var req struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		tpl, err := repo.DuplicateBillTemplate(r.Context(), societyID, templateID, req.Name, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			response.Error(w, http.StatusBadRequest, "DUPLICATE_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, tpl, nil)
	}
}

func PublishBillTemplateHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		templateID, err := uuid.Parse(chi.URLParam(r, "templateId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid template ID", reqID)
			return
		}

		ver, err := repo.PublishBillTemplate(r.Context(), societyID, templateID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if err == society.ErrTemplateNotFound {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Template not found", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "PUBLISH_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, ver, nil)
	}
}

func ArchiveBillTemplateHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		templateID, err := uuid.Parse(chi.URLParam(r, "templateId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid template ID", reqID)
			return
		}

		err = repo.ArchiveBillTemplate(r.Context(), societyID, templateID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if err == society.ErrTemplateNotFound {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Template not found", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "ARCHIVE_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{"status": "ARCHIVED"}, nil)
	}
}

func ListBillTemplateVersionsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		templateID, err := uuid.Parse(chi.URLParam(r, "templateId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid template ID", reqID)
			return
		}

		versions, err := repo.ListBillTemplateVersions(r.Context(), societyID, templateID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list template versions", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"versions": versions,
		}, nil)
	}
}

// ------------------------------------------------------------------------------
// 2. Charge Head Handlers
// ------------------------------------------------------------------------------

func ListChargeHeadsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		activeOnly := r.URL.Query().Get("active") == "true"
		heads, err := repo.ListChargeHeads(r.Context(), societyID, activeOnly)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list charge heads", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"charge_heads": heads,
		}, nil)
	}
}

func CreateChargeHeadHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		var req society.CreateChargeHeadParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		head, err := repo.CreateChargeHead(r.Context(), societyID, req, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if err == society.ErrChargeHeadCodeExists {
				response.Error(w, http.StatusConflict, "CODE_EXISTS", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "CREATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, head, nil)
	}
}

func UpdateChargeHeadHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		headID, err := uuid.Parse(chi.URLParam(r, "headId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid charge head ID", reqID)
			return
		}

		var req society.UpdateChargeHeadParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		head, err := repo.UpdateChargeHead(r.Context(), societyID, headID, req, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if err == society.ErrChargeHeadNotFound {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Charge head not found", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, head, nil)
	}
}

func ArchiveChargeHeadHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		headID, err := uuid.Parse(chi.URLParam(r, "headId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid charge head ID", reqID)
			return
		}

		err = repo.ArchiveChargeHead(r.Context(), societyID, headID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if err == society.ErrChargeHeadNotFound {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Charge head not found", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "ARCHIVE_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{"status": "INACTIVE"}, nil)
	}
}

// ------------------------------------------------------------------------------
// 3. Unit Billing Overrides Handlers
// ------------------------------------------------------------------------------

func ListUnitBillingOverridesHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		unitID, err := uuid.Parse(chi.URLParam(r, "unitId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid unit ID", reqID)
			return
		}

		overrides, err := repo.ListUnitBillingOverrides(r.Context(), societyID, unitID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list unit overrides", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"overrides": overrides,
		}, nil)
	}
}

func CreateUnitBillingOverrideHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		unitID, err := uuid.Parse(chi.URLParam(r, "unitId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid unit ID", reqID)
			return
		}

		var req society.CreateUnitBillingOverrideParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		ov, err := repo.CreateUnitBillingOverride(r.Context(), societyID, unitID, req, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			response.Error(w, http.StatusBadRequest, "CREATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, ov, nil)
	}
}

func UpdateUnitBillingOverrideHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		overrideID, err := uuid.Parse(chi.URLParam(r, "overrideId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid override ID", reqID)
			return
		}

		var req struct {
			OverrideValue *float64 `json:"override_value"`
			Status        *string  `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		err = repo.UpdateUnitBillingOverride(r.Context(), societyID, overrideID, req.OverrideValue, req.Status, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if err == society.ErrUnitOverrideNotFound {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Override not found", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{"status": "UPDATED"}, nil)
	}
}

func DeleteUnitBillingOverrideHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		overrideID, err := uuid.Parse(chi.URLParam(r, "overrideId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid override ID", reqID)
			return
		}

		err = repo.DeleteUnitBillingOverride(r.Context(), societyID, overrideID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if err == society.ErrUnitOverrideNotFound {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Override not found", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "DELETE_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{"status": "DELETED"}, nil)
	}
}

// ------------------------------------------------------------------------------
// 4. Billing Batches & Calculation Handlers
// ------------------------------------------------------------------------------

func ListBillingBatchesHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		batches, err := repo.ListBillingBatches(r.Context(), societyID, limit, offset)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list billing batches", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"batches": batches,
		}, nil)
	}
}

func GetBillingBatchDetailsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		batchID, err := uuid.Parse(chi.URLParam(r, "batchId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid batch ID", reqID)
			return
		}

		batch, err := repo.GetBillingBatch(r.Context(), societyID, batchID)
		if err != nil {
			if err == society.ErrBillingBatchNotFound {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing batch not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get batch details", reqID)
			return
		}

		response.Success(w, http.StatusOK, batch, nil)
	}
}

func CreateBillingBatchHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		var req society.CreateBillingBatchParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		batch, err := repo.CreateBillingBatch(r.Context(), societyID, req, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			response.Error(w, http.StatusBadRequest, "CREATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, batch, nil)
	}
}

func PreviewBillingBatchHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		batchID, err := uuid.Parse(chi.URLParam(r, "batchId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid batch ID", reqID)
			return
		}

		preview, err := repo.PreviewBillingBatch(r.Context(), societyID, batchID)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "PREVIEW_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, preview, nil)
	}
}

func GenerateBillingBatchDraftsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		batchID, err := uuid.Parse(chi.URLParam(r, "batchId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid batch ID", reqID)
			return
		}

		batch, err := repo.GenerateBillingBatchDrafts(r.Context(), societyID, batchID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if err == society.ErrBatchAlreadyFinalized {
				response.Error(w, http.StatusConflict, "BATCH_FINALIZED", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "GENERATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, batch, nil)
	}
}

func FinalizeBillingBatchHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		batchID, err := uuid.Parse(chi.URLParam(r, "batchId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid batch ID", reqID)
			return
		}

		batch, err := repo.FinalizeBillingBatch(r.Context(), societyID, batchID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			response.Error(w, http.StatusBadRequest, "FINALIZATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, batch, nil)
	}
}

func CancelBillingBatchHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		batchID, err := uuid.Parse(chi.URLParam(r, "batchId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid batch ID", reqID)
			return
		}

		var req struct {
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if strings.TrimSpace(req.Reason) == "" {
			req.Reason = "Cancelled by administrator"
		}

		err = repo.CancelBillingBatch(r.Context(), societyID, batchID, req.Reason, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			response.Error(w, http.StatusBadRequest, "CANCELLATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{"status": "CANCELLED"}, nil)
	}
}
