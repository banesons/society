package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

// CORS configures cross-origin resource sharing for frontend communication.
func CORS(allowedOrigins []string) func(next http.Handler) http.Handler {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:5173", "http://localhost:4173"}
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", RequestIDHeader},
		ExposedHeaders:   []string{"Link", RequestIDHeader},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
