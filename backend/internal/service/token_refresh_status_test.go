//go:build unit

package service

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildTokenRefreshStatusSuccessUsesExpiryPolicyWindow(t *testing.T) {
	attemptedAt := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	expiresAt := attemptedAt.Add(2 * time.Hour)
	account := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"expires_at": expiresAt.Unix(),
		},
	}

	status := BuildTokenRefreshStatus(account, TokenRefreshTriggerBackground, nil, attemptedAt, 30*time.Minute, 5*time.Minute)

	require.Equal(t, TokenRefreshResultSuccess, status.LastResult)
	require.Equal(t, TokenRefreshTriggerBackground, status.Trigger)
	require.Equal(t, attemptedAt, status.LastAttemptAt)
	require.NotNil(t, status.NextWindowStart)
	require.NotNil(t, status.NextWindowEnd)
	require.Equal(t, expiresAt.Add(-30*time.Minute), *status.NextWindowStart)
	require.Equal(t, expiresAt.Add(-25*time.Minute), *status.NextWindowEnd)
	require.Empty(t, status.Error)
}

func TestBuildTokenRefreshStatusFailureRedactsErrorAndUsesNextScanWindow(t *testing.T) {
	attemptedAt := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	status := BuildTokenRefreshStatus(
		&Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth},
		TokenRefreshTriggerManual,
		errors.New("invalid_grant refresh_token=relay-secret"),
		attemptedAt,
		30*time.Minute,
		5*time.Minute,
	)

	require.Equal(t, TokenRefreshResultFailed, status.LastResult)
	require.Equal(t, TokenRefreshTriggerManual, status.Trigger)
	require.Contains(t, status.Error, "refresh_token=***")
	require.NotContains(t, status.Error, "relay-secret")
	require.Equal(t, attemptedAt.Add(5*time.Minute), *status.NextWindowStart)
	require.Equal(t, attemptedAt.Add(10*time.Minute), *status.NextWindowEnd)
}

func TestAccountTokenRefreshStatusRoundTripUsesSingleExtraOwner(t *testing.T) {
	account := &Account{Extra: map[string]any{}}
	status := TokenRefreshStatus{
		LastAttemptAt: time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
		LastResult:    TokenRefreshResultSuccess,
		Trigger:       TokenRefreshTriggerManual,
	}

	ApplyTokenRefreshStatus(account, status)
	parsed := account.GetTokenRefreshStatus()

	require.NotNil(t, parsed)
	require.Equal(t, status.LastAttemptAt, parsed.LastAttemptAt)
	require.Equal(t, status.LastResult, parsed.LastResult)
	require.Equal(t, status.Trigger, parsed.Trigger)
	require.Contains(t, account.Extra, TokenRefreshStatusExtraKey)
}

func TestShouldTrackTokenRefreshStatusOnlyForMaintainedOAuthAccounts(t *testing.T) {
	require.True(t, ShouldTrackTokenRefreshStatus(&Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth}))
	require.True(t, ShouldTrackTokenRefreshStatus(&Account{Platform: PlatformAnthropic, Type: AccountTypeSetupToken}))
	require.True(t, ShouldTrackTokenRefreshStatus(&Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth}))
	require.False(t, ShouldTrackTokenRefreshStatus(&Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}))
	require.False(t, ShouldTrackTokenRefreshStatus(&Account{Platform: PlatformGemini, Type: AccountTypeOAuth}))
}
