package openai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultModelsIncludeBareGPT56Alias(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-5.6")
}

func TestDefaultModelsIncludeGPT6Astra(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-6-astra")
	require.Contains(t, DefaultModelIDs(), "gpt-6")
	var displayName string
	for _, model := range DefaultModels {
		if model.ID == "gpt-6-astra" {
			displayName = model.DisplayName
			break
		}
	}
	require.Equal(t, "GPT-6 Astra", displayName)
}

func TestSelectableModelIDsUseCanonicalGPTIDs(t *testing.T) {
	selectable := SelectableModelIDs()
	require.NotContains(t, selectable, "gpt-5.6")
	require.NotContains(t, selectable, "gpt-6")
	require.Contains(t, selectable, "gpt-5.6-sol")
	require.Contains(t, selectable, "gpt-6-astra")

	// Preserve aliases in the broader model set so existing clients remain routable.
	require.Contains(t, DefaultModelIDs(), "gpt-5.6")
	require.Contains(t, DefaultModelIDs(), "gpt-6")
}

func TestDefaultModelsPreferConcreteGPT56SolForAccountTests(t *testing.T) {
	require.NotEmpty(t, DefaultModels)
	require.Equal(t, "gpt-5.6-sol", DefaultModels[0].ID)
}

func TestDefaultModelsIncludeGPTImage25(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-image-2.5-flare")
	require.Contains(t, DefaultModelIDs(), "gpt-image-2.5-sunburst")
}
