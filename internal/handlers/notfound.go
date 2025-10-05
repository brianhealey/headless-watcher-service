package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// NotFoundHandler handles all unmatched routes (404)
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	// Read request body
	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	// Log the 404 in detail
	log404Request(r, bodyBytes)

	// Return 404 response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  "Not Found",
		"path":   r.URL.Path,
		"method": r.Method,
	})
}

func log404Request(r *http.Request, body []byte) {
	log.Println("================================================================================")
	log.Println("404 NOT FOUND - Unmatched Route")
	log.Println("================================================================================")
	log.Printf("Timestamp:   %s", time.Now().Format(time.RFC3339))
	log.Printf("Action:      %s %s", r.Method, r.URL.Path)
	if r.URL.RawQuery != "" {
		log.Printf("Query:       %s", r.URL.RawQuery)
	}
	log.Printf("Full URL:    %s", r.URL.String())
	log.Printf("Remote Addr: %s", r.RemoteAddr)
	log.Printf("User-Agent:  %s", r.Header.Get("User-Agent"))

	// Log all headers
	log.Println("--------------------------------------------------------------------------------")
	log.Println("REQUEST HEADERS")
	log.Println("--------------------------------------------------------------------------------")
	for name, values := range r.Header {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}

	// Log body if present
	if len(body) > 0 {
		log.Println("--------------------------------------------------------------------------------")
		log.Println("REQUEST BODY")
		log.Println("--------------------------------------------------------------------------------")
		log.Printf("Length: %d bytes", len(body))

		// Try to pretty-print as JSON
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			if formatted, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
				log.Println(string(formatted))
			} else {
				// Not JSON or couldn't format, print raw
				if len(body) > 1024 {
					log.Printf("%s... (truncated, %d total bytes)", string(body[:1024]), len(body))
				} else {
					log.Println(string(body))
				}
			}
		} else {
			// Not JSON, print raw
			if len(body) > 1024 {
				log.Printf("%s... (truncated, %d total bytes)", string(body[:1024]), len(body))
			} else {
				log.Println(string(body))
			}
		}
	}

	log.Println("================================================================================")
	log.Println()
}
