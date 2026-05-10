package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestImagePlaygroundOpenAIImagesAdapterUsesHandlerPathAndReturnsResultJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			require.Equal(t, int64(9), id)
			return apiKey, nil
		},
		invokeImages: func(c *gin.Context) {
			require.Equal(t, "/v1/images/generations", c.Request.URL.Path)
			require.Equal(t, "application/json", c.GetHeader("Content-Type"))
			require.Empty(t, c.GetHeader("Authorization"))
			gotKey, ok := middleware.GetAPIKeyFromContext(c)
			require.True(t, ok)
			require.Equal(t, apiKey.ID, gotKey.ID)
			subject, ok := middleware.GetAuthSubjectFromContext(c)
			require.True(t, ok)
			require.Equal(t, int64(7), subject.UserID)
			require.Equal(t, 3, subject.Concurrency)
			role, ok := middleware.GetUserRoleFromContext(c)
			require.True(t, ok)
			require.Equal(t, service.RoleUser, role)
			c.Header("X-Upstream-Request-ID", "req_123")
			c.JSON(http.StatusOK, gin.H{"created": 1700000000, "data": []gin.H{{"b64_json": "aW1hZ2U="}}})
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		ID:          101,
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.NoError(t, err)
	require.JSONEq(t, `{"created":1700000000,"data":[{"b64_json":"aW1hZ2U="}]}`, string(result))
}

func TestImagePlaygroundOpenAIImagesAdapterInjectsLoadedSubscriptionContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	apiKey.User.Subscriptions = nil
	apiKey.Group.SubscriptionType = service.SubscriptionTypeSubscription
	subscription := &service.UserSubscription{
		ID:        44,
		UserID:    7,
		GroupID:   groupID,
		Status:    service.SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return apiKey, nil
		},
		loadSubscription: func(ctx context.Context, userID, requestedGroupID int64) (*service.UserSubscription, error) {
			require.Equal(t, int64(7), userID)
			require.Equal(t, groupID, requestedGroupID)
			return subscription, nil
		},
		validateSubscription: func(sub *service.UserSubscription, group *service.Group) (bool, error) {
			require.Equal(t, int64(44), sub.ID)
			require.Equal(t, groupID, group.ID)
			return true, nil
		},
		maintainSubscription: func(sub *service.UserSubscription) {
			require.Equal(t, int64(44), sub.ID)
		},
		invokeImages: func(c *gin.Context) {
			got, ok := middleware.GetSubscriptionFromContext(c)
			require.True(t, ok)
			require.Equal(t, int64(44), got.ID)
			c.JSON(http.StatusOK, gin.H{"data": []gin.H{{"url": "data:image/png;base64,aW1hZ2U="}}})
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.NoError(t, err)
	require.True(t, gjson.GetBytes(result, "data.0.url").Exists())
}

func TestImagePlaygroundOpenAIImagesAdapterRequiresSubscriptionLoaderForSubscriptionGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	apiKey.Group.SubscriptionType = service.SubscriptionTypeSubscription
	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return apiKey, nil
		},
		invokeImages: func(c *gin.Context) {
			t.Fatal("handler should not be invoked without a subscription loader")
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.Nil(t, result)
	var execErr service.ImagePlaygroundTaskExecutionError
	require.True(t, errors.As(err, &execErr))
	require.Equal(t, "subscription_loader_missing", execErr.Code)
}

func TestImagePlaygroundOpenAIImagesAdapterRejectsSubscriptionLimitError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	apiKey.Group.SubscriptionType = service.SubscriptionTypeSubscription
	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return apiKey, nil
		},
		loadSubscription: func(ctx context.Context, userID, requestedGroupID int64) (*service.UserSubscription, error) {
			return &service.UserSubscription{ID: 44, UserID: userID, GroupID: requestedGroupID, Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		validateSubscription: func(sub *service.UserSubscription, group *service.Group) (bool, error) {
			return false, service.ErrDailyLimitExceeded
		},
		invokeImages: func(c *gin.Context) {
			t.Fatal("handler should not be invoked when subscription limits fail")
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.Nil(t, result)
	var execErr service.ImagePlaygroundTaskExecutionError
	require.True(t, errors.As(err, &execErr))
	require.Equal(t, "usage_limit_exceeded", execErr.Code)
}

func TestImagePlaygroundOpenAIImagesAdapterTouchesAPIKeyOnAuthorizedExecution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	var touchedID int64
	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return apiKey, nil
		},
		touchAPIKey: func(ctx context.Context, id int64) error {
			touchedID = id
			return nil
		},
		invokeImages: func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []gin.H{{"b64_json": "aW1hZ2U="}}})
		},
	}

	_, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.NoError(t, err)
	require.Equal(t, int64(9), touchedID)
}

func TestImagePlaygroundOpenAIImagesAdapterRejectsUnsupportedPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	apiKey.Group.Platform = service.PlatformGemini
	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return apiKey, nil
		},
		invokeImages: func(c *gin.Context) {
			t.Fatal("handler should not be invoked for unsupported platform")
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.Nil(t, result)
	var execErr service.ImagePlaygroundTaskExecutionError
	require.True(t, errors.As(err, &execErr))
	require.Equal(t, "unsupported_platform", execErr.Code)
}

func TestImagePlaygroundOpenAIImagesAdapterRejectsIPRestrictedAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	apiKey.IPWhitelist = []string{"127.0.0.1"}
	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return apiKey, nil
		},
		invokeImages: func(c *gin.Context) {
			t.Fatal("handler should not be invoked for ip-restricted api keys")
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.Nil(t, result)
	var execErr service.ImagePlaygroundTaskExecutionError
	require.True(t, errors.As(err, &execErr))
	require.Equal(t, "ip_restricted_api_key", execErr.Code)
}

func TestImagePlaygroundOpenAIImagesAdapterDefaultPathCallsImagesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	apiKey.Group.AllowImageGeneration = false
	adapter := &openAIImagesTaskAdapter{
		handler: &OpenAIGatewayHandler{
			gatewayService:      &service.OpenAIGatewayService{},
			billingCacheService: &service.BillingCacheService{},
			apiKeyService:       &service.APIKeyService{},
			concurrencyHelper:   &ConcurrencyHelper{concurrencyService: &service.ConcurrencyService{}},
		},
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return apiKey, nil
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw","size":"1024x1024"}`),
	})

	require.Nil(t, result)
	var execErr service.ImagePlaygroundTaskExecutionError
	require.True(t, errors.As(err, &execErr))
	require.Equal(t, "handler_status_403", execErr.Code)
	require.Contains(t, execErr.Message, service.ImageGenerationPermissionMessage())
}

func TestImagePlaygroundOpenAIImagesAdapterReturnsExecutionErrorForNon2XXWithSafeMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return apiKey, nil
		},
		invokeImages: func(c *gin.Context) {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": gin.H{
					"message": "upstream failed Authorization: Bearer sk-secret-token aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			})
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.Nil(t, result)
	var execErr service.ImagePlaygroundTaskExecutionError
	require.True(t, errors.As(err, &execErr))
	require.Equal(t, "handler_status_502", execErr.Code)
	require.NotContains(t, execErr.Message, "sk-secret-token")
	require.NotContains(t, execErr.Message, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.LessOrEqual(t, len(execErr.Message), 700)
}

func TestImagePlaygroundOpenAIImagesAdapterRejects2XXWithoutImageResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return apiKey, nil
		},
		invokeImages: func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"created": 1700000000, "data": []gin.H{{"revised_prompt": "draw"}}})
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.Nil(t, result)
	var execErr service.ImagePlaygroundTaskExecutionError
	require.True(t, errors.As(err, &execErr))
	require.Equal(t, "invalid_image_result", execErr.Code)
}

func TestImagePlaygroundOpenAIImagesAdapterReturnsExecutionErrorForHandlerMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return apiKey, nil
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.Nil(t, result)
	var execErr service.ImagePlaygroundTaskExecutionError
	require.True(t, errors.As(err, &execErr))
	require.Equal(t, "handler_missing", execErr.Code)
}

func TestImagePlaygroundOpenAIImagesAdapterReturnsExecutionErrorForLoadKeyFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return nil, fmt.Errorf("db failed Authorization: Bearer sk-secret-token")
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.Nil(t, result)
	var execErr service.ImagePlaygroundTaskExecutionError
	require.True(t, errors.As(err, &execErr))
	require.Equal(t, "api_key_load_failed", execErr.Code)
	require.NotContains(t, execErr.Message, "sk-secret-token")
}

func TestImagePlaygroundOpenAIImagesAdapterReturnsExecutionErrorForConcurrencyReject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := imagePlaygroundAdapterTestAPIKey(7, 9, groupID)
	adapter := &openAIImagesTaskAdapter{
		loadAPIKey: func(ctx context.Context, id int64) (*service.APIKey, error) {
			return apiKey, nil
		},
		invokeImages: func(c *gin.Context) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"type": "rate_limit_error", "message": "Image generation concurrency limit exceeded, please retry later"}})
		},
	}

	result, err := adapter.ExecuteImagePlaygroundTask(context.Background(), service.ImagePlaygroundTask{
		UserID:      7,
		APIKeyID:    9,
		Endpoint:    service.ImagePlaygroundEndpointGenerations,
		RequestJSON: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
	})

	require.Nil(t, result)
	var execErr service.ImagePlaygroundTaskExecutionError
	require.True(t, errors.As(err, &execErr))
	require.Equal(t, "handler_status_429", execErr.Code)
	require.Contains(t, execErr.Message, "429")
	require.Contains(t, execErr.Message, "concurrency limit")
}

func imagePlaygroundAdapterTestAPIKey(userID, apiKeyID, groupID int64) *service.APIKey {
	now := time.Now()
	return &service.APIKey{
		ID:      apiKeyID,
		UserID:  userID,
		Key:     "sk-test",
		Status:  service.StatusAPIKeyActive,
		GroupID: &groupID,
		User: &service.User{
			ID:          userID,
			Role:        service.RoleUser,
			Status:      service.StatusActive,
			Balance:     100,
			Concurrency: 3,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Group: &service.Group{
			ID:                   groupID,
			Platform:             service.PlatformOpenAI,
			AllowImageGeneration: true,
		},
	}
}
