package service

import "testing"

func TestIsGPT6Model(t *testing.T) {
	for _, model := range []string{"gpt-6", "gpt-6-astra", "openai/gpt-6-xhigh"} {
		if !IsGPT6Model(model) {
			t.Fatalf("IsGPT6Model(%q) = false", model)
		}
	}
	for _, model := range []string{"gpt-5.4", "deepseek-v4-pro", "my-gpt-6-compatible"} {
		if IsGPT6Model(model) {
			t.Fatalf("IsGPT6Model(%q) = true", model)
		}
	}
}
