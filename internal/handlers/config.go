package handlers

import "github.com/brianhealey/sensecap-server/internal/config"

// Global configuration (will be set by main.go)
var cfg *config.Config

// SetConfig sets the global configuration for handlers
func SetConfig(c *config.Config) {
	cfg = c
}
