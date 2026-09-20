package fingerprint

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/hkdf"
)

type Crypto struct {
	masterKey []byte
	salt      []byte
}

func NewCrypto(masterKeyHex, salt string) (*Crypto, error) {
	mk, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode master key hex: %w", err)
	}
	if len(mk) < 16 {
		return nil, errors.New("master key too short")
	}
	return &Crypto{
		masterKey: mk,
		salt:      []byte(salt),
	}, nil
}

func (c *Crypto) deriveKey(info string) ([]byte, error) {
	h := hkdf.New(sha256.New, c.masterKey, c.salt, []byte(info))
	out := make([]byte, 32)
	if _, err := io.ReadFull(h, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Crypto) EncryptTemplate(plaintext []byte) (ciphertext []byte, err error) {
	key, err := c.deriveKey("fingerprint-template-v1")
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := gcm.Seal(nil, nonce, plaintext, []byte("stmarys-fp-tpl"))
	return append(nonce, out...), nil
}

func (c *Crypto) DecryptTemplate(ciphertext []byte) ([]byte, error) {
	key, err := c.deriveKey("fingerprint-template-v1")
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(ciphertext) <= ns {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ciphertext[:ns]
	ct := ciphertext[ns:]
	return gcm.Open(nil, nonce, ct, []byte("stmarys-fp-tpl"))
}

func HashDeviceSecret(secret string) string {
	mac := hmac.New(sha256.New, []byte("stmarys-device-secret-v1"))
	mac.Write([]byte(secret))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyDeviceHMAC checks an HMAC-SHA256 over the full canonical message the
// device signed (see BuildHeartbeatMessage / BuildScanMessage), not just a
// bare timestamp. Signing only the timestamp would let anyone who observes a
// single valid (timestamp, hmac) pair — e.g. by sniffing one heartbeat —
// replay it forever with an arbitrary attacker-chosen payload (different
// student, different scan_type, fabricated match score) as long as they
// reuse that timestamp. Binding the signature to the actual request content
// makes tampering with any field invalidate it; callers must additionally
// enforce a freshness window (see IsTimestampFresh) to bound replay of an
// unmodified, legitimately-signed message.
func VerifyDeviceHMAC(message string, receivedMAC, secretHash string) bool {
	mac := hmac.New(sha256.New, []byte(secretHash))
	mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(receivedMAC))
}

// IsTimestampFresh bounds how old (or how far in the future) a signed
// request's timestamp may be, so a captured, still-validly-signed request
// can only be replayed within a short window rather than indefinitely.
func IsTimestampFresh(unixTS int64, now time.Time, window time.Duration) bool {
	delta := now.Unix() - unixTS
	if delta < 0 {
		delta = -delta
	}
	return time.Duration(delta)*time.Second <= window
}

// BuildHeartbeatMessage is the canonical string a device must HMAC (keyed by
// its secret hash) for POST /devices/{id}/heartbeat.
func BuildHeartbeatMessage(deviceID string, timestamp int64) string {
	return fmt.Sprintf("heartbeat|%s|%d", deviceID, timestamp)
}

// BuildScanMessage is the canonical string a device must HMAC (keyed by its
// secret hash) for POST /scan. studentID/staffID should be the raw request
// values (empty string when absent) so the signature covers exactly what
// was sent, in a fixed field order.
func BuildScanMessage(deviceID, scanType, studentID, staffID string, timestamp int64) string {
	return fmt.Sprintf("scan|%s|%s|%s|%s|%d", deviceID, scanType, studentID, staffID, timestamp)
}

func GenerateDeviceSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
