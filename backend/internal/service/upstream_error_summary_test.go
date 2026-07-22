package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildFailoverExhaustedClientMessageIncludesUpstreamSummary(t *testing.T) {
	err := &UpstreamFailoverError{
		StatusCode:   http.StatusForbidden,
		ResponseBody: []byte(`{"error":{"message":"model unavailable for relay account"}}`),
	}

	got := BuildFailoverExhaustedClientMessage("Service temporarily unavailable", err)
	require.Equal(t, "Service temporarily unavailable. Last upstream error: 403 model unavailable for relay account", got)
}

func TestBuildFailoverExhaustedClientMessageRedactsSecrets(t *testing.T) {
	err := &UpstreamFailoverError{
		StatusCode:   http.StatusUnauthorized,
		ResponseBody: []byte(`{"error":{"message":"bad key sk-test-secret and refresh_token=rt-secret"}}`),
	}

	got := BuildFailoverExhaustedClientMessage("Service temporarily unavailable", err)
	require.Contains(t, got, "401")
	require.NotContains(t, got, "sk-test-secret")
	require.NotContains(t, got, "rt-secret")
}

func TestBuildFailoverExhaustedClientMessageRedactsBearerToken(t *testing.T) {
	err := &UpstreamFailoverError{
		StatusCode:   http.StatusUnauthorized,
		ResponseBody: []byte(`{"error":{"message":"authorization: Bearer relay-secret"}}`),
	}

	got := BuildFailoverExhaustedClientMessage("Service temporarily unavailable", err)
	require.NotContains(t, got, "relay-secret")
	require.Contains(t, got, "authorization=***")
}

func TestDescribeUpstreamFailoverErrorClassifiesModelNotFound(t *testing.T) {
	details := DescribeUpstreamFailoverError(&UpstreamFailoverError{
		StatusCode:   http.StatusNotFound,
		ResponseBody: []byte(`{"error":{"code":"model_not_found","message":"model glm-5.2 not found; api_key=relay-secret"}}`),
	})

	require.Equal(t, "upstream_model_not_found", details.Code)
	require.Equal(t, http.StatusNotFound, details.StatusCode)
	require.Contains(t, details.Reason, "model glm-5.2 not found")
	require.NotContains(t, details.Reason, "relay-secret")
}

func TestDescribeUpstreamFailoverErrorKeepsEndpoint404AsUpstreamError(t *testing.T) {
	details := DescribeUpstreamFailoverError(&UpstreamFailoverError{
		StatusCode:   http.StatusNotFound,
		ResponseBody: []byte(`{"error":{"message":"endpoint /v1/responses not found"}}`),
	})

	require.Equal(t, "upstream_error", details.Code)
	require.Equal(t, http.StatusNotFound, details.StatusCode)
	require.Equal(t, "endpoint /v1/responses not found", details.Reason)
}
