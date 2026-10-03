// Package signedlink issues and checks the tamper-proof tokens printed as QR
// codes on school documents (receipts, statements, results, certificates).
//
// A token is a compact binary record signed with the school's Ed25519
// private key, so anyone holding the public key — the /verify page, another
// school, an auditor — can confirm a document is genuine without trusting
// the paper it's printed on:
//
//	version(1) | key id(1) | purpose(1) | expiry(4) | record id(16) | extra(n) || signature(64)
//
// expiry is Unix seconds (big-endian uint32); 0 means the token never
// expires, which is what printed documents use. The purpose byte binds a
// token to one kind of document so a receipt token can never be replayed as a
// results token. The key id lets the key be rotated: retired public keys stay
// in the Keyring so documents printed under them still verify.
package signedlink

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const version byte = 1

// headerLen is everything before the optional extra bytes.
const headerLen = 1 + 1 + 1 + 4 + 16

// maxExtra keeps tokens small enough for a QR code a phone camera reads at
// receipt-printer resolution.
const maxExtra = 32

// context separates these signatures from anything else the same key could
// ever be asked to sign.
var context = []byte("stmarys-signedlink-v1")

// Purpose says what kind of document a token vouches for.
type Purpose byte

const (
	PurposeReceipt     Purpose = 1
	PurposeStatement   Purpose = 2
	PurposeResults     Purpose = 3
	PurposeCertificate Purpose = 4
)

func (p Purpose) String() string {
	switch p {
	case PurposeReceipt:
		return "receipt"
	case PurposeStatement:
		return "statement"
	case PurposeResults:
		return "results"
	case PurposeCertificate:
		return "certificate"
	}
	return "unknown"
}

var (
	ErrMalformed  = errors.New("signedlink: malformed token")
	ErrUnknownKey = errors.New("signedlink: unknown signing key")
	ErrSignature  = errors.New("signedlink: bad signature")
	ErrExpired    = errors.New("signedlink: token expired")
	ErrPurpose    = errors.New("signedlink: token issued for a different purpose")
	ErrNoKey      = errors.New("signedlink: no signing key configured")
)

// Claims is what a verified token asserts.
type Claims struct {
	KeyID    byte
	Purpose  Purpose
	RecordID uuid.UUID
	Expires  time.Time // zero = never
	Extra    []byte
}

// Keyring holds the current private key (for issuing) and every public key
// still trusted for verification, current and retired.
type Keyring struct {
	currentID byte
	private   ed25519.PrivateKey
	public    map[byte]ed25519.PublicKey
}

// NewKeyring builds a keyring that signs with priv under kid. retired maps
// older key ids to their public keys.
func NewKeyring(kid byte, priv ed25519.PrivateKey, retired map[byte]ed25519.PublicKey) (*Keyring, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("signedlink: private key must be %d bytes", ed25519.PrivateKeySize)
	}
	if kid == 0 {
		return nil, errors.New("signedlink: key id 0 is reserved")
	}
	pub := map[byte]ed25519.PublicKey{kid: priv.Public().(ed25519.PublicKey)}
	for id, k := range retired {
		if id == kid {
			return nil, fmt.Errorf("signedlink: retired key id %d clashes with the current key", id)
		}
		if len(k) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("signedlink: retired key %d is not an Ed25519 public key", id)
		}
		pub[id] = k
	}
	return &Keyring{currentID: kid, private: priv, public: pub}, nil
}

// KeyringFromEnv loads:
//
//	SIGNING_KEY_PATH      PEM (PKCS#8) Ed25519 private key — required
//	SIGNING_KEY_ID        1–255, default 1
//	SIGNING_RETIRED_KEYS  optional "kid:base64url-pubkey,kid:..." for rotated keys
//
// It returns (nil, ErrNoKey) when SIGNING_KEY_PATH is unset, and refuses a key
// file that is readable by group or others.
func KeyringFromEnv() (*Keyring, error) {
	path := os.Getenv("SIGNING_KEY_PATH")
	if path == "" {
		return nil, ErrNoKey
	}
	kid := byte(1)
	if v := os.Getenv("SIGNING_KEY_ID"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 255 {
			return nil, fmt.Errorf("signedlink: SIGNING_KEY_ID must be 1-255, got %q", v)
		}
		kid = byte(n)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("signedlink: read private key: %w", err)
	}
	// Anyone who can read this file can forge receipts and result slips.
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		return nil, fmt.Errorf("signedlink: private key %s has mode %04o — it must not be readable by group/others (chmod 600 %s)", path, perm, path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("signedlink: read private key: %w", err)
	}
	priv, err := ParsePrivateKeyPEM(raw)
	if err != nil {
		return nil, err
	}
	retired, err := parseRetired(os.Getenv("SIGNING_RETIRED_KEYS"))
	if err != nil {
		return nil, err
	}
	return NewKeyring(kid, priv, retired)
}

// ParsePrivateKeyPEM reads a PKCS#8 "PRIVATE KEY" block holding an Ed25519 key.
func ParsePrivateKeyPEM(raw []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("signedlink: private key file is not PEM")
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("signedlink: parse private key: %w", err)
	}
	priv, ok := k.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("signedlink: private key is not Ed25519")
	}
	return priv, nil
}

// MarshalPrivateKeyPEM is the inverse of ParsePrivateKeyPEM (used by keygen).
func MarshalPrivateKeyPEM(priv ed25519.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func parseRetired(s string) (map[byte]ed25519.PublicKey, error) {
	out := map[byte]ed25519.PublicKey{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, key, ok := strings.Cut(part, ":")
		n, err := strconv.Atoi(id)
		if !ok || err != nil || n < 1 || n > 255 {
			return nil, fmt.Errorf("signedlink: bad SIGNING_RETIRED_KEYS entry %q", part)
		}
		pub, err := base64.RawURLEncoding.DecodeString(key)
		if err != nil || len(pub) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("signedlink: bad public key for retired key %d", n)
		}
		out[byte(n)] = ed25519.PublicKey(pub)
	}
	return out, nil
}

// Issue signs a token for record under purpose. ttl 0 means it never expires.
func (k *Keyring) Issue(purpose Purpose, record uuid.UUID, ttl time.Duration, extra []byte) (string, error) {
	if k == nil {
		return "", ErrNoKey
	}
	if len(extra) > maxExtra {
		return "", fmt.Errorf("signedlink: extra data limited to %d bytes", maxExtra)
	}
	var exp uint32
	if ttl > 0 {
		exp = uint32(time.Now().Add(ttl).Unix())
	}
	msg := make([]byte, headerLen, headerLen+len(extra)+ed25519.SignatureSize)
	msg[0] = version
	msg[1] = k.currentID
	msg[2] = byte(purpose)
	binary.BigEndian.PutUint32(msg[3:7], exp)
	copy(msg[7:23], record[:])
	msg = append(msg, extra...)
	sig := ed25519.Sign(k.private, signed(msg))
	return base64.RawURLEncoding.EncodeToString(append(msg, sig...)), nil
}

// Verify checks token's signature and expiry and that it was issued for want.
func (k *Keyring) Verify(token string, want Purpose) (*Claims, error) {
	if k == nil {
		return nil, ErrNoKey
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil || len(raw) < headerLen+ed25519.SignatureSize || len(raw) > headerLen+maxExtra+ed25519.SignatureSize {
		return nil, ErrMalformed
	}
	if raw[0] != version {
		return nil, ErrMalformed
	}
	msg, sig := raw[:len(raw)-ed25519.SignatureSize], raw[len(raw)-ed25519.SignatureSize:]
	pub, ok := k.public[raw[1]]
	if !ok {
		return nil, ErrUnknownKey
	}
	// Signature before anything else is trusted — including purpose/expiry.
	if !ed25519.Verify(pub, signed(msg), sig) {
		return nil, ErrSignature
	}
	c := &Claims{KeyID: raw[1], Purpose: Purpose(raw[2])}
	copy(c.RecordID[:], raw[7:23])
	if exp := binary.BigEndian.Uint32(raw[3:7]); exp != 0 {
		c.Expires = time.Unix(int64(exp), 0)
		if time.Now().After(c.Expires) {
			return nil, ErrExpired
		}
	}
	if c.Purpose != want {
		return nil, ErrPurpose
	}
	if len(msg) > headerLen {
		c.Extra = append([]byte(nil), msg[headerLen:]...)
	}
	return c, nil
}

func signed(msg []byte) []byte {
	return append(append([]byte(nil), context...), msg...)
}

// PublicKey is one entry of the published key set.
type PublicKey struct {
	KeyID     int    `json:"kid"`
	Algorithm string `json:"alg"`
	Key       string `json:"key"` // base64url, raw 32-byte Ed25519 public key
	Current   bool   `json:"current"`
}

// PublicKeys lists every trusted public key, for /.well-known/stmarys-keys.json.
func (k *Keyring) PublicKeys() []PublicKey {
	if k == nil {
		return nil
	}
	out := make([]PublicKey, 0, len(k.public))
	for id := 1; id <= 255; id++ {
		if pub, ok := k.public[byte(id)]; ok {
			out = append(out, PublicKey{
				KeyID:     id,
				Algorithm: "Ed25519",
				Key:       base64.RawURLEncoding.EncodeToString(pub),
				Current:   byte(id) == k.currentID,
			})
		}
	}
	return out
}
