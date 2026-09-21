package service

import (
	"time"
)

// SubscriptionCacheData represents cached subscription data
type SubscriptionCacheData struct {
	Status          string
	ExpiresAt       time.Time
	DailyUsage      float64
	WeeklyUsage     float64
	MonthlyUsage    float64
	QuotaUSD        float64
	QuotaUsedUSD    float64
	UsageMultiplier float64
	PlanName        string
	Version         int64
}
