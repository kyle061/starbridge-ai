package geminicli

import "github.com/Wei-Shaw/sub2api/internal/pkg/modelcatalog"

// Model represents a selectable Gemini model for UI/testing purposes.
// Keep JSON fields consistent with existing frontend expectations.
type Model struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

// DefaultModels is the curated Gemini model list used by the admin UI "test account" flow.
var DefaultModels = []Model{
	{ID: "gemini-2.5-flash", Type: "model", DisplayName: "Gemini 2.5 Flash", CreatedAt: ""},
	{ID: "gemini-2.5-flash-image", Type: "model", DisplayName: "Gemini 2.5 Flash Image", CreatedAt: ""},
	{ID: "gemini-2.5-pro", Type: "model", DisplayName: "Gemini 2.5 Pro", CreatedAt: ""},
	{ID: "gemini-3.5-flash", Type: "model", DisplayName: "Gemini 3.5 Flash", CreatedAt: ""},
	{ID: "gemini-3.1-flash-image", Type: "model", DisplayName: "Gemini 3.1 Flash Image", CreatedAt: ""},
}

// GoogleOneModels is the conservative model set exposed for legacy Google One
// OAuth accounts. Newer subscription models are served through Antigravity
// OAuth rather than the retired consumer Gemini CLI / Code Assist channel.
var GoogleOneModels = []Model{
	{ID: "gemini-2.5-flash", Type: "model", DisplayName: "Gemini 2.5 Flash", CreatedAt: ""},
	{ID: "gemini-2.5-pro", Type: "model", DisplayName: "Gemini 2.5 Pro", CreatedAt: ""},
	{ID: "gemini-2.0-flash", Type: "model", DisplayName: "Gemini 2.0 Flash", CreatedAt: ""},
}

// SelectableModels is the stable Gemini catalogue used by UI and static
// fallbacks. Google One has a separate legacy catalogue below.
func SelectableModels() []Model {
	byID := make(map[string]Model, len(DefaultModels))
	for _, model := range DefaultModels {
		byID[model.ID] = model
	}
	models := make([]Model, 0, len(modelcatalog.ModelsForPlatform("gemini")))
	for _, id := range modelcatalog.ModelsForPlatform("gemini") {
		if model, ok := byID[id]; ok {
			models = append(models, model)
		}
	}
	return models
}

// GoogleOneModelMapping returns a new whitelist map for each account so callers
// cannot mutate the package-level catalog.
func GoogleOneModelMapping() map[string]string {
	mapping := make(map[string]string, len(GoogleOneModels))
	for _, model := range GoogleOneModels {
		mapping[model.ID] = model.ID
	}
	return mapping
}

// DefaultTestModel is the default model to preselect in test flows.
const DefaultTestModel = "gemini-2.5-flash"
