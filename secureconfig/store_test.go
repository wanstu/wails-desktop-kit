package secureconfig

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/zalando/go-keyring"
)

type memoryBackend struct {
	mu    sync.Mutex
	items map[string]string
	err   error
}

func newMemoryBackend() *memoryBackend {
	return &memoryBackend{items: map[string]string{}}
}

func (m *memoryBackend) key(service, user string) string { return service + "\x00" + user }

func (m *memoryBackend) Get(service, user string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return "", m.err
	}
	value, ok := m.items[m.key(service, user)]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return value, nil
}

func (m *memoryBackend) Set(service, user, password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.items[m.key(service, user)] = password
	return nil
}

func (m *memoryBackend) Delete(service, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	key := m.key(service, user)
	if _, ok := m.items[key]; !ok {
		return keyring.ErrNotFound
	}
	delete(m.items, key)
	return nil
}

func TestPutGetDoesNotPersistPlaintext(t *testing.T) {
	backend := newMemoryBackend()
	configDir := t.TempDir()
	store, err := NewAt("ssh-client", configDir, backend)
	if err != nil {
		t.Fatal(err)
	}

	secret := []byte("correct horse battery staple")
	if err := store.Put("profile:123:password", secret); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("profile:123:password")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatalf("got %q want %q", got, secret)
	}

	entries, err := os.ReadDir(store.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one encrypted file, got %d", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(store.Dir(), entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, secret) {
		t.Fatal("encrypted file contains plaintext secret")
	}
	if strings.Contains(string(data), "profile:123:password") {
		t.Fatal("encrypted file leaked logical secret name")
	}
}

func TestDifferentStoresCannotDecryptCopiedCiphertext(t *testing.T) {
	backendA := newMemoryBackend()
	dirA := t.TempDir()
	storeA, err := NewAt("ssh-client", dirA, backendA)
	if err != nil {
		t.Fatal(err)
	}
	if err := storeA.Put("password", []byte("secret")); err != nil {
		t.Fatal(err)
	}

	backendB := newMemoryBackend()
	dirB := t.TempDir()
	storeB, err := NewAt("ssh-client", dirB, backendB)
	if err != nil {
		t.Fatal(err)
	}
	// Create an unrelated master key in B first.
	if err := storeB.Put("other", []byte("other")); err != nil {
		t.Fatal(err)
	}

	source := storeA.pathFor("password")
	target := storeB.pathFor("password")
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := storeB.Get("password"); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("expected ErrCorrupt with wrong master key, got %v", err)
	}
}

func TestAssociatedDataBindsSecretName(t *testing.T) {
	backend := newMemoryBackend()
	store, err := NewAt("ssh-client", t.TempDir(), backend)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("source-name", []byte("secret")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(store.pathFor("source-name"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.pathFor("renamed"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("renamed"); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("expected authenticated name binding, got %v", err)
	}
}

func TestMissingMasterKeyDoesNotCreateReplacementOnRead(t *testing.T) {
	backend := newMemoryBackend()
	store, err := NewAt("ssh-client", t.TempDir(), backend)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("password", []byte("secret")); err != nil {
		t.Fatal(err)
	}
	delete(backend.items, backend.key(store.service, masterUser))

	if _, err := store.Get("password"); !errors.Is(err, ErrMasterKeyMissing) {
		t.Fatalf("expected ErrMasterKeyMissing, got %v", err)
	}
	if _, ok := backend.items[backend.key(store.service, masterUser)]; ok {
		t.Fatal("read path silently created a replacement master key")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	backend := newMemoryBackend()
	store, err := NewAt("frp-client", t.TempDir(), backend)
	if err != nil {
		t.Fatal(err)
	}
	type credentials struct {
		Token string `json:"token"`
		Key   string `json:"key"`
	}
	want := credentials{Token: "abc", Key: "xyz"}
	if err := store.SaveJSON("api", want); err != nil {
		t.Fatal(err)
	}
	var got credentials
	if err := store.LoadJSON("api", &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestDeleteAndExists(t *testing.T) {
	store, err := NewAt("adm", t.TempDir(), newMemoryBackend())
	if err != nil {
		t.Fatal(err)
	}
	exists, err := store.Exists("secret")
	if err != nil || exists {
		t.Fatalf("initial exists=%v err=%v", exists, err)
	}
	if err := store.Put("secret", []byte("x")); err != nil {
		t.Fatal(err)
	}
	exists, err = store.Exists("secret")
	if err != nil || !exists {
		t.Fatalf("after put exists=%v err=%v", exists, err)
	}
	if err := store.Delete("secret"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("secret"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCredentialBackendFailureNeverFallsBackToPlaintext(t *testing.T) {
	backend := newMemoryBackend()
	backend.err = errors.New("credential store locked")
	store, err := NewAt("ssh-client", t.TempDir(), backend)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("password", []byte("secret")); !errors.Is(err, ErrSecureStorageUnavailable) {
		t.Fatalf("expected secure-storage error, got %v", err)
	}
	entries, err := os.ReadDir(store.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unexpected plaintext/fallback files: %#v", entries)
	}
}

func TestTamperDetection(t *testing.T) {
	store, err := NewAt("ssh-client", t.TempDir(), newMemoryBackend())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("password", []byte("secret")); err != nil {
		t.Fatal(err)
	}
	path := store.pathFor("password")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record encryptedFile
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(record.Ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[0] ^= 0x01
	record.Ciphertext = base64.RawStdEncoding.EncodeToString(ciphertext)
	data, _ = json.Marshal(record)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("password"); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("expected tamper detection, got %v", err)
	}
}
