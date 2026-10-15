package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"chs/backend/internal/middleware"
	"chs/backend/internal/platform"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

// ListDocumentCategoriesHandler returns categories for the society.
func ListDocumentCategoriesHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		includeInactive := r.URL.Query().Get("include_inactive") == "true"
		categories, err := societyRepo.ListCategories(r.Context(), socID, includeInactive)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve categories", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]any{
			"categories": categories,
		}, nil)
	}
}

// CreateDocumentCategoryHandler adds a new document category to the society.
func CreateDocumentCategoryHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		userID, userOk := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())
		if !ok || !userOk {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing context", reqID)
			return
		}

		if role != society.RoleSocietyAdmin && role != "SUPER_ADMIN" {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only administrators can create categories", reqID)
			return
		}

		var input struct {
			Name         string  `json:"name"`
			Description  *string `json:"description"`
			DisplayOrder int     `json:"display_order"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		cat, err := societyRepo.CreateCategory(r.Context(), socID, input.Name, input.Description, input.DisplayOrder, userID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, map[string]any{
			"category": cat,
		}, nil)
	}
}

// ListDocumentsHandler returns filtered, paginated documents based on caller permissions.
func ListDocumentsHandler(societyRepo *society.Repository, platformRepo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		userID, userOk := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())
		if !ok || !userOk {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing context", reqID)
			return
		}

		scope, err := societyRepo.ResolveUserClassificationScope(r.Context(), socID, userID, role)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve permissions", reqID)
			return
		}

		q := r.URL.Query()
		var filter society.ListDocumentsFilter

		if catIDStr := q.Get("category_id"); catIDStr != "" {
			if id, err := uuid.Parse(catIDStr); err == nil {
				filter.CategoryID = &id
			}
		}
		if bIDStr := q.Get("building_id"); bIDStr != "" {
			if id, err := uuid.Parse(bIDStr); err == nil {
				filter.BuildingID = &id
			}
		}
		if wIDStr := q.Get("wing_id"); wIDStr != "" {
			if id, err := uuid.Parse(wIDStr); err == nil {
				filter.WingID = &id
			}
		}
		if uIDStr := q.Get("unit_id"); uIDStr != "" {
			if id, err := uuid.Parse(uIDStr); err == nil {
				filter.UnitID = &id
			}
		}
		if mIDStr := q.Get("member_id"); mIDStr != "" {
			if id, err := uuid.Parse(mIDStr); err == nil {
				filter.MemberID = &id
			}
		}

		filter.Classification = q.Get("classification")
		filter.Status = q.Get("status")
		filter.Search = q.Get("search")

		if lStr := q.Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil {
				filter.Limit = l
			}
		}
		if oStr := q.Get("offset"); oStr != "" {
			if o, err := strconv.Atoi(oStr); err == nil {
				filter.Offset = o
			}
		}

		result, err := societyRepo.ListDocuments(r.Context(), socID, filter, scope)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve documents", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]any{
			"documents": result.Items,
			"total":     result.Total,
			"limit":     result.Limit,
			"offset":    result.Offset,
		}, nil)
	}
}

// CreateDocumentHandler accepts multipart/form-data upload and creates document + version 1.
func CreateDocumentHandler(societyRepo *society.Repository, platformRepo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		userID, userOk := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())
		if !ok || !userOk {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing context", reqID)
			return
		}

		scope, err := societyRepo.ResolveUserClassificationScope(r.Context(), socID, userID, role)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve permissions", reqID)
			return
		}

		// Residents cannot upload documents
		if scope == society.ClassificationSocietyResidents {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Residents cannot upload official society documents", reqID)
			return
		}

		// Enforce bounded request body size accounting for 10MB original file + 2MB multipart overhead
		r.Body = http.MaxBytesReader(w, r.Body, 12*1024*1024)
		if err := r.ParseMultipartForm(12 * 1024 * 1024); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) || strings.Contains(err.Error(), "request body too large") {
				response.Error(w, http.StatusBadRequest, "FILE_TOO_LARGE", "Request payload exceeds maximum allowed size of 12MB", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Failed to parse multipart form: malformed or truncated payload", reqID)
			return
		}

		title := r.FormValue("title")
		desc := r.FormValue("description")
		catIDStr := r.FormValue("category_id")
		classification := r.FormValue("classification")
		changeSummary := r.FormValue("change_summary")

		if title == "" || catIDStr == "" || classification == "" {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Title, category_id, and classification are required", reqID)
			return
		}

		catID, err := uuid.Parse(catIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid category identifier", reqID)
			return
		}

		// Committee members can ONLY upload COMMITTEE or SOCIETY_RESIDENTS documents
		if scope == society.ClassificationCommittee && classification == society.ClassificationSocAdminOnly {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Committee members cannot create administrator-only documents", reqID)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "File attachment is required", reqID)
			return
		}
		defer file.Close()

		// Read file bytes with limit check (MaxOriginalFileSize + 1)
		limitedReader := io.LimitReader(file, society.MaxOriginalFileSize+1)
		content, err := io.ReadAll(limitedReader)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Failed to read uploaded file", reqID)
			return
		}
		if len(content) > society.MaxOriginalFileSize {
			response.Error(w, http.StatusBadRequest, "FILE_TOO_LARGE", "File size exceeds 10MB limit", reqID)
			return
		}
		if len(content) == 0 {
			response.Error(w, http.StatusBadRequest, "EMPTY_FILE", "Uploaded file is empty", reqID)
			return
		}

		var descPtr *string
		if desc != "" {
			descPtr = &desc
		}
		var changePtr *string
		if changeSummary != "" {
			changePtr = &changeSummary
		}

		params := society.CreateDocumentParams{
			CategoryID:     catID,
			Title:          title,
			Description:    descPtr,
			Classification: classification,
			FileName:       header.Filename,
			FileSize:       int64(len(content)),
			MimeType:       header.Header.Get("Content-Type"),
			FileContent:    content,
			ChangeSummary:  changePtr,
			ActorID:        userID,
			ClientIP:       r.RemoteAddr,
			UserAgent:      r.UserAgent(),
		}

		// Optional associations
		if bStr := r.FormValue("building_id"); bStr != "" {
			if id, err := uuid.Parse(bStr); err == nil {
				params.BuildingID = &id
			}
		}
		if wStr := r.FormValue("wing_id"); wStr != "" {
			if id, err := uuid.Parse(wStr); err == nil {
				params.WingID = &id
			}
		}
		if uStr := r.FormValue("unit_id"); uStr != "" {
			if id, err := uuid.Parse(uStr); err == nil {
				params.UnitID = &id
			}
		}
		if mStr := r.FormValue("member_id"); mStr != "" {
			if id, err := uuid.Parse(mStr); err == nil {
				params.MemberID = &id
			}
		}

		doc, err := societyRepo.CreateDocument(r.Context(), socID, params)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrUnsupportedFileType):
				response.Error(w, http.StatusBadRequest, "UNSUPPORTED_FILE_TYPE", err.Error(), reqID)
			case errors.Is(err, society.ErrMimeMismatch):
				response.Error(w, http.StatusBadRequest, "MIME_MISMATCH", err.Error(), reqID)
			case errors.Is(err, society.ErrFileExceedsOriginalLimit), errors.Is(err, society.ErrFileTooLarge):
				response.Error(w, http.StatusBadRequest, "FILE_TOO_LARGE", err.Error(), reqID)
			case errors.Is(err, society.ErrFileExceedsAcceptedLimit):
				response.Error(w, http.StatusBadRequest, "FILE_EXCEEDS_ACCEPTED_LIMIT", err.Error(), reqID)
			case errors.Is(err, society.ErrCategoryNotFound):
				response.Error(w, http.StatusBadRequest, "CATEGORY_NOT_FOUND", err.Error(), reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]any{
			"document": doc,
		}, nil)
	}
}

// GetDocumentDetailsHandler returns metadata for a single document if caller is authorized.
func GetDocumentDetailsHandler(societyRepo *society.Repository, platformRepo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		userID, userOk := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())
		if !ok || !userOk {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing context", reqID)
			return
		}

		docIDStr := chi.URLParam(r, "documentId")
		docID, err := uuid.Parse(docIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid document identifier", reqID)
			return
		}

		doc, err := societyRepo.GetDocumentByID(r.Context(), socID, docID)
		if err != nil {
			if errors.Is(err, society.ErrDocumentNotFound) {
				response.Error(w, http.StatusNotFound, "DOCUMENT_NOT_FOUND", "Document not found in this society", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get document", reqID)
			return
		}

		scope, err := societyRepo.ResolveUserClassificationScope(r.Context(), socID, userID, role)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve permissions", reqID)
			return
		}

		// Enforce classification access
		if doc.Classification == society.ClassificationSocAdminOnly && scope != society.ClassificationSocAdminOnly && scope != "SUPER_ADMIN" {
			response.Error(w, http.StatusForbidden, "DOCUMENT_ACCESS_DENIED", "This document is restricted to society administrators", reqID)
			return
		}
		if doc.Classification == society.ClassificationCommittee && scope == society.ClassificationSocietyResidents {
			response.Error(w, http.StatusForbidden, "DOCUMENT_ACCESS_DENIED", "This document is restricted to committee members", reqID)
			return
		}
		if doc.Status == society.DocumentStatusArchived && scope == society.ClassificationSocietyResidents {
			response.Error(w, http.StatusNotFound, "DOCUMENT_NOT_FOUND", "Document not found", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]any{
			"document": doc,
		}, nil)
	}
}

// UpdateDocumentHandler updates mutable metadata of a document.
func UpdateDocumentHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		userID, userOk := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())
		if !ok || !userOk {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing context", reqID)
			return
		}

		if role != society.RoleSocietyAdmin && role != "SUPER_ADMIN" {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only administrators can update documents", reqID)
			return
		}

		docIDStr := chi.URLParam(r, "documentId")
		docID, err := uuid.Parse(docIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid document identifier", reqID)
			return
		}

		var input struct {
			Title          string     `json:"title"`
			Description    *string    `json:"description"`
			CategoryID     uuid.UUID  `json:"category_id"`
			Classification string     `json:"classification"`
			BuildingID     *uuid.UUID `json:"building_id"`
			WingID         *uuid.UUID `json:"wing_id"`
			UnitID         *uuid.UUID `json:"unit_id"`
			MemberID       *uuid.UUID `json:"member_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		params := society.UpdateDocumentParams{
			DocumentID:     docID,
			Title:          input.Title,
			Description:    input.Description,
			CategoryID:     input.CategoryID,
			Classification: input.Classification,
			BuildingID:     input.BuildingID,
			WingID:         input.WingID,
			UnitID:         input.UnitID,
			MemberID:       input.MemberID,
			ActorID:        userID,
			ClientIP:       r.RemoteAddr,
			UserAgent:      r.UserAgent(),
		}

		doc, err := societyRepo.UpdateDocument(r.Context(), socID, params)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]any{
			"document": doc,
		}, nil)
	}
}

// UploadDocumentVersionHandler adds a new version to an existing document.
func UploadDocumentVersionHandler(societyRepo *society.Repository, platformRepo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		userID, userOk := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())
		if !ok || !userOk {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing context", reqID)
			return
		}

		docIDStr := chi.URLParam(r, "documentId")
		docID, err := uuid.Parse(docIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid document identifier", reqID)
			return
		}

		scope, err := societyRepo.ResolveUserClassificationScope(r.Context(), socID, userID, role)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve permissions", reqID)
			return
		}

		// Verify document exists and check permissions
		doc, err := societyRepo.GetDocumentByID(r.Context(), socID, docID)
		if err != nil {
			if errors.Is(err, society.ErrDocumentNotFound) {
				response.Error(w, http.StatusNotFound, "DOCUMENT_NOT_FOUND", "Document not found in this society", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query document", reqID)
			return
		}

		// Permission to add version:
		if scope == society.ClassificationSocietyResidents {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Residents cannot upload document versions", reqID)
			return
		}
		if doc.Classification == society.ClassificationSocAdminOnly && scope != society.ClassificationSocAdminOnly && scope != "SUPER_ADMIN" {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only administrators can upload versions for administrator-only documents", reqID)
			return
		}

		// Enforce bounded request body size accounting for 10MB original file + 2MB multipart overhead
		r.Body = http.MaxBytesReader(w, r.Body, 12*1024*1024)
		if err := r.ParseMultipartForm(12 * 1024 * 1024); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) || strings.Contains(err.Error(), "request body too large") {
				response.Error(w, http.StatusBadRequest, "FILE_TOO_LARGE", "Request payload exceeds maximum allowed size of 12MB", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Failed to parse multipart form: malformed or truncated payload", reqID)
			return
		}

		changeSummary := r.FormValue("change_summary")
		file, header, err := r.FormFile("file")
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "File is required", reqID)
			return
		}
		defer file.Close()

		limitedReader := io.LimitReader(file, society.MaxOriginalFileSize+1)
		content, err := io.ReadAll(limitedReader)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Failed to read uploaded file", reqID)
			return
		}
		if len(content) > society.MaxOriginalFileSize {
			response.Error(w, http.StatusBadRequest, "FILE_TOO_LARGE", "File size exceeds 10MB limit", reqID)
			return
		}
		if len(content) == 0 {
			response.Error(w, http.StatusBadRequest, "EMPTY_FILE", "Uploaded file is empty", reqID)
			return
		}

		var changePtr *string
		if changeSummary != "" {
			changePtr = &changeSummary
		}

		params := society.UploadVersionParams{
			DocumentID:    docID,
			FileName:      header.Filename,
			FileSize:      int64(len(content)),
			MimeType:      header.Header.Get("Content-Type"),
			FileContent:   content,
			ChangeSummary: changePtr,
			ActorID:       userID,
			ClientIP:      r.RemoteAddr,
			UserAgent:     r.UserAgent(),
		}

		ver, err := societyRepo.UploadVersion(r.Context(), socID, params)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrUnsupportedFileType):
				response.Error(w, http.StatusBadRequest, "UNSUPPORTED_FILE_TYPE", err.Error(), reqID)
			case errors.Is(err, society.ErrMimeMismatch):
				response.Error(w, http.StatusBadRequest, "MIME_MISMATCH", err.Error(), reqID)
			case errors.Is(err, society.ErrFileExceedsOriginalLimit), errors.Is(err, society.ErrFileTooLarge):
				response.Error(w, http.StatusBadRequest, "FILE_TOO_LARGE", err.Error(), reqID)
			case errors.Is(err, society.ErrFileExceedsAcceptedLimit):
				response.Error(w, http.StatusBadRequest, "FILE_EXCEEDS_ACCEPTED_LIMIT", err.Error(), reqID)
			case errors.Is(err, society.ErrMaxVersionsReached):
				response.Error(w, http.StatusBadRequest, "MAX_VERSIONS_REACHED", err.Error(), reqID)
			case errors.Is(err, society.ErrDocumentArchived):
				response.Error(w, http.StatusBadRequest, "DOCUMENT_ARCHIVED", err.Error(), reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]any{
			"version": ver,
		}, nil)
	}
}

// ListDocumentVersionsHandler returns version history for a document.
func ListDocumentVersionsHandler(societyRepo *society.Repository, platformRepo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		userID, userOk := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())
		if !ok || !userOk {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing context", reqID)
			return
		}

		docIDStr := chi.URLParam(r, "documentId")
		docID, err := uuid.Parse(docIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid document identifier", reqID)
			return
		}

		doc, err := societyRepo.GetDocumentByID(r.Context(), socID, docID)
		if err != nil {
			if errors.Is(err, society.ErrDocumentNotFound) {
				response.Error(w, http.StatusNotFound, "DOCUMENT_NOT_FOUND", "Document not found in this society", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query document", reqID)
			return
		}

		scope, err := societyRepo.ResolveUserClassificationScope(r.Context(), socID, userID, role)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve permissions", reqID)
			return
		}

		// Enforce classification access
		if doc.Classification == society.ClassificationSocAdminOnly && scope != society.ClassificationSocAdminOnly && scope != "SUPER_ADMIN" {
			response.Error(w, http.StatusForbidden, "DOCUMENT_ACCESS_DENIED", "This document is restricted to society administrators", reqID)
			return
		}
		if doc.Classification == society.ClassificationCommittee && scope == society.ClassificationSocietyResidents {
			response.Error(w, http.StatusForbidden, "DOCUMENT_ACCESS_DENIED", "This document is restricted to committee members", reqID)
			return
		}

		versions, err := societyRepo.ListDocumentVersions(r.Context(), socID, docID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list versions", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]any{
			"versions": versions,
		}, nil)
	}
}

// DownloadDocumentVersionHandler streams the requested version file binary to the client.
func DownloadDocumentVersionHandler(societyRepo *society.Repository, platformRepo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		userID, userOk := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())
		if !ok || !userOk {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing context", reqID)
			return
		}

		docIDStr := chi.URLParam(r, "documentId")
		docID, err := uuid.Parse(docIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid document identifier", reqID)
			return
		}

		verIDStr := chi.URLParam(r, "versionId")
		verID, err := uuid.Parse(verIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid version identifier", reqID)
			return
		}

		// 1. Verify document belongs to society
		doc, err := societyRepo.GetDocumentByID(r.Context(), socID, docID)
		if err != nil {
			if errors.Is(err, society.ErrDocumentNotFound) {
				response.Error(w, http.StatusNotFound, "DOCUMENT_NOT_FOUND", "Document not found in this society", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query document", reqID)
			return
		}

		// 2. Enforce classification
		scope, err := societyRepo.ResolveUserClassificationScope(r.Context(), socID, userID, role)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve permissions", reqID)
			return
		}

		if doc.Classification == society.ClassificationSocAdminOnly && scope != society.ClassificationSocAdminOnly && scope != "SUPER_ADMIN" {
			response.Error(w, http.StatusForbidden, "DOCUMENT_ACCESS_DENIED", "This document is restricted to society administrators", reqID)
			return
		}
		if doc.Classification == society.ClassificationCommittee && scope == society.ClassificationSocietyResidents {
			response.Error(w, http.StatusForbidden, "DOCUMENT_ACCESS_DENIED", "This document is restricted to committee members", reqID)
			return
		}

		// 3. Get version record
		version, err := societyRepo.GetDocumentVersionByID(r.Context(), socID, docID, verID)
		if err != nil {
			if errors.Is(err, society.ErrDocumentVersionNotFound) {
				response.Error(w, http.StatusNotFound, "VERSION_NOT_FOUND", "Version not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query version", reqID)
			return
		}

		// 4. Download file from private Supabase Storage
		stream, _, _, err := societyRepo.DownloadFileFromStorage(r.Context(), version.StoragePath)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "STORAGE_ERROR", "Failed to retrieve document binary", reqID)
			return
		}
		defer stream.Close()

		// 5. Header sanitization and Content-Disposition
		inline := r.URL.Query().Get("inline") == "true"
		sanitizedName, _ := society.SanitizeFilename(version.FileName)

		w.Header().Set("Content-Type", version.MimeType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Length", strconv.FormatInt(version.FileSize, 10))

		// Inline preview only permitted for safe previewable formats (PDF, images)
		canInline := inline && (version.MimeType == "application/pdf" || strings.HasPrefix(version.MimeType, "image/"))
		if canInline {
			w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", sanitizedName))
		} else {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", sanitizedName))
		}

		// Stream content
		if _, err := io.Copy(w, stream); err != nil {
			// streaming interrupted
			return
		}

		// Audit log download
		_ = societyRepo.RecordDocumentDownloadAudit(r.Context(), socID, docID, verID, userID, r.RemoteAddr, r.UserAgent())
	}
}

// ArchiveDocumentHandler archives a document with a mandatory reason.
func ArchiveDocumentHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		userID, userOk := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())
		if !ok || !userOk {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing context", reqID)
			return
		}

		if role != society.RoleSocietyAdmin && role != "SUPER_ADMIN" {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only administrators can archive documents", reqID)
			return
		}

		docIDStr := chi.URLParam(r, "documentId")
		docID, err := uuid.Parse(docIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid document identifier", reqID)
			return
		}

		var input struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		doc, err := societyRepo.ArchiveDocument(r.Context(), socID, docID, input.Reason, userID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]any{
			"document": doc,
		}, nil)
	}
}

// RestoreDocumentHandler restores an archived document to active state.
func RestoreDocumentHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		userID, userOk := middleware.GetUserID(r.Context())
		role, _ := middleware.GetSocietyRole(r.Context())
		if !ok || !userOk {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing context", reqID)
			return
		}

		if role != society.RoleSocietyAdmin && role != "SUPER_ADMIN" {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only administrators can restore documents", reqID)
			return
		}

		docIDStr := chi.URLParam(r, "documentId")
		docID, err := uuid.Parse(docIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid document identifier", reqID)
			return
		}

		doc, err := societyRepo.RestoreDocument(r.Context(), socID, docID, userID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]any{
			"document": doc,
		}, nil)
	}
}

// GetDocumentStorageMetricsHandler returns storage observability metrics for the society.
func GetDocumentStorageMetricsHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		metrics, err := societyRepo.GetDocumentStorageMetrics(r.Context(), socID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve storage metrics", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]any{
			"metrics": metrics,
		}, nil)
	}
}
