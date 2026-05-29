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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	accountBackupDefaultRetainCount     = 5
	accountBackupDefaultDebounceSeconds = 120
	accountBackupFileSuffix             = ".json.enc"
	accountBackupDataPageCap            = 1000
)

var ErrAccountBackupNotFound = infraerrors.NotFound("ACCOUNT_BACKUP_NOT_FOUND", "account backup snapshot not found")

type AccountBackupExporter func(ctx context.Context) ([]byte, error)

type AccountBackupConfig struct {
	Enabled       bool
	Dir           string
	PublicKeyPEM  []byte
	PublicKeyFile string
	RetainCount   int
	Debounce      time.Duration
	Now           func() time.Time
}

type AccountBackupSnapshot struct {
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
	Reason    string    `json:"reason,omitempty"`
}

type AccountCredentialsChangeNotifier interface {
	NotifyCredentialsChanged(ctx context.Context, reason string)
}

type AccountBackupService struct {
	cfg      AccountBackupConfig
	exporter AccountBackupExporter
	pubKey   *rsa.PublicKey

	mu          sync.Mutex
	timer       *time.Timer
	pending     bool
	lastReason  string
	stopped     bool
	stopCh      chan struct{}
	timerDoneCh chan struct{}
}

type accountBackupEnvelope struct {
	Version      int    `json:"version"`
	Algorithm    string `json:"algorithm"`
	CreatedAt    string `json:"created_at"`
	Reason       string `json:"reason,omitempty"`
	EncryptedKey string `json:"encrypted_key"`
	Nonce        string `json:"nonce"`
	Ciphertext   string `json:"ciphertext"`
	Tag          string `json:"tag"`
}

type accountBackupDataPayload struct {
	Type       string                     `json:"type"`
	Version    int                        `json:"version"`
	ExportedAt string                     `json:"exported_at"`
	Proxies    []accountBackupDataProxy   `json:"proxies"`
	Accounts   []accountBackupDataAccount `json:"accounts"`
}

type accountBackupDataProxy struct {
	ProxyKey string `json:"proxy_key"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Status   string `json:"status"`
}

type accountBackupDataAccount struct {
	Name               string         `json:"name"`
	Notes              *string        `json:"notes,omitempty"`
	Platform           string         `json:"platform"`
	Type               string         `json:"type"`
	Credentials        map[string]any `json:"credentials"`
	Extra              map[string]any `json:"extra,omitempty"`
	ProxyKey           *string        `json:"proxy_key,omitempty"`
	Concurrency        int            `json:"concurrency"`
	Priority           int            `json:"priority"`
	RateMultiplier     *float64       `json:"rate_multiplier,omitempty"`
	ExpiresAt          *int64         `json:"expires_at,omitempty"`
	AutoPauseOnExpired *bool          `json:"auto_pause_on_expired,omitempty"`
}

func NewAccountBackupService(cfg AccountBackupConfig, exporter AccountBackupExporter) *AccountBackupService {
	normalized := normalizeAccountBackupConfig(cfg)
	s := &AccountBackupService{
		cfg:         normalized,
		exporter:    exporter,
		stopCh:      make(chan struct{}),
		timerDoneCh: make(chan struct{}),
	}
	if normalized.Enabled {
		pubKey, err := parseAccountBackupPublicKey(normalized)
		if err != nil {
			slog.Warn("account_backup.disabled_public_key_invalid", "error", err)
			s.cfg.Enabled = false
		} else {
			s.pubKey = pubKey
		}
	}
	return s
}

func NewAccountBackupServiceFromEnv(adminSvc AdminService) *AccountBackupService {
	return NewAccountBackupService(AccountBackupConfigFromEnv(), NewAccountBackupExporter(adminSvc))
}

func AccountBackupConfigFromEnv() AccountBackupConfig {
	enabled := parseBoolEnv("ACCOUNT_JSON_BACKUP_ENABLED")
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if dataDir == "" {
		dataDir = "."
	}
	backupDir := strings.TrimSpace(os.Getenv("ACCOUNT_JSON_BACKUP_DIR"))
	if backupDir == "" {
		backupDir = filepath.Join(dataDir, "account_backups")
	}

	return AccountBackupConfig{
		Enabled:       enabled,
		Dir:           backupDir,
		PublicKeyFile: strings.TrimSpace(os.Getenv("ACCOUNT_JSON_BACKUP_PUBLIC_KEY_FILE")),
		RetainCount:   parsePositiveIntEnv("ACCOUNT_JSON_BACKUP_SERVER_RETAIN_COUNT", accountBackupDefaultRetainCount),
		Debounce:      time.Duration(parsePositiveIntEnv("ACCOUNT_JSON_BACKUP_DEBOUNCE_SECONDS", accountBackupDefaultDebounceSeconds)) * time.Second,
	}
}

func NewAccountBackupExporter(adminSvc AdminService) AccountBackupExporter {
	return func(ctx context.Context) ([]byte, error) {
		payload, err := buildAccountBackupPayload(ctx, adminSvc)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(payload, "", "  ")
	}
}

func (s *AccountBackupService) NotifyCredentialsChanged(ctx context.Context, reason string) {
	if s == nil || !s.isEnabled() {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.pending = true
	s.lastReason = sanitizeAccountBackupReason(reason)
	if s.cfg.Debounce <= 0 {
		go s.createDebouncedSnapshot()
		return
	}
	if s.timer == nil {
		s.timer = time.AfterFunc(s.cfg.Debounce, s.createDebouncedSnapshot)
		return
	}
	s.timer.Reset(s.cfg.Debounce)
}

func (s *AccountBackupService) CreateSnapshotNow(ctx context.Context, reason string) (*AccountBackupSnapshot, error) {
	if s == nil || !s.isEnabled() {
		return nil, nil
	}
	if s.exporter == nil {
		return nil, errors.New("account backup exporter is not configured")
	}
	if err := os.MkdirAll(s.cfg.Dir, 0700); err != nil {
		return nil, fmt.Errorf("create account backup dir: %w", err)
	}

	plaintext, err := s.exporter(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	encrypted, err := s.encryptPayload(plaintext, sanitizeAccountBackupReason(reason), now)
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("account-data-%s%s", now.Format("20060102T150405.000000000Z"), accountBackupFileSuffix)
	path := filepath.Join(s.cfg.Dir, name)
	if err := os.WriteFile(path, encrypted, 0600); err != nil {
		return nil, fmt.Errorf("write account backup snapshot: %w", err)
	}
	if err := s.enforceRetention(ctx); err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return snapshotFromFileInfo(name, info, reason), nil
}

func (s *AccountBackupService) ListSnapshots(ctx context.Context) ([]AccountBackupSnapshot, error) {
	if s == nil || !s.isEnabled() {
		return []AccountBackupSnapshot{}, nil
	}
	entries, err := os.ReadDir(s.cfg.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return []AccountBackupSnapshot{}, nil
	}
	if err != nil {
		return nil, err
	}
	snapshots := make([]AccountBackupSnapshot, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isAccountBackupFileName(entry.Name()) {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil, infoErr
		}
		snapshots = append(snapshots, *snapshotFromFileInfo(entry.Name(), info, ""))
	}
	sortAccountBackupSnapshots(snapshots)
	return snapshots, nil
}

func (s *AccountBackupService) ReadSnapshotFile(ctx context.Context, name string) ([]byte, error) {
	if s == nil || !s.isEnabled() || !isAccountBackupFileName(name) {
		return nil, ErrAccountBackupNotFound
	}
	cleanName := filepath.Base(name)
	if cleanName != name {
		return nil, ErrAccountBackupNotFound
	}
	data, err := os.ReadFile(filepath.Join(s.cfg.Dir, cleanName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrAccountBackupNotFound
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *AccountBackupService) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	if s.timer != nil {
		s.timer.Stop()
	}
	close(s.stopCh)
	s.mu.Unlock()
}

func (s *AccountBackupService) isEnabled() bool {
	return s != nil && s.cfg.Enabled && s.pubKey != nil
}

func (s *AccountBackupService) now() time.Time {
	if s.cfg.Now != nil {
		return s.cfg.Now()
	}
	return time.Now()
}

func (s *AccountBackupService) createDebouncedSnapshot() {
	s.mu.Lock()
	if s.stopped || !s.pending {
		s.mu.Unlock()
		return
	}
	reason := s.lastReason
	s.pending = false
	s.mu.Unlock()

	if _, err := s.CreateSnapshotNow(context.Background(), reason); err != nil {
		slog.Warn("account_backup.snapshot_failed", "reason", reason, "error", err)
	}
}

func (s *AccountBackupService) encryptPayload(plaintext []byte, reason string, now time.Time) ([]byte, error) {
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	sealed := gcm.Seal(nil, nonce, plaintext, nil)
	tagSize := gcm.Overhead()
	ciphertext := sealed[:len(sealed)-tagSize]
	tag := sealed[len(sealed)-tagSize:]

	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, s.pubKey, aesKey, nil)
	if err != nil {
		return nil, err
	}
	envelope := accountBackupEnvelope{
		Version:      1,
		Algorithm:    "RSA-OAEP-SHA256+A256GCM",
		CreatedAt:    now.Format(time.RFC3339Nano),
		Reason:       reason,
		EncryptedKey: base64.StdEncoding.EncodeToString(encryptedKey),
		Nonce:        base64.StdEncoding.EncodeToString(nonce),
		Ciphertext:   base64.StdEncoding.EncodeToString(ciphertext),
		Tag:          base64.StdEncoding.EncodeToString(tag),
	}
	return json.MarshalIndent(envelope, "", "  ")
}

func (s *AccountBackupService) enforceRetention(ctx context.Context) error {
	if s.cfg.RetainCount <= 0 {
		return nil
	}
	snapshots, err := s.ListSnapshots(ctx)
	if err != nil {
		return err
	}
	if len(snapshots) <= s.cfg.RetainCount {
		return nil
	}
	for _, snapshot := range snapshots[s.cfg.RetainCount:] {
		if err := os.Remove(filepath.Join(s.cfg.Dir, snapshot.Name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func buildAccountBackupPayload(ctx context.Context, adminSvc AdminService) (accountBackupDataPayload, error) {
	payload := accountBackupDataPayload{
		Type:       "sub2api-data",
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Proxies:    []accountBackupDataProxy{},
		Accounts:   []accountBackupDataAccount{},
	}
	if adminSvc == nil {
		return payload, nil
	}

	accounts, err := listAllAccountBackupAccounts(ctx, adminSvc)
	if err != nil {
		return payload, err
	}
	proxies, err := resolveAccountBackupProxies(ctx, adminSvc, accounts)
	if err != nil {
		return payload, err
	}

	proxyKeyByID := make(map[int64]string, len(proxies))
	for _, proxy := range proxies {
		key := buildAccountBackupProxyKey(proxy.Protocol, proxy.Host, proxy.Port, proxy.Username, proxy.Password)
		proxyKeyByID[proxy.ID] = key
		payload.Proxies = append(payload.Proxies, accountBackupDataProxy{
			ProxyKey: key,
			Name:     proxy.Name,
			Protocol: proxy.Protocol,
			Host:     proxy.Host,
			Port:     proxy.Port,
			Username: proxy.Username,
			Password: proxy.Password,
			Status:   proxy.Status,
		})
	}

	for _, account := range accounts {
		var proxyKey *string
		if account.ProxyID != nil {
			if key, ok := proxyKeyByID[*account.ProxyID]; ok {
				proxyKey = &key
			}
		}
		var expiresAt *int64
		if account.ExpiresAt != nil {
			v := account.ExpiresAt.Unix()
			expiresAt = &v
		}
		payload.Accounts = append(payload.Accounts, accountBackupDataAccount{
			Name:               account.Name,
			Notes:              account.Notes,
			Platform:           account.Platform,
			Type:               account.Type,
			Credentials:        account.Credentials,
			Extra:              account.Extra,
			ProxyKey:           proxyKey,
			Concurrency:        account.Concurrency,
			Priority:           account.Priority,
			RateMultiplier:     account.RateMultiplier,
			ExpiresAt:          expiresAt,
			AutoPauseOnExpired: &account.AutoPauseOnExpired,
		})
	}
	return payload, nil
}

func listAllAccountBackupAccounts(ctx context.Context, adminSvc AdminService) ([]Account, error) {
	page := 1
	var out []Account
	for {
		items, total, err := adminSvc.ListAccounts(ctx, page, accountBackupDataPageCap, "", "", "", "", 0, "", "name", "asc")
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if len(out) >= int(total) || len(items) == 0 {
			return out, nil
		}
		page++
	}
}

func resolveAccountBackupProxies(ctx context.Context, adminSvc AdminService, accounts []Account) ([]Proxy, error) {
	seen := make(map[int64]struct{})
	ids := make([]int64, 0)
	for _, account := range accounts {
		if account.ProxyID == nil || *account.ProxyID <= 0 {
			continue
		}
		if _, ok := seen[*account.ProxyID]; ok {
			continue
		}
		seen[*account.ProxyID] = struct{}{}
		ids = append(ids, *account.ProxyID)
	}
	if len(ids) == 0 {
		return []Proxy{}, nil
	}
	return adminSvc.GetProxiesByIDs(ctx, ids)
}

func normalizeAccountBackupConfig(cfg AccountBackupConfig) AccountBackupConfig {
	cfg.Dir = strings.TrimSpace(cfg.Dir)
	if cfg.Dir == "" {
		cfg.Dir = filepath.Join(".", "account_backups")
	}
	if cfg.RetainCount <= 0 {
		cfg.RetainCount = accountBackupDefaultRetainCount
	}
	if cfg.Debounce < 0 {
		cfg.Debounce = 0
	}
	return cfg
}

func parseAccountBackupPublicKey(cfg AccountBackupConfig) (*rsa.PublicKey, error) {
	pemBytes := cfg.PublicKeyPEM
	if len(pemBytes) == 0 && strings.TrimSpace(cfg.PublicKeyFile) != "" {
		data, err := os.ReadFile(cfg.PublicKeyFile)
		if err != nil {
			return nil, err
		}
		pemBytes = data
	}
	if len(pemBytes) == 0 {
		return nil, errors.New("public key is empty")
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid PEM public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}
	return rsaPub, nil
}

func snapshotFromFileInfo(name string, info os.FileInfo, reason string) *AccountBackupSnapshot {
	return &AccountBackupSnapshot{
		Name:      name,
		SizeBytes: info.Size(),
		CreatedAt: info.ModTime().UTC(),
		Reason:    sanitizeAccountBackupReason(reason),
	}
}

func sortAccountBackupSnapshots(snapshots []AccountBackupSnapshot) {
	sort.SliceStable(snapshots, func(i, j int) bool {
		if snapshots[i].CreatedAt.Equal(snapshots[j].CreatedAt) {
			return snapshots[i].Name > snapshots[j].Name
		}
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})
}

func isAccountBackupFileName(name string) bool {
	return filepath.Base(name) == name && strings.HasPrefix(name, "account-data-") && strings.HasSuffix(name, accountBackupFileSuffix)
}

func buildAccountBackupProxyKey(protocol, host string, port int, username, password string) string {
	return fmt.Sprintf("%s|%s|%d|%s|%s", strings.TrimSpace(protocol), strings.TrimSpace(host), port, strings.TrimSpace(username), strings.TrimSpace(password))
}

func sanitizeAccountBackupReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) > 80 {
		reason = reason[:80]
	}
	return reason
}

func parseBoolEnv(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parsePositiveIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
