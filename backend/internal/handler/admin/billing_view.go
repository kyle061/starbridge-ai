package admin

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/gin-gonic/gin"
)

const customerBillingView = "customer"

func isCustomerBillingView(c *gin.Context) bool {
	return c != nil && strings.EqualFold(strings.TrimSpace(c.Query("billing_view")), customerBillingView)
}

func projectAdminUsageStats(stats *usagestats.UsageStats) {
	if stats == nil {
		return
	}
	stats.TotalCost = stats.TotalActualCost
	if stats.TotalAccountCost != nil {
		stats.TotalAccountCost = &stats.TotalActualCost
	}
	for i := range stats.Endpoints {
		stats.Endpoints[i].Cost = stats.Endpoints[i].ActualCost
	}
	for i := range stats.UpstreamEndpoints {
		stats.UpstreamEndpoints[i].Cost = stats.UpstreamEndpoints[i].ActualCost
	}
	for i := range stats.EndpointPaths {
		stats.EndpointPaths[i].Cost = stats.EndpointPaths[i].ActualCost
	}
}

func customerUsageStats(stats *usagestats.UsageStats) *usagestats.UsageStats {
	if stats == nil {
		return nil
	}
	out := *stats
	out.Endpoints = append([]usagestats.EndpointStat(nil), stats.Endpoints...)
	out.UpstreamEndpoints = append([]usagestats.EndpointStat(nil), stats.UpstreamEndpoints...)
	out.EndpointPaths = append([]usagestats.EndpointStat(nil), stats.EndpointPaths...)
	projectAdminUsageStats(&out)
	return &out
}

func projectAdminDashboardStats(stats *usagestats.DashboardStats) {
	if stats == nil {
		return
	}
	stats.TotalCost = stats.TotalActualCost
	stats.TodayCost = stats.TodayActualCost
	stats.TotalAccountCost = stats.TotalActualCost
	stats.TodayAccountCost = stats.TodayActualCost
}

func customerDashboardStats(stats *usagestats.DashboardStats) *usagestats.DashboardStats {
	if stats == nil {
		return nil
	}
	out := *stats
	projectAdminDashboardStats(&out)
	return &out
}

func projectAdminTrend(values []usagestats.TrendDataPoint) {
	for i := range values {
		values[i].Cost = values[i].ActualCost
	}
}

func customerTrend(values []usagestats.TrendDataPoint) []usagestats.TrendDataPoint {
	out := append([]usagestats.TrendDataPoint(nil), values...)
	projectAdminTrend(out)
	return out
}

func projectAdminModels(values []usagestats.ModelStat) {
	for i := range values {
		values[i].Cost = values[i].ActualCost
		values[i].AccountCost = values[i].ActualCost
	}
}

func customerModels(values []usagestats.ModelStat) []usagestats.ModelStat {
	out := append([]usagestats.ModelStat(nil), values...)
	projectAdminModels(out)
	return out
}

func projectAdminGroups(values []usagestats.GroupStat) {
	for i := range values {
		values[i].Cost = values[i].ActualCost
		values[i].AccountCost = values[i].ActualCost
	}
}

func customerGroups(values []usagestats.GroupStat) []usagestats.GroupStat {
	out := append([]usagestats.GroupStat(nil), values...)
	projectAdminGroups(out)
	return out
}

func projectAdminUserTrend(values []usagestats.UserUsageTrendPoint) {
	for i := range values {
		values[i].Cost = values[i].ActualCost
	}
}

func customerUserTrend(values []usagestats.UserUsageTrendPoint) []usagestats.UserUsageTrendPoint {
	out := append([]usagestats.UserUsageTrendPoint(nil), values...)
	projectAdminUserTrend(out)
	return out
}

func projectAdminUserBreakdown(values []usagestats.UserBreakdownItem) {
	for i := range values {
		values[i].Cost = values[i].ActualCost
		values[i].AccountCost = values[i].ActualCost
	}
}

func customerUserBreakdown(values []usagestats.UserBreakdownItem) []usagestats.UserBreakdownItem {
	out := append([]usagestats.UserBreakdownItem(nil), values...)
	projectAdminUserBreakdown(out)
	return out
}
