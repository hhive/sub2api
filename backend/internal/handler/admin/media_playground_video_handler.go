package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type MediaPlaygroundVideoHandler struct {
	settingService *service.SettingService
	httpClient     *http.Client
}

type mediaPlaygroundVideoModelRequest struct {
	DisplayName     string  `json:"display_name"`
	Model           string  `json:"model"`
	ProviderName    string  `json:"provider_name"`
	APIMode         string  `json:"api_mode"`
	UpstreamBaseURL string  `json:"upstream_base_url"`
	UpstreamAPIKey  string  `json:"upstream_api_key"`
	PriceQuota      float64 `json:"price_quota"`
	BillingMode     string  `json:"billing_mode"`
	RefundEnabled   bool    `json:"refund_enabled"`
	TimeoutSeconds  int     `json:"timeout_seconds"`
	Enabled         bool    `json:"enabled"`
	SortOrder       int     `json:"sort_order"`
}

func NewMediaPlaygroundVideoHandler(settingService *service.SettingService) *MediaPlaygroundVideoHandler {
	return &MediaPlaygroundVideoHandler{
		settingService: settingService,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (h *MediaPlaygroundVideoHandler) ListModels(c *gin.Context) {
	h.proxy(c, http.MethodGet, "/api/admin/media/video/models", nil)
}

func (h *MediaPlaygroundVideoHandler) ListUpstreamRequests(c *gin.Context) {
	query := c.Request.URL.RawQuery
	path := "/api/admin/media/video/upstream-requests"
	if query != "" {
		path += "?" + query
	}
	h.proxy(c, http.MethodGet, path, nil)
}

func (h *MediaPlaygroundVideoHandler) CreateModel(c *gin.Context) {
	var req mediaPlaygroundVideoModelRequest
	if err := bindAllowedVideoModelRequest(c, &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.proxy(c, http.MethodPost, "/api/admin/media/video/models", req)
}

func (h *MediaPlaygroundVideoHandler) UpdateModel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid model id")
		return
	}
	var req mediaPlaygroundVideoModelRequest
	if err := bindAllowedVideoModelRequest(c, &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.proxy(c, http.MethodPatch, fmt.Sprintf("/api/admin/media/video/models/%d", id), req)
}

func (h *MediaPlaygroundVideoHandler) DeleteModel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid model id")
		return
	}
	h.proxy(c, http.MethodDelete, fmt.Sprintf("/api/admin/media/video/models/%d", id), nil)
}

func bindAllowedVideoModelRequest(c *gin.Context, out *mediaPlaygroundVideoModelRequest) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	disallowed := []string{"id", "media_type", "upstream_api_key_mask", "studio_model_id", "model_kind", "input_schema_json", "payload_mapping_json", "api_key", "submit_path", "status_path_template", "content_path_template"}
	for _, key := range disallowed {
		if _, ok := raw[key]; ok {
			return fmt.Errorf("%s is not configurable here", key)
		}
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if strings.TrimSpace(out.APIMode) == "" {
		return fmt.Errorf("api_mode is required")
	}
	return nil
}

func (h *MediaPlaygroundVideoHandler) proxy(c *gin.Context, method, path string, payload any) {
	baseURL, err := h.videoPlaygroundAdminTarget(c)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			response.InternalError(c, err.Error())
			return
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), method, baseURL+path, body)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		response.InternalError(c, fmt.Sprintf("media playground video request failed: %v", err))
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		response.Error(c, resp.StatusCode, "media playground video error: "+strings.TrimSpace(string(respBody)))
		return
	}
	var data any
	if len(strings.TrimSpace(string(respBody))) > 0 {
		if err := json.Unmarshal(respBody, &data); err != nil {
			response.InternalError(c, "invalid media playground video response")
			return
		}
	}
	response.Success(c, data)
}

func (h *MediaPlaygroundVideoHandler) videoPlaygroundAdminTarget(_ *gin.Context) (string, error) {
	if baseURL := strings.TrimSpace(os.Getenv("IMAGE_PLAYGROUND_ADMIN_BASE_URL")); baseURL != "" {
		return strings.TrimRight(baseURL, "/"), nil
	}
	if baseURL := strings.TrimSpace(os.Getenv("IMAGE_PLAYGROUND_GO_BASE_URL")); baseURL != "" {
		return strings.TrimRight(baseURL, "/"), nil
	}
	return "http://127.0.0.1:3304", nil
}
