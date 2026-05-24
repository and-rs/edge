package config

import (
	"os"
	"path/filepath"
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
	AuthMode         string
	BaseURL          string
	Model            string
	ReasoningEffort  string
	APIKey           string
	Organization     string
	Project          string
	OAuthProfilePath string
	Timeout          time.Duration
	DebugLogging     bool
}

func Load() Config {
	addr := envString("STINT_ADDR", ":8080")
	aiBaseURL := strings.TrimRight(envString("STINT_AI_BASE_URL", "https://api.openai.com/v1"), "/")
	timeoutSeconds := envPositiveInt("STINT_AI_TIMEOUT_SECONDS", 20)
	cacheSeconds := envPositiveInt("STINT_SIGNALS_CACHE_SECONDS", 20)
	oauthProfilePath := os.Getenv("STINT_AI_OAUTH_PROFILE")
	if oauthProfilePath == "" {
		configDir, err := os.UserConfigDir()
		if err == nil {
			oauthProfilePath = filepath.Join(configDir, "iridium-edge", "openai-oauth.json")
		} else {
			oauthProfilePath = filepath.Join(".", ".openai-oauth.json")
		}
	}

	authMode := envString("STINT_AI_AUTH_MODE", "api-key")
	model := os.Getenv("STINT_AI_MODEL")
	if authMode == "api-key" {
		if apiModel := os.Getenv("STINT_AI_API_MODEL"); apiModel != "" {
			model = apiModel
		}
	}

	return Config{
		Addr:            addr,
		SignalsCacheTTL: time.Duration(cacheSeconds) * time.Second,
		AI: AIConfig{
			AuthMode:         authMode,
			BaseURL:          aiBaseURL,
			Model:            model,
			ReasoningEffort:  os.Getenv("STINT_AI_REASONING_EFFORT"),
			APIKey:           os.Getenv("STINT_AI_API_KEY"),
			Organization:     os.Getenv("STINT_AI_ORG_ID"),
			Project:          os.Getenv("STINT_AI_PROJECT_ID"),
			OAuthProfilePath: oauthProfilePath,
			Timeout:          time.Duration(timeoutSeconds) * time.Second,
			DebugLogging:     envBool("STINT_DEBUG_AI", false),
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
