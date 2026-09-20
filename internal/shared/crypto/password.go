package crypto

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

const BcryptCost = 12

func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func VerifyPassword(plain, storedHash string) (needsRehash bool, err error) {
	if strings.HasPrefix(storedHash, "pbkdf2_sha256$") {
		ok := verifyPBKDF2(plain, storedHash)
		if !ok {
			return false, errors.New("invalid password")
		}
		return true, nil
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(plain)); err != nil {
		return false, err
	}
	return false, nil
}

func verifyPBKDF2(plain, stored string) bool {
	parts := strings.SplitN(stored, "$", 4)
	if len(parts) != 4 {
		return false
	}
	algo, iterStr, salt, b64hash := parts[0], parts[1], parts[2], parts[3]
	if algo != "pbkdf2_sha256" {
		return false
	}
	iterations, err := strconv.Atoi(iterStr)
	if err != nil {
		return false
	}
	expectedHash, err := base64.StdEncoding.DecodeString(b64hash)
	if err != nil {
		return false
	}
	computed := pbkdf2.Key([]byte(plain), []byte(salt), iterations, len(expectedHash), sha256.New)
	return subtle.ConstantTimeCompare(computed, expectedHash) == 1
}

const (
	minLen  = 8
	maxLen  = 128
)

func ValidatePasswordStrength(pw string) error {
	if len(pw) < minLen {
		return fmt.Errorf("password must be at least %d characters", minLen)
	}
	if len(pw) > maxLen {
		return fmt.Errorf("password must be at most %d characters", maxLen)
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range pw {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("password must contain at least one uppercase letter, one lowercase letter, and one digit")
	}
	return nil
}
