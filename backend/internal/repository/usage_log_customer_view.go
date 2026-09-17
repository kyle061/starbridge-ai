package repository

import "fmt"

// usageLogTokenExpr returns the persisted token count, optionally weighted by
// the effective customer multiplier captured on the usage row. Rounding each
// row matches the per-record customer DTO projection and avoids leaking raw
// token totals from mixed-rate aggregates.
func usageLogTokenExpr(column, alias string, customerView bool) string {
	qualified := column
	rate := "rate_multiplier"
	if alias != "" {
		qualified = alias + "." + column
		rate = alias + ".rate_multiplier"
	}
	if !customerView {
		return qualified
	}
	return fmt.Sprintf(
		"CAST(ROUND(COALESCE(%s, 0) * CASE WHEN COALESCE(%s, 0) > 0 THEN %s ELSE 1 END) AS BIGINT)",
		qualified, rate, rate,
	)
}

type usageLogTokenExpressions struct {
	input        string
	output       string
	cacheCreate  string
	cacheRead    string
	total        string
}

func newUsageLogTokenExpressions(alias string, customerView bool) usageLogTokenExpressions {
	input := usageLogTokenExpr("input_tokens", alias, customerView)
	output := usageLogTokenExpr("output_tokens", alias, customerView)
	cacheCreate := usageLogTokenExpr("cache_creation_tokens", alias, customerView)
	cacheRead := usageLogTokenExpr("cache_read_tokens", alias, customerView)
	return usageLogTokenExpressions{
		input:       input,
		output:      output,
		cacheCreate: cacheCreate,
		cacheRead:   cacheRead,
		total:       input + " + " + output + " + " + cacheCreate + " + " + cacheRead,
	}
}
