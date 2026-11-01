// Package secretbox encrypts tenant secrets stored in PostgreSQL with
// AES-256-GCM. The master key never lives in the database: it is injected
// (Secret Manager in production) and versioned so it can be rotated.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	ErrNotConfigured = errors.New("secret encryption key not configured")
	ErrUnknownKey    = errors.New("secret encrypted with an unknown key version")
)

type Keyring struct {
	current int
	keys    map[int][]byte
}

// New builds a keyring; keys maps version → 32-byte key.
func New(current int, keys map[int][]byte) (*Keyring, error) {
	if _, ok := keys[current]; !ok {
		return nil, ErrNotConfigured
	}
	for version, key := range keys {
		if len(key) != 32 {
			return nil, fmt.Errorf("secret key version %d must be 32 bytes", version)
		}
	}
	return &Keyring{current: current, keys: keys}, nil
}

// FromEnv reads <prefix>_MASTER_KEY (base64, 32 bytes), <prefix>_KEY_VERSION
// (default 1) and <prefix>_PREVIOUS_KEYS ("version:base64,…") for rotation.
// It returns ErrNotConfigured when no master key is set.
func FromEnv(prefix string) (*Keyring, error) {
	master := strings.TrimSpace(os.Getenv(prefix + "_MASTER_KEY"))
	if master == "" {
		return nil, ErrNotConfigured
	}
	version := 1
	if raw := strings.TrimSpace(os.Getenv(prefix + "_KEY_VERSION")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return nil, fmt.Errorf("%s_KEY_VERSION must be a positive integer", prefix)
		}
		version = parsed
	}
	keys := map[int][]byte{}
	key, err := base64.StdEncoding.DecodeString(master)
	if err != nil {
		return nil, fmt.Errorf("%s_MASTER_KEY must be base64", prefix)
	}
	keys[version] = key
	for _, item := range strings.Split(os.Getenv(prefix+"_PREVIOUS_KEYS"), ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, ":", 2)
		previous, errVersion := strconv.Atoi(parts[0])
		if errVersion != nil || len(parts) != 2 {
			return nil, fmt.Errorf("%s_PREVIOUS_KEYS must be version:base64 pairs", prefix)
		}
		decoded, errKey := base64.StdEncoding.DecodeString(parts[1])
		if errKey != nil {
			return nil, fmt.Errorf("%s_PREVIOUS_KEYS must be base64", prefix)
		}
		keys[previous] = decoded
	}
	return New(version, keys)
}

// Seal encrypts with the current key. aad binds the ciphertext to its owner
// (e.g. organization and profile) so it cannot be moved to another tenant.
func (k *Keyring) Seal(plaintext, aad []byte) (int, []byte, []byte, error) {
	if k == nil {
		return 0, nil, nil, ErrNotConfigured
	}
	gcm, err := k.aead(k.current)
	if err != nil {
		return 0, nil, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return 0, nil, nil, err
	}
	return k.current, nonce, gcm.Seal(nil, nonce, plaintext, aad), nil
}

func (k *Keyring) Open(version int, nonce, ciphertext, aad []byte) ([]byte, error) {
	if k == nil {
		return nil, ErrNotConfigured
	}
	gcm, err := k.aead(version)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, aad)
}

func (k *Keyring) aead(version int) (cipher.AEAD, error) {
	key, ok := k.keys[version]
	if !ok {
		return nil, ErrUnknownKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
