//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type onyxLaunchServiceStub struct {
	createUserID                int64
	createResult                *service.OnyxLaunchResult
	createErr                   error
	createImagePlaygroundUserID int64
	createImagePlaygroundResult *service.OnyxLaunchResult
	createImagePlaygroundErr    error
	createLobeHubUserID         int64
	createLobeHubResult         *service.OnyxLaunchResult
	createLobeHubErr            error
	createVideoPlaygroundUserID int64
	createVideoPlaygroundResult *service.OnyxLaunchResult
	createVideoPlaygroundErr    error
	consumeToken                string
	consumeResult               *service.OnyxLaunchPayload
	consumeErr                  error
	consumeVideoToken           string
	consumeVideoResult          *service.VideoPlaygroundLaunchPayload
	consumeVideoErr             error
}

func (s *onyxLaunchServiceStub) CreateLaunch(ctx context.Context, userID int64) (*service.OnyxLaunchResult, error) {
	s.createUserID = userID
	if s.createErr != nil {
		return nil, s.createErr
	}
	return s.createResult, nil
}

func (s *onyxLaunchServiceStub) CreateImagePlaygroundLaunch(ctx context.Context, userID int64) (*service.OnyxLaunchResult, error) {
	s.createImagePlaygroundUserID = userID
	if s.createImagePlaygroundErr != nil {
		return nil, s.createImagePlaygroundErr
	}
	return s.createImagePlaygroundResult, nil
}

func (s *onyxLaunchServiceStub) CreateLobeHubLaunch(ctx context.Context, userID int64) (*service.OnyxLaunchResult, error) {
	s.createLobeHubUserID = userID
	if s.createLobeHubErr != nil {
		return nil, s.createLobeHubErr
	}
	return s.createLobeHubResult, nil
}

func (s *onyxLaunchServiceStub) CreateVideoPlaygroundLaunch(ctx context.Context, userID int64) (*service.OnyxLaunchResult, error) {
	s.createVideoPlaygroundUserID = userID
	if s.createVideoPlaygroundErr != nil {
		return nil, s.createVideoPlaygroundErr
	}
	return s.createVideoPlaygroundResult, nil
}

func (s *onyxLaunchServiceStub) ConsumeLaunch(ctx context.Context, token string) (*service.OnyxLaunchPayload, error) {
	s.consumeToken = token
	if s.consumeErr != nil {
		return nil, s.consumeErr
	}
	return s.consumeResult, nil
}

func (s *onyxLaunchServiceStub) ConsumeVideoPlaygroundLaunch(ctx context.Context, token string) (*service.VideoPlaygroundLaunchPayload, error) {
	s.consumeVideoToken = token
	if s.consumeVideoErr != nil {
		return nil, s.consumeVideoErr
	}
	return s.consumeVideoResult, nil
}

func TestOnyxHandler_Launch_ReturnsRedirectURLForAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	launchSvc := &onyxLaunchServiceStub{
		createResult: &service.OnyxLaunchResult{RedirectURL: "https://onyx.example.com/api/sub2api/exchange?token=abc"},
	}
	handler := NewOnyxHandler(launchSvc, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/onyx/launch", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.Launch(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(42), launchSvc.createUserID)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			RedirectURL string `json:"redirect_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "https://onyx.example.com/api/sub2api/exchange?token=abc", resp.Data.RedirectURL)
}

func TestOnyxHandler_Launch_ReturnsUnauthorizedWithoutAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewOnyxHandler(&onyxLaunchServiceStub{}, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/onyx/launch", nil)

	handler.Launch(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestOnyxHandler_ImagePlaygroundLaunch_ReturnsRedirectURLForAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	launchSvc := &onyxLaunchServiceStub{
		createImagePlaygroundResult: &service.OnyxLaunchResult{RedirectURL: "https://xiaoni-ai.zle.ee/image_playground?apiMode=images"},
	}
	handler := NewOnyxHandler(launchSvc, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/image-playground/launch", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})

	handler.ImagePlaygroundLaunch(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(42), launchSvc.createImagePlaygroundUserID)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			RedirectURL string `json:"redirect_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "https://xiaoni-ai.zle.ee/image_playground?apiMode=images", resp.Data.RedirectURL)
}

func TestOnyxHandler_ImagePlaygroundLaunch_ReturnsUnauthorizedWithoutAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewOnyxHandler(&onyxLaunchServiceStub{}, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/image-playground/launch", nil)

	handler.ImagePlaygroundLaunch(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestOnyxHandler_Exchange_ReturnsUnauthorizedWhenSecretMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	launchSvc := &onyxLaunchServiceStub{}
	handler := NewOnyxHandler(launchSvc, newOnyxTestSettingService("exchange-secret"))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/onyx/exchange", bytes.NewBufferString(`{"token":"launch-token"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Exchange(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Empty(t, launchSvc.consumeToken)
}

func TestOnyxHandler_Exchange_ReturnsPayloadWhenSecretValid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	launchSvc := &onyxLaunchServiceStub{
		consumeResult: &service.OnyxLaunchPayload{
			UserID:         42,
			Email:          "user@example.com",
			Username:       "onyx-user",
			APIKeyID:       1001,
			APIKey:         "sk-user-key",
			APIBaseURL:     "https://sub2api.example.com",
			TextModelName:  "gpt-5.5-mini",
			ImageModelName: "gpt-image-2",
		},
	}
	handler := NewOnyxHandler(launchSvc, newOnyxTestSettingService("exchange-secret"))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/onyx/exchange", bytes.NewBufferString(`{"token":"launch-token"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set(onyxExchangeSecretHeader, "exchange-secret")

	handler.Exchange(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "launch-token", launchSvc.consumeToken)

	var resp struct {
		Code int                       `json:"code"`
		Data service.OnyxLaunchPayload `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(42), resp.Data.UserID)
	require.Equal(t, "user@example.com", resp.Data.Email)
	require.Equal(t, "sk-user-key", resp.Data.APIKey)
	require.Equal(t, "gpt-image-2", resp.Data.ImageModelName)
}

func TestOnyxHandler_Exchange_ReturnsBadRequestWhenTokenEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	launchSvc := &onyxLaunchServiceStub{}
	handler := NewOnyxHandler(launchSvc, newOnyxTestSettingService("exchange-secret"))

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/onyx/exchange", bytes.NewBufferString(`{"token":"  "}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set(onyxExchangeSecretHeader, "exchange-secret")

	handler.Exchange(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Empty(t, launchSvc.consumeToken)
}

func newOnyxTestSettingService(secret string) *service.SettingService {
	return service.NewSettingService(&settingRepoStub{values: map[string]string{
		service.SettingKeyOnyxExchangeSecret: secret,
	}}, &config.Config{})
}

type settingRepoStub struct {
	values map[string]string
}

func (s *settingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (s *settingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *settingRepoStub) Set(ctx context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *settingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *settingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *settingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	result := make(map[string]string, len(s.values))
	for key, value := range s.values {
		result[key] = value
	}
	return result, nil
}

func (s *settingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}
