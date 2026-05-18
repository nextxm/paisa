package config

import (
	"testing"
)

func TestMissingPriceLoggingConfigDefaults(t *testing.T) {
	err := LoadConfig([]byte(`
journal_path: /tmp/test.journal
db_path: /tmp/test.db
`), "")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	cfg := GetConfig()

	if cfg.DisableMissingPriceLogging != false {
		t.Errorf("DisableMissingPriceLogging = %v, want false", cfg.DisableMissingPriceLogging)
	}
	if IsMissingPriceLoggingDisabled() != false {
		t.Errorf("IsMissingPriceLoggingDisabled() = %v, want false", IsMissingPriceLoggingDisabled())
	}
}

func TestMissingPriceLoggingConfigCustom(t *testing.T) {
	err := LoadConfig([]byte(`
journal_path: /tmp/test.journal
db_path: /tmp/test.db
disable_missing_price_logging: true
`), "")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	cfg := GetConfig()

	if cfg.DisableMissingPriceLogging != true {
		t.Errorf("DisableMissingPriceLogging = %v, want true", cfg.DisableMissingPriceLogging)
	}
	if IsMissingPriceLoggingDisabled() != true {
		t.Errorf("IsMissingPriceLoggingDisabled() = %v, want true", IsMissingPriceLoggingDisabled())
	}
}
