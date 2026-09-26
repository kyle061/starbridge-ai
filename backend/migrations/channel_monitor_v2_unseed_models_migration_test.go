package migrations

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelMonitorV2UnseedModelsMatchesFactoryInventories(t *testing.T) {
	seedSQL, err := FS.ReadFile("197_channel_monitor_v2_seed_popular_models.sql")
	require.NoError(t, err)
	unseedSQL, err := FS.ReadFile("254_channel_monitor_v2_unseed_model_inventory.sql")
	require.NoError(t, err)
	match := regexp.MustCompile(`(?s)\$platforms\$(.*?)\$platforms\$`).FindSubmatch(seedSQL)
	require.Len(t, match, 2)
	var factory []struct {
		Platform string   `json:"platform"`
		Models   []string `json:"models"`
	}
	require.NoError(t, json.Unmarshal(match[1], &factory))
	require.Len(t, factory, 6)
	for _, platform := range factory {
		sort.Strings(platform.Models)
		fingerprint := fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(platform.Models, "\x1f"))))
		require.Contains(t, string(unseedSQL), fmt.Sprintf("('%s', '%s')", platform.Platform, fingerprint))
	}
	require.Contains(t, string(unseedSQL), `ORDER BY model COLLATE "C"`)
	require.Contains(t, string(unseedSQL), "cfg.platforms IS DISTINCT FROM cleaned.platforms")
	require.Contains(t, string(unseedSQL), "ELSE entry.value END")
}
