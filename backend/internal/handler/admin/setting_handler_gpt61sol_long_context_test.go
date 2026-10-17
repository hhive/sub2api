package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 覆盖设置项的完整往返：请求省略必须保持现值（否则每次保存都会把开关重置成默认值），
// 显式布尔必须写入，非法类型必须 400 且不写库，响应必须回带该字段供前端回填。
func TestSettingHandler_GPT61SolLongContextBillingRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := service.SettingKeyOpenAIGPT61SolLongContextBillingEnabled
	for _, tt := range []struct {
		name    string
		initial map[string]string
		body    string
		stored  string
		value   any
		status  int
	}{
		{"absent keeps default off", nil, `{}`, "false", false, http.StatusOK},
		{"enable", nil, `{"openai_gpt61sol_long_context_billing_enabled":true}`, "true", true, http.StatusOK},
		{"disable is explicit", map[string]string{key: "true"}, `{"openai_gpt61sol_long_context_billing_enabled":false}`, "false", false, http.StatusOK},
		{"omitted preserves stored", map[string]string{key: "true"}, `{}`, "true", true, http.StatusOK},
		{"string is rejected", nil, `{"openai_gpt61sol_long_context_billing_enabled":"yes"}`, "", nil, http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &settingHandlerRepoStub{values: tt.initial}
			svc := service.NewSettingService(repo, &config.Config{})
			handler := NewSettingHandler(svc, nil, nil, nil, nil, nil, nil, nil)

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			handler.UpdateSettings(c)

			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			if tt.status != http.StatusOK {
				require.Empty(t, repo.lastUpdates)
				return
			}
			require.Equal(t, tt.stored, repo.values[key])

			var body struct {
				Data map[string]any `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tt.value, body.Data[key])
		})
	}
}
