package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
)

const defaultAdminExternalAppsPath = "/etc/sub2api/admin-external-apps.json"

const maxAdminExternalAppIDLength = 64

var adminExternalAppIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type AdminExternalApp struct {
	AppID          string `json:"app_id"`
	LabelEN        string `json:"label_en"`
	LabelZH        string `json:"label_zh"`
	LaunchURL      string `json:"launch_url"`
	ExchangeSecret string `json:"exchange_secret"`
	Enabled        bool   `json:"enabled"`
	SortOrder      int    `json:"sort_order"`
}

type AdminExternalAppRegistry interface {
	List() []AdminExternalApp
	Get(appID string) (AdminExternalApp, bool)
}

type fileAdminExternalAppRegistry struct {
	apps map[string]AdminExternalApp
}

func ProvideAdminExternalAppRegistry() (AdminExternalAppRegistry, error) {
	configPath := strings.TrimSpace(os.Getenv("ADMIN_EXTERNAL_APPS_CONFIG_PATH"))
	if configPath == "" {
		configPath = defaultAdminExternalAppsPath
	}
	return LoadAdminExternalAppRegistry(configPath)
}

func LoadAdminExternalAppRegistry(configPath string) (AdminExternalAppRegistry, error) {
	file, err := os.Open(configPath)
	if errors.Is(err, os.ErrNotExist) {
		return &fileAdminExternalAppRegistry{apps: map[string]AdminExternalApp{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load admin external app registry: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat admin external app registry: %w", err)
	}
	if info.Mode().Perm()&0137 != 0 {
		return nil, errors.New("admin external app registry permissions must not exceed 0640")
	}

	var apps []AdminExternalApp
	decoder := json.NewDecoder(io.LimitReader(file, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&apps); err != nil {
		return nil, errors.New("invalid admin external app registry JSON")
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, errors.New("invalid admin external app registry JSON")
	}

	registry := &fileAdminExternalAppRegistry{apps: make(map[string]AdminExternalApp, len(apps))}
	for i := range apps {
		app := apps[i]
		if err := validateAdminExternalApp(app); err != nil {
			return nil, fmt.Errorf("invalid admin external app registry entry %d: %w", i, err)
		}
		if _, exists := registry.apps[app.AppID]; exists {
			return nil, fmt.Errorf("invalid admin external app registry entry %d: duplicate app_id", i)
		}
		registry.apps[app.AppID] = app
	}
	return registry, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return errors.New("unexpected trailing data")
}

func validateAdminExternalApp(app AdminExternalApp) error {
	if !IsValidAdminExternalAppID(app.AppID) {
		return errors.New("app_id must contain only lowercase letters, digits, and hyphens")
	}
	if strings.TrimSpace(app.LabelEN) == "" || strings.TrimSpace(app.LabelZH) == "" {
		return errors.New("labels are required")
	}
	if len(app.ExchangeSecret) < 32 {
		return errors.New("exchange_secret is too weak")
	}
	parsed, err := url.Parse(app.LaunchURL)
	if err != nil || parsed.Host == "" || parsed.Path == "" || parsed.Path == "/" {
		return errors.New("launch_url is invalid")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("launch_url must not contain userinfo, query, or fragment")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())) {
		return errors.New("launch_url must use HTTPS")
	}
	return nil
}

func IsValidAdminExternalAppID(appID string) bool {
	return len(appID) <= maxAdminExternalAppIDLength && adminExternalAppIDPattern.MatchString(appID)
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (r *fileAdminExternalAppRegistry) List() []AdminExternalApp {
	apps := make([]AdminExternalApp, 0, len(r.apps))
	for _, app := range r.apps {
		apps = append(apps, app)
	}
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].SortOrder == apps[j].SortOrder {
			return apps[i].AppID < apps[j].AppID
		}
		return apps[i].SortOrder < apps[j].SortOrder
	})
	return apps
}

func (r *fileAdminExternalAppRegistry) Get(appID string) (AdminExternalApp, bool) {
	app, ok := r.apps[appID]
	return app, ok
}
