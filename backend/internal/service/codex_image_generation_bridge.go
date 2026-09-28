package service

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/tidwall/gjson"
)

var groupImageModelID = regexp.MustCompile(`^gpt-image-[a-zA-Z0-9._-]+$`)

func normalizeGroupDefaultImageModel(platform, model string) (string, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", nil
	}
	if platform != PlatformOpenAI || len(model) > 100 || !groupImageModelID.MatchString(model) {
		return "", fmt.Errorf("default_image_model requires an OpenAI group and a gpt-image model ID")
	}
	return model, nil
}

func groupImageModel(group *Group) string {
	if group != nil && group.Platform == PlatformOpenAI && groupImageModelID.MatchString(group.DefaultImageModel) {
		return group.DefaultImageModel
	}
	return openAIImagesDefaultModel
}

func setGroupDefaultImageToolModel(body map[string]any, group *Group) bool {
	if group == nil || group.DefaultImageModel == "" || !GroupAllowsImageGeneration(group) {
		return false
	}
	model := groupImageModel(group)
	changed := false
	var visit func(any)
	visit = func(raw any) {
		tools, ok := raw.([]any)
		if !ok {
			return
		}
		for _, entry := range tools {
			tool, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			if tool["type"] == "image_generation" && strings.TrimSpace(firstNonEmptyString(tool["model"])) == "" {
				tool["model"] = model
				changed = true
			}
			visit(tool["tools"])
		}
	}
	visit(body["tools"])
	if items, ok := body["input"].([]any); ok {
		for _, raw := range items {
			if item, ok := raw.(map[string]any); ok && item["type"] == "additional_tools" {
				visit(item["tools"])
			}
		}
	}
	return changed
}

// Validate hosted image tools as well as the top-level model: the scheduler's
// text-model check alone cannot authorize a second model inside tools.
func validateOpenAIImageModelAuthorization(account *Account, body []byte, group *Group) error {
	if account == nil || !account.IsOpenAI() {
		return nil
	}
	mapping := account.GetModelMapping()
	if len(mapping) == 0 {
		return nil
	}
	check := func(model string) error {
		if mappingSupportsRequestedModel(mapping, model) {
			return nil
		}
		return fmt.Errorf("image model %q is not authorized for this account", model)
	}
	if model := strings.TrimSpace(gjson.GetBytes(body, "model").String()); isOpenAIImageGenerationModel(model) {
		if err := check(model); err != nil {
			return err
		}
	}
	var visitTools func(gjson.Result) error
	visitTools = func(tools gjson.Result) error {
		for _, tool := range tools.Array() {
			if tool.Get("type").String() == "image_generation" {
				model := strings.TrimSpace(tool.Get("model").String())
				if !isOpenAIImageGenerationModel(model) {
					model = groupImageModel(group)
				}
				if err := check(model); err != nil {
					return err
				}
			}
			if err := visitTools(tool.Get("tools")); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visitTools(gjson.GetBytes(body, "tools")); err != nil {
		return err
	}
	for _, item := range gjson.GetBytes(body, "input").Array() {
		if item.Get("type").String() == "additional_tools" {
			if err := visitTools(item.Get("tools")); err != nil {
				return err
			}
		}
	}
	return nil
}

// The group enables the feature; account model authorization still limits
// which image model can be offered. Passthrough must not bypass this check.
func codexImageGenerationModel(account *Account, group *Group) string {
	if account == nil || !account.IsOpenAI() {
		return ""
	}
	mapping := account.GetModelMapping()
	preferred := groupImageModel(group)
	if len(mapping) == 0 {
		return preferred
	}
	if mapped, ok := resolveRequestedModelInMapping(mapping, preferred); ok && isOpenAIImageGenerationModel(mapped) {
		return mapped
	}
	if group != nil && group.DefaultImageModel != "" {
		return "" // A configured model must not silently fall back to a different one.
	}
	models := make([]string, 0, len(mapping))
	for _, model := range mapping {
		if isOpenAIImageGenerationModel(model) && !strings.Contains(model, "*") {
			models = append(models, model)
		}
	}
	sort.Strings(models)
	if len(models) > 0 {
		return models[0]
	}
	return ""
}

const codexImageAPIAvailableMarker = "<starbridge-image-api-available>"

const codexImageAPIAvailableInstructions = codexImageAPIAvailableMarker + `
For an image the user requested, use an available image tool, or the installed starbridge-imagegen skill with the current provider. The current key authorizes this route. Do not ask the user for another provider or key. Show the actual image and any requested answer; do not narrate internal tools, provider routing, credentials or skill names. Do not generate for text-only requests.
</starbridge-image-api-available>`

func ensureCodexImageGenerationBridge(body map[string]any, account *Account, group *Group, responsesLite bool) bool {
	defaultSet := setGroupDefaultImageToolModel(body, group)
	if responsesLite {
		if account == nil || !account.IsOpenAI() || isCodexSparkModel(firstNonEmptyString(body["model"])) || hasCodexImageGenerationFunctionTool(body) {
			return defaultSet
		}
		existing, ok := body["instructions"].(string)
		if body["instructions"] != nil && !ok {
			return defaultSet
		}
		if strings.Contains(existing, codexImageAPIAvailableMarker) {
			return defaultSet
		}
		body["instructions"] = strings.TrimSpace(existing + "\n\n" + codexImageAPIAvailableInstructions + "\nUse image model " + groupImageModel(group) + " when the user has not chosen one.")
		return true
	}
	model := codexImageGenerationModel(account, group)
	if model == "" {
		return defaultSet
	}
	injected := ensureOpenAIResponsesImageGenerationTool(body)
	modified := injected || defaultSet
	if injected {
		tools := body["tools"].([]any)
		tools[len(tools)-1].(map[string]any)["model"] = model
	}
	if ensureOpenAIResponsesImageGenerationToolChoiceAuto(body) {
		modified = true
	}
	if normalizeOpenAIResponsesImageGenerationTools(body) {
		modified = true
	}
	if injected && applyCodexImageGenerationBridgeInstructions(body) {
		modified = true
	}
	return modified
}
