package server

import (
	"context"
	"net/http"
	"time"

	"chs/backend/internal/database"
	"chs/backend/internal/response"
)

// DatabaseHealthStatus reflects database dependency connectivity.
type DatabaseHealthStatus struct {
	Connected bool   `json:"connected"`
	LatencyMs *int64 `json:"latency_ms,omitempty"`
	Message   string `json:"message,omitempty"`
}

// HealthData represents the health check response payload.
type HealthData struct {
	Status   string               `json:"status"`
	Version  string               `json:"version"`
	Database DatabaseHealthStatus `json:"database"`
}

// HealthHandler returns the health status of the application and database.
func HealthHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		dbStatus := DatabaseHealthStatus{
			Connected: false,
		}

		if db != nil && db.IsConnected() {
			latency, err := db.Ping(ctx)
			if err != nil {
				dbStatus.Message = "database connection error"
			} else {
				dbStatus.Connected = true
				dbStatus.LatencyMs = &latency
				dbStatus.Message = "operational"
			}
		} else {
			dbStatus.Message = "no database configured"
		}

		data := HealthData{
			Status:   "healthy",
			Version:  "1.0.0",
			Database: dbStatus,
		}

		response.Success(w, http.StatusOK, data, nil)
	}
}
