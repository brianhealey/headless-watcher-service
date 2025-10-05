package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"time"
)

// Logger middleware logs incoming requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		log.Printf("=> %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(rw, r)

		// Log completion with status code
		duration := time.Since(start)
		log.Printf("<= %s %s completed in %v (status: %d)", r.Method, r.URL.Path, duration, rw.statusCode)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// AuthValidator middleware validates the Authorization header
// For now, it just logs the token but doesn't enforce validation
func AuthValidator(requiredToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if requiredToken != "" {
				// If a required token is configured, validate it
				if authHeader != requiredToken {
					log.Printf("ERROR: Invalid or missing Authorization header (expected: %s, got: %s)",
						requiredToken, authHeader)
					http.Error(w, `{"code": 401}`, http.StatusUnauthorized)
					return
				}
			}

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// DeviceEUIValidator middleware validates the API-OBITER-DEVICE-EUI header
func DeviceEUIValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deviceEUI := r.Header.Get("API-OBITER-DEVICE-EUI")

		if deviceEUI == "" {
			log.Println("WARN: Missing API-OBITER-DEVICE-EUI header")
			// For now, just log the warning but allow the request through
		} else if len(deviceEUI) != 16 {
			log.Printf("WARN: Invalid API-OBITER-DEVICE-EUI header (expected 16 chars, got %d): %s",
				len(deviceEUI), deviceEUI)
			// For now, just log the warning but allow the request through
		}

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// CORS middleware adds CORS headers for development
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, API-OBITER-DEVICE-EUI")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// NotFoundLogger middleware logs 404 errors with full request details
func NotFoundLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture response
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(rw, r)

		// If 404, log detailed information
		if rw.statusCode == http.StatusNotFound {
			log.Println("================================================================================")
			log.Println("404 NOT FOUND - Unmatched Route")
			log.Println("================================================================================")
			log.Printf("Method:      %s", r.Method)
			log.Printf("URL:         %s", r.URL.String())
			log.Printf("Path:        %s", r.URL.Path)
			if r.URL.RawQuery != "" {
				log.Printf("Query:       %s", r.URL.RawQuery)
			}
			log.Printf("Remote Addr: %s", r.RemoteAddr)
			log.Printf("User-Agent:  %s", r.Header.Get("User-Agent"))

			// Log all headers
			log.Println("Headers:")
			for name, values := range r.Header {
				for _, value := range values {
					log.Printf("  %s: %s", name, value)
				}
			}

			// Try to read and log body (if any)
			if r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				if err == nil && len(bodyBytes) > 0 {
					log.Println("Body:")
					log.Printf("  Length: %d bytes", len(bodyBytes))
					// Limit body output to 1KB
					if len(bodyBytes) > 1024 {
						log.Printf("  Content (first 1KB): %s...", string(bodyBytes[:1024]))
					} else {
						log.Printf("  Content: %s", string(bodyBytes))
					}
					// Restore body for potential further processing
					r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			log.Println("================================================================================")
			log.Println()
		}
	})
}
