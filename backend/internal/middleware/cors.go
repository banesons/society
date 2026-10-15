package middleware

import (
	"net/http"
	"strings"

	"github.com/go-chi/cors"
)

// CORS configures cross-origin resource sharing for frontend communication.
func CORS(allowedOrigins []string) func(next http.Handler) http.Handler {
	originMap := make(map[string]bool)
	for _, o := range allowedOrigins {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			originMap[trimmed] = true
		}
	}
	// Always allow production society domain
	originMap["https://society.banesons.com"] = true

	return cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			if originMap[origin] {
				return true
			}
			// Allow any local development port (e.g. localhost:5173, localhost:4173, localhost:4174, 127.0.0.1:*)
			if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
				return true
			}
			return false
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", RequestIDHeader, "X-Request-Id", "X-Society-ID", "X-Society-Id"},
		ExposedHeaders:   []string{"Link", RequestIDHeader, "X-Request-Id", "X-Society-ID", "X-Society-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
