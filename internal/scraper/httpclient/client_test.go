package httpclient

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nextxm/paisa/internal/config"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureHook struct {
	entries []*log.Entry
}

func (h *captureHook) Levels() []log.Level {
	return log.AllLevels
}

func (h *captureHook) Fire(entry *log.Entry) error {
	copyData := make(log.Fields, len(entry.Data))
	for key, value := range entry.Data {
		copyData[key] = value
	}
	h.entries = append(h.entries, &log.Entry{Message: entry.Message, Level: entry.Level, Data: copyData})
	return nil
}

func TestDo_LogsProviderHTTPWhenEnabled(t *testing.T) {
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db\nprovider_debug_http: true\n"), ""))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "hello=world", r.URL.RawQuery)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "ping", string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	hook := &captureHook{}
	logger := log.StandardLogger()
	logger.AddHook(hook)
	defer func() {
		hooks := logger.Hooks[log.InfoLevel]
		for i, existing := range hooks {
			if existing == hook {
				logger.Hooks[log.InfoLevel] = append(hooks[:i], hooks[i+1:]...)
				break
			}
		}
	}()

	req, err := http.NewRequest(http.MethodPost, server.URL+"?hello=world", io.NopCloser(bytes.NewReader([]byte("ping"))))
	require.NoError(t, err)
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader([]byte("ping"))), nil
	}
	req.Header.Set("Authorization", "secret")

	resp, err := Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Len(t, hook.entries, 2)
	assert.Equal(t, "Provider HTTP request", hook.entries[0].Message)
	assert.Equal(t, "ping", hook.entries[0].Data["request_body"])
	assert.Equal(t, []string{"[REDACTED]"}, hook.entries[0].Data["headers"].(map[string][]string)["Authorization"])
	assert.Equal(t, "Provider HTTP response", hook.entries[1].Message)
	assert.Equal(t, "{\"ok\":true}", hook.entries[1].Data["response_body"])
	assert.Equal(t, 200, hook.entries[1].Data["status_code"])
}

func TestDo_DoesNotLogProviderHTTPWhenDisabled(t *testing.T) {
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db\nprovider_debug_http: false\n"), ""))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	hook := &captureHook{}
	logger := log.StandardLogger()
	logger.AddHook(hook)
	defer func() {
		hooks := logger.Hooks[log.InfoLevel]
		for i, existing := range hooks {
			if existing == hook {
				logger.Hooks[log.InfoLevel] = append(hooks[:i], hooks[i+1:]...)
				break
			}
		}
	}()

	resp, err := Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Empty(t, hook.entries)
}
