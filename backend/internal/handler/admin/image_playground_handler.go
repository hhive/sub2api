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

type ImagePlaygroundHandler struct {
	settingService *service.SettingService
	httpClient     *http.Client
}

type imagePlaygroundModelRequest struct {
	DisplayName     string   `json:"display_name"`
	Model           string   `json:"model"`
	APIMode         string   `json:"api_mode"`
	ProviderName    string   `json:"provider_name"`
	UpstreamBaseURL string   `json:"upstream_base_url"`
	UpstreamAPIKey  string   `json:"upstream_api_key"`
	Price1K         float64  `json:"price_1k"`
	Price2K         float64  `json:"price_2k"`
	Price4K         float64  `json:"price_4k"`
	SupportedSizes  []string `json:"supported_sizes"`
	TimeoutSeconds  int      `json:"timeout_seconds"`
	Enabled         bool     `json:"enabled"`
	SortOrder       int      `json:"sort_order"`
}

func NewImagePlaygroundHandler(settingService *service.SettingService) *ImagePlaygroundHandler {
	return &ImagePlaygroundHandler{
		settingService: settingService,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (h *ImagePlaygroundHandler) ListModels(c *gin.Context) {
	h.proxy(c, http.MethodGet, "/api/admin/models", nil)
}

func (h *ImagePlaygroundHandler) CreateModel(c *gin.Context) {
	var req imagePlaygroundModelRequest
	if err := bindAllowedImageModelRequest(c, &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.proxy(c, http.MethodPost, "/api/admin/models", req)
}

func (h *ImagePlaygroundHandler) UpdateModel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid model id")
		return
	}
	var req imagePlaygroundModelRequest
	if err := bindAllowedImageModelRequest(c, &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.proxy(c, http.MethodPatch, fmt.Sprintf("/api/admin/models/%d", id), req)
}

func (h *ImagePlaygroundHandler) DeleteModel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid model id")
		return
	}
	h.proxy(c, http.MethodDelete, fmt.Sprintf("/api/admin/models/%d", id), nil)
}

func bindAllowedImageModelRequest(c *gin.Context, out *imagePlaygroundModelRequest) error {
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

func (h *ImagePlaygroundHandler) proxy(c *gin.Context, method, path string, payload any) {
	baseURL, err := h.imagePlaygroundAdminTarget(c)
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
		response.InternalError(c, fmt.Sprintf("image playground request failed: %v", err))
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		response.Error(c, resp.StatusCode, "image playground error: "+strings.TrimSpace(string(respBody)))
		return
	}
	var data any
	if len(strings.TrimSpace(string(respBody))) > 0 {
		if err := json.Unmarshal(respBody, &data); err != nil {
			response.InternalError(c, "invalid image playground response")
			return
		}
	}
	response.Success(c, data)
}

func (h *ImagePlaygroundHandler) imagePlaygroundAdminTarget(c *gin.Context) (string, error) {
	if baseURL := strings.TrimSpace(os.Getenv("IMAGE_PLAYGROUND_ADMIN_BASE_URL")); baseURL != "" {
		return strings.TrimRight(baseURL, "/"), nil
	}
	if baseURL := strings.TrimSpace(os.Getenv("IMAGE_PLAYGROUND_GO_BASE_URL")); baseURL != "" {
		return strings.TrimRight(baseURL, "/"), nil
	}
	return "http://127.0.0.1:3304", nil
}
