package admin

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const relayMonitorPrioritySecretHeader = "X-Sub2API-Relay-Monitor-Secret"

func requireRelayMonitorPrioritySecret(c *gin.Context) bool {
	expected := strings.TrimSpace(os.Getenv("SUB2API_RELAY_MONITOR_PRIORITY_SECRET"))
	provided := strings.TrimSpace(c.GetHeader(relayMonitorPrioritySecretHeader))
	if len(expected) < 32 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "relay monitor priority integration unavailable"})
		return false
	}
	expectedHash := sha256.Sum256([]byte(expected))
	providedHash := sha256.Sum256([]byte(provided))
	if subtle.ConstantTimeCompare(expectedHash[:], providedHash[:]) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return false
	}
	return true
}

// GetRelayMonitorPriority exposes only the scheduling priority to the dedicated Monitor integration.
func (h *AccountHandler) GetRelayMonitorPriority(c *gin.Context) {
	if !requireRelayMonitorPrioritySecret(c) {
		return
	}
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

// SetRelayMonitorPriority applies a narrow idempotent compare-before-update operation.
func (h *AccountHandler) SetRelayMonitorPriority(c *gin.Context) {
	if !requireRelayMonitorPrioritySecret(c) {
		return
	}
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
