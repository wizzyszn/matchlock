package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	AccessCookieName  = "matchlock_access"
	RefreshCookieName = "matchlock_refresh"
)

type AccessClaims struct {
	UserID uuid.UUID `json:"uid"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// TokenConfig holds signing and TTL settings.
type TokenConfig struct {
	AccessSecret  []byte
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	MagicLinkTTL  time.Duration
	CookieSecure  bool
	CookieDomain  string
	FrontendURL   string
}

func NewAccessToken(cfg TokenConfig, userID uuid.UUID, email string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(cfg.AccessTTL)
	claims := AccessClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(cfg.AccessSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, exp, nil
}

func ParseAccessToken(cfg TokenConfig, raw string) (AccessClaims, error) {
	var claims AccessClaims
	parsed, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return cfg.AccessSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return AccessClaims{}, err
	}
	if !parsed.Valid {
		return AccessClaims{}, fmt.Errorf("invalid access token")
	}
	return claims, nil
}

func NewOpaqueToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("rand: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	hash = HashToken(raw)
	return raw, hash, nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}