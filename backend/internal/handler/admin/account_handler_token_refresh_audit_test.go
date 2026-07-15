//go:build unit

package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type manualRefreshAuditAdminService struct {
	*stubAdminService
	account      *service.Account
	extraUpdates map[string]any
}

func (s *manualRefreshAuditAdminService) UpdateAccount(_ context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error) {
	updated := *s.account
	updated.ID = id
	updated.Credentials = input.Credentials
	s.account = &updated
	return s.account, nil
}

func (s *manualRefreshAuditAdminService) UpdateAccountExtra(_ context.Context, _ int64, updates map[string]any) error {
	s.extraUpdates = updates
	return nil
}

type manualRefreshAuditClaudeClient struct {
	response *oauth.TokenResponse
	err      error
}

func (c *manualRefreshAuditClaudeClient) GetOrganizationUUID(context.Context, string, string) (string, error) {
	return "", errors.New("not implemented")
}

func (c *manualRefreshAuditClaudeClient) GetAuthorizationCode(context.Context, string, string, string, string, string, string) (string, error) {
	return "", errors.New("not implemented")
}

func (c *manualRefreshAuditClaudeClient) ExchangeCodeForToken(context.Context, string, string, string, string, bool) (*oauth.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *manualRefreshAuditClaudeClient) RefreshToken(context.Context, string, string) (*oauth.TokenResponse, error) {
	return c.response, c.err
}

func TestAccountHandlerRefreshSingleAccountRecordsManualSuccess(t *testing.T) {
	account := &service.Account{
		ID:       501,
		Platform: service.PlatformAnthropic,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"refresh_token": "old-refresh",
		},
		Extra: map[string]any{},
	}
	adminSvc := &manualRefreshAuditAdminService{stubAdminService: newStubAdminService(), account: account}
	oauthSvc := service.NewOAuthService(nil, &manualRefreshAuditClaudeClient{response: &oauth.TokenResponse{
		AccessToken: "new-access", RefreshToken: "new-refresh", TokenType: "Bearer", ExpiresIn: 7200,
	}})
	defer oauthSvc.Stop()
	h := &AccountHandler{
		adminService: adminSvc,
		oauthService: oauthSvc,
		tokenRefreshConfig: &config.TokenRefreshConfig{
			CheckIntervalMinutes: 5, RefreshBeforeExpiryHours: 0.5,
		},
	}

	updated, _, err := h.refreshSingleAccount(context.Background(), account)

	require.NoError(t, err)
	audit := updated.GetTokenRefreshStatus()
	require.NotNil(t, audit)
	require.Equal(t, service.TokenRefreshResultSuccess, audit.LastResult)
	require.Equal(t, service.TokenRefreshTriggerManual, audit.Trigger)
	require.Contains(t, adminSvc.extraUpdates, service.TokenRefreshStatusExtraKey)
}

func TestAccountHandlerRefreshSingleAccountRecordsManualFailure(t *testing.T) {
	account := &service.Account{
		ID:       502,
		Platform: service.PlatformAnthropic,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"refresh_token": "old-refresh",
		},
		Extra: map[string]any{},
	}
	adminSvc := &manualRefreshAuditAdminService{stubAdminService: newStubAdminService(), account: account}
	oauthSvc := service.NewOAuthService(nil, &manualRefreshAuditClaudeClient{
		err: errors.New("invalid_grant refresh_token=relay-secret"),
	})
	defer oauthSvc.Stop()
	h := &AccountHandler{
		adminService: adminSvc,
		oauthService: oauthSvc,
		tokenRefreshConfig: &config.TokenRefreshConfig{
			CheckIntervalMinutes: 5, RefreshBeforeExpiryHours: 0.5,
		},
	}

	_, _, err := h.refreshSingleAccount(context.Background(), account)

	require.Error(t, err)
	audit := account.GetTokenRefreshStatus()
	require.NotNil(t, audit)
	require.Equal(t, service.TokenRefreshResultFailed, audit.LastResult)
	require.Equal(t, service.TokenRefreshTriggerManual, audit.Trigger)
	require.Contains(t, audit.Error, "refresh_token=***")
	require.NotContains(t, audit.Error, "relay-secret")
}
