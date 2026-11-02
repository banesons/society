package server

import (
	"encoding/json"
	"errors"
	"fmt"
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
			if errors.Is(err, society.ErrBillingBatchNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing batch not found", reqID)
				return
			}
			if errors.Is(err, society.ErrBatchAlreadyProcessing) {
				response.Error(w, http.StatusConflict, "BATCH_ALREADY_PROCESSING", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrBatchAlreadyFinalized) {
				response.Error(w, http.StatusConflict, "BATCH_FINALIZED", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrBatchCancelled) {
				response.Error(w, http.StatusConflict, "BATCH_CANCELLED", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrInvalidBatchTransition) {
				response.Error(w, http.StatusConflict, "INVALID_TRANSITION", err.Error(), reqID)
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
			if errors.Is(err, society.ErrBillingBatchNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing batch not found", reqID)
				return
			}
			if errors.Is(err, society.ErrBatchAlreadyProcessing) {
				response.Error(w, http.StatusConflict, "BATCH_ALREADY_PROCESSING", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrBatchAlreadyFinalized) {
				response.Error(w, http.StatusConflict, "BATCH_FINALIZED", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrBatchCancelled) {
				response.Error(w, http.StatusConflict, "BATCH_CANCELLED", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrInvalidBatchTransition) {
				response.Error(w, http.StatusConflict, "INVALID_TRANSITION", err.Error(), reqID)
				return
			}
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
			if errors.Is(err, society.ErrBillingBatchNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing batch not found", reqID)
				return
			}
			if errors.Is(err, society.ErrBatchAlreadyProcessing) {
				response.Error(w, http.StatusConflict, "BATCH_ALREADY_PROCESSING", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "CANCELLATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{"status": "CANCELLED"}, nil)
	}
}

func RetryFailedBatchUnitsHandler(repo *society.Repository) http.HandlerFunc {
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

		batch, err := repo.RetryFailedBatchUnits(r.Context(), societyID, batchID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrBillingBatchNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing batch not found", reqID)
				return
			}
			if errors.Is(err, society.ErrBatchAlreadyProcessing) {
				response.Error(w, http.StatusConflict, "BATCH_ALREADY_PROCESSING", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrBatchAlreadyFinalized) {
				response.Error(w, http.StatusConflict, "BATCH_FINALIZED", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrBatchCancelled) {
				response.Error(w, http.StatusConflict, "BATCH_CANCELLED", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrInvalidBatchTransition) {
				response.Error(w, http.StatusConflict, "INVALID_TRANSITION", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrNoFailedUnitsToRetry) {
				response.Error(w, http.StatusBadRequest, "NO_FAILED_UNITS", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "RETRY_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, batch, nil)
	}
}

func GetBatchProgressHandler(repo *society.Repository) http.HandlerFunc {
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

		progress, err := repo.GetBatchProgress(r.Context(), societyID, batchID)
		if err != nil {
			if errors.Is(err, society.ErrBillingBatchNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing batch not found", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "PROGRESS_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, progress, nil)
	}
}

func ListBatchUnitsHandler(repo *society.Repository) http.HandlerFunc {
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

		statusFilter := r.URL.Query().Get("status")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		units, err := repo.ListBatchUnits(r.Context(), societyID, batchID, statusFilter, limit, offset)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "LIST_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{"units": units}, nil)
	}
}

// ------------------------------------------------------------------------------
// 5. PDF Generation & Artifact Handlers
// ------------------------------------------------------------------------------

func GenerateBillPDFHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
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

		artifact, _, err := repo.GenerateBillPDF(r.Context(), societyID, billID, actorID)
		if err != nil {
			if errors.Is(err, society.ErrBillNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Maintenance bill not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "PDF_GENERATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, artifact, nil)
	}
}

func DownloadBillPDFHandler(repo *society.Repository) http.HandlerFunc {
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

		userID, _ := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())

		allowed, err := repo.CanUserAccessBill(r.Context(), societyID, billID, userID, role)
		if err != nil || !allowed {
			response.Error(w, http.StatusForbidden, "ACCESS_DENIED", "You do not have permission to view or download this bill", reqID)
			return
		}

		artifact, pdfBytes, err := repo.GetBillPDF(r.Context(), societyID, billID)
		if err != nil {
			if errors.Is(err, society.ErrBillNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Maintenance bill not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "PDF_FETCH_FAILED", err.Error(), reqID)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", artifact.FileName))
		w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
		w.Header().Set("ETag", fmt.Sprintf("%q", artifact.SHA256Checksum))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdfBytes)
	}
}

func GenerateBatchPDFsHandler(repo *society.Repository) http.HandlerFunc {
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

		artifact, _, err := repo.GenerateBatchPDFs(r.Context(), societyID, batchID, actorID)
		if err != nil {
			if errors.Is(err, society.ErrBillingBatchNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing batch not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "BATCH_PDF_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, artifact, nil)
	}
}

func DownloadBatchZipHandler(repo *society.Repository) http.HandlerFunc {
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

		artifact, zipBytes, err := repo.GetBatchZipArtifact(r.Context(), societyID, batchID)
		if err != nil {
			if errors.Is(err, society.ErrBillingBatchNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing batch not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "BATCH_ZIP_FAILED", err.Error(), reqID)
			return
		}

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", artifact.FileName))
		w.Header().Set("Content-Length", strconv.Itoa(len(zipBytes)))
		w.Header().Set("ETag", fmt.Sprintf("%q", artifact.SHA256Checksum))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipBytes)
	}
}

func ListBillArtifactsHandler(repo *society.Repository) http.HandlerFunc {
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

		userID, _ := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())

		allowed, err := repo.CanUserAccessBill(r.Context(), societyID, billID, userID, role)
		if err != nil || !allowed {
			response.Error(w, http.StatusForbidden, "ACCESS_DENIED", "You do not have permission to view artifacts for this bill", reqID)
			return
		}

		artifacts, err := repo.ListBillPDFArtifacts(r.Context(), societyID, billID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{"artifacts": artifacts}, nil)
	}
}

func GetArtifactDetailsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		artifactID, err := uuid.Parse(chi.URLParam(r, "artifactId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid artifact ID", reqID)
			return
		}

		artifact, fileBytes, err := repo.GetArtifactByID(r.Context(), societyID, artifactID)
		if err != nil {
			if errors.Is(err, society.ErrArtifactNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Artifact not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error(), reqID)
			return
		}

		if r.URL.Query().Get("download") == "true" && len(fileBytes) > 0 {
			w.Header().Set("Content-Type", artifact.MimeType)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", artifact.FileName))
			w.Header().Set("Content-Length", strconv.Itoa(len(fileBytes)))
			w.Header().Set("ETag", fmt.Sprintf("%q", artifact.SHA256Checksum))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fileBytes)
			return
		}

		response.Success(w, http.StatusOK, artifact, nil)
	}
}

// --- Billing Automation Scheduler Handlers (Phase 7.2) ---

func CreateBillingScheduleHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		var params society.CreateBillingScheduleParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON payload", reqID)
			return
		}

		sched, err := repo.CreateBillingSchedule(r.Context(), societyID, params, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrTemplateNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing template not found", reqID)
				return
			}
			if errors.Is(err, society.ErrDuplicateScheduleName) {
				response.Error(w, http.StatusConflict, "DUPLICATE_NAME", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrInvalidRecurrenceType) ||
				errors.Is(err, society.ErrInvalidDayOfMonth) ||
				errors.Is(err, society.ErrInvalidDueDays) ||
				errors.Is(err, society.ErrInvalidTimezone) {
				response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, sched, nil)
	}
}

func ListBillingSchedulesHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		schedules, err := repo.ListBillingSchedules(r.Context(), societyID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{"schedules": schedules}, nil)
	}
}

func GetBillingScheduleHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID", reqID)
			return
		}

		sched, err := repo.GetBillingSchedule(r.Context(), societyID, scheduleID)
		if err != nil {
			if errors.Is(err, society.ErrScheduleNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing schedule not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, sched, nil)
	}
}

func UpdateBillingScheduleHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID", reqID)
			return
		}

		var params society.UpdateBillingScheduleParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON payload", reqID)
			return
		}

		sched, err := repo.UpdateBillingSchedule(r.Context(), societyID, scheduleID, params, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrScheduleNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing schedule not found", reqID)
				return
			}
			if errors.Is(err, society.ErrTemplateNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing template not found", reqID)
				return
			}
			if errors.Is(err, society.ErrDuplicateScheduleName) {
				response.Error(w, http.StatusConflict, "DUPLICATE_NAME", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, sched, nil)
	}
}

func PauseBillingScheduleHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID", reqID)
			return
		}

		sched, err := repo.SetBillingScheduleStatus(r.Context(), societyID, scheduleID, "PAUSED", actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrScheduleNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing schedule not found", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "ACTION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, sched, nil)
	}
}

func ResumeBillingScheduleHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID", reqID)
			return
		}

		sched, err := repo.SetBillingScheduleStatus(r.Context(), societyID, scheduleID, "ACTIVE", actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrScheduleNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing schedule not found", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "ACTION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, sched, nil)
	}
}

func DisableBillingScheduleHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID", reqID)
			return
		}

		sched, err := repo.SetBillingScheduleStatus(r.Context(), societyID, scheduleID, "DISABLED", actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrScheduleNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing schedule not found", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "ACTION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, sched, nil)
	}
}

func RunBillingScheduleNowHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID", reqID)
			return
		}

		exec, err := repo.ExecuteScheduleRun(r.Context(), societyID, scheduleID, "MANUAL", actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrScheduleNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing schedule not found", reqID)
				return
			}
			if errors.Is(err, society.ErrScheduleAlreadyRunning) {
				response.Error(w, http.StatusConflict, "SCHEDULE_ALREADY_RUNNING", err.Error(), reqID)
				return
			}
			if errors.Is(err, society.ErrScheduleDisabled) {
				response.Error(w, http.StatusBadRequest, "SCHEDULE_DISABLED", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "EXECUTION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, exec, nil)
	}
}

func ListScheduleExecutionsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID", reqID)
			return
		}

		limit := 20
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
				limit = l
			}
		}
		offset := 0
		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
				offset = o
			}
		}

		executions, err := repo.ListScheduleExecutions(r.Context(), societyID, scheduleID, limit, offset)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{"executions": executions}, nil)
	}
}

func GetBillingSchedulesHealthHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		health, err := repo.GetSchedulerHealth(r.Context(), societyID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "HEALTH_QUERY_FAILED", "Failed to retrieve scheduler health", reqID)
			return
		}

		response.Success(w, http.StatusOK, health, nil)
	}
}

func GetScheduleExecutionDetailsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID", reqID)
			return
		}
		executionID, err := uuid.Parse(chi.URLParam(r, "executionId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid execution ID", reqID)
			return
		}

		exec, err := repo.GetScheduleExecution(r.Context(), societyID, scheduleID, executionID)
		if err != nil {
			if errors.Is(err, society.ErrExecutionNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Schedule execution not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "QUERY_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, exec, nil)
	}
}

func RecoverScheduleLockHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID", reqID)
			return
		}

		sched, err := repo.RecoverScheduleLock(r.Context(), societyID, scheduleID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrScheduleNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing schedule not found", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "LOCK_RECOVERY_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, sched, nil)
	}
}

func RetryScheduleExecutionHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID", reqID)
			return
		}
		executionID, err := uuid.Parse(chi.URLParam(r, "executionId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid execution ID", reqID)
			return
		}

		newExec, err := repo.RetryScheduleExecution(r.Context(), societyID, scheduleID, executionID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrExecutionNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Schedule execution not found", reqID)
				return
			}
			if errors.Is(err, society.ErrFinalizedBatchImmutable) {
				response.Error(w, http.StatusConflict, "FINALIZED_BATCH_IMMUTABLE", "Cannot retry execution for an already finalized billing batch", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "RETRY_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, newExec, nil)
	}
}

func RerunMissedScheduleHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		societyID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}
		scheduleID, err := uuid.Parse(chi.URLParam(r, "scheduleId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid schedule ID", reqID)
			return
		}

		exec, err := repo.RerunMissedSchedule(r.Context(), societyID, scheduleID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrScheduleNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Billing schedule not found", reqID)
				return
			}
			if errors.Is(err, society.ErrScheduleDisabled) {
				response.Error(w, http.StatusBadRequest, "SCHEDULE_DISABLED", "Cannot re-run disabled billing schedule", reqID)
				return
			}
			if errors.Is(err, society.ErrScheduleAlreadyRunning) {
				response.Error(w, http.StatusConflict, "SCHEDULE_ALREADY_RUNNING", "Billing schedule is currently running", reqID)
				return
			}
			if errors.Is(err, society.ErrFinalizedBatchImmutable) {
				response.Error(w, http.StatusConflict, "FINALIZED_BATCH_IMMUTABLE", "Cannot re-run schedule: period batch is already finalized", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "RERUN_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, exec, nil)
	}
}

