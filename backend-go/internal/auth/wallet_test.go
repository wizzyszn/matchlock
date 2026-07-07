package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
)

func TestVerifyWalletLinkSignatureHappyPath(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	pk := solana.PublicKeyFromBytes(pub)
	userID := "550e8400-e29b-41d4-a716-446655440000"
	issued := time.Now().UTC()
	message := BuildWalletLinkMessage(userID, pk.String(), issued)
	sig := ed25519.Sign(priv, []byte(message))

	err = VerifyWalletLinkSignature(userID, pk.String(), message, base64.StdEncoding.EncodeToString(sig), walletLinkTTL)
	if err != nil {
		t.Fatalf("expected valid signature: %v", err)
	}
}

func TestVerifyWalletLinkSignatureRejectsUserMismatch(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	pk := solana.PublicKeyFromBytes(pub)
	message := BuildWalletLinkMessage("user-a", pk.String(), time.Now().UTC())
	sig := ed25519.Sign(priv, []byte(message))

	err = VerifyWalletLinkSignature("user-b", pk.String(), message, base64.StdEncoding.EncodeToString(sig), walletLinkTTL)
	if err == nil {
		t.Fatal("expected user mismatch error")
	}
}