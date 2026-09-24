package admin

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UserTierHandler 管理端「等级与权益」配置接口。
// 档位与权益全部是管理端数据；代码里只有权益类型与触发类型两层枚举，不写死任何档位常量或阈值。
type UserTierHandler struct {
	tierService *service.UserTierService
}

// NewUserTierHandler 创建管理端等级配置处理器
func NewUserTierHandler(tierService *service.UserTierService) *UserTierHandler {
	return &UserTierHandler{tierService: tierService}
}

// ---------- DTO ----------

type tierBenefitResponse struct {
	ID             int64   `json:"id"`
	BenefitType    string  `json:"benefit_type"`
	Amount         float64 `json:"amount"`
	ValidityDays   int     `json:"validity_days"`
	GroupID        int64   `json:"group_id"`
	GroupName      string  `json:"group_name,omitempty"`
	RateMultiplier float64 `json:"rate_multiplier"`
	Enabled        bool    `json:"enabled"`
	SortOrder      int     `json:"sort_order"`
}

type tierResponse struct {
	ID           int64                 `json:"id"`
	Code         string                `json:"code"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	SortOrder    int                   `json:"sort_order"`
	TriggerType  string                `json:"trigger_type"`
	ThresholdUSD *float64              `json:"threshold_usd"`
	Enabled      bool                  `json:"enabled"`
	AwardCount   int64                 `json:"award_count"`
	Benefits     []tierBenefitResponse `json:"benefits"`
	CreatedAt    string                `json:"created_at"`
	UpdatedAt    string                `json:"updated_at"`
}

type tierBenefitRequest struct {
	ID             int64   `json:"id"`
	BenefitType    string  `json:"benefit_type" binding:"required"`
	Amount         float64 `json:"amount"`
	ValidityDays   int     `json:"validity_days"`
	GroupID        int64   `json:"group_id"`
	RateMultiplier float64 `json:"rate_multiplier"`
	Enabled        *bool   `json:"enabled"`
}

type createTierRequest struct {
	Code         string               `json:"code" binding:"required"`
	Name         string               `json:"name" binding:"required"`
	Description  string               `json:"description"`
	TriggerType  string               `json:"trigger_type" binding:"required"`
	ThresholdUSD *float64             `json:"threshold_usd"`
	Enabled      *bool                `json:"enabled"`
	Benefits     []tierBenefitRequest `json:"benefits"`
}

type updateTierRequest struct {
	Code         *string               `json:"code"`
	Name         *string               `json:"name"`
	Description  *string               `json:"description"`
	TriggerType  *string               `json:"trigger_type"`
	ThresholdUSD *float64              `json:"threshold_usd"`
	Enabled      *bool                 `json:"enabled"`
	Benefits     *[]tierBenefitRequest `json:"benefits"`
}

type reorderTiersRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}

type userTierEffectResponse struct {
	ID             int64          `json:"id"`
	BenefitKey     string         `json:"benefit_key"`
	BenefitType    string         `json:"benefit_type"`
	Amount         *float64       `json:"amount,omitempty"`
	GroupID        *int64         `json:"group_id,omitempty"`
	RateMultiplier *float64       `json:"rate_multiplier,omitempty"`
	Status         string         `json:"status"`
	Attempts       int            `json:"attempts"`
	LastError      string         `json:"last_error,omitempty"`
	Detail         map[string]any `json:"detail,omitempty"`
	AppliedAt      string         `json:"applied_at,omitempty"`
}

type userTierAwardResponse struct {
	ID                int64                    `json:"id"`
	TierCode          string                   `json:"tier_code"`
	TierNameSnapshot  string                   `json:"tier_name_snapshot"`
	ThresholdSnapshot *float64                 `json:"threshold_snapshot,omitempty"`
	AchievedAt        string                   `json:"achieved_at"`
	Source            string                   `json:"source"`
	Effects           []userTierEffectResponse `json:"effects"`
}

// adminUserTierResponse 只读查看某用户的等级与领取记录（供运营核对，不提供任何写入口）
type adminUserTierResponse struct {
	UserID         int64                    `json:"user_id"`
	ConsumedAmount float64                  `json:"consumed_amount"`
	Tiers          []adminUserTierTierState `json:"tiers"`
	Awards         []userTierAwardResponse  `json:"awards"`
	Effects        []userTierEffectResponse `json:"effects"`
}

type adminUserTierTierState struct {
	TierID    int64  `json:"tier_id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Achieved  bool   `json:"achieved"`
	Claimed   bool   `json:"claimed"`
	Claimable bool   `json:"claimable"`
}

// ---------- 转换 ----------

func tierToResponse(tier service.UserTier) tierResponse {
	out := tierResponse{
		ID:           tier.ID,
		Code:         tier.Code,
		Name:         tier.Name,
		Description:  tier.Description,
		SortOrder:    tier.SortOrder,
		TriggerType:  tier.TriggerType,
		ThresholdUSD: tier.ThresholdUSD,
		Enabled:      tier.Enabled,
		AwardCount:   tier.AwardCount,
		Benefits:     make([]tierBenefitResponse, 0, len(tier.Benefits)),
		CreatedAt:    tier.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    tier.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	for _, benefit := range tier.Benefits {
		item := tierBenefitResponse{
			ID:          benefit.ID,
			BenefitType: benefit.BenefitType,
			Enabled:     benefit.Enabled,
			SortOrder:   benefit.SortOrder,
		}
		if benefit.BalanceCredit != nil {
			item.Amount = benefit.BalanceCredit.Amount
			item.ValidityDays = benefit.BalanceCredit.ValidityDays
		}
		if benefit.GroupRate != nil {
			item.GroupID = benefit.GroupRate.GroupID
			item.RateMultiplier = benefit.GroupRate.RateMultiplier
		}
		out.Benefits = append(out.Benefits, item)
	}
	return out
}

func benefitRequestsToInput(reqs []tierBenefitRequest) []service.UserTierBenefitInput {
	out := make([]service.UserTierBenefitInput, 0, len(reqs))
	for _, req := range reqs {
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		out = append(out, service.UserTierBenefitInput{
			ID:             req.ID,
			BenefitType:    req.BenefitType,
			Amount:         req.Amount,
			ValidityDays:   req.ValidityDays,
			GroupID:        req.GroupID,
			RateMultiplier: req.RateMultiplier,
			Enabled:        enabled,
		})
	}
	return out
}

func effectToResponse(effect service.UserTierEffect) userTierEffectResponse {
	out := userTierEffectResponse{
		ID:             effect.ID,
		BenefitKey:     effect.BenefitKey,
		BenefitType:    effect.BenefitType,
		Amount:         effect.Amount,
		GroupID:        effect.GroupID,
		RateMultiplier: effect.RateMultiplier,
		Status:         effect.Status,
		Attempts:       effect.Attempts,
		LastError:      effect.LastError,
		Detail:         effect.Detail,
	}
	if effect.AppliedAt != nil && !effect.AppliedAt.IsZero() {
		out.AppliedAt = effect.AppliedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return out
}

type tierSwitchRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// ---------- handlers ----------

// GetFeatureSwitch GET /admin/tiers/switch
// 返回整个用户等级体系的总开关（缺省=开启）。
func (h *UserTierHandler) GetFeatureSwitch(c *gin.Context) {
	response.Success(c, gin.H{"enabled": h.tierService.IsFeatureEnabled(c.Request.Context())})
}

// UpdateFeatureSwitch PUT /admin/tiers/switch
// 关闭后：用户端入口与页面不可见、不再触发新的等级权益；已发放的授予与权益不回收。
func (h *UserTierHandler) UpdateFeatureSwitch(c *gin.Context) {
	var req tierSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.tierService.SetFeatureEnabled(c.Request.Context(), *req.Enabled); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"enabled": *req.Enabled})
}

// ListTiers GET /admin/tiers
func (h *UserTierHandler) ListTiers(c *gin.Context) {
	includeDisabled := c.Query("include_disabled") != "false"
	tiers, err := h.tierService.ListTiers(c.Request.Context(), includeDisabled)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]tierResponse, 0, len(tiers))
	for _, tier := range tiers {
		out = append(out, tierToResponse(tier))
	}
	response.Success(c, out)
}

// CreateTier POST /admin/tiers
func (h *UserTierHandler) CreateTier(c *gin.Context) {
	var req createTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	input := service.UserTierInput{
		Code:         &req.Code,
		Name:         &req.Name,
		Description:  &req.Description,
		TriggerType:  &req.TriggerType,
		ThresholdUSD: req.ThresholdUSD,
		Enabled:      req.Enabled,
	}
	benefits := benefitRequestsToInput(req.Benefits)
	input.Benefits = &benefits

	tier, err := h.tierService.CreateTier(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tierToResponse(*tier))
}

// UpdateTier PUT /admin/tiers/:id
func (h *UserTierHandler) UpdateTier(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req updateTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	input := service.UserTierInput{
		Code:         req.Code,
		Name:         req.Name,
		Description:  req.Description,
		TriggerType:  req.TriggerType,
		ThresholdUSD: req.ThresholdUSD,
		Enabled:      req.Enabled,
	}
	if req.Benefits != nil {
		benefits := benefitRequestsToInput(*req.Benefits)
		input.Benefits = &benefits
	}

	tier, err := h.tierService.UpdateTier(c.Request.Context(), id, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tierToResponse(*tier))
}

// ReorderTiers PUT /admin/tiers/reorder
func (h *UserTierHandler) ReorderTiers(c *gin.Context) {
	var req reorderTiersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.tierService.ReorderTiers(c.Request.Context(), req.IDs); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Reorder successful"})
}

// DeleteTier DELETE /admin/tiers/:id
// 已产生授予记录的等级只能停用，不能物理删除。
func (h *UserTierHandler) DeleteTier(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.tierService.DeleteTier(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Tier deleted successfully"})
}

// GetUserTier GET /admin/users/:id/tier
// 只读：查看某用户的累计消费、等级状态、授予与发放记录（不提供任何写入口）。
func (h *UserTierHandler) GetUserTier(c *gin.Context) {
	userID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	view, awards, effects, err := h.tierService.GetUserTierAdminView(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := adminUserTierResponse{
		UserID:         userID,
		ConsumedAmount: view.ConsumedAmount,
		Tiers:          make([]adminUserTierTierState, 0, len(view.Tiers)),
		Awards:         make([]userTierAwardResponse, 0, len(awards)),
		Effects:        make([]userTierEffectResponse, 0, len(effects)),
	}
	for _, state := range view.Tiers {
		out.Tiers = append(out.Tiers, adminUserTierTierState{
			TierID:    state.TierID,
			Code:      state.Code,
			Name:      state.Name,
			Achieved:  state.Achieved,
			Claimed:   state.Claimed,
			Claimable: state.Claimable,
		})
	}
	effectsByAward := make(map[int64][]userTierEffectResponse, len(awards))
	for _, effect := range effects {
		effectsByAward[effect.AwardID] = append(effectsByAward[effect.AwardID], effectToResponse(effect))
		out.Effects = append(out.Effects, effectToResponse(effect))
	}
	for _, award := range awards {
		item := userTierAwardResponse{
			ID:                award.ID,
			TierCode:          award.TierCode,
			TierNameSnapshot:  award.TierNameSnapshot,
			ThresholdSnapshot: award.ThresholdSnapshot,
			AchievedAt:        award.AchievedAt.Format(time.RFC3339),
			Source:            award.Source,
			Effects:           effectsByAward[award.ID],
		}
		out.Awards = append(out.Awards, item)
	}
	response.Success(c, out)
}
