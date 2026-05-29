package service

import (
	"encoding/json"
	"net/mail"
	"strings"
)

var DefaultLobeHubAllowedEmails = []string{
	"oncethewindblows@gmail.com",
	"136469770@qq.com",
	"1643689728@qq.com",
	"2910703711@qq.com",
	"1312623967@qq.com",
}

const defaultLobeHubAllowedEmailsValue = "oncethewindblows@gmail.com\n136469770@qq.com\n1643689728@qq.com\n2910703711@qq.com\n1312623967@qq.com"

func ParseLobeHubAllowedEmails(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}

	items := parseLobeHubAllowedEmailItems(raw)
	normalized := normalizeLobeHubAllowedEmails(items)
	if len(normalized) == 0 {
		return []string{}
	}
	return normalized
}

func NormalizeLobeHubAllowedEmailsValue(raw string) string {
	return strings.Join(ParseLobeHubAllowedEmails(raw), "\n")
}

func lobeHubAllowedEmailsForSettingsView(settings map[string]string) string {
	raw, ok := settings[SettingKeyLobeHubAllowedEmails]
	if !ok {
		return defaultLobeHubAllowedEmailsValue
	}
	return raw
}

func parseLobeHubAllowedEmailItems(raw string) []string {
	var jsonItems []string
	if strings.HasPrefix(raw, "[") && json.Unmarshal([]byte(raw), &jsonItems) == nil {
		return jsonItems
	}

	splitter := func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';'
	}
	return strings.FieldsFunc(raw, splitter)
}

func normalizeLobeHubAllowedEmails(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		email := strings.ToLower(strings.TrimSpace(item))
		if email == "" {
			continue
		}
		addr, err := mail.ParseAddress(email)
		if err != nil || addr.Address != email || !strings.Contains(email, "@") {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		out = append(out, email)
	}
	return out
}
