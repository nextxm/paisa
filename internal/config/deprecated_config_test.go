package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig_IgnoresDeprecatedFireflyKeys(t *testing.T) {
	yamlContent := []byte(`
journal_path: /tmp/main.ledger
db_path: /tmp/paisa.db
firefly:
  url: https://firefly.example.com
  token: secret
  ignore_accounts:
    - Cash
labs:
  firefly_reconcile: true
`)

	err := LoadConfig(yamlContent, "/tmp/paisa.yaml")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"firefly", "labs.firefly_reconcile"}, GetDeprecatedConfigKeys())
}
