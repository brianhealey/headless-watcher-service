package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	AI       AIConfig
	Auth     AuthConfig
	API      APIConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// APIConfig holds external API endpoint configuration
type APIConfig struct {
	BaseURL string // Base URL for external API calls (e.g., "http://localhost:8834")
	Schema  string // URL schema (http or https)
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path string
}

// AIConfig holds AI service URLs and models
type AIConfig struct {
	WhisperURL   string
	OllamaURL    string
	OllamaModel  string
	LLaVAModel   string
	PiperURL     string
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Token   string
	Enabled bool
}

// Load reads configuration from flags and environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Define flags
	port := flag.String("port", "8834", "Server port")
	host := flag.String("host", "localhost", "Server host")
	token := flag.String("token", "", "Required authentication token (optional)")
	dbPath := flag.String("db", "sensecap.db", "Path to SQLite database file")

	whisperURL := flag.String("whisper-url", "http://localhost:5000", "Whisper STT service URL")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama LLM service URL")
	ollamaModel := flag.String("ollama-model", "llama3.1:8b-instruct-q4_1", "Ollama model name")
	llavaModel := flag.String("llava-model", "llava:7b", "LLaVA vision model name")
	piperURL := flag.String("piper-url", "http://localhost:5000", "Piper TTS service URL")

	apiSchema := flag.String("api-schema", "http", "API URL schema (http or https)")
	apiBaseURL := flag.String("api-base-url", "", "API base URL (defaults to http://host:port)")

	flag.Parse()

	// Override with environment variables if set
	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}
	if envHost := os.Getenv("HOST"); envHost != "" {
		*host = envHost
	}
	if envToken := os.Getenv("AUTH_TOKEN"); envToken != "" {
		*token = envToken
	}
	if envDB := os.Getenv("DB_PATH"); envDB != "" {
		*dbPath = envDB
	}
	if envWhisper := os.Getenv("WHISPER_URL"); envWhisper != "" {
		*whisperURL = envWhisper
	}
	if envOllama := os.Getenv("OLLAMA_URL"); envOllama != "" {
		*ollamaURL = envOllama
	}
	if envOllamaModel := os.Getenv("OLLAMA_MODEL"); envOllamaModel != "" {
		*ollamaModel = envOllamaModel
	}
	if envLLaVA := os.Getenv("LLAVA_MODEL"); envLLaVA != "" {
		*llavaModel = envLLaVA
	}
	if envPiper := os.Getenv("PIPER_URL"); envPiper != "" {
		*piperURL = envPiper
	}
	if envAPISchema := os.Getenv("API_SCHEMA"); envAPISchema != "" {
		*apiSchema = envAPISchema
	}
	if envAPIBaseURL := os.Getenv("API_BASE_URL"); envAPIBaseURL != "" {
		*apiBaseURL = envAPIBaseURL
	}

	// Build default API base URL if not provided
	if *apiBaseURL == "" {
		*apiBaseURL = fmt.Sprintf("%s://%s:%s", *apiSchema, *host, *port)
	}

	// Build config
	cfg.Server = ServerConfig{
		Port:         *port,
		Host:         *host,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	cfg.Database = DatabaseConfig{
		Path: *dbPath,
	}

	cfg.AI = AIConfig{
		WhisperURL:  *whisperURL,
		OllamaURL:   *ollamaURL,
		OllamaModel: *ollamaModel,
		LLaVAModel:  *llavaModel,
		PiperURL:    *piperURL,
	}

	cfg.Auth = AuthConfig{
		Token:   *token,
		Enabled: *token != "",
	}

	cfg.API = APIConfig{
		BaseURL: *apiBaseURL,
		Schema:  *apiSchema,
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}
	if c.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}
	if c.AI.WhisperURL == "" {
		return fmt.Errorf("whisper URL cannot be empty")
	}
	if c.AI.OllamaURL == "" {
		return fmt.Errorf("ollama URL cannot be empty")
	}
	if c.AI.PiperURL == "" {
		return fmt.Errorf("piper URL cannot be empty")
	}
	return nil
}
