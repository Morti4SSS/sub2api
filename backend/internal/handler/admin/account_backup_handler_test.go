package admin

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountBackupHandlerListAndDownload(t *testing.T) {
	router, backupService := setupAccountBackupRouter(t)
	snapshot, err := backupService.CreateSnapshotNow(context.Background(), "manual_refresh")
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/backups", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var listResp struct {
		Code int `json:"code"`
		Data struct {
			Items []service.AccountBackupSnapshot `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listResp))
	require.Equal(t, 0, listResp.Code)
	require.Len(t, listResp.Data.Items, 1)
	require.Equal(t, snapshot.Name, listResp.Data.Items[0].Name)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/backups/"+snapshot.Name, nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Header().Get("Content-Disposition"), snapshot.Name)
	require.NotContains(t, rec.Body.String(), "rt-secret")
	require.Contains(t, rec.Body.String(), "encrypted_key")
}

func TestAccountBackupHandlerRejectsInvalidSnapshotName(t *testing.T) {
	router, _ := setupAccountBackupRouter(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/backups/../secret.json.enc", nil)
	router.ServeHTTP(rec, req)
	require.NotEqual(t, http.StatusOK, rec.Code)
}

func setupAccountBackupRouter(t *testing.T) (*gin.Engine, *service.AccountBackupService) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	backupService := service.NewAccountBackupService(service.AccountBackupConfig{
		Enabled:      true,
		Dir:          t.TempDir(),
		PublicKeyPEM: generateAccountBackupHandlerPublicKey(t),
		RetainCount:  5,
	}, func(ctx context.Context) ([]byte, error) {
		return []byte(`{"accounts":[{"credentials":{"refresh_token":"rt-secret"}}]}`), nil
	})
	t.Cleanup(backupService.Stop)

	handler := NewBackupHandler(nil, nil, backupService)
	router.GET("/api/v1/admin/accounts/backups", handler.ListAccountBackups)
	router.GET("/api/v1/admin/accounts/backups/:name", handler.DownloadAccountBackup)
	return router, backupService
}

func generateAccountBackupHandlerPublicKey(t *testing.T) []byte {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
}

func TestAccountBackupHandlerRejectsPathSeparators(t *testing.T) {
	router, _ := setupAccountBackupRouter(t)

	for _, name := range []string{"..%2fsecret.json.enc", strings.ReplaceAll("..\\secret.json.enc", "\\", "%5c")} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/backups/"+name, nil)
		router.ServeHTTP(rec, req)
		require.NotEqual(t, http.StatusOK, rec.Code)
	}
}
