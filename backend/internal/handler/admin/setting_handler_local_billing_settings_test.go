//go:build unit

package admin

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

var localBillingSettingKeys = []string{
	service.SettingKeyFirstRechargeBonusEnabled,
	service.SettingKeyFirstRechargeBonusAmount,
	service.SettingKeyFirstRechargeBonusValidityDays,
	service.SettingKeyAffiliateSubscriptionRebateMultiplier,
	service.SettingKeyAffiliateTieredRebateEnabled,
	service.SettingKeyAffiliateTier2MinPaidInvitees,
	service.SettingKeyAffiliateTier3MinPaidInvitees,
	service.SettingKeyAffiliateTier2MultiplierPercent,
	service.SettingKeyAffiliateTier3MultiplierPercent,
}

func localBillingSettingsRepoValues() map[string]string {
	return map[string]string{
		service.SettingKeyPromoCodeEnabled:                      "true",
		service.SettingKeyFirstRechargeBonusEnabled:             "true",
		service.SettingKeyFirstRechargeBonusAmount:              "12.50000000",
		service.SettingKeyFirstRechargeBonusValidityDays:        "45",
		service.SettingKeyAffiliateSubscriptionRebateMultiplier: "75.50000000",
		service.SettingKeyAffiliateTieredRebateEnabled:          "true",
		service.SettingKeyAffiliateTier2MinPaidInvitees:         "12",
		service.SettingKeyAffiliateTier3MinPaidInvitees:         "40",
		service.SettingKeyAffiliateTier2MultiplierPercent:       "135.50000000",
		service.SettingKeyAffiliateTier3MultiplierPercent:       "175.25000000",
	}
}

func newLocalBillingSettingHandler(repo *settingHandlerRepoStub) *SettingHandler {
	svc := service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	return NewSettingHandler(svc, nil, nil, nil, nil, nil, nil, nil)
}

func getLocalBillingSettings(t *testing.T, handler *SettingHandler) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)

	handler.GetSettings(c)

	return rec.Code, decodeLocalBillingSettingsResponse(t, rec)
}

func updateLocalBillingSettings(t *testing.T, handler *SettingHandler, body map[string]any) (int, map[string]any) {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	return rec.Code, decodeLocalBillingSettingsResponse(t, rec)
}

func updateLocalBillingSettingsRaw(t *testing.T, handler *SettingHandler, body map[string]any) (int, map[string]json.RawMessage) {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateSettings(c)

	var resp struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp), rec.Body.String())
	return rec.Code, resp.Data
}

func decodeLocalBillingSettingsResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp), rec.Body.String())
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok, "response data is %T: %s", resp.Data, rec.Body.String())
	return data
}

func requireLocalBillingSettingsResponse(t *testing.T, data map[string]any, expected map[string]any) {
	t.Helper()
	for _, key := range localBillingSettingKeys {
		require.Contains(t, data, key)
		require.Equal(t, expected[key], data[key], key)
	}
}

func TestSettingHandler_GetSettingsIncludesLocalBillingSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: localBillingSettingsRepoValues()}

	code, data := getLocalBillingSettings(t, newLocalBillingSettingHandler(repo))

	require.Equal(t, http.StatusOK, code)
	requireLocalBillingSettingsResponse(t, data, map[string]any{
		service.SettingKeyFirstRechargeBonusEnabled:             true,
		service.SettingKeyFirstRechargeBonusAmount:              12.5,
		service.SettingKeyFirstRechargeBonusValidityDays:        float64(45),
		service.SettingKeyAffiliateSubscriptionRebateMultiplier: 75.5,
		service.SettingKeyAffiliateTieredRebateEnabled:          true,
		service.SettingKeyAffiliateTier2MinPaidInvitees:         float64(12),
		service.SettingKeyAffiliateTier3MinPaidInvitees:         float64(40),
		service.SettingKeyAffiliateTier2MultiplierPercent:       135.5,
		service.SettingKeyAffiliateTier3MultiplierPercent:       175.25,
	})
}

func TestSettingHandler_UpdateSettingsPersistsLocalBillingSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("persists all fields and returns normalized service values", func(t *testing.T) {
		repo := &settingHandlerRepoStub{values: localBillingSettingsRepoValues()}
		code, data := updateLocalBillingSettings(t, newLocalBillingSettingHandler(repo), map[string]any{
			"first_recharge_bonus_enabled":             true,
			"first_recharge_bonus_amount":              -2.5,
			"first_recharge_bonus_validity_days":       -3,
			"affiliate_subscription_rebate_multiplier": 150,
			"affiliate_tiered_rebate_enabled":          true,
			"affiliate_tier2_min_paid_invitees":        50,
			"affiliate_tier3_min_paid_invitees":        40,
			"affiliate_tier2_multiplier_percent":       -1,
			"affiliate_tier3_multiplier_percent":       600,
		})

		require.Equal(t, http.StatusOK, code)
		require.Equal(t, "true", repo.values[service.SettingKeyFirstRechargeBonusEnabled])
		require.Equal(t, "0.00000000", repo.values[service.SettingKeyFirstRechargeBonusAmount])
		require.Equal(t, "0", repo.values[service.SettingKeyFirstRechargeBonusValidityDays])
		require.Equal(t, "100.00000000", repo.values[service.SettingKeyAffiliateSubscriptionRebateMultiplier])
		require.Equal(t, "true", repo.values[service.SettingKeyAffiliateTieredRebateEnabled])
		require.Equal(t, "50", repo.values[service.SettingKeyAffiliateTier2MinPaidInvitees])
		require.Equal(t, "51", repo.values[service.SettingKeyAffiliateTier3MinPaidInvitees])
		require.Equal(t, "0.00000000", repo.values[service.SettingKeyAffiliateTier2MultiplierPercent])
		require.Equal(t, "500.00000000", repo.values[service.SettingKeyAffiliateTier3MultiplierPercent])
		requireLocalBillingSettingsResponse(t, data, map[string]any{
			service.SettingKeyFirstRechargeBonusEnabled:             true,
			service.SettingKeyFirstRechargeBonusAmount:              float64(0),
			service.SettingKeyFirstRechargeBonusValidityDays:        float64(0),
			service.SettingKeyAffiliateSubscriptionRebateMultiplier: float64(100),
			service.SettingKeyAffiliateTieredRebateEnabled:          true,
			service.SettingKeyAffiliateTier2MinPaidInvitees:         float64(50),
			service.SettingKeyAffiliateTier3MinPaidInvitees:         float64(51),
			service.SettingKeyAffiliateTier2MultiplierPercent:       float64(0),
			service.SettingKeyAffiliateTier3MultiplierPercent:       float64(500),
		})
	})

	t.Run("normalizes negative subscription multiplier", func(t *testing.T) {
		repo := &settingHandlerRepoStub{values: localBillingSettingsRepoValues()}
		code, data := updateLocalBillingSettings(t, newLocalBillingSettingHandler(repo), map[string]any{
			"affiliate_subscription_rebate_multiplier": -20,
		})

		require.Equal(t, http.StatusOK, code)
		require.Equal(t, "0.00000000", repo.values[service.SettingKeyAffiliateSubscriptionRebateMultiplier])
		require.Equal(t, float64(0), data[service.SettingKeyAffiliateSubscriptionRebateMultiplier])
	})

	t.Run("normalizes platform maximum to cross JSON safe bounds", func(t *testing.T) {
		repo := &settingHandlerRepoStub{values: localBillingSettingsRepoValues()}
		tier3Max := math.MaxInt
		const maxJavaScriptSafeInteger int64 = 1<<53 - 1
		if int64(tier3Max) > maxJavaScriptSafeInteger {
			maxSafeInteger := maxJavaScriptSafeInteger
			tier3Max = int(maxSafeInteger)
		}
		tier2Expected := strconv.Itoa(tier3Max - 1)
		tier3Expected := strconv.Itoa(tier3Max)
		code, data := updateLocalBillingSettingsRaw(t, newLocalBillingSettingHandler(repo), map[string]any{
			"affiliate_tier2_min_paid_invitees": math.MaxInt,
			"affiliate_tier3_min_paid_invitees": math.MaxInt,
		})

		require.Equal(t, http.StatusOK, code)
		require.Equal(t, tier2Expected, repo.values[service.SettingKeyAffiliateTier2MinPaidInvitees])
		require.Equal(t, tier3Expected, repo.values[service.SettingKeyAffiliateTier3MinPaidInvitees])
		require.Equal(t, tier2Expected, string(data[service.SettingKeyAffiliateTier2MinPaidInvitees]))
		require.Equal(t, tier3Expected, string(data[service.SettingKeyAffiliateTier3MinPaidInvitees]))
	})
}

func TestSettingHandler_UpdateSettingsPreservesOmittedLocalBillingSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("omitted fields retain previous settings", func(t *testing.T) {
		repo := &settingHandlerRepoStub{values: localBillingSettingsRepoValues()}
		code, data := updateLocalBillingSettings(t, newLocalBillingSettingHandler(repo), map[string]any{
			"promo_code_enabled": false,
		})

		require.Equal(t, http.StatusOK, code)
		expectedStored := localBillingSettingsRepoValues()
		for _, key := range localBillingSettingKeys {
			require.Equal(t, expectedStored[key], repo.values[key], key)
		}
		requireLocalBillingSettingsResponse(t, data, map[string]any{
			service.SettingKeyFirstRechargeBonusEnabled:             true,
			service.SettingKeyFirstRechargeBonusAmount:              12.5,
			service.SettingKeyFirstRechargeBonusValidityDays:        float64(45),
			service.SettingKeyAffiliateSubscriptionRebateMultiplier: 75.5,
			service.SettingKeyAffiliateTieredRebateEnabled:          true,
			service.SettingKeyAffiliateTier2MinPaidInvitees:         float64(12),
			service.SettingKeyAffiliateTier3MinPaidInvitees:         float64(40),
			service.SettingKeyAffiliateTier2MultiplierPercent:       135.5,
			service.SettingKeyAffiliateTier3MultiplierPercent:       175.25,
		})
	})

	t.Run("explicit false and zero are not treated as omitted", func(t *testing.T) {
		repo := &settingHandlerRepoStub{values: localBillingSettingsRepoValues()}
		code, data := updateLocalBillingSettings(t, newLocalBillingSettingHandler(repo), map[string]any{
			"first_recharge_bonus_enabled":             false,
			"first_recharge_bonus_amount":              0,
			"first_recharge_bonus_validity_days":       0,
			"affiliate_subscription_rebate_multiplier": 0,
			"affiliate_tiered_rebate_enabled":          false,
			"affiliate_tier2_min_paid_invitees":        0,
			"affiliate_tier3_min_paid_invitees":        0,
			"affiliate_tier2_multiplier_percent":       0,
			"affiliate_tier3_multiplier_percent":       0,
		})

		require.Equal(t, http.StatusOK, code)
		require.Equal(t, "false", repo.values[service.SettingKeyFirstRechargeBonusEnabled])
		require.Equal(t, "0.00000000", repo.values[service.SettingKeyFirstRechargeBonusAmount])
		require.Equal(t, "0", repo.values[service.SettingKeyFirstRechargeBonusValidityDays])
		require.Equal(t, "0.00000000", repo.values[service.SettingKeyAffiliateSubscriptionRebateMultiplier])
		require.Equal(t, "false", repo.values[service.SettingKeyAffiliateTieredRebateEnabled])
		require.Equal(t, "10", repo.values[service.SettingKeyAffiliateTier2MinPaidInvitees])
		require.Equal(t, "30", repo.values[service.SettingKeyAffiliateTier3MinPaidInvitees])
		require.Equal(t, "0.00000000", repo.values[service.SettingKeyAffiliateTier2MultiplierPercent])
		require.Equal(t, "0.00000000", repo.values[service.SettingKeyAffiliateTier3MultiplierPercent])
		requireLocalBillingSettingsResponse(t, data, map[string]any{
			service.SettingKeyFirstRechargeBonusEnabled:             false,
			service.SettingKeyFirstRechargeBonusAmount:              float64(0),
			service.SettingKeyFirstRechargeBonusValidityDays:        float64(0),
			service.SettingKeyAffiliateSubscriptionRebateMultiplier: float64(0),
			service.SettingKeyAffiliateTieredRebateEnabled:          false,
			service.SettingKeyAffiliateTier2MinPaidInvitees:         float64(10),
			service.SettingKeyAffiliateTier3MinPaidInvitees:         float64(30),
			service.SettingKeyAffiliateTier2MultiplierPercent:       float64(0),
			service.SettingKeyAffiliateTier3MultiplierPercent:       float64(0),
		})
	})
}

func TestSettingHandler_DiffSettingsIncludesLocalBillingSettings(t *testing.T) {
	before := &service.SystemSettings{
		FirstRechargeBonusEnabled:             false,
		FirstRechargeBonusAmount:              5,
		FirstRechargeBonusValidityDays:        3,
		AffiliateSubscriptionRebateMultiplier: 80,
		AffiliateTieredRebateEnabled:          false,
		AffiliateTier2MinPaidInvitees:         10,
		AffiliateTier3MinPaidInvitees:         30,
		AffiliateTier2MultiplierPercent:       120,
		AffiliateTier3MultiplierPercent:       150,
	}
	after := *before
	after.FirstRechargeBonusEnabled = true
	after.FirstRechargeBonusAmount = 6
	after.FirstRechargeBonusValidityDays = 4
	after.AffiliateSubscriptionRebateMultiplier = 81
	after.AffiliateTieredRebateEnabled = true
	after.AffiliateTier2MinPaidInvitees = 11
	after.AffiliateTier3MinPaidInvitees = 31
	after.AffiliateTier2MultiplierPercent = 121
	after.AffiliateTier3MultiplierPercent = 151

	changed := diffSettings(before, &after, nil, nil, UpdateSettingsRequest{})

	require.ElementsMatch(t, localBillingSettingKeys, changed)
	require.Len(t, changed, len(localBillingSettingKeys))
	seen := make(map[string]struct{}, len(changed))
	for _, key := range changed {
		_, duplicate := seen[key]
		require.False(t, duplicate, "duplicate audit field %q", key)
		seen[key] = struct{}{}
	}

	onlyAmountChanged := *before
	onlyAmountChanged.FirstRechargeBonusAmount = 7
	require.Equal(t,
		[]string{service.SettingKeyFirstRechargeBonusAmount},
		diffSettings(before, &onlyAmountChanged, nil, nil, UpdateSettingsRequest{}),
	)
	require.Empty(t, diffSettings(before, before, nil, nil, UpdateSettingsRequest{}))
}
