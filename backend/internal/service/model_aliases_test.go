package service

import "testing"

func TestIsGPT6Model(t *testing.T) {
	for _, model := range []string{"gpt-6", "gpt-6-astra", "gpt-6.1-sol", "gpt-6.2-sol", "openai/gpt-6.1-sol", "openai/gpt-6-xhigh"} {
		if !IsGPT6Model(model) {
			t.Fatalf("IsGPT6Model(%q) = false", model)
		}
	}
	for _, model := range []string{"gpt-5.4", "gpt-60", "gpt-6.x-sol", "gpt-6.", "gpt-6.1fake", "deepseek-v4-pro", "my-gpt-6-compatible"} {
		if IsGPT6Model(model) {
			t.Fatalf("IsGPT6Model(%q) = true", model)
		}
	}
}
