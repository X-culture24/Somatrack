package signedlink

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newRing(t *testing.T, kid byte, retired map[byte]ed25519.PublicKey) (*Keyring, ed25519.PrivateKey) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	k, err := NewKeyring(kid, priv, retired)
	if err != nil {
		t.Fatal(err)
	}
	return k, priv
}

func TestRoundTrip(t *testing.T) {
	k, _ := newRing(t, 1, nil)
	id := uuid.New()
	tok, err := k.Issue(PurposeReceipt, id, 0, []byte{0, 0, 0, 0, 0, 6, 26, 128})
	if err != nil {
		t.Fatal(err)
	}
	// 23 header + 8 extra + 64 signature = 95 bytes → 127 base64url chars.
	if len(tok) != 127 {
		t.Errorf("token length = %d", len(tok))
	}
	c, err := k.Verify(tok, PurposeReceipt)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if c.RecordID != id || c.Purpose != PurposeReceipt || !c.Expires.IsZero() || len(c.Extra) != 8 {
		t.Errorf("claims = %+v", c)
	}
}

func TestWrongPurposeRejected(t *testing.T) {
	k, _ := newRing(t, 1, nil)
	tok, _ := k.Issue(PurposeReceipt, uuid.New(), 0, nil)
	if _, err := k.Verify(tok, PurposeResults); !errors.Is(err, ErrPurpose) {
		t.Fatalf("want ErrPurpose, got %v", err)
	}
}

func TestTamperRejected(t *testing.T) {
	k, _ := newRing(t, 1, nil)
	tok, _ := k.Issue(PurposeReceipt, uuid.New(), 0, []byte{1, 2, 3})
	raw, _ := base64.RawURLEncoding.DecodeString(tok)
	for i := range raw {
		bad := append([]byte(nil), raw...)
		bad[i] ^= 0x01
		if _, err := k.Verify(base64.RawURLEncoding.EncodeToString(bad), PurposeReceipt); err == nil {
			t.Fatalf("flipping byte %d still verified", i)
		}
	}
}

func TestForeignKeyRejected(t *testing.T) {
	school, _ := newRing(t, 1, nil)
	forger, _ := newRing(t, 1, nil) // same key id, different key
	tok, _ := forger.Issue(PurposeReceipt, uuid.New(), 0, nil)
	if _, err := school.Verify(tok, PurposeReceipt); !errors.Is(err, ErrSignature) {
		t.Fatalf("want ErrSignature, got %v", err)
	}
}

func TestExpiry(t *testing.T) {
	k, _ := newRing(t, 1, nil)
	tok, _ := k.Issue(PurposeReceipt, uuid.New(), -time.Hour, nil) // negative ttl → never expires
	if _, err := k.Verify(tok, PurposeReceipt); err != nil {
		t.Fatalf("non-positive ttl must mean no expiry: %v", err)
	}
	tok, _ = k.Issue(PurposeReceipt, uuid.New(), time.Hour, nil)
	c, err := k.Verify(tok, PurposeReceipt)
	if err != nil || c.Expires.IsZero() {
		t.Fatalf("hour ttl: %+v %v", c, err)
	}
}

func TestRotationKeepsOldDocumentsValid(t *testing.T) {
	old, oldPriv := newRing(t, 1, nil)
	printed, _ := old.Issue(PurposeCertificate, uuid.New(), 0, nil)

	next, _ := newRing(t, 2, map[byte]ed25519.PublicKey{1: oldPriv.Public().(ed25519.PublicKey)})
	if _, err := next.Verify(printed, PurposeCertificate); err != nil {
		t.Fatalf("document signed with retired key must still verify: %v", err)
	}
	fresh, _ := next.Issue(PurposeCertificate, uuid.New(), 0, nil)
	if _, err := old.Verify(fresh, PurposeCertificate); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("old ring must not know key 2: %v", err)
	}
	if n := len(next.PublicKeys()); n != 2 {
		t.Fatalf("published keys = %d, want 2", n)
	}
}

func TestMalformed(t *testing.T) {
	k, _ := newRing(t, 1, nil)
	for _, tok := range []string{"", "not base64!", "AAAA", base64.RawURLEncoding.EncodeToString(make([]byte, 200))} {
		if _, err := k.Verify(tok, PurposeReceipt); !errors.Is(err, ErrMalformed) {
			t.Errorf("Verify(%q) = %v, want ErrMalformed", tok, err)
		}
	}
}

func TestKeyringFromEnv(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	pemBytes, err := MarshalPrivateKeyPEM(priv)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "signing.pem")
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	_, oldPriv, _ := ed25519.GenerateKey(rand.Reader)
	oldPub := base64.RawURLEncoding.EncodeToString(oldPriv.Public().(ed25519.PublicKey))

	t.Setenv("SIGNING_KEY_PATH", path)
	t.Setenv("SIGNING_KEY_ID", "3")
	t.Setenv("SIGNING_RETIRED_KEYS", "2:"+oldPub)
	k, err := KeyringFromEnv()
	if err != nil {
		t.Fatalf("KeyringFromEnv: %v", err)
	}
	if keys := k.PublicKeys(); len(keys) != 2 || keys[1].KeyID != 3 || !keys[1].Current {
		t.Fatalf("public keys = %+v", keys)
	}

	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := KeyringFromEnv(); err == nil {
		t.Fatal("world-readable key: want error, got nil")
	}

	t.Setenv("SIGNING_KEY_PATH", "")
	if _, err := KeyringFromEnv(); !errors.Is(err, ErrNoKey) {
		t.Fatalf("unset path: want ErrNoKey, got %v", err)
	}
}
