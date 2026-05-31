package config

import "testing"

func TestLifeGoalsLoadFromConfig(t *testing.T) {
	err := LoadConfig([]byte(`
journal_path: /tmp/test.journal
db_path: /tmp/test.db
goals:
  life:
    - name: House Down Payment
      icon: home
      type: milestone
      target_amount: 5000000
      target_date: 2030-06
      priority: 10
      funded_by: ["Assets:Investments:*"]
    - name: Family Vacation
      icon: beach
      type: recurring
      start_date: 2027-01
      end_date: 2029-12
      frequency: yearly
      monthly_allocation: 200000
      inflation_rate: 8
      priority: 2
      funded_by: ["Assets:Checking"]
`), "")
	if err != nil {
		t.Fatalf("Failed to load config with life goals: %v", err)
	}

	cfg := GetConfig()
	if len(cfg.Goals.Life) != 2 {
		t.Fatalf("len(cfg.Goals.Life) = %d, want 2", len(cfg.Goals.Life))
	}

	if cfg.Goals.Life[0].Type != "milestone" {
		t.Fatalf("first life goal type = %q, want milestone", cfg.Goals.Life[0].Type)
	}
	if cfg.Goals.Life[0].TargetDate != "2030-06" {
		t.Fatalf("first life goal target date = %q, want 2030-06", cfg.Goals.Life[0].TargetDate)
	}
	if cfg.Goals.Life[1].Frequency != "yearly" {
		t.Fatalf("second life goal frequency = %q, want yearly", cfg.Goals.Life[1].Frequency)
	}
	if cfg.Goals.Life[1].InflationRate == nil || *cfg.Goals.Life[1].InflationRate != 8 {
		t.Fatalf("second life goal inflation rate = %v, want 8", cfg.Goals.Life[1].InflationRate)
	}
}
