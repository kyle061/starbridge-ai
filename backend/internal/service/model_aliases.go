package service

import "strings"

// IsGPT6Model matches the public GPT6 family, including its reasoning suffixes.
// It intentionally does not match arbitrary models containing "gpt-6".
func IsGPT6Model(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if i := strings.LastIndex(model, "/"); i >= 0 {
		model = model[i+1:]
	}
	return model == "gpt-6" || strings.HasPrefix(model, "gpt-6-")
}
