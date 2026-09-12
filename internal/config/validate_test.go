package config

import "testing"

func TestValidateAllowsEmptyMonitorsForAdmin(t *testing.T) {
	if err := (&AppConfig{Admin: AdminConfig{Enabled: true}}).validate(); err != nil {
		t.Fatalf("启用管理后台时空监测列表应该允许启动: %v", err)
	}

	if err := (&AppConfig{}).validate(); err == nil {
		t.Fatal("未启用 onboarding 或 admin 时空监测列表应该被拒绝")
	}
}

func TestCheckRuntimeModelIDs(t *testing.T) {
	// 全有 model_id → nil
	ok := []ServiceConfig{{Provider: "P", Service: "cc", Channel: "c", Model: "Opus", ModelID: "md_x"}}
	if err := CheckRuntimeModelIDs(ok); err != nil {
		t.Fatalf("全有 id 不该报错: %v", err)
	}
	// 缺 model_id → error
	bad := []ServiceConfig{{Provider: "P", Service: "cc", Channel: "c", Model: "Opus", ModelID: ""}}
	if err := CheckRuntimeModelIDs(bad); err == nil {
		t.Fatal("缺 model_id 必须报错")
	}
	// 空 monitors（onboarding-zero-monitors 启动）→ nil
	if err := CheckRuntimeModelIDs(nil); err != nil {
		t.Fatalf("空 monitors 不该报错: %v", err)
	}
}
