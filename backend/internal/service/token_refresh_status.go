package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
)

const (
	TokenRefreshStatusExtraKey = "token_refresh_status"

	TokenRefreshResultSuccess = "success"
	TokenRefreshResultFailed  = "failed"

	TokenRefreshTriggerManual     = "manual"
	TokenRefreshTriggerBackground = "background"

	DefaultTokenRefreshCheckInterval = 5 * time.Minute
	DefaultTokenRefreshBeforeExpiry  = 30 * time.Minute
)

// TokenRefreshStatus is the sole persisted refresh audit contract. It records
// real attempts only; token presence never implies a successful refresh.
type TokenRefreshStatus struct {
	LastAttemptAt   time.Time  `json:"last_attempt_at"`
	LastResult      string     `json:"last_result"`
	Trigger         string     `json:"trigger"`
	Error           string     `json:"error,omitempty"`
	NextWindowStart *time.Time `json:"next_window_start,omitempty"`
	NextWindowEnd   *time.Time `json:"next_window_end,omitempty"`
}

// ShouldTrackTokenRefreshStatus limits this custom audit to the maintained
// Claude/OpenAI OAuth and Setup Token account types.
func ShouldTrackTokenRefreshStatus(account *Account) bool {
	if account == nil || (account.Type != AccountTypeOAuth && account.Type != AccountTypeSetupToken) {
		return false
	}
	return account.Platform == PlatformAnthropic || account.Platform == PlatformOpenAI
}

// BuildTokenRefreshStatus creates one sanitized audit record and estimates the
// next background scan window from the configured policy.
func BuildTokenRefreshStatus(
	account *Account,
	trigger string,
	refreshErr error,
	attemptedAt time.Time,
	refreshWindow time.Duration,
	checkInterval time.Duration,
) TokenRefreshStatus {
	if attemptedAt.IsZero() {
		attemptedAt = time.Now()
	}
	attemptedAt = attemptedAt.UTC()
	if checkInterval <= 0 {
		checkInterval = DefaultTokenRefreshCheckInterval
	}
	if refreshWindow < 0 {
		refreshWindow = DefaultTokenRefreshBeforeExpiry
	}

	status := TokenRefreshStatus{
		LastAttemptAt: attemptedAt,
		LastResult:    TokenRefreshResultSuccess,
		Trigger:       trigger,
	}
	if refreshErr != nil {
		status.LastResult = TokenRefreshResultFailed
		status.Error = truncateString(logredact.RedactText(
			refreshErr.Error(), "token", "api_key", "authorization", "cookie",
		), 220)
	}

	if refreshErr == nil && account != nil {
		if expiresAt := account.GetCredentialAsTime("expires_at"); expiresAt != nil {
			start := expiresAt.UTC().Add(-refreshWindow)
			if start.After(attemptedAt) {
				end := start.Add(checkInterval)
				expiresUTC := expiresAt.UTC()
				if end.After(expiresUTC) {
					end = expiresUTC
				}
				status.NextWindowStart = &start
				status.NextWindowEnd = &end
				return status
			}
		}
	}

	start := attemptedAt.Add(checkInterval)
	end := start.Add(checkInterval)
	status.NextWindowStart = &start
	status.NextWindowEnd = &end
	return status
}

// TokenRefreshStatusExtraUpdate builds the atomic JSONB update owned by
// extra.token_refresh_status.
func TokenRefreshStatusExtraUpdate(status TokenRefreshStatus) map[string]any {
	return map[string]any{TokenRefreshStatusExtraKey: status}
}

// ApplyTokenRefreshStatus updates an in-memory account after persistence so
// the current response and scheduler snapshot can expose the same audit.
func ApplyTokenRefreshStatus(account *Account, status TokenRefreshStatus) {
	if account == nil {
		return
	}
	updates := TokenRefreshStatusExtraUpdate(status)
	value, ok := updates[TokenRefreshStatusExtraKey]
	if !ok {
		return
	}
	if account.Extra == nil {
		account.Extra = make(map[string]any)
	}
	account.Extra[TokenRefreshStatusExtraKey] = value
}

// GetTokenRefreshStatus parses only the canonical extra owner and rejects
// incomplete or unknown result/trigger values.
func (a *Account) GetTokenRefreshStatus() *TokenRefreshStatus {
	if a == nil || a.Extra == nil {
		return nil
	}
	raw, ok := a.Extra[TokenRefreshStatusExtraKey]
	if !ok || raw == nil {
		return nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var status TokenRefreshStatus
	if err := json.Unmarshal(encoded, &status); err != nil {
		return nil
	}
	status.LastResult = strings.TrimSpace(status.LastResult)
	status.Trigger = strings.TrimSpace(status.Trigger)
	if status.LastAttemptAt.IsZero() ||
		(status.LastResult != TokenRefreshResultSuccess && status.LastResult != TokenRefreshResultFailed) ||
		(status.Trigger != TokenRefreshTriggerManual && status.Trigger != TokenRefreshTriggerBackground) {
		return nil
	}
	status.Error = truncateString(logredact.RedactText(
		status.Error, "token", "api_key", "authorization", "cookie",
	), 220)
	return &status
}
