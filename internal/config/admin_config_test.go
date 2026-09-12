package config

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"monitor/internal/apikey"
)

func TestNormalizeAdminConfigRequiresSecuritySettings(t *testing.T) {
	validHashBytes, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	base := AdminConfig{
		Enabled:       true,
		Username:      "admin",
		PasswordHash:  string(validHashBytes),
		SessionSecret: "session-secret",
		EncryptionKey: strings.Repeat("ab", 32),
		SessionTTL:    "24h",
	}

	if err := (&AppConfig{Admin: base}).normalizeAdminConfig(); err != nil {
		t.Fatalf("valid admin config rejected: %v", err)
	}

	for name, mutate := range map[string]func(*AdminConfig){
		"username":       func(c *AdminConfig) { c.Username = "" },
		"password hash":  func(c *AdminConfig) { c.PasswordHash = "plain-text" },
		"session secret": func(c *AdminConfig) { c.SessionSecret = "" },
		"encryption key": func(c *AdminConfig) { c.EncryptionKey = "" },
		"session ttl":    func(c *AdminConfig) { c.SessionTTL = "0s" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := base
			mutate(&cfg)
			if err := (&AppConfig{Admin: cfg}).normalizeAdminConfig(); err == nil {
				t.Fatal("expected invalid admin config to be rejected")
			}
		})
	}
}

func TestDecryptMonitorAPIKeysPrefersEncryptedValue(t *testing.T) {
	cipher, err := apikey.NewKeyCipher(strings.Repeat("cd", 32))
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cipher.Encrypt("encrypted-secret")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &AppConfig{Admin: AdminConfig{EncryptionKey: strings.Repeat("cd", 32)}}
	monitors := []ServiceConfig{{APIKey: "legacy-secret", APIKeyEncrypted: encrypted}}
	if err := cfg.decryptMonitorAPIKeys(monitors); err != nil {
		t.Fatal(err)
	}
	if monitors[0].APIKey != "encrypted-secret" {
		t.Fatalf("encrypted API Key should win, got %q", monitors[0].APIKey)
	}
}

func TestClonePreservesAdminConfig(t *testing.T) {
	original := &AppConfig{Admin: AdminConfig{
		Enabled:            true,
		Username:           "admin",
		PasswordHash:       "hash",
		SessionTTL:         "24h",
		SessionTTLDuration: 24,
		SessionSecret:      "session",
		EncryptionKey:      "encryption",
	}}
	clone := original.clone()
	if clone.Admin != original.Admin {
		t.Fatalf("clone 丢失 Admin 配置: got %+v want %+v", clone.Admin, original.Admin)
	}
}
