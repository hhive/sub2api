package vendorhall

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ListParams struct {
	Window    string
	Search    string
	Group     string
	Status    string
	SortBy    string
	SortOrder string
	Page      int
	PageSize  int
}

var windowDurations = map[string]time.Duration{
	"3h": 3 * time.Hour, "24h": 24 * time.Hour, "3d": 3 * 24 * time.Hour,
}

var supportedSorts = map[string]bool{
	"availability": true, "cache_hit_rate": true, "user_ttft": true, "requests": true, "updated_at": true,
}

func ParseListParams(values url.Values) (ListParams, error) {
	params := ListParams{
		Window: strings.TrimSpace(values.Get("window")), Search: strings.TrimSpace(values.Get("search")),
		Group: strings.TrimSpace(values.Get("group")), Status: strings.ToLower(strings.TrimSpace(values.Get("status"))),
		SortBy: strings.ToLower(strings.TrimSpace(values.Get("sort_by"))), SortOrder: strings.ToLower(strings.TrimSpace(values.Get("sort_order"))),
		Page: 1, PageSize: 20,
	}
	if params.Window == "" {
		params.Window = "24h"
	}
	if params.Status == "" {
		params.Status = "all"
	}
	if params.SortBy == "" {
		params.SortBy = "availability"
	}
	if params.SortOrder == "" {
		params.SortOrder = "desc"
	}
	if _, ok := windowDurations[params.Window]; !ok {
		return ListParams{}, fmt.Errorf("unsupported window")
	}
	if !supportedSorts[params.SortBy] {
		return ListParams{}, fmt.Errorf("unsupported sort")
	}
	if params.SortOrder != "asc" && params.SortOrder != "desc" {
		return ListParams{}, fmt.Errorf("unsupported sort order")
	}
	switch params.Status {
	case "all", "schedulable", "paused", "disabled", "unavailable":
	default:
		return ListParams{}, fmt.Errorf("unsupported status")
	}
	if raw := values.Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return ListParams{}, fmt.Errorf("invalid page")
		}
		params.Page = value
	}
	if raw := values.Get("page_size"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return ListParams{}, fmt.Errorf("invalid page size")
		}
		if value > 100 {
			value = 100
		}
		params.PageSize = value
	}
	return params, nil
}
