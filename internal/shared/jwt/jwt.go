package jwtutil

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/stmaryskabete/lms/internal/shared/types"
)

const (
	ScopePortal = "portal"
	ScopeNova   = "nova"
)

type Manager struct {
	secret        []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	rotateRefresh bool
}

type TokenPair struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

func NewManager(secret string, accessTTL, refreshTTL time.Duration, rotateRefresh bool) *Manager {
	return &Manager{
		secret:        []byte(secret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		rotateRefresh: rotateRefresh,
	}
}

func (m *Manager) IssueTokenPair(user *types.User, scope string) (*TokenPair, error) {
	if scope != ScopePortal && scope != ScopeNova {
		return nil, errors.New("invalid scope")
	}
	accessJti := randomJTI()
	refreshJti := randomJTI()
	now := time.Now()

	accessClaims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"role":    string(user.Role),
		"portal":  string(user.Portal),
		"scope":   scope,
		"exp":     now.Add(m.accessTTL).Unix(),
		"iat":     now.Unix(),
		"jti":     accessJti,
	}
	access, err := m.sign(accessClaims)
	if err != nil {
		return nil, fmt.Errorf("sign access: %w", err)
	}
	refreshClaims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"role":    string(user.Role),
		"portal":  string(user.Portal),
		"scope":   scope,
		"exp":     now.Add(m.refreshTTL).Unix(),
		"iat":     now.Unix(),
		"jti":     refreshJti,
		"type":    "refresh",
	}
	refresh, err := m.sign(refreshClaims)
	if err != nil {
		return nil, fmt.Errorf("sign refresh: %w", err)
	}
	return &TokenPair{Access: access, Refresh: refresh}, nil
}

func (m *Manager) RotateRefresh(refreshToken string) (*TokenPair, error) {
	claims, err := m.ValidateRefresh(refreshToken)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(claims["user_id"].(string))
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}
	user := &types.User{
		ID:     uid,
		Role:   types.Role(claims["role"].(string)),
		Portal: types.PortalType(claims["portal"].(string)),
	}
	scope, _ := claims["scope"].(string)
	return m.IssueTokenPair(user, scope)
}

func (m *Manager) sign(claims jwt.MapClaims) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(m.secret)
}

func (m *Manager) parse(tokenString string) (jwt.MapClaims, error) {
	tok, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

func (m *Manager) ValidateAccess(tokenString string, expectedScope string) (*types.AuthClaims, error) {
	claims, err := m.parse(tokenString)
	if err != nil {
		return nil, err
	}
	scope, _ := claims["scope"].(string)
	if expectedScope != "" && scope != expectedScope {
		return nil, fmt.Errorf("invalid scope: expected %s", expectedScope)
	}
	uidStr, _ := claims["user_id"].(string)
	uid, err := uuid.Parse(uidStr)
	if err != nil {
		return nil, fmt.Errorf("parse user_id: %w", err)
	}
	exp, _ := claims["exp"].(float64)
	iat, _ := claims["iat"].(float64)
	jti, _ := claims["jti"].(string)
	return &types.AuthClaims{
		UserID: uid,
		Role:   types.Role(claims["role"].(string)),
		Portal: types.PortalType(claims["portal"].(string)),
		Scope:  scope,
		Exp:    int64(exp),
		Iat:    int64(iat),
		Jti:    jti,
	}, nil
}

func (m *Manager) ValidateRefresh(tokenString string) (jwt.MapClaims, error) {
	claims, err := m.parse(tokenString)
	if err != nil {
		return nil, err
	}
	ttype, _ := claims["type"].(string)
	if ttype != "refresh" {
		return nil, errors.New("not a refresh token")
	}
	return claims, nil
}

func randomJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
