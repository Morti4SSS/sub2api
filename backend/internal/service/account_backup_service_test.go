package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testAccountBackupEnvelope struct {
	EncryptedKey string `json:"encrypted_key"`
	Nonce        string `json:"nonce"`
	Ciphertext   string `json:"ciphertext"`
	Tag          string `json:"tag"`
}

func TestAccountBackupServiceCreateSnapshotEncryptsPayload(t *testing.T) {
	privateKey, publicKeyPEM := generateAccountBackupTestKey(t)
	dir := t.TempDir()
	plaintext := []byte(`{"accounts":[{"credentials":{"refresh_token":"rt-secret"}}]}`)

	service := NewAccountBackupService(AccountBackupConfig{
		Enabled:      true,
		Dir:          dir,
		PublicKeyPEM: publicKeyPEM,
		RetainCount:  5,
		Now:          func() time.Time { return time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC) },
	}, func(ctx context.Context) ([]byte, error) {
		return plaintext, nil
	})
	defer service.Stop()

	snapshot, err := service.CreateSnapshotNow(context.Background(), "manual_refresh")
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.True(t, strings.HasSuffix(snapshot.Name, ".json.enc"))

	raw, err := os.ReadFile(filepath.Join(dir, snapshot.Name))
	require.NoError(t, err)
	require.NotContains(t, string(raw), "rt-secret")
	require.Equal(t, plaintext, decryptAccountBackupTestEnvelope(t, privateKey, raw))
}

func TestAccountBackupServiceRetentionKeepsNewestSnapshots(t *testing.T) {
	_, publicKeyPEM := generateAccountBackupTestKey(t)
	dir := t.TempDir()
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)

	service := NewAccountBackupService(AccountBackupConfig{
		Enabled:      true,
		Dir:          dir,
		PublicKeyPEM: publicKeyPEM,
		RetainCount:  2,
		Now: func() time.Time {
			now = now.Add(time.Second)
			return now
		},
	}, func(ctx context.Context) ([]byte, error) {
		return []byte(`{"accounts":[]}`), nil
	})
	defer service.Stop()

	first, err := service.CreateSnapshotNow(context.Background(), "first")
	require.NoError(t, err)
	second, err := service.CreateSnapshotNow(context.Background(), "second")
	require.NoError(t, err)
	third, err := service.CreateSnapshotNow(context.Background(), "third")
	require.NoError(t, err)

	snapshots, err := service.ListSnapshots(context.Background())
	require.NoError(t, err)
	require.Len(t, snapshots, 2)
	require.Equal(t, third.Name, snapshots[0].Name)
	require.Equal(t, second.Name, snapshots[1].Name)
	require.NoFileExists(t, filepath.Join(dir, first.Name))
}

func TestAccountBackupServiceNotifyCredentialsChangedDebouncesSnapshots(t *testing.T) {
	_, publicKeyPEM := generateAccountBackupTestKey(t)
	dir := t.TempDir()
	var exports atomic.Int64

	service := NewAccountBackupService(AccountBackupConfig{
		Enabled:      true,
		Dir:          dir,
		PublicKeyPEM: publicKeyPEM,
		RetainCount:  5,
		Debounce:     20 * time.Millisecond,
	}, func(ctx context.Context) ([]byte, error) {
		exports.Add(1)
		return []byte(`{"accounts":[]}`), nil
	})
	defer service.Stop()

	service.NotifyCredentialsChanged(context.Background(), "first")
	service.NotifyCredentialsChanged(context.Background(), "second")
	service.NotifyCredentialsChanged(context.Background(), "third")

	require.Eventually(t, func() bool {
		snapshots, err := service.ListSnapshots(context.Background())
		return err == nil && len(snapshots) == 1 && exports.Load() == 1
	}, time.Second, 10*time.Millisecond)
}

func generateAccountBackupTestKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	return privateKey, publicKeyPEM
}

func decryptAccountBackupTestEnvelope(t *testing.T, privateKey *rsa.PrivateKey, raw []byte) []byte {
	t.Helper()

	var envelope testAccountBackupEnvelope
	require.NoError(t, json.Unmarshal(raw, &envelope))

	encryptedKey, err := base64.StdEncoding.DecodeString(envelope.EncryptedKey)
	require.NoError(t, err)
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedKey, nil)
	require.NoError(t, err)

	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	require.NoError(t, err)
	ciphertextBytes, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	require.NoError(t, err)
	tag, err := base64.StdEncoding.DecodeString(envelope.Tag)
	require.NoError(t, err)

	block, err := aes.NewCipher(aesKey)
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	plaintext, err := gcm.Open(nil, nonce, append(ciphertextBytes, tag...), nil)
	require.NoError(t, err)
	return plaintext
}
