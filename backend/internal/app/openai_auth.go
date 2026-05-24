package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/index/stint/backend/internal/config"
)

const (
	openAIAuthClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	openAIAuthURL      = "https://auth.openai.com/oauth/authorize"
	openAITokenURL     = "https://auth.openai.com/oauth/token"
	openAIRedirectURI  = "http://127.0.0.1:1455/auth/callback"
)

type openAIOAuthProfile struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	TokenType    string    `json:"token_type"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type openAITokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
}

type openAIOAuthTokenSource struct {
	cfg config.AIConfig
	mu  sync.Mutex
}

func LoginOpenAI(cfg config.AIConfig, output io.Writer) error {
	state, err := randomOAuthValue(32)
	if err != nil {
		return err
	}
	verifier, err := randomOAuthValue(48)
	if err != nil {
		return err
	}
	challenge := sha256.Sum256([]byte(verifier))
	challengeValue := base64.RawURLEncoding.EncodeToString(challenge[:])

	callbackCode := make(chan string, 1)
	callbackError := make(chan error, 1)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/callback" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("state") != state {
			callbackError <- fmt.Errorf("OpenAI OAuth state mismatch")
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		if oauthError := r.URL.Query().Get("error"); oauthError != "" {
			callbackError <- fmt.Errorf("OpenAI OAuth error: %s", oauthError)
			http.Error(w, oauthError, http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			callbackError <- fmt.Errorf("OpenAI OAuth callback missing code")
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte("OpenAI login complete. Return to terminal."))
		callbackCode <- code
	})}

	listener, err := net.Listen("tcp", "127.0.0.1:1455")
	if err != nil {
		return fmt.Errorf("listen for OpenAI OAuth callback: %w", err)
	}
	defer listener.Close()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			callbackError <- err
		}
	}()

	authURL := buildOpenAIAuthURL(state, challengeValue)
	_, _ = fmt.Fprintf(output, "OpenAI login URL:\n%s\n", authURL)
	_ = openBrowser(authURL)

	select {
	case err := <-callbackError:
		return err
	case code := <-callbackCode:
		profile, err := exchangeOpenAICode(cfg, code, verifier)
		if err != nil {
			return err
		}
		if err := saveOpenAIProfile(cfg.OAuthProfilePath, profile); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(output, "saved OpenAI OAuth profile at %s\n", cfg.OAuthProfilePath)
		return nil
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("timed out waiting for OpenAI OAuth callback")
	}
}

func PrintOpenAIStatus(cfg config.AIConfig, output io.Writer) error {
	profile, err := loadOpenAIProfile(cfg.OAuthProfilePath)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(output, "profile: %s\n", cfg.OAuthProfilePath)
	_, _ = fmt.Fprintf(output, "expires: %s\n", profile.ExpiresAt.Format(time.RFC3339))
	_, _ = fmt.Fprintf(output, "has_refresh_token: %t\n", profile.RefreshToken != "")
	return nil
}

func LogoutOpenAI(cfg config.AIConfig) error {
	if err := os.Remove(cfg.OAuthProfilePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func newOpenAIOAuthTokenSource(cfg config.AIConfig) *openAIOAuthTokenSource {
	return &openAIOAuthTokenSource{cfg: cfg}
}

func (s *openAIOAuthTokenSource) accessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, err := loadOpenAIProfile(s.cfg.OAuthProfilePath)
	if err != nil {
		return "", err
	}
	if profile.AccessToken != "" && time.Until(profile.ExpiresAt) > time.Minute {
		return profile.AccessToken, nil
	}
	if profile.RefreshToken == "" {
		return "", fmt.Errorf("OpenAI OAuth profile missing refresh token")
	}

	nextProfile, err := refreshOpenAIProfile(ctx, s.cfg, profile)
	if err != nil {
		return "", err
	}
	if err := saveOpenAIProfile(s.cfg.OAuthProfilePath, nextProfile); err != nil {
		return "", err
	}
	return nextProfile.AccessToken, nil
}

func buildOpenAIAuthURL(state string, challenge string) string {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", openAIAuthClientID)
	params.Set("redirect_uri", openAIRedirectURI)
	params.Set("scope", "openid profile email offline_access")
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	params.Set("state", state)
	params.Set("id_token_add_organizations", "true")
	params.Set("codex_cli_simplified_flow", "true")
	return openAIAuthURL + "?" + params.Encode()
}

func exchangeOpenAICode(cfg config.AIConfig, code string, verifier string) (openAIOAuthProfile, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("redirect_uri", openAIRedirectURI)
	values.Set("client_id", openAIAuthClientID)
	values.Set("code_verifier", verifier)
	return exchangeOpenAITokenRequest(cfg, values)
}

func refreshOpenAIProfile(ctx context.Context, cfg config.AIConfig, profile openAIOAuthProfile) (openAIOAuthProfile, error) {
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", profile.RefreshToken)
	values.Set("client_id", openAIAuthClientID)
	nextProfile, err := exchangeOpenAITokenRequestWithContext(ctx, cfg, values)
	if err != nil {
		return openAIOAuthProfile{}, err
	}
	if nextProfile.RefreshToken == "" {
		nextProfile.RefreshToken = profile.RefreshToken
	}
	if nextProfile.IDToken == "" {
		nextProfile.IDToken = profile.IDToken
	}
	return nextProfile, nil
}

func exchangeOpenAITokenRequest(cfg config.AIConfig, values url.Values) (openAIOAuthProfile, error) {
	return exchangeOpenAITokenRequestWithContext(context.Background(), cfg, values)
}

func exchangeOpenAITokenRequestWithContext(ctx context.Context, cfg config.AIConfig, values url.Values) (openAIOAuthProfile, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, openAITokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return openAIOAuthProfile{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := &http.Client{Timeout: cfg.Timeout}
	response, err := httpClient.Do(request)
	if err != nil {
		return openAIOAuthProfile{}, err
	}
	defer response.Body.Close()

	var tokenResponse openAITokenResponse
	if err := json.NewDecoder(response.Body).Decode(&tokenResponse); err != nil {
		return openAIOAuthProfile{}, err
	}
	if response.StatusCode >= http.StatusBadRequest {
		return openAIOAuthProfile{}, fmt.Errorf("OpenAI OAuth token status %d", response.StatusCode)
	}
	if tokenResponse.AccessToken == "" {
		return openAIOAuthProfile{}, fmt.Errorf("OpenAI OAuth token missing access_token")
	}

	return openAIOAuthProfile{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
		IDToken:      tokenResponse.IDToken,
		TokenType:    tokenResponse.TokenType,
		Scope:        tokenResponse.Scope,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second),
	}, nil
}

func loadOpenAIProfile(path string) (openAIOAuthProfile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return openAIOAuthProfile{}, err
	}
	var profile openAIOAuthProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return openAIOAuthProfile{}, err
	}
	return profile, nil
}

func saveOpenAIProfile(path string, profile openAIOAuthProfile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o600)
}

func randomOAuthValue(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func openBrowser(target string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	if err := command.Start(); err != nil {
		return nil
	}
	return nil
}
