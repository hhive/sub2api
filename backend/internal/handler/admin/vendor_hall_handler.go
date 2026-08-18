package admin

import (
	"errors"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/vendorhall"
	"github.com/gin-gonic/gin"
)

// SetVendorHallLister is a narrow test/assembly hook; production uses the
// lazy environment-backed service installed by NewOpsHandler.
func (h *OpsHandler) SetVendorHallLister(lister vendorhall.Lister) {
	h.vendorHallService = lister
}

// ListVendorHallAccounts serves Monitor's read-only account metrics without
// making the Monitor database a startup dependency for Sub2API.
func (h *OpsHandler) ListVendorHallAccounts(c *gin.Context) {
	params, err := vendorhall.ParseListParams(c.Request.URL.Query())
	if err != nil {
		response.BadRequest(c, "Invalid vendor hall query")
		return
	}
	if h.vendorHallService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Vendor hall data is unavailable")
		return
	}
	result, err := h.vendorHallService.List(c.Request.Context(), params)
	if err != nil {
		if errors.Is(err, vendorhall.ErrUnavailable) {
			response.Error(c, http.StatusServiceUnavailable, "Vendor hall data is unavailable")
			return
		}
		response.InternalError(c, "Failed to load vendor hall data")
		return
	}
	response.Success(c, result)
}
