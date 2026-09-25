package handler

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// MyTierHandler 用户端「等级权益」接口。
// DTO 只暴露称呼/进度/权益/领取状态，不暴露内部字段（tier_id 之外的配置细节、effect 内部状态）。
type MyTierHandler struct {
	tierService *service.UserTierService
}

// NewMyTierHandler 创建用户端「等级权益」处理器
func NewMyTierHandler(tierService *service.UserTierService) *MyTierHandler {
	return &MyTierHandler{tierService: tierService}
}

// ---------- DTO ----------

type userTierBenefitDTO struct {
	BenefitType    string  `json:"benefit_type"`
	Amount         float64 `json:"amount,omitempty"`
	ValidityDays   int     `json:"validity_days,omitempty"`
	ExpiresAt      string  `json:"expires_at,omitempty"`
	GroupID        int64   `json:"group_id,omitempty"`
	GroupName      string  `json:"group_name,omitempty"`
	RateMultiplier float64 `json:"rate_multiplier,omitempty"`
	Status         string  `json:"status"`
	AppliedAt      string  `json:"applied_at,omitempty"`
	SkippedReason  string  `json:"skipped_reason,omitempty"`
}

type userTierStateDTO struct {
	TierID       int64                `json:"tier_id"`
	Code         string               `json:"code"`
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	TriggerType  string               `json:"trigger_type"`
	ThresholdUSD *float64             `json:"threshold_usd,omitempty"`
	Achieved     bool                 `json:"achieved"`
	Claimed      bool                 `json:"claimed"`
	Claimable    bool                 `json:"claimable"`
	ClaimedAt    string               `json:"claimed_at,omitempty"`
	Source       string               `json:"source,omitempty"`
	Benefits     []userTierBenefitDTO `json:"benefits"`
}

type userTierViewDTO struct {
	// Enabled 总开关：false 时 tiers 为空，前端据此隐藏入口与页面内容
	Enabled         bool               `json:"enabled"`
	ConsumedAmount  float64            `json:"consumed_amount"`
	CurrentTier     *userTierStateDTO  `json:"current_tier"`
	NextTier        *userTierStateDTO  `json:"next_tier"`
	RemainingToNext *float64           `json:"remaining_to_next"`
	Tiers           []userTierStateDTO `json:"tiers"`
	ClaimableCount  int                `json:"claimable_count"`
}

type userTierClaimRequest struct {
	TierID int64 `json:"tier_id" binding:"required"`
}

type userTierAppliedBenefitDTO struct {
	BenefitType    string  `json:"benefit_type"`
	Amount         float64 `json:"amount,omitempty"`
	ValidityDays   int     `json:"validity_days,omitempty"`
	ExpiresAt      string  `json:"expires_at,omitempty"`
	GroupID        int64   `json:"group_id,omitempty"`
	GroupName      string  `json:"group_name,omitempty"`
	RateMultiplier float64 `json:"rate_multiplier,omitempty"`
	SkippedReason  string  `json:"skipped_reason,omitempty"`
}

type userTierClaimResponse struct {
	TierID         int64                       `json:"tier_id"`
	Code           string                      `json:"code"`
	Name           string                      `json:"name"`
	AlreadyClaimed bool                        `json:"already_claimed"`
	Applied        []userTierAppliedBenefitDTO `json:"applied"`
}

func formatTierTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func tierBenefitToDTO(benefit service.UserTierBenefitState) userTierBenefitDTO {
	dto := userTierBenefitDTO{
		BenefitType:   benefit.BenefitType,
		Status:        benefit.Status,
		AppliedAt:     formatTierTime(benefit.AppliedAt),
		SkippedReason: benefit.SkippedReason,
	}
	if benefit.BalanceCredit != nil {
		dto.Amount = benefit.BalanceCredit.Amount
		dto.ValidityDays = benefit.BalanceCredit.ValidityDays
	}
	if benefit.GroupRate != nil {
		dto.GroupID = benefit.GroupRate.GroupID
		dto.GroupName = benefit.GroupName
		dto.RateMultiplier = benefit.GroupRate.RateMultiplier
	}
	return dto
}

func tierStateToDTO(state service.UserTierClaimState) userTierStateDTO {
	dto := userTierStateDTO{
		TierID:       state.TierID,
		Code:         state.Code,
		Name:         state.Name,
		Description:  state.Description,
		TriggerType:  state.TriggerType,
		ThresholdUSD: state.ThresholdUSD,
		Achieved:     state.Achieved,
		Claimed:      state.Claimed,
		Claimable:    state.Claimable,
		ClaimedAt:    formatTierTime(state.ClaimedAt),
		Source:       state.Source,
		Benefits:     make([]userTierBenefitDTO, 0, len(state.Benefits)),
	}
	for _, benefit := range state.Benefits {
		dto.Benefits = append(dto.Benefits, tierBenefitToDTO(benefit))
	}
	return dto
}

func tierViewToDTO(view *service.UserTierView) userTierViewDTO {
	dto := userTierViewDTO{
		Enabled:        view.Enabled,
		Tiers:          make([]userTierStateDTO, 0, len(view.Tiers)),
		ConsumedAmount: view.ConsumedAmount,
		ClaimableCount: view.ClaimableCount,
	}
	if view.CurrentTier != nil {
		current := tierStateToDTO(*view.CurrentTier)
		dto.CurrentTier = &current
	}
	if view.NextTier != nil {
		next := tierStateToDTO(*view.NextTier)
		dto.NextTier = &next
	}
	dto.RemainingToNext = view.RemainingToNext
	for _, state := range view.Tiers {
		dto.Tiers = append(dto.Tiers, tierStateToDTO(state))
	}
	return dto
}

// ---------- handlers ----------

// GetMyTier GET /user/tier
// 返回当前用户等级、累计消费、各档权益与领取状态。口径只含兑换额度消费（不含订阅）。
func (h *MyTierHandler) GetMyTier(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.tierService == nil {
		response.Success(c, userTierViewDTO{Tiers: []userTierStateDTO{}})
		return
	}
	view, err := h.tierService.GetUserTierView(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tierViewToDTO(view))
}

// ClaimTier POST /user/tier/claim
// body 只带档位 id：金额与倍率一律由服务端按配置决定，客户端不传权益内容。
func (h *MyTierHandler) ClaimTier(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req userTierClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if h == nil || h.tierService == nil {
		response.InternalError(c, "tier service is not configured")
		return
	}

	result, err := h.tierService.ClaimTier(c.Request.Context(), subject.UserID, req.TierID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := userTierClaimResponse{
		TierID:         result.TierID,
		Code:           result.Code,
		Name:           result.Name,
		AlreadyClaimed: result.AlreadyClaimed,
		Applied:        make([]userTierAppliedBenefitDTO, 0, len(result.Applied)),
	}
	for _, applied := range result.Applied {
		out.Applied = append(out.Applied, userTierAppliedBenefitDTO{
			BenefitType:    applied.BenefitType,
			Amount:         applied.Amount,
			ValidityDays:   applied.ValidityDays,
			ExpiresAt:      formatTierTime(applied.ExpiresAt),
			GroupID:        applied.GroupID,
			GroupName:      applied.GroupName,
			RateMultiplier: applied.RateMultiplier,
			SkippedReason:  applied.SkippedReason,
		})
	}
	response.Success(c, out)
}
