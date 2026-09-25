package dto

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// profitControlJSONFields 是分组利润控制的三个 JSON 字段，只能出现在管理员 DTO 中。
var profitControlJSONFields = []string{
	"profit_control_enabled",
	"profit_min_margin",
	"profit_safety_buffer",
}

func profitControlServiceGroup() *service.Group {
	return &service.Group{
		ID:                   7,
		Name:                 "profit-gated",
		Platform:             service.PlatformAnthropic,
		RateMultiplier:       2.0,
		Status:               service.StatusActive,
		ProfitControlEnabled: true,
		ProfitMinMargin:      0.3,
		ProfitSafetyBuffer:   0.05,
	}
}

func marshalToMap(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

// TestGroupFromServiceOmitsProfitControl 钉死普通用户侧的分组 DTO 不泄露利润控制配置。
func TestGroupFromServiceOmitsProfitControl(t *testing.T) {
	for name, got := range map[string]any{
		"GroupFromService":        GroupFromService(profitControlServiceGroup()),
		"GroupFromServiceShallow": GroupFromServiceShallow(profitControlServiceGroup()),
	} {
		fields := marshalToMap(t, got)
		for _, f := range profitControlJSONFields {
			if _, ok := fields[f]; ok {
				t.Errorf("%s: 普通用户 DTO 不得包含 %q", name, f)
			}
		}
		if _, ok := fields["rate_multiplier"]; ok {
			t.Errorf("%s: 普通用户 DTO 不得包含 rate_multiplier", name)
		}
	}
}

// TestGroupFromServiceAdminIncludesProfitControl 钉死管理端仍能读写利润控制配置。
func TestGroupFromServiceAdminIncludesProfitControl(t *testing.T) {
	admin := GroupFromServiceAdmin(profitControlServiceGroup())
	if admin.ProfitControlEnabled != true || admin.ProfitMinMargin != 0.3 || admin.ProfitSafetyBuffer != 0.05 {
		t.Fatalf("管理员 DTO 未透传利润控制配置: %+v", admin)
	}
	fields := marshalToMap(t, admin)
	for _, f := range profitControlJSONFields {
		if _, ok := fields[f]; !ok {
			t.Errorf("管理员 DTO 应包含 %q", f)
		}
	}
}

func TestGroupFromServiceAdminKeepsNestedRateWithoutRecursiveGroups(t *testing.T) {
	group := profitControlServiceGroup()
	group.AccountGroups = []service.AccountGroup{{AccountID: 9, GroupID: group.ID, Group: group}}

	admin := GroupFromServiceAdmin(group)
	if len(admin.AccountGroups) != 1 || admin.AccountGroups[0].Group == nil {
		t.Fatalf("expected one nested admin group reference: %+v", admin)
	}
	if admin.AccountGroups[0].Group.RateMultiplier != group.RateMultiplier {
		t.Fatalf("nested group rate multiplier was lost: %+v", admin.AccountGroups[0].Group)
	}
	if len(admin.AccountGroups[0].Group.AccountGroups) != 0 {
		t.Fatal("nested group references must not recursively expand account groups")
	}
	marshalToMap(t, admin)
}
