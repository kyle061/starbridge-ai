package service

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

var groupImageModelID = regexp.MustCompile(`^gpt-image-[a-zA-Z0-9._-]+$`)

func normalizeGroupDefaultImageModel(platform, model string) (string, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", nil
	}
	if (platform != PlatformOpenAI && platform != PlatformComposite) || len(model) > 100 || !groupImageModelID.MatchString(model) {
		return "", fmt.Errorf("default_image_model requires an OpenAI or composite group and a gpt-image model ID")
	}
	return model, nil
}

func groupImageModel(group *Group) string {
	if group != nil && (group.Platform == PlatformOpenAI || group.Platform == PlatformComposite) && groupImageModelID.MatchString(group.DefaultImageModel) {
		return group.DefaultImageModel
	}
	return openAIImagesDefaultModel
}

func setMissingImageToolModel(body map[string]any, model string) bool {
	if model == "" {
		return false
	}
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

func setGroupDefaultImageToolModel(body map[string]any, group *Group) bool {
	if group == nil || group.DefaultImageModel == "" || !GroupAllowsImageGeneration(group) {
		return false
	}
	return setMissingImageToolModel(body, groupImageModel(group))
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
		for _, upstreamModel := range mapping {
			if upstreamModel == model {
				return nil
			}
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
					model = codexImageGenerationModel(account, group)
					if model == "" {
						return fmt.Errorf("image generation is not authorized for this account")
					}
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

func imageModelVersion(model string) (int, int) {
	parts := strings.Split(strings.TrimPrefix(model, "gpt-image-"), "-")
	version := strings.SplitN(parts[0], ".", 3)
	major, err := strconv.Atoi(version[0])
	if err != nil {
		return -1, -1
	}
	if len(version) < 2 {
		return major, 0
	}
	minor, err := strconv.Atoi(version[1])
	if err != nil {
		return major, 0
	}
	return major, minor
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
	if group != nil && group.DefaultImageModel != "" {
		if mapped, ok := resolveRequestedModelInMapping(mapping, preferred); ok && isOpenAIImageGenerationModel(mapped) {
			return mapped
		}
		return "" // A configured model must not silently fall back to a different one.
	}
	models := make([]string, 0, len(mapping))
	for _, model := range mapping {
		if IsGPTImageGenerationModel(model) && !strings.Contains(model, "*") {
			models = append(models, model)
		}
	}
	for _, candidate := range []string{"gpt-image-2.5-flare", "gpt-image-2.5-sunburst", openAIImagesDefaultModel} {
		if mapped, ok := resolveRequestedModelInMapping(mapping, candidate); ok && IsGPTImageGenerationModel(mapped) && !strings.Contains(mapped, "*") {
			models = append(models, mapped)
		}
	}
	sort.Slice(models, func(i, j int) bool {
		majorI, minorI := imageModelVersion(models[i])
		majorJ, minorJ := imageModelVersion(models[j])
		if majorI != majorJ {
			return majorI > majorJ
		}
		if minorI != minorJ {
			return minorI > minorJ
		}
		return models[i] < models[j]
	})
	if len(models) > 0 {
		return models[0]
	}
	return ""
}

const codexImageAPIAvailableMarker = "<starbridge-image-api-available>"

const codexImageAPIAvailableInstructions = codexImageAPIAvailableMarker + `
For an image the user explicitly requested, use an available image tool, or the installed starbridge-imagegen skill with the currently selected Starbridge provider. The request authorizes this image prompt through that configured provider; do not ask again to switch to API mode, send the prompt, or supply another provider or key. Honor any separate runtime approval requirement or denial. Do not send unrelated files or use another provider. Show the actual image and requested answer without narrating internal tools or routing. Do not generate for text-only requests.
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
		modelGuidance := "If no image model was requested, choose an authorized image model from this key's /v1/models listing; prefer gpt-image-2.5, then gpt-image-2. Never guess a model that the key cannot use."
		if group != nil && group.DefaultImageModel != "" {
			modelGuidance = "If no image model was requested, use the group's configured image model " + groupImageModel(group) + "."
		}
		body["instructions"] = strings.TrimSpace(existing + "\n\n" + codexImageAPIAvailableInstructions + "\n" + modelGuidance)
		return true
	}
	model := codexImageGenerationModel(account, group)
	if model == "" {
		return defaultSet
	}
	if setMissingImageToolModel(body, model) {
		defaultSet = true
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
