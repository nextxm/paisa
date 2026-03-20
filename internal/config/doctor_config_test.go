package config

import (
"testing"
)

func TestDoctorConfigDefaults(t *testing.T) {
err := LoadConfig([]byte(`
journal_path: /tmp/test.journal
db_path: /tmp/test.db
`), "")
if err != nil {
t.Fatalf("Failed to load config: %v", err)
}
cfg := GetConfig()

// All rules should be enabled by default
if cfg.Doctor.NegativeBalance.Enabled != Yes {
t.Errorf("NegativeBalance.Enabled = %v, want %v", cfg.Doctor.NegativeBalance.Enabled, Yes)
}
if cfg.Doctor.NonCreditAccount.Enabled != Yes {
t.Errorf("NonCreditAccount.Enabled = %v, want %v", cfg.Doctor.NonCreditAccount.Enabled, Yes)
}
if cfg.Doctor.NonDebitAccount.Enabled != Yes {
t.Errorf("NonDebitAccount.Enabled = %v, want %v", cfg.Doctor.NonDebitAccount.Enabled, Yes)
}
if cfg.Doctor.ExchangePriceMissing.Enabled != Yes {
t.Errorf("ExchangePriceMissing.Enabled = %v, want %v", cfg.Doctor.ExchangePriceMissing.Enabled, Yes)
}
if cfg.Doctor.UnitPriceMismatch.Enabled != Yes {
t.Errorf("UnitPriceMismatch.Enabled = %v, want %v", cfg.Doctor.UnitPriceMismatch.Enabled, Yes)
}
if cfg.Doctor.AssetAllocationMissing.Enabled != Yes {
t.Errorf("AssetAllocationMissing.Enabled = %v, want %v", cfg.Doctor.AssetAllocationMissing.Enabled, Yes)
}

// Default patterns should be set
if len(cfg.Doctor.NegativeBalance.Pattern) != 1 || cfg.Doctor.NegativeBalance.Pattern[0] != "Assets:%" {
t.Errorf("NegativeBalance.Pattern = %v, want [Assets:%%]", cfg.Doctor.NegativeBalance.Pattern)
}
if len(cfg.Doctor.NonCreditAccount.Pattern) != 1 || cfg.Doctor.NonCreditAccount.Pattern[0] != "Income:%" {
t.Errorf("NonCreditAccount.Pattern = %v, want [Income:%%]", cfg.Doctor.NonCreditAccount.Pattern)
}
if len(cfg.Doctor.NonDebitAccount.Pattern) != 1 || cfg.Doctor.NonDebitAccount.Pattern[0] != "Expenses:%" {
t.Errorf("NonDebitAccount.Pattern = %v, want [Expenses:%%]", cfg.Doctor.NonDebitAccount.Pattern)
}
if len(cfg.Doctor.AssetAllocationMissing.Pattern) != 1 || cfg.Doctor.AssetAllocationMissing.Pattern[0] != "Assets:%" {
t.Errorf("AssetAllocationMissing.Pattern = %v, want [Assets:%%]", cfg.Doctor.AssetAllocationMissing.Pattern)
}
}

func TestDoctorConfigCustom(t *testing.T) {
err := LoadConfig([]byte(`
journal_path: /tmp/test.journal
db_path: /tmp/test.db
doctor:
  negative_balance:
    enabled: "no"
    pattern: ["Assets:Savings:%", "Assets:Investments:%"]
  non_credit_account:
    enabled: "no"
  exchange_price_missing:
    enabled: "no"
`), "")
if err != nil {
t.Fatalf("Failed to load config: %v", err)
}
cfg := GetConfig()

// Disabled rules
if cfg.Doctor.NegativeBalance.Enabled != No {
t.Errorf("NegativeBalance.Enabled = %v, want %v", cfg.Doctor.NegativeBalance.Enabled, No)
}
if cfg.Doctor.NonCreditAccount.Enabled != No {
t.Errorf("NonCreditAccount.Enabled = %v, want %v", cfg.Doctor.NonCreditAccount.Enabled, No)
}
if cfg.Doctor.ExchangePriceMissing.Enabled != No {
t.Errorf("ExchangePriceMissing.Enabled = %v, want %v", cfg.Doctor.ExchangePriceMissing.Enabled, No)
}

// Custom patterns
if len(cfg.Doctor.NegativeBalance.Pattern) != 2 ||
cfg.Doctor.NegativeBalance.Pattern[0] != "Assets:Savings:%" ||
cfg.Doctor.NegativeBalance.Pattern[1] != "Assets:Investments:%" {
t.Errorf("NegativeBalance.Pattern = %v, want [Assets:Savings:%% Assets:Investments:%%]", cfg.Doctor.NegativeBalance.Pattern)
}

// Rules not explicitly set should still default
if cfg.Doctor.NonDebitAccount.Enabled != Yes {
t.Errorf("NonDebitAccount.Enabled = %v, want %v (should default)", cfg.Doctor.NonDebitAccount.Enabled, Yes)
}
if cfg.Doctor.UnitPriceMismatch.Enabled != Yes {
t.Errorf("UnitPriceMismatch.Enabled = %v, want %v (should default)", cfg.Doctor.UnitPriceMismatch.Enabled, Yes)
}
}
