package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayHandleFailoverExhaustedClassifiesUpstreamModelNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, EndpointMessages, nil)

	h := &GatewayHandler{}
	h.handleFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode:   http.StatusNotFound,
		ResponseBody: []byte(`{"error":{"code":"model_not_found","message":"model glm-5.2 not found; api_key=relay-secret"}}`),
	}, service.PlatformAnthropic, false)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	var payload struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "upstream_model_not_found", payload.Error.Type)
	require.Contains(t, payload.Error.Message, "404")
	require.Contains(t, payload.Error.Message, "model glm-5.2 not found")
	require.NotContains(t, payload.Error.Message, "relay-secret")
}

func TestOpenAIHandleFailoverExhaustedClassifiesUpstreamModelNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, EndpointResponses, nil)

	h := &OpenAIGatewayHandler{}
	h.handleFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode:   http.StatusNotFound,
		ResponseBody: []byte(`{"error":{"code":"model_not_found","message":"unknown model gpt-relay-pro; token=relay-secret"}}`),
	}, false)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	var payload struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "upstream_model_not_found", payload.Error.Type)
	require.Contains(t, payload.Error.Message, "404")
	require.Contains(t, payload.Error.Message, "unknown model gpt-relay-pro")
	require.NotContains(t, payload.Error.Message, "relay-secret")
}

func TestGatewayHandleFailoverExhaustedKeepsEndpoint404AsUpstreamError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, EndpointMessages, nil)

	h := &GatewayHandler{}
	h.handleFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode:   http.StatusNotFound,
		ResponseBody: []byte(`{"error":{"message":"endpoint /v1/messages not found"}}`),
	}, service.PlatformAnthropic, false)

	var payload struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "upstream_error", payload.Error.Type)
}
