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

type VideoPlaygroundHandler struct {
	settingService *service.SettingService
	httpClient     *http.Client
}

type videoPlaygroundModelRequest struct {
	DisplayName     string  `json:"display_name"`
	Model           string  `json:"model"`
	ProviderName    string  `json:"provider_name"`
	UpstreamBaseURL string  `json:"upstream_base_url"`
	UpstreamAPIKey  string  `json:"upstream_api_key"`
	PriceQuota      float64 `json:"price_quota"`
	BillingMode     string  `json:"billing_mode"`
	RefundEnabled   bool    `json:"refund_enabled"`
	TimeoutSeconds  int     `json:"timeout_seconds"`
	Enabled         bool    `json:"enabled"`
	SortOrder       int     `json:"sort_order"`
	StudioModelID   string  `json:"studio_model_id"`
	ModelKind       string  `json:"model_kind"`
	InputSchemaJSON string  `json:"input_schema_json"`
	PayloadMapJSON  string  `json:"payload_mapping_json"`
}

func NewVideoPlaygroundHandler(settingService *service.SettingService) *VideoPlaygroundHandler {
	return &VideoPlaygroundHandler{
		settingService: settingService,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (h *VideoPlaygroundHandler) ListModels(c *gin.Context) {
	h.proxy(c, http.MethodGet, "/api/admin/models", nil)
}

func (h *VideoPlaygroundHandler) ListUpstreamRequests(c *gin.Context) {
	query := c.Request.URL.RawQuery
	path := "/api/admin/upstream-requests"
	if query != "" {
		path += "?" + query
	}
	h.proxy(c, http.MethodGet, path, nil)
}

func (h *VideoPlaygroundHandler) CreateModel(c *gin.Context) {
	var req videoPlaygroundModelRequest
	if err := bindAllowedVideoModelRequest(c, &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.proxy(c, http.MethodPost, "/api/admin/models", req)
}

func (h *VideoPlaygroundHandler) UpdateModel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid model id")
		return
	}
	var req videoPlaygroundModelRequest
	if err := bindAllowedVideoModelRequest(c, &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.proxy(c, http.MethodPatch, fmt.Sprintf("/api/admin/models/%d", id), req)
}

func (h *VideoPlaygroundHandler) DeleteModel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid model id")
		return
	}
	h.proxy(c, http.MethodDelete, fmt.Sprintf("/api/admin/models/%d", id), nil)
}

func bindAllowedVideoModelRequest(c *gin.Context, out *videoPlaygroundModelRequest) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	disallowed := []string{"api_key", "submit_path", "status_path_template", "content_path_template"}
	for _, key := range disallowed {
		if _, ok := raw[key]; ok {
			return fmt.Errorf("%s is not configurable here", key)
		}
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func (h *VideoPlaygroundHandler) proxy(c *gin.Context, method, path string, payload any) {
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
		response.InternalError(c, fmt.Sprintf("video playground request failed: %v", err))
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		response.Error(c, resp.StatusCode, "video playground error: "+strings.TrimSpace(string(respBody)))
		return
	}
	var data any
	if len(strings.TrimSpace(string(respBody))) > 0 {
		if err := json.Unmarshal(respBody, &data); err != nil {
			response.InternalError(c, "invalid video playground response")
			return
		}
	}
	response.Success(c, data)
}

func (h *VideoPlaygroundHandler) videoPlaygroundAdminTarget(c *gin.Context) (string, error) {
	if baseURL := strings.TrimSpace(os.Getenv("VIDEO_PLAYGROUND_ADMIN_BASE_URL")); baseURL != "" {
		return strings.TrimRight(baseURL, "/"), nil
	}
	if baseURL := strings.TrimSpace(os.Getenv("VIDEO_PLAYGROUND_GO_BASE_URL")); baseURL != "" {
		return strings.TrimRight(baseURL, "/"), nil
	}
	baseURL := "http://127.0.0.1:3303"
	settings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err == nil && strings.TrimSpace(settings.VideoPlaygroundBaseURL) != "" {
		if strings.Contains(settings.VideoPlaygroundBaseURL, "video.xiaoni-ai.top") {
			return baseURL, nil
		}
		return strings.TrimRight(strings.TrimSpace(settings.VideoPlaygroundBaseURL), "/"), nil
	}
	if err != nil {
		return "", err
	}
	return baseURL, nil
}
