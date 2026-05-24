package service

func applyGroupRateCorrectionMultiplier(group *Group, visibleMultiplier float64) float64 {
	if group == nil {
		return visibleMultiplier
	}
	correction := group.RateCorrectionMultiplier
	if correction < 0 {
		return 0
	}
	if correction == 0 {
		correction = 1
	}
	return visibleMultiplier * correction
}

func resolveImageRateMultiplier(apiKey *APIKey, effectiveGroupMultiplier float64) float64 {
	if apiKey != nil && apiKey.Group != nil && apiKey.Group.ImageRateIndependent {
		if apiKey.Group.ImageRateMultiplier < 0 {
			return 0
		}
		return apiKey.Group.ImageRateMultiplier
	}
	return effectiveGroupMultiplier
}
