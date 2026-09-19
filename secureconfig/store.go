package secureconfig

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wanstu/wails-desktop-kit/atomicfile"
	"github.com/wanstu/wails-desktop-kit/paths"
	"github.com/zalando/go-keyring"
)

const (
	currentVersion = 1
	algorithm      = "AES-256-GCM"
	masterUser     = "secure-config-master-key-v1"
)

var (
	ErrNotFound                 = errors.New("secureconfig: secret not found")
	ErrSecureStorageUnavailable = errors.New("secureconfig: system credential store unavailable")
	ErrMasterKeyMissing         = errors.New("secureconfig: master key missing")
	ErrCorrupt                  = errors.New("secureconfig: encrypted data is invalid or was tampered with")
)

// Backend is the minimal system credential-store contract used by Secure Config.
// The built-in backend uses Windows Credential Manager, macOS Keychain, and
// Linux Secret Service via github.com/zalando/go-keyring.
type Backend interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type systemBackend struct{}

func (systemBackend) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (systemBackend) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (systemBackend) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

type encryptedFile struct {
	Version    int    `json:"version"`
	Algorithm  string `json:"algorithm"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// Store encrypts arbitrary small/medium configuration blobs on disk while
// keeping the randomly generated master key in the operating-system credential
// store. Copying the app config directory alone is therefore insufficient to
// decrypt the sensitive payloads.
type Store struct {
	mu      sync.Mutex
	appID   string
	dir     string
	service string
	backend Backend
}

// New creates a Store under ~/.config/<appID>/secure.
func New(appID string) (*Store, error) {
	configDir, err := paths.EnsureConfigDir(appID)
	if err != nil {
		return nil, err
	}
	return NewAt(appID, configDir, systemBackend{})
}

// NewAt creates a Store using an explicit app config directory and credential
// backend. It is primarily useful for tests and controlled integrations.
func NewAt(appID, configDir string, backend Backend) (*Store, error) {
	if err := paths.ValidateAppID(appID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(configDir) == "" {
		return nil, errors.New("secureconfig: config dir is required")
	}
	if !filepath.IsAbs(configDir) {
		return nil, fmt.Errorf("secureconfig: config dir must be absolute: %q", configDir)
	}
	if backend == nil {
		return nil, errors.New("secureconfig: credential backend is required")
	}
	dir := filepath.Join(configDir, "secure")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("secureconfig: create secure dir: %w", err)
	}
	return &Store{
		appID:   appID,
		dir:     dir,
		service: "wails-desktop-kit/" + appID,
		backend: backend,
	}, nil
}

// Dir returns the on-disk secure payload directory.
func (s *Store) Dir() string { return s.dir }

// Put encrypts and atomically stores plaintext under name.
func (s *Store) Put(name string, plaintext []byte) error {
	if err := validateName(name); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := s.loadOrCreateMasterKey()
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("secureconfig: create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("secureconfig: create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("secureconfig: generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, s.additionalData(name))
	record := encryptedFile{
		Version:    currentVersion,
		Algorithm:  algorithm,
		Nonce:      base64.RawStdEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawStdEncoding.EncodeToString(ciphertext),
	}
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("secureconfig: encode encrypted record: %w", err)
	}
	data = append(data, '\n')
	return writeAtomic(s.pathFor(name), data)
}

// Get decrypts and returns the secret stored under name.
func (s *Store) Get(name string) ([]byte, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.pathFor(name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("secureconfig: read encrypted record: %w", err)
	}

	key, err := s.loadMasterKey()
	if err != nil {
		return nil, err
	}
	return s.decrypt(name, key, data)
}

// Delete removes one encrypted payload. It intentionally retains the app master
// key so other secrets remain decryptable.
func (s *Store) Delete(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.Remove(s.pathFor(name))
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("secureconfig: delete encrypted record: %w", err)
	}
	return nil
}

// Exists reports whether an encrypted payload exists. It does not access the
// credential store or decrypt the payload.
func (s *Store) Exists(name string) (bool, error) {
	if err := validateName(name); err != nil {
		return false, err
	}
	_, err := os.Stat(s.pathFor(name))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("secureconfig: stat encrypted record: %w", err)
}

// SaveJSON marshals value as JSON and stores the encrypted bytes.
func (s *Store) SaveJSON(name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("secureconfig: encode JSON: %w", err)
	}
	return s.Put(name, data)
}

// LoadJSON decrypts name and unmarshals the JSON into target.
func (s *Store) LoadJSON(name string, target any) error {
	if target == nil {
		return errors.New("secureconfig: JSON target is required")
	}
	data, err := s.Get(name)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("secureconfig: decode JSON: %w", err)
	}
	return nil
}

func (s *Store) decrypt(name string, key, data []byte) ([]byte, error) {
	var record encryptedFile
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("%w: decode record: %v", ErrCorrupt, err)
	}
	if record.Version != currentVersion || record.Algorithm != algorithm {
		return nil, fmt.Errorf("%w: unsupported version or algorithm", ErrCorrupt)
	}
	nonce, err := base64.RawStdEncoding.DecodeString(record.Nonce)
	if err != nil {
		return nil, fmt.Errorf("%w: decode nonce", ErrCorrupt)
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(record.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: decode ciphertext", ErrCorrupt)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid master key", ErrCorrupt)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: initialize GCM", ErrCorrupt)
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("%w: invalid nonce length", ErrCorrupt)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, s.additionalData(name))
	if err != nil {
		return nil, fmt.Errorf("%w: authentication failed", ErrCorrupt)
	}
	return plaintext, nil
}

func (s *Store) loadOrCreateMasterKey() ([]byte, error) {
	key, err := s.loadMasterKey()
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, ErrMasterKeyMissing) {
		return nil, err
	}

	key = make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("secureconfig: generate master key: %w", err)
	}
	encoded := base64.RawStdEncoding.EncodeToString(key)
	if err := s.backend.Set(s.service, masterUser, encoded); err != nil {
		return nil, fmt.Errorf("%w: save master key: %v", ErrSecureStorageUnavailable, err)
	}
	return key, nil
}

func (s *Store) loadMasterKey() ([]byte, error) {
	encoded, err := s.backend.Get(s.service, masterUser)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) || errors.Is(err, ErrNotFound) {
			return nil, ErrMasterKeyMissing
		}
		return nil, fmt.Errorf("%w: load master key: %v", ErrSecureStorageUnavailable, err)
	}
	key, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("%w: invalid stored master key", ErrCorrupt)
	}
	return key, nil
}

func (s *Store) additionalData(name string) []byte {
	return []byte("wails-desktop-kit/secureconfig/v1\x00" + s.appID + "\x00" + name)
}

func (s *Store) pathFor(name string) string {
	sum := sha256.Sum256([]byte(name))
	return filepath.Join(s.dir, hex.EncodeToString(sum[:])+".json.enc")
}

func validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("secureconfig: secret name is required")
	}
	if len(name) > 256 {
		return errors.New("secureconfig: secret name is too long")
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	if err := atomicfile.Write(path, data, 0o600); err != nil {
		return fmt.Errorf("secureconfig: write encrypted record: %w", err)
	}
	return nil
}
