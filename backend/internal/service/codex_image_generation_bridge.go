package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tidwall/gjson"
)

// Validate hosted image tools as well as the top-level model: the scheduler's
// text-model check alone cannot authorize a second model inside tools.
func validateOpenAIImageModelAuthorization(account *Account, body []byte) error {
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
					model = openAIImagesDefaultModel
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
func codexImageGenerationModel(account *Account) string {
	if account == nil || !account.IsOpenAI() {
		return ""
	}
	mapping := account.GetModelMapping()
	if len(mapping) == 0 {
		return openAIImagesDefaultModel
	}
	if mapped, ok := resolveRequestedModelInMapping(mapping, openAIImagesDefaultModel); ok && isOpenAIImageGenerationModel(mapped) {
		return mapped
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
This key's group permits image generation through the current Starbridge provider. Responses Lite cannot attach a hosted image tool. For an image the user requested, use a callable local image_gen tool if present; otherwise use the installed starbridge-imagegen skill with the selected provider's existing credential. The gateway can select an image-capable account within the group when the text account cannot generate images. Do not require OpenArt or a new OPENAI_API_KEY because a built-in tool is missing. Never print credentials; save and inspect the image before reporting completion. Do not generate for text-only requests.
</starbridge-image-api-available>`

func ensureCodexImageGenerationBridge(body map[string]any, account *Account, responsesLite bool) bool {
	if responsesLite {
		if account == nil || !account.IsOpenAI() || isCodexSparkModel(firstNonEmptyString(body["model"])) || hasCodexImageGenerationFunctionTool(body) {
			return false
		}
		existing, ok := body["instructions"].(string)
		if body["instructions"] != nil && !ok {
			return false
		}
		if strings.Contains(existing, codexImageAPIAvailableMarker) {
			return false
		}
		body["instructions"] = strings.TrimSpace(existing + "\n\n" + codexImageAPIAvailableInstructions)
		return true
	}
	model := codexImageGenerationModel(account)
	if model == "" {
		return false
	}
	injected := ensureOpenAIResponsesImageGenerationTool(body)
	modified := injected
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
