package service

import (
	"encoding/json"
	"strconv"
	"strings"
)

func accountRateMultiplierForUsage(account *Account, groupID *int64) float64 {
	base := 1.0
	if account != nil {
		base = account.BillingRateMultiplier()
	}
	if accountUpstreamID(account) <= 0 {
		return base
	}
	if groupID == nil {
		return 1
	}
	remoteRate := upstreamRuntimeRemoteGroupRateMultiplier(account, *groupID)
	if remoteRate <= 0 {
		return 1
	}
	return remoteRate
}

func openAIAccountRateMultiplierForUsage(account *Account, groupID *int64) float64 {
	return accountRateMultiplierForUsage(account, groupID)
}

func upstreamRuntimeRemoteGroupRateMultiplier(account *Account, localGroupID int64) float64 {
	if account == nil || account.Credentials == nil || localGroupID <= 0 {
		return 0
	}
	key := strconv.FormatInt(localGroupID, 10)
	rawRates, ok := account.Credentials["upstream_group_rate_multipliers"].(map[string]any)
	if !ok {
		return 0
	}
	return parseFloatLike(rawRates[key])
}

func parseFloatLike(raw any) float64 {
	switch v := raw.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	default:
		return 0
	}
}
