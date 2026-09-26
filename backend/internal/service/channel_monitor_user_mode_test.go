package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildUserViewFromSummaryIncludesCheckMode(t *testing.T) {
	view := buildUserViewFromSummary(&ChannelMonitor{
		ID:        1,
		CheckMode: MonitorCheckModeQuotaProbe,
	}, MonitorStatusSummary{}, nil, nil)
	require.Equal(t, MonitorCheckModeQuotaProbe, view.CheckMode)

	legacyView := buildUserViewFromSummary(&ChannelMonitor{ID: 2}, MonitorStatusSummary{}, nil, nil)
	require.Equal(t, MonitorCheckModeProbe, legacyView.CheckMode)
}
