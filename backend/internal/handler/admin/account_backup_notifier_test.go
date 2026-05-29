package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type accountBackupNotifierRecorder struct {
	mu      sync.Mutex
	reasons []string
}

func (r *accountBackupNotifierRecorder) NotifyCredentialsChanged(ctx context.Context, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reasons = append(r.reasons, reason)
}

func (r *accountBackupNotifierRecorder) Reasons() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.reasons...)
}

type accountBackupFailingAdminService struct {
	*stubAdminService
	failAllUpdates bool
}

func (s *accountBackupFailingAdminService) UpdateAccount(ctx context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error) {
	if s.failAllUpdates {
		return nil, errors.New("database error")
	}
	return s.stubAdminService.UpdateAccount(ctx, id, input)
}

func TestBatchUpdateCredentialsNotifiesAccountBackupAfterSuccessfulWrites(t *testing.T) {
	router, handler := setupAccountBackupNotifyRouter(&accountBackupFailingAdminService{stubAdminService: newStubAdminService()})
	notifier := &accountBackupNotifierRecorder{}
	handler.SetAccountBackupNotifier(notifier)

	body, _ := json.Marshal(BatchUpdateCredentialsRequest{
		AccountIDs: []int64{1, 2},
		Field:      "account_uuid",
		Value:      "test-uuid",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-update-credentials", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{"batch_update_credentials"}, notifier.Reasons())
}

func TestBatchUpdateCredentialsDoesNotNotifyAccountBackupWhenAllWritesFail(t *testing.T) {
	router, handler := setupAccountBackupNotifyRouter(&accountBackupFailingAdminService{
		stubAdminService: newStubAdminService(),
		failAllUpdates:   true,
	})
	notifier := &accountBackupNotifierRecorder{}
	handler.SetAccountBackupNotifier(notifier)

	body, _ := json.Marshal(BatchUpdateCredentialsRequest{
		AccountIDs: []int64{1, 2},
		Field:      "account_uuid",
		Value:      "test-uuid",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-update-credentials", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, notifier.Reasons())
}

func setupAccountBackupNotifyRouter(adminSvc service.AdminService) (*gin.Engine, *AccountHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/batch-update-credentials", handler.BatchUpdateCredentials)
	return router, handler
}
