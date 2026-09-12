package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"monitor/internal/apikey"
)

func TestProxyProfileStoreEncryptsAndPreservesURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), ProxyProfilesFileName)
	cipher, err := apikey.NewKeyCipher(strings.Repeat("ab", 32))
	if err != nil {
		t.Fatal(err)
	}
	store := NewProxyProfileStore(path, cipher)

	created, err := store.Create("日本出口", "socks5://user:secret@proxy.example:1080")
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("代理配置 ID 不能为空")
	}

	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	diskText := string(disk)
	if strings.Contains(diskText, "secret") || strings.Contains(diskText, "proxy.example") {
		t.Fatalf("代理明文泄露到磁盘: %s", diskText)
	}
	if !strings.Contains(diskText, "url_encrypted:") {
		t.Fatalf("磁盘配置缺少 url_encrypted: %s", diskText)
	}

	profiles, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].URL != "socks5://user:secret@proxy.example:1080" {
		t.Fatalf("解密后的代理配置不正确: %+v", profiles)
	}

	updated, err := store.Update(created.ID, "日本出口改名", "")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "日本出口改名" || updated.URL != created.URL {
		t.Fatalf("空 URL 更新不应清除原地址: %+v", updated)
	}
}

func TestProxyProfileStoreReadsLegacyPlaintext(t *testing.T) {
	path := filepath.Join(t.TempDir(), ProxyProfilesFileName)
	content := "proxies:\n  - id: legacy\n    name: 旧代理\n    url: http://proxy.example:8080\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	store := NewProxyProfileStore(path, nil)
	profiles, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].URL != "http://proxy.example:8080" {
		t.Fatalf("旧明文代理读取失败: %+v", profiles)
	}
}

func TestResolveMonitorProxyProfiles(t *testing.T) {
	monitors := []ServiceConfig{
		{Provider: "acme", Service: "cx", Channel: "main", ProxyProfile: "japan"},
		{Parent: "acme/cx/main"},
	}
	if err := ResolveMonitorProxyProfiles(monitors, map[string]string{"japan": "http://proxy.example:8080"}); err != nil {
		t.Fatal(err)
	}
	if monitors[0].Proxy != "http://proxy.example:8080" {
		t.Fatalf("Proxy = %q", monitors[0].Proxy)
	}
	if err := ResolveMonitorProxyProfiles(monitors, map[string]string{}); err == nil {
		t.Fatal("引用不存在的代理配置应报错")
	}
}
