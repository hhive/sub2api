package admin

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type vendorHallAccountPauser interface {
	PauseVendorHallAccount(context.Context, int64) (time.Time, error)
}

// PauseVendorHallAccount temporarily removes one account from scheduling for
// the fixed one-hour administrator policy.
func (h *AccountHandler) PauseVendorHallAccount(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	pauser, ok := h.adminService.(vendorHallAccountPauser)
	if !ok {
		response.Error(c, http.StatusServiceUnavailable, "Account pause is unavailable")
		return
	}
	until, err := pauser.PauseVendorHallAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"account_id": accountID, "temp_unschedulable_until": until.UTC()})
}
