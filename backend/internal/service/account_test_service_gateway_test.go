//go:build unit

package service

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestNormalizeAccountTestPromptRejectsProbeWords(t *testing.T) {
	for _, prompt := range []string{"hi", "hello", "ping", "test", " HI "} {
		_, err := normalizeAccountTestPrompt(prompt, defaultClaudeTestPrompt)
		require.Error(t, err, prompt)
	}

	normalized, err := normalizeAccountTestPrompt("  Explain why this API connection is working.  ", defaultClaudeTestPrompt)
	require.NoError(t, err)
	require.Equal(t, "Explain why this API connection is working.", normalized)
}

func TestAccountTestServiceClaudeUsesFixedProductionGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	account := &Account{
		ID:          301,
		Name:        "Claude relay A",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Status:      "inactive",
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "test-key",
			"base_url": "https://relay.example.com",
		},
		Extra: map[string]any{
			"anthropic_passthrough": true,
			"claude_code_routes": []any{map[string]any{
				"shell_model": "claude-opus-4-8", "upstream_model": "glm-5.2",
				"thinking": map[string]any{
					"mode": "levels", "target_field": "reasoning_effort", "levels": []any{"off", "on"},
				},
			}},
		},
	}
	repo := &openAIAccountTestRepo{mockAccountRepoForGemini: mockAccountRepoForGemini{
		accountsByID: map[int64]*Account{account.ID: account},
	}}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"message_start","message":{"id":"msg_test","model":"glm-5.2"}}`,
			"",
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"connected"}}`,
			"",
			`data: {"type":"message_stop"}`,
			"",
		}, "\n"))),
	}}
	cfg := &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}}
	gateway := &GatewayService{
		accountRepo:         repo,
		httpUpstream:        upstream,
		cfg:                 cfg,
		tlsFPProfileService: &TLSFingerprintProfileService{},
	}
	svc := &AccountTestService{accountRepo: repo, gatewayService: gateway}
	c, recorder := newTestContext()

	err := svc.TestAccountConnection(
		c,
		account.ID,
		"claude-opus-4-8",
		"Explain why this API connection is working.",
		AccountTestModeDefault,
	)

	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, claude.DefaultHeaders["User-Agent"], upstream.lastReq.Header.Get("User-Agent"))
	require.Equal(t, "glm-5.2", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "on", gjson.GetBytes(upstream.lastBody, "reasoning_effort").String())
	require.Equal(t, "Explain why this API connection is working.", gjson.GetBytes(upstream.lastBody, "messages.0.content.0.text").String())
	require.Contains(t, recorder.Body.String(), `"type":"diagnostics"`)
	require.Contains(t, recorder.Body.String(), `"account_id":301`)
	require.Contains(t, recorder.Body.String(), `"client_identity":"claude_code_cli"`)
	require.Contains(t, recorder.Body.String(), `"requested_model":"claude-opus-4-8"`)
	require.Contains(t, recorder.Body.String(), `"upstream_model":"glm-5.2"`)
	require.Contains(t, recorder.Body.String(), `"passthrough":true`)
	require.Contains(t, recorder.Body.String(), `"thinking_target_value":"on"`)
	require.Contains(t, recorder.Body.String(), `"upstream_http_status":200`)
	require.Contains(t, recorder.Body.String(), `"success":true`)
}

func TestAccountTestServiceOpenAIUsesFixedProductionGatewayAndCodexIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	account := &Account{
		ID:          302,
		Name:        "OpenAI relay A",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      "inactive",
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "test-key",
			"base_url": "https://openai-relay.example.com/v1",
		},
		Extra: map[string]any{"openai_passthrough": true},
	}
	repo := &openAIAccountTestRepo{mockAccountRepoForGemini: mockAccountRepoForGemini{
		accountsByID: map[int64]*Account{account.ID: account},
	}}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"response.output_text.delta","delta":"connected"}`,
			"",
			`data: {"type":"response.completed","response":{"id":"resp_test"}}`,
			"",
		}, "\n"))),
	}}
	cfg := &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}}
	openAIGateway := &OpenAIGatewayService{
		accountRepo:      repo,
		httpUpstream:     upstream,
		cfg:              cfg,
		codexDetector:    NewOpenAICodexClientRestrictionDetector(cfg),
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
	}
	svc := &AccountTestService{accountRepo: repo, openAIGatewayService: openAIGateway}
	c, recorder := newTestContext()

	err := svc.TestAccountConnection(c, account.ID, "gpt-5.4", "", AccountTestModeDefault)

	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, codexCLIUserAgent, upstream.lastReq.Header.Get("User-Agent"))
	require.Equal(t, "codex_cli_rs", upstream.lastReq.Header.Get("Originator"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(upstream.lastBody, &body))
	input := body["input"].([]any)
	content := input[0].(map[string]any)["content"].([]any)
	prompt := content[0].(map[string]any)["text"].(string)
	require.Equal(t, defaultOpenAITextTestPrompt, prompt)
	require.NotContains(t, []string{"hi", "hello", "ping", "test"}, strings.ToLower(prompt))
	require.Contains(t, recorder.Body.String(), `"type":"diagnostics"`)
	require.Contains(t, recorder.Body.String(), `"account_id":302`)
	require.Contains(t, recorder.Body.String(), `"client_identity":"codex_cli"`)
	require.Contains(t, recorder.Body.String(), `"passthrough":true`)
	require.Contains(t, recorder.Body.String(), `"upstream_http_status":200`)
	require.Contains(t, recorder.Body.String(), `"success":true`)
}

func TestAccountTestServiceClaudeReportsSanitizedUpstreamModelNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	account := &Account{
		ID:          303,
		Name:        "Claude relay failure",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "test-key", "base_url": "https://relay.example.com"},
		Extra: map[string]any{
			"anthropic_passthrough": true,
			"claude_code_routes": []any{map[string]any{
				"shell_model": "claude-opus-4-8", "upstream_model": "glm-5.2",
			}},
		},
	}
	repo := &openAIAccountTestRepo{mockAccountRepoForGemini: mockAccountRepoForGemini{
		accountsByID: map[int64]*Account{account.ID: account},
	}}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"model_not_found","message":"model glm-5.2 not found; api_key=relay-secret"}}`)),
	}}
	gateway := &GatewayService{
		accountRepo:         repo,
		httpUpstream:        upstream,
		cfg:                 &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		tlsFPProfileService: &TLSFingerprintProfileService{},
	}
	svc := &AccountTestService{accountRepo: repo, gatewayService: gateway}
	c, recorder := newTestContext()

	err := svc.TestAccountConnection(c, account.ID, "claude-opus-4-8", "Explain whether this connection is working.", AccountTestModeDefault)

	require.Error(t, err)
	body := recorder.Body.String()
	require.Contains(t, body, `"type":"diagnostics"`)
	require.Contains(t, body, `"upstream_model":"glm-5.2"`)
	require.Contains(t, body, `"upstream_http_status":404`)
	require.Contains(t, body, `"upstream_error_code":"upstream_model_not_found"`)
	require.Contains(t, body, `"upstream_error_reason":"model glm-5.2 not found; api_key=***"`)
	require.NotContains(t, body, "relay-secret")
}

func TestAccountTestServiceOpenAIReportsSanitizedUpstreamError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	account := &Account{
		ID:          304,
		Name:        "OpenAI relay failure",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "test-key", "base_url": "https://openai-relay.example.com/v1"},
		Extra:       map[string]any{"openai_passthrough": true},
	}
	repo := &openAIAccountTestRepo{mockAccountRepoForGemini: mockAccountRepoForGemini{
		accountsByID: map[int64]*Account{account.ID: account},
	}}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"relay temporarily unavailable; token=relay-secret"}}`)),
	}}
	openAIGateway := &OpenAIGatewayService{
		accountRepo:      repo,
		httpUpstream:     upstream,
		cfg:              &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		codexDetector:    NewOpenAICodexClientRestrictionDetector(nil),
		openaiWSResolver: NewOpenAIWSProtocolResolver(nil),
		toolCorrector:    NewCodexToolCorrector(),
	}
	svc := &AccountTestService{accountRepo: repo, openAIGatewayService: openAIGateway}
	c, recorder := newTestContext()

	err := svc.TestAccountConnection(c, account.ID, "gpt-5.4", "Explain whether this connection is working.", AccountTestModeDefault)

	require.Error(t, err)
	body := recorder.Body.String()
	require.Contains(t, body, `"type":"diagnostics"`)
	require.Contains(t, body, `"upstream_model":"gpt-5.4"`)
	require.Contains(t, body, `"upstream_http_status":503`)
	require.Contains(t, body, `"upstream_error_code":"upstream_error"`)
	require.Contains(t, body, `"upstream_error_reason":"relay temporarily unavailable; token=***"`)
	require.NotContains(t, body, "relay-secret")
}
