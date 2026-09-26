// Package modelcatalog contains the small, public model catalogue used by
// selectors and static model-list fallbacks. Runtime request handling still
// accepts explicitly configured custom model IDs.
package modelcatalog

import "strings"

var catalogs = map[string][]string{
	"openai": {
		// Keep gpt-5.6 and gpt-6 aliases routable, but list their concrete models only.
		"gpt-5.5", "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna",
		"gpt-6-astra", "gpt-6-sol", "gpt-6-luna",
		"gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst",
	},
	"anthropic": {
		"claude-fable-5-1", "claude-fable-5", "claude-opus-4-5-20251101",
		"claude-opus-4-6", "claude-opus-4-7", "claude-opus-4-8", "claude-opus-5",
		"claude-sonnet-4-5-20250929", "claude-sonnet-4-6", "claude-sonnet-5",
		"claude-haiku-4-5-20251001",
	},
	"gemini": {
		"gemini-3.1-flash-image", "gemini-2.5-flash-image", "gemini-3.5-flash",
		"gemini-2.5-flash", "gemini-2.5-pro",
	},
	"antigravity": {
		"claude-fable-5-1", "claude-fable-5", "claude-opus-4-6", "claude-opus-4-6-thinking",
		"claude-opus-4-7", "claude-opus-4-8", "claude-opus-4-5-thinking",
		"claude-sonnet-4-6", "claude-sonnet-4-5", "claude-sonnet-4-5-thinking",
		"gemini-3.1-flash-image", "gemini-2.5-flash-image", "gemini-2.5-flash",
		"gemini-2.5-flash-lite", "gemini-2.5-flash-thinking", "gemini-2.5-pro",
		"gemini-3-flash", "gemini-3-pro-high", "gemini-3-pro-low",
		"gemini-3.1-pro-high", "gemini-3.1-pro-low",
		"gemini-3.6-flash", "gemini-3.6-flash-high", "gemini-3.6-flash-low",
		"gemini-3.6-flash-medium", "gemini-3.6-flash-tiered",
		"gemini-3.7-flash", "gemini-3.7-flash-high", "gemini-3.7-flash-low",
		"gemini-3.7-flash-medium", "gemini-3.7-flash-tiered",
		"gemini-3.8-flash", "gemini-3.8-flash-high", "gemini-3.8-flash-low",
		"gemini-3.8-flash-medium", "gemini-3.8-flash-tiered",
	},
	"grok": {
		"grok-4.6", "grok-4.5", "grok-4.3", "grok-build-0.1", "grok-composer-2.5-fast",
		"grok-4.20-0309-reasoning", "grok-4.20-0309-non-reasoning",
		"grok-4.20-multi-agent-0309", "grok-imagine-image-quality", "grok-imagine-image",
		"grok-imagine-image-2.0", "grok-imagine-video", "grok-imagine-video-1.5",
	},
	"zhipu": {
		"glm-4.5", "glm-4.5-air", "glm-4.5-flash", "glm-4.6", "glm-4.7", "glm-4.7-flash",
		"glm-5", "glm-5-turbo", "glm-5.1", "glm-5.2", "glm-5.3", "glm-5.3-flash",
	},
	"qwen": {
		"qwen-turbo", "qwen-plus", "qwen-max", "qwen-long", "qwen3-235b-a22b",
	},
	"deepseek": {
		"deepseek-chat", "deepseek-reasoner", "deepseek-v4-pro", "deepseek-v4-flash", "deepseek-flash",
	},
	"mistral": {
		"mistral-small-latest", "mistral-medium-latest", "mistral-large-latest",
		"codestral-latest", "pixtral-large-latest",
	},
	"meta":     {"llama-3.3-70b-instruct"},
	"cohere":   {"command-a-03-2025", "command-r", "command-r-plus"},
	"yi":       {"yi-large", "yi-large-turbo", "yi-vision"},
	"moonshot": {"kimi-latest", "kimi-for-coding", "kimi-k2"},
	"doubao": {
		"doubao-1.5-pro-256k", "doubao-1.5-pro-32k", "doubao-1.5-lite-32k",
		"doubao-1.5-pro-vision-32k", "doubao-1.5-thinking-pro",
	},
	"minimax": {
		"MiniMax-M3", "MiniMax-M2.7", "MiniMax-M2.7-highspeed", "MiniMax-M2.5",
		"MiniMax-M2.5-highspeed", "MiniMax-M2.1", "MiniMax-M2.1-highspeed",
	},
	"baidu": {
		"ernie-4.0-8k-latest", "ernie-4.0-turbo-8k", "ernie-speed-128k",
		"ernie-speed-pro-128k", "ernie-lite-pro-128k",
	},
	"spark":      {"spark-desk-v4.0", "spark-pro", "spark-max", "spark-ultra"},
	"hunyuan":    {"hunyuan-pro", "hunyuan-turbo", "hunyuan-large", "hunyuan-vision", "hunyuan-code"},
	"perplexity": {"sonar", "sonar-pro", "sonar-reasoning"},
	"opencode_go": {
		"grok-4.6", "gpt-5.6-luna", "glm-5.3-flash", "glm-5.3", "glm-5.2", "glm-5.1",
		"kimi-k3", "kimi-k2.7-code", "kimi-k2.6", "longcat-2.0", "deepseek-v4-pro",
		"deepseek-v4-flash", "mimo-v2.5", "mimo-v2.5-pro", "minimax-m3", "minimax-m2.7",
		"minimax-m2.5", "qwen3.8-max", "qwen3.8-flash", "qwen3.7-max", "qwen3.7-plus",
		"qwen3.6-plus", "hy3",
	},
}

// ModelsForPlatform returns a defensive copy so callers cannot mutate the
// catalogue shared by concurrent handlers.
func ModelsForPlatform(platform string) []string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform == "claude" || platform == "bedrock" {
		platform = "anthropic"
	}
	if platform == "kimi" {
		platform = "moonshot"
	}
	if platform == "xai" {
		platform = "grok"
	}
	if platform == "composite" {
		return compositeModels()
	}
	models := catalogs[platform]
	return append([]string(nil), models...)
}

func compositeModels() []string {
	seen := make(map[string]struct{})
	models := make([]string, 0)
	for _, platform := range []string{
		"anthropic", "gemini", "openai", "antigravity", "grok", "zhipu", "qwen",
		"deepseek", "mistral", "meta", "cohere", "yi", "moonshot", "doubao", "minimax",
		"baidu", "spark", "hunyuan", "perplexity", "opencode_go",
	} {
		for _, model := range catalogs[platform] {
			if _, ok := seen[model]; ok {
				continue
			}
			seen[model] = struct{}{}
			models = append(models, model)
		}
	}
	return models
}

func IsSupported(platform, model string) bool {
	model = strings.TrimSpace(model)
	if strings.HasPrefix(model, "models/") {
		model = strings.TrimPrefix(model, "models/")
	}
	for _, candidate := range ModelsForPlatform(platform) {
		if candidate == model {
			return true
		}
	}
	return false
}

// IsVisibleOpenAIModel keeps the public OpenAI family on the curated list,
// while leaving explicitly configured compatible-provider IDs alone. This is
// a discovery policy only; legacy IDs can still be routed by existing keys.
func IsVisibleOpenAIModel(model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	for _, id := range catalogs["openai"] {
		if model == id {
			return true
		}
	}
	model = strings.TrimPrefix(strings.ToLower(model), "models/")
	model = strings.TrimPrefix(model, "openai/")
	if strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "codex") || strings.HasPrefix(model, "chatgpt") {
		return false
	}
	return !(len(model) > 1 && model[0] == 'o' && model[1] >= '0' && model[1] <= '9')
}

func FilterVisibleOpenAIModels(models []string) []string {
	visible := make([]string, 0, len(models))
	for _, model := range models {
		if IsVisibleOpenAIModel(model) {
			visible = append(visible, model)
		}
	}
	return visible
}

// FilterSupportedModels removes IDs that are not in the curated public
// catalogue. Unknown platforms are left untouched because they may represent
// an OpenAI-compatible custom upstream rather than a first-party provider.
func FilterSupportedModels(platform string, models []string) []string {
	supported := ModelsForPlatform(platform)
	if len(supported) == 0 {
		return append([]string(nil), models...)
	}
	allowed := make(map[string]struct{}, len(supported))
	for _, id := range supported {
		allowed[id] = struct{}{}
	}
	filtered := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		model = strings.TrimSpace(strings.TrimPrefix(model, "models/"))
		if model == "" {
			continue
		}
		if _, ok := allowed[model]; !ok {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		filtered = append(filtered, model)
	}
	return filtered
}
