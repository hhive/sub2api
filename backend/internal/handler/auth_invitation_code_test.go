//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type invitationSettingRepoStub struct {
	values map[string]string
}

func (s *invitationSettingRepoStub) Get(_ context.Context, key string) (*service.Setting, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (s *invitationSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *invitationSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *invitationSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *invitationSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *invitationSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	result := make(map[string]string, len(s.values))
	for key, value := range s.values {
		result[key] = value
	}
	return result, nil
}

func (s *invitationSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type invitationRedeemRepoStub struct {
	service.RedeemCodeRepository

	code *service.RedeemCode
	err  error
}

func (s *invitationRedeemRepoStub) GetByCode(context.Context, string) (*service.RedeemCode, error) {
	if s.err != nil {
		return nil, s.err
	}
	cloned := *s.code
	return &cloned, nil
}

type validateInvitationEnvelope struct {
	Code int `json:"code"`
	Data struct {
		Valid     bool   `json:"valid"`
		Kind      string `json:"kind"`
		ErrorCode string `json:"error_code"`
	} `json:"data"`
}

func callValidateInvitationCode(t *testing.T, redeemRepo *invitationRedeemRepoStub, code string) validateInvitationEnvelope {
	t.Helper()
	settingSvc := service.NewSettingService(&invitationSettingRepoStub{values: map[string]string{
		"invitation_code_enabled": "true",
	}}, &config.Config{})
	redeemSvc := service.NewRedeemService(redeemRepo, nil, nil, nil, nil, nil, nil, nil, nil, settingSvc)
	h := &AuthHandler{settingSvc: settingSvc, redeemService: redeemSvc}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/validate", h.ValidateInvitationCode)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(`{"code":"`+code+`"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var envelope validateInvitationEnvelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	return envelope
}

func TestValidateInvitationCodeMarksUnknownCodeAsAffiliateCandidate(t *testing.T) {
	envelope := callValidateInvitationCode(t, &invitationRedeemRepoStub{
		err: service.ErrRedeemCodeNotFound,
	}, "4W52TTV5G5A3")

	require.False(t, envelope.Data.Valid)
	require.Equal(t, "affiliate_candidate", envelope.Data.Kind)
	require.Equal(t, "INVITATION_CODE_NOT_FOUND", envelope.Data.ErrorCode)
}

func TestValidateInvitationCodeReportsUsedInvitation(t *testing.T) {
	envelope := callValidateInvitationCode(t, &invitationRedeemRepoStub{
		code: &service.RedeemCode{ID: 1, Code: "INV-USED", Type: service.RedeemTypeInvitation, Status: service.StatusUsed},
	}, "INV-USED")

	require.False(t, envelope.Data.Valid)
	require.Equal(t, "invitation", envelope.Data.Kind)
	require.Equal(t, "INVITATION_CODE_USED", envelope.Data.ErrorCode)
}

func TestValidateInvitationCodeReportsExpiredInvitation(t *testing.T) {
	expiredAt := time.Now().Add(-time.Minute)
	envelope := callValidateInvitationCode(t, &invitationRedeemRepoStub{
		code: &service.RedeemCode{
			ID:        2,
			Code:      "INV-EXPIRED",
			Type:      service.RedeemTypeInvitation,
			Status:    service.StatusUnused,
			ExpiresAt: &expiredAt,
		},
	}, "INV-EXPIRED")

	require.False(t, envelope.Data.Valid)
	require.Equal(t, "invitation", envelope.Data.Kind)
	require.Equal(t, "INVITATION_CODE_EXPIRED", envelope.Data.ErrorCode)
}

func TestValidateInvitationCodeReportsExpiredStatus(t *testing.T) {
	envelope := callValidateInvitationCode(t, &invitationRedeemRepoStub{
		code: &service.RedeemCode{ID: 5, Code: "INV-STATUS-EXPIRED", Type: service.RedeemTypeInvitation, Status: service.StatusExpired},
	}, "INV-STATUS-EXPIRED")

	require.False(t, envelope.Data.Valid)
	require.Equal(t, "invitation", envelope.Data.Kind)
	require.Equal(t, "INVITATION_CODE_EXPIRED", envelope.Data.ErrorCode)
}

func TestValidateInvitationCodeAcceptsUsableInvitation(t *testing.T) {
	envelope := callValidateInvitationCode(t, &invitationRedeemRepoStub{
		code: &service.RedeemCode{ID: 3, Code: "INV-OK", Type: service.RedeemTypeInvitation, Status: service.StatusUnused},
	}, "INV-OK")

	require.True(t, envelope.Data.Valid)
	require.Equal(t, "invitation", envelope.Data.Kind)
	require.Empty(t, envelope.Data.ErrorCode)
}

func TestValidateInvitationCodeRejectsOtherRedeemTypes(t *testing.T) {
	envelope := callValidateInvitationCode(t, &invitationRedeemRepoStub{
		code: &service.RedeemCode{ID: 4, Code: "BAL-1", Type: service.RedeemTypeBalance, Status: service.StatusUnused, Value: 10},
	}, "BAL-1")

	require.False(t, envelope.Data.Valid)
	require.Empty(t, envelope.Data.Kind)
	require.Equal(t, "INVITATION_CODE_INVALID", envelope.Data.ErrorCode)
}
