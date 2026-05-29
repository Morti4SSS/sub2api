package admin

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildTokenRefreshState_BackgroundFailure(t *testing.T) {
	handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &config.Config{
		TokenRefresh: config.TokenRefreshConfig{
			CheckIntervalMinutes:     5,
			RefreshBeforeExpiryHours: 12,
		},
	})
	expiresAt := time.Now().Add(6 * time.Hour).Unix()
	account := &service.Account{
		ID:       1,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"expires_at": expiresAt,
			"_token_refresh_background_last_checked_at":        "2026-05-29T04:20:39Z",
			"_token_refresh_background_last_error_at":          "2026-05-29T04:20:39Z",
			"_token_refresh_background_last_error":             "invalid_grant",
			"_token_refresh_background_refresh_before_seconds": int64(1800),
		},
	}

	state := handler.buildTokenRefreshState(account)

	require.NotNil(t, state)
	require.Equal(t, int64(5*60), state.CheckIntervalSeconds)
	require.Equal(t, int64(12*60*60), state.RefreshBeforeSeconds)
	require.NotNil(t, state.AccessTokenExpiresAt)
	require.Equal(t, expiresAt, state.AccessTokenExpiresAt.Unix())
	require.NotNil(t, state.RefreshWindowStartsAt)
	require.NotNil(t, state.BackgroundLastCheckedAt)
	require.NotNil(t, state.BackgroundLastErrorAt)
	require.Equal(t, "invalid_grant", state.BackgroundLastError)
	require.Equal(t, "failed", state.RiskLevel)
}

func TestBuildTokenRefreshState_OpenAIResponseMissingRTIsHighRisk(t *testing.T) {
	handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &config.Config{
		TokenRefresh: config.TokenRefreshConfig{
			CheckIntervalMinutes:     5,
			RefreshBeforeExpiryHours: 12,
		},
	})
	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	account := &service.Account{
		ID:       2,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"expires_at":                     expiresAt,
			"_token_refresh_last_source":     "manual_single",
			"_token_refresh_last_result":     "success",
			"_token_refresh_last_attempt_at": "2026-05-19T10:00:00Z",
			"_token_refresh_last_success_at": "2026-05-19T10:00:00Z",
			"_token_refresh_at_status":       "refreshed",
			"_token_refresh_rt_status":       "response_missing_preserved_old",
		},
	}

	state := handler.buildTokenRefreshState(account)

	require.NotNil(t, state)
	require.Equal(t, "manual_single", state.LastSource)
	require.Equal(t, "success", state.LastResult)
	require.Equal(t, "refreshed", state.AccessTokenStatus)
	require.Equal(t, "response_missing_preserved_old", state.RefreshTokenStatus)
	require.Equal(t, "high_risk", state.RiskLevel)
	require.NotNil(t, state.RefreshWindowStartsAt)
	require.Equal(t, time.Unix(expiresAt, 0).Add(-12*time.Hour).Unix(), state.RefreshWindowStartsAt.Unix())
}
