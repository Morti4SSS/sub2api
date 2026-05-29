package service

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type recordingCredentialsChangeNotifier struct {
	mu      sync.Mutex
	reasons []string
}

func (n *recordingCredentialsChangeNotifier) NotifyCredentialsChanged(ctx context.Context, reason string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.reasons = append(n.reasons, reason)
}

func (n *recordingCredentialsChangeNotifier) Reasons() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string(nil), n.reasons...)
}

func TestTokenRefreshPostRefreshActionsNotifiesAccountBackup(t *testing.T) {
	notifier := &recordingCredentialsChangeNotifier{}
	svc := &TokenRefreshService{}
	svc.SetAccountBackupNotifier(notifier)

	svc.postRefreshActions(context.Background(), &Account{
		ID:          1,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Credentials: map[string]any{"access_token": "access-token"},
	})

	require.Equal(t, []string{"background_token_refresh"}, notifier.Reasons())
}
