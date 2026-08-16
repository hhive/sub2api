package admin

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetRelayMonitorPriority exposes only the scheduling priority to the dedicated Monitor integration.
func (h *AccountHandler) GetRelayMonitorPriority(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}
	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"account_id": account.ID, "priority": account.Priority})
}

type relayMonitorPriorityUpdateRequest struct {
	ExpectedPriority int `json:"expected_priority"`
	TargetPriority   int `json:"target_priority"`
}

type relayMonitorPriorityCompareAndSwapper interface {
	CompareAndSwapAccountPriority(context.Context, int64, int, int) (int, bool, error)
}

type relayMonitorPriorityCapPauser interface {
	PauseRelayMonitorPriorityCappedAccount(context.Context, int64, time.Duration) (time.Time, error)
}

type relayMonitorPriorityCapPauseRequest struct {
	DurationSeconds int `json:"duration_seconds"`
}

// SetRelayMonitorPriority applies a narrow idempotent compare-before-update operation.
func (h *AccountHandler) SetRelayMonitorPriority(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}
	var request relayMonitorPriorityUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.ExpectedPriority < 0 || request.TargetPriority < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid priority update"})
		return
	}
	updater, ok := h.adminService.(relayMonitorPriorityCompareAndSwapper)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "priority integration unavailable"})
		return
	}
	current, updated, err := updater.CompareAndSwapAccountPriority(c.Request.Context(), accountID, request.ExpectedPriority, request.TargetPriority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "priority update failed"})
		return
	}
	if !updated && current != request.TargetPriority {
		c.JSON(http.StatusConflict, gin.H{"error": "priority changed", "priority": current})
		return
	}
	c.JSON(http.StatusOK, gin.H{"account_id": accountID, "priority": request.TargetPriority, "idempotent": !updated})
}

// PauseRelayMonitorPriorityCappedAccount temporarily removes a capped account from scheduling.
func (h *AccountHandler) PauseRelayMonitorPriorityCappedAccount(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}
	var request relayMonitorPriorityCapPauseRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.DurationSeconds < 60 || request.DurationSeconds > 3600 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pause duration"})
		return
	}
	pauser, ok := h.adminService.(relayMonitorPriorityCapPauser)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "priority integration unavailable"})
		return
	}
	until, err := pauser.PauseRelayMonitorPriorityCappedAccount(
		c.Request.Context(), accountID, time.Duration(request.DurationSeconds)*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "account pause failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"account_id": accountID, "temp_unschedulable_until": until.UTC()})
}
