package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// decodeErrorEnvelope reads the response body and returns the nested ErrorDetail.
func decodeErrorEnvelope(t *testing.T, rec *httptest.ResponseRecorder) ErrorDetail {
	t.Helper()
	var env struct {
		Error ErrorDetail `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&env))
	return env.Error
}

func TestErrResponse_Shape(t *testing.T) {
	h := ErrResponse(ErrCodeInvalidRequest, "bad input")
	raw, err := json.Marshal(h)
	require.NoError(t, err)

	var env struct {
		Error ErrorDetail `json:"error"`
	}
	require.NoError(t, json.Unmarshal(raw, &env))

	assert.Equal(t, ErrCodeInvalidRequest, env.Error.Code)
	assert.Equal(t, "bad input", env.Error.Message)
}

func TestErrResponse_NoExtraKeys(t *testing.T) {
	h := ErrResponse(ErrCodeInternalError, "oops")
	raw, _ := json.Marshal(h)

	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &top))

	// Only "error" key at the top level.
	assert.Len(t, top, 1)
	_, hasError := top["error"]
	assert.True(t, hasError)
}

func TestConvenienceHelpers(t *testing.T) {
	cases := []struct {
		name     string
		fn       func(string) gin.H
		wantCode ErrorCode
	}{
		{"ErrInvalidRequest", ErrInvalidRequest, ErrCodeInvalidRequest},
		{"ErrInternal", ErrInternal, ErrCodeInternalError},
		{"ErrUnauthorized", ErrUnauthorized, ErrCodeUnauthorized},
		{"ErrTooManyRequests", ErrTooManyRequests, ErrCodeTooManyRequests},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := tc.fn("some message")
			raw, _ := json.Marshal(h)
			var env struct {
				Error ErrorDetail `json:"error"`
			}
			require.NoError(t, json.Unmarshal(raw, &env))
			assert.Equal(t, tc.wantCode, env.Error.Code)
			assert.Equal(t, "some message", env.Error.Message)
		})
	}
}

func TestRespondError_WritesJSON(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "test error")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
	assert.Equal(t, "test error", detail.Message)
}

func TestAbortWithError_StopsChain(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		AbortWithError(c, http.StatusUnauthorized, ErrCodeUnauthorized, "not allowed")
	}, func(c *gin.Context) {
		// This handler must NOT be reached.
		t.Error("second handler must not be called after AbortWithError")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeUnauthorized, detail.Code)
}

func TestBindJSONOrError_ValidInput(t *testing.T) {
	type Payload struct {
		Name string `json:"name"`
	}
	var called bool

	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		var p Payload
		if !BindJSONOrError(c, &p) {
			return
		}
		called = true
		c.JSON(http.StatusOK, gin.H{"name": p.Name})
	})

	body := strings.NewReader(`{"name":"alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called)
}

func TestBindJSONOrError_InvalidInput(t *testing.T) {
	type Payload struct {
		Name string `json:"name" binding:"required"`
	}

	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		var p Payload
		if !BindJSONOrError(c, &p) {
			return
		}
		t.Error("handler must not proceed after bind failure")
	})

	// Send malformed JSON.
	body := strings.NewReader(`{not json}`)
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

func TestErrorCodeConstants_Values(t *testing.T) {
	assert.Equal(t, ErrorCode("INVALID_REQUEST"), ErrCodeInvalidRequest)
	assert.Equal(t, ErrorCode("INTERNAL_ERROR"), ErrCodeInternalError)
	assert.Equal(t, ErrorCode("UNAUTHORIZED"), ErrCodeUnauthorized)
	assert.Equal(t, ErrorCode("TOO_MANY_REQUESTS"), ErrCodeTooManyRequests)
}
