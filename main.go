package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/brianhealey/sensecap-server/database"
	"github.com/brianhealey/sensecap-server/handlers"
	"github.com/brianhealey/sensecap-server/middleware"
	"github.com/gorilla/mux"
)

const (
	defaultPort  = "8834"
	defaultToken = ""
)

func main() {
	// Parse command-line flags
	port := flag.String("port", defaultPort, "Server port")
	token := flag.String("token", defaultToken, "Required authentication token (optional)")
	dbPath := flag.String("db", "sensecap.db", "Path to SQLite database file")
	flag.Parse()

	// Override with environment variables if set
	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}
	if envToken := os.Getenv("AUTH_TOKEN"); envToken != "" {
		*token = envToken
	}
	if envDB := os.Getenv("DB_PATH"); envDB != "" {
		*dbPath = envDB
	}

	// Initialize database
	if err := database.Initialize(*dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create router
	r := mux.NewRouter()

	// Apply global middleware
	r.Use(middleware.CORS)
	r.Use(middleware.Logger)
	r.Use(middleware.DeviceEUIValidator)

	// V1 API routes
	v1 := r.PathPrefix("/v1").Subrouter()

	// Apply authentication middleware if token is configured
	if *token != "" {
		log.Printf("Authentication enabled with token: %s", *token)
		v1.Use(middleware.AuthValidator(*token))
	} else {
		log.Println("WARNING: Authentication disabled (no token configured)")
	}

	// Register V1 endpoints
	v1.HandleFunc("/notification/event", handlers.NotificationHandler).Methods("POST")
	v1.HandleFunc("/watcher/vision", handlers.VisionHandler).Methods("POST")

	// V2 API routes
	v2 := r.PathPrefix("/v2").Subrouter()

	// Apply authentication middleware to v2 if token is configured
	if *token != "" {
		v2.Use(middleware.AuthValidator(*token))
	}

	// Register V2 endpoints
	v2.HandleFunc("/watcher/talk/audio_stream", handlers.AudioStreamHandler).Methods("POST")
	v2.HandleFunc("/watcher/talk/view_task_detail", handlers.TaskDetailHandler).Methods("POST")

	// Health check endpoint (no auth required)
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","service":"sensecap-local-server"}`)
	}).Methods("GET")

	// Catch-all 404 handler - must be last
	r.PathPrefix("/").HandlerFunc(handlers.NotFoundHandler)

	// Print startup information
	printBanner(*port, *token)

	// Start server
	addr := ":" + *port
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func printBanner(port, token string) {
	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("  SenseCAP Watcher Local Server")
	fmt.Println("================================================================================")
	fmt.Println()
	fmt.Println("Server Configuration:")
	fmt.Printf("  Port:           %s\n", port)
	if token != "" {
		fmt.Printf("  Auth Token:     %s\n", token)
		fmt.Println("  Authentication: ENABLED")
	} else {
		fmt.Println("  Auth Token:     (not configured)")
		fmt.Println("  Authentication: DISABLED")
	}
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  V1 API:")
	fmt.Printf("    POST http://localhost:%s/v1/notification/event\n", port)
	fmt.Printf("    POST http://localhost:%s/v1/watcher/vision\n", port)
	fmt.Println("  V2 API:")
	fmt.Printf("    POST http://localhost:%s/v2/watcher/talk/audio_stream\n", port)
	fmt.Printf("    POST http://localhost:%s/v2/watcher/talk/view_task_detail\n", port)
	fmt.Println("  Health:")
	fmt.Printf("    GET  http://localhost:%s/health\n", port)
	fmt.Println()
	fmt.Println("Configuration Headers Required:")
	fmt.Println("  Authorization:            <token>              (if auth enabled)")
	fmt.Println("  API-OBITER-DEVICE-EUI:    <16-char hex EUI>")
	fmt.Println()
	fmt.Println("To configure your SenseCAP Watcher device:")
	fmt.Println()
	fmt.Println("  AT+localservice={\"data\":{\"notification_proxy\":{")
	fmt.Printf("    \"switch\":1,\"url\":\"http://<your-ip>:%s\",\"token\":\"%s\"}}}\n", port, token)
	fmt.Println()
	fmt.Println("  AT+localservice={\"data\":{\"image_analyzer\":{")
	fmt.Printf("    \"switch\":1,\"url\":\"http://<your-ip>:%s\",\"token\":\"%s\"}}}\n", port, token)
	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println()
	log.Println("Server ready to receive requests...")
	fmt.Println()
}
