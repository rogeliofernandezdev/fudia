package secretbox

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func key(b byte) []byte { return bytes.Repeat([]byte{b}, 32) }

func TestSealOpenBindsOwnerAndRotates(t *testing.T) {
	ring, err := New(1, map[int][]byte{1: key(1)})
	if err != nil {
		t.Fatal(err)
	}
	version, nonce, sealed, err := ring.Seal([]byte("clave SOL"), []byte("org-a|profile-a"))
	if err != nil || version != 1 || bytes.Contains(sealed, []byte("clave SOL")) {
		t.Fatalf("seal: %v %d", err, version)
	}
	if plain, err := ring.Open(version, nonce, sealed, []byte("org-a|profile-a")); err != nil || string(plain) != "clave SOL" {
		t.Fatalf("open: %q %v", plain, err)
	}
	if _, err := ring.Open(version, nonce, sealed, []byte("org-b|profile-a")); err == nil {
		t.Fatal("a secret moved to another tenant must not open")
	}
	rotated, err := New(2, map[int][]byte{1: key(1), 2: key(2)})
	if err != nil {
		t.Fatal(err)
	}
	if plain, err := rotated.Open(version, nonce, sealed, []byte("org-a|profile-a")); err != nil || string(plain) != "clave SOL" {
		t.Fatalf("previous key must still open: %v", err)
	}
	if v, _, _, _ := rotated.Seal([]byte("x"), nil); v != 2 {
		t.Fatalf("new secrets use the current key, got %d", v)
	}
	if _, err := (*Keyring)(nil).Open(1, nonce, sealed, nil); err != ErrNotConfigured {
		t.Fatalf("missing keyring: %v", err)
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("TEST_SB_MASTER_KEY", "")
	if _, err := FromEnv("TEST_SB"); err != ErrNotConfigured {
		t.Fatalf("no key: %v", err)
	}
	t.Setenv("TEST_SB_MASTER_KEY", base64.StdEncoding.EncodeToString(key(3)))
	t.Setenv("TEST_SB_KEY_VERSION", "3")
	t.Setenv("TEST_SB_PREVIOUS_KEYS", "1:"+base64.StdEncoding.EncodeToString(key(1)))
	ring, err := FromEnv("TEST_SB")
	if err != nil || ring.current != 3 || len(ring.keys) != 2 {
		t.Fatalf("env keyring: %v", err)
	}
	t.Setenv("TEST_SB_MASTER_KEY", base64.StdEncoding.EncodeToString([]byte("short")))
	if _, err := FromEnv("TEST_SB"); err == nil {
		t.Fatal("a short key must be rejected")
	}
}
