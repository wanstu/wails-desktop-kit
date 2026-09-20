package secureconfig

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

const DefaultRandomSecretBytes = 32

// RandomBytes returns size cryptographically secure random bytes.
//
// It is the library equivalent of reading random bytes from OpenSSL or the
// operating system CSPRNG, without requiring an external openssl executable.
func RandomBytes(size int) ([]byte, error) {
	if size <= 0 {
		return nil, errors.New("secureconfig: random size must be greater than zero")
	}
	value := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, value); err != nil {
		return nil, fmt.Errorf("secureconfig: generate random bytes: %w", err)
	}
	return value, nil
}

// RandomHex returns size random bytes encoded as lowercase hexadecimal text.
// RandomHex(32) is format-compatible with: openssl rand -hex 32.
func RandomHex(size int) (string, error) {
	value, err := RandomBytes(size)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

// RandomBase64URL returns size random bytes encoded as unpadded base64url text.
func RandomBase64URL(size int) (string, error) {
	value, err := RandomBytes(size)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

// RandomSecretHex returns the Kit default 256-bit random secret as lowercase
// hexadecimal text.
func RandomSecretHex() (string, error) {
	return RandomHex(DefaultRandomSecretBytes)
}
