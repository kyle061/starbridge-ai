package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserMonitorResponsesIncludeCheckMode(t *testing.T) {
	view := userMonitorViewToItem(&service.UserMonitorView{
		ID:        1,
		CheckMode: service.MonitorCheckModeProbe,
	}, false)
	require.Equal(t, service.MonitorCheckModeProbe, view.CheckMode)

	detail := userMonitorDetailToResponse(&service.UserMonitorDetail{
		ID:        1,
		CheckMode: service.MonitorCheckModeQuota,
	})
	require.Equal(t, service.MonitorCheckModeQuota, detail.CheckMode)
}
