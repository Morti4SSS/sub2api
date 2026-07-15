//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayServiceHandleErrorResponseWritesCanonicalModelNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	response := &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"model_not_found","message":"model glm-5.2 not found; api_key=relay-secret"}}`)),
	}
	account := &Account{ID: 401, Name: "Claude relay", Platform: PlatformAnthropic, Type: AccountTypeAPIKey}

	result, err := (&GatewayService{}).handleErrorResponse(context.Background(), response, c, account, "claude-opus-4-8")

	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusBadGateway, recorder.Code)
	var payload struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, UpstreamModelNotFoundErrorCode, payload.Error.Type)
	require.Contains(t, payload.Error.Message, "404")
	require.Contains(t, payload.Error.Message, "model glm-5.2 not found")
	require.NotContains(t, payload.Error.Message, "relay-secret")
}

func TestOpenAIGatewayServiceHandlePassthroughErrorWritesCanonicalModelNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	response := &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"model_not_found","message":"unknown model gpt-relay-pro; token=relay-secret"}}`)),
	}
	account := &Account{ID: 402, Name: "OpenAI relay", Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	err := (&OpenAIGatewayService{}).handleErrorResponsePassthrough(context.Background(), response, c, account, nil)

	require.Error(t, err)
	require.Equal(t, http.StatusBadGateway, recorder.Code)
	var payload struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, UpstreamModelNotFoundErrorCode, payload.Error.Type)
	require.Contains(t, payload.Error.Message, "404")
	require.Contains(t, payload.Error.Message, "unknown model gpt-relay-pro")
	require.NotContains(t, payload.Error.Message, "relay-secret")
}
