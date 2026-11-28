package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ananthakumaran/paisa/web"
	"github.com/stretchr/testify/require"
)

func TestUIRoutingPrimaryAndV1(t *testing.T) {
	origIndex := web.Index
	origIndexV1 := web.IndexV1
	t.Cleanup(func() {
		web.Index = origIndex
		web.IndexV1 = origIndexV1
	})

	web.Index = "<html><body>primary-ui</body></html>"
	web.IndexV1 = "<html><body>v1-ui</body></html>"

	db := openTestDB(t)
	router := Build(db, false)

	cases := []struct {
		path     string
		expected string
	}{
		{path: "/", expected: web.Index},
		{path: "/transactions", expected: web.Index},
		{path: "/v2", expected: web.Index},
		{path: "/v1", expected: web.IndexV1},
		{path: "/v1/", expected: web.IndexV1},
		{path: "/v1/assets/networth", expected: web.IndexV1},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "path=%s", tc.path)
		require.Equal(t, tc.expected, w.Body.String(), "path=%s", tc.path)
	}
}
