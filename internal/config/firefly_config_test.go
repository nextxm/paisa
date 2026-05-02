package config

import (
	"testing"
)

func TestFireflyConfigDefaults(t *testing.T) {
	err := LoadConfig([]byte(`
journal_path: /tmp/test.journal
db_path: /tmp/test.db
`), "")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	cfg := GetConfig()

	if cfg.Labs.FireflyReconcile != false {
		t.Errorf("FireflyReconcile = %v, want false", cfg.Labs.FireflyReconcile)
	}
	if len(cfg.Firefly.IgnoreAccounts) != 0 {
		t.Errorf("IgnoreAccounts = %v, want empty", cfg.Firefly.IgnoreAccounts)
	}
}

func TestFireflyConfigCustom(t *testing.T) {
	err := LoadConfig([]byte(`
journal_path: /tmp/test.journal
db_path: /tmp/test.db
firefly:
  url: "https://firefly.example.com"
  token: "secret-token"
  ignore_accounts: ["Bank A", "Bank B"]
labs:
  firefly_reconcile: true
`), "")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	cfg := GetConfig()

	if cfg.Labs.FireflyReconcile != true {
		t.Errorf("FireflyReconcile = %v, want true", cfg.Labs.FireflyReconcile)
	}
	if cfg.Firefly.URL != "https://firefly.example.com" {
		t.Errorf("URL = %v, want https://firefly.example.com", cfg.Firefly.URL)
	}
	if cfg.Firefly.Token != "secret-token" {
		t.Errorf("Token = %v, want secret-token", cfg.Firefly.Token)
	}
	if len(cfg.Firefly.IgnoreAccounts) != 2 || cfg.Firefly.IgnoreAccounts[0] != "Bank A" {
		t.Errorf("IgnoreAccounts = %v, want [Bank A Bank B]", cfg.Firefly.IgnoreAccounts)
	}
}
