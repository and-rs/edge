package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr            string
	AI              AIConfig
	SignalsCacheTTL time.Duration
}

type AIConfig struct {
	AuthMode     string
	BaseURL      string
	Model        string
	APIKey       string
	Organization string
	Project      string
	Timeout      time.Duration
	DebugLogging bool
}

func Load() Config {
	addr := envString("EDGE_ADDR", ":8080")
	aiBaseURL := strings.TrimRight(envString("EDGE_AI_BASE_URL", "https://api.openai.com/v1"), "/")
	timeoutSeconds := envPositiveInt("EDGE_AI_TIMEOUT_SECONDS", 20)
	cacheSeconds := envPositiveInt("EDGE_SIGNALS_CACHE_SECONDS", 20)

	return Config{
		Addr:            addr,
		SignalsCacheTTL: time.Duration(cacheSeconds) * time.Second,
		AI: AIConfig{
			AuthMode:     envString("EDGE_AI_AUTH_MODE", "api-key"),
			BaseURL:      aiBaseURL,
			Model:        os.Getenv("EDGE_AI_MODEL"),
			APIKey:       os.Getenv("EDGE_AI_API_KEY"),
			Organization: os.Getenv("EDGE_AI_ORG_ID"),
			Project:      os.Getenv("EDGE_AI_PROJECT_ID"),
			Timeout:      time.Duration(timeoutSeconds) * time.Second,
			DebugLogging: envBool("EDGE_DEBUG_AI", false),
		},
	}
}

func envString(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envPositiveInt(key string, fallback int) int {
	if raw := os.Getenv(key); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}

	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
