package service

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
)

const (
	UserAppScopeKeysRead   = "keys:read"
	UserAppScopeModelsRead = "models:read"
	UserAppScopeChatInvoke = "chat:invoke"
)

var userAppSecretEnvPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

type userAppIntegration struct {
	AppID     string   `json:"app_id"`
	Enabled   bool     `json:"enabled"`
	LaunchURL string   `json:"launch_url"`
	SecretEnv string   `json:"secret_env"`
	Scopes    []string `json:"scopes"`
	secret    [sha256.Size]byte
}

func loadUserAppIntegrations(configPath string) (map[string]userAppIntegration, error) {
	registry := make(map[string]userAppIntegration)
	if configPath == "" {
		return registry, nil
	}
	file, err := os.Open(configPath)
	if errors.Is(err, os.ErrNotExist) {
		return registry, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load user app integration registry: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat user app integration registry: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0137 != 0 || info.Size() > 1<<20 {
		return nil, errors.New("user app integration registry must be a regular file up to 1 MiB with permissions up to 0640")
	}
	var apps []userAppIntegration
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&apps) != nil || apps == nil || ensureJSONEOF(decoder) != nil {
		return nil, errors.New("invalid user app integration registry JSON")
	}

	references := make(map[string]bool)
	digests := make(map[[sha256.Size]byte]bool)
	for i, app := range apps {
		if !IsValidAdminExternalAppID(app.AppID) || registry[app.AppID].AppID != "" {
			return nil, fmt.Errorf("user app integration entry %d has invalid or duplicate app_id", i)
		}
		if !validUserAppLaunchURL(app.LaunchURL) {
			return nil, fmt.Errorf("user app integration entry %d has invalid HTTPS launch_url", i)
		}
		if app.Scopes == nil {
			return nil, fmt.Errorf("user app integration entry %d must declare scopes", i)
		}
		for j, scope := range app.Scopes {
			if !validUserAppScope(scope) || slices.Contains(app.Scopes[:j], scope) {
				return nil, fmt.Errorf("user app integration entry %d has invalid or duplicate scope", i)
			}
		}
		if !userAppSecretEnvPattern.MatchString(app.SecretEnv) || references[app.SecretEnv] {
			return nil, fmt.Errorf("user app integration entry %d has invalid or duplicate secret_env", i)
		}
		references[app.SecretEnv] = true
		secret := os.Getenv(app.SecretEnv)
		if app.Enabled || secret != "" {
			app.secret = sha256.Sum256([]byte(secret))
			if len(secret) < 32 || strings.TrimSpace(secret) != secret || digests[app.secret] {
				return nil, fmt.Errorf("user app integration entry %d requires a strong independent secret", i)
			}
			digests[app.secret] = true
		}
		registry[app.AppID] = app
	}
	return registry, nil
}

func validUserAppLaunchURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" &&
		strings.HasPrefix(parsed.Path, "/") && !strings.HasSuffix(parsed.Path, "/") && parsed.User == nil &&
		!strings.ContainsAny(raw, "?#\\;,* \t\r\n") && parsed.RawPath == ""
}

func validUserAppScope(scope string) bool {
	switch scope {
	case UserAppScopeKeysRead, UserAppScopeModelsRead, UserAppScopeChatInvoke:
		return true
	default:
		return false
	}
}
