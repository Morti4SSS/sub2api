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
