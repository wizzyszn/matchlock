package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	cfg := TokenConfig{
		AccessSecret: []byte("test-jwt-access-secret-at-least-32-chars-long"),
		AccessTTL:    time.Minute,
	}
	userID := uuid.New()
	token, _, err := NewAccessToken(cfg, userID, "player@example.com")
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}
	claims, err := ParseAccessToken(cfg, token)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims.UserID != userID || claims.Email != "player@example.com" {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestOpaqueTokenUnique(t *testing.T) {
	a, ha, err := NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	b, hb, err := NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	if a == b || ha == hb {
		t.Fatal("expected unique tokens")
	}
	if HashToken(a) != ha {
		t.Fatal("hash mismatch")
	}
}