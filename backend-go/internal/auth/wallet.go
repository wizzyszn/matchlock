package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
)

const walletLinkTTL = 10 * time.Minute

// BuildWalletLinkMessage returns the nonce message a user must sign to link a wallet.
func BuildWalletLinkMessage(userID, pubkey string, issuedAt time.Time) string {
	return fmt.Sprintf(
		"Matchlock wallet link\nUser: %s\nWallet: %s\nIssued: %s",
		userID,
		pubkey,
		issuedAt.UTC().Format(time.RFC3339),
	)
}

func parseWalletLinkMessage(message string) (userID, wallet, issuedLine string, err error) {
	if !strings.HasPrefix(message, "Matchlock wallet link\n") {
		return "", "", "", fmt.Errorf("invalid wallet link message prefix")
	}
	for _, line := range strings.Split(message, "\n") {
		switch {
		case strings.HasPrefix(line, "User: "):
			userID = strings.TrimPrefix(line, "User: ")
		case strings.HasPrefix(line, "Wallet: "):
			wallet = strings.TrimPrefix(line, "Wallet: ")
		case strings.HasPrefix(line, "Issued: "):
			issuedLine = strings.TrimPrefix(line, "Issued: ")
		}
	}
	if userID == "" || wallet == "" {
		return "", "", "", fmt.Errorf("wallet link message missing user or wallet")
	}
	if issuedLine == "" {
		return "", "", "", fmt.Errorf("wallet link message missing issued timestamp")
	}
	return userID, wallet, issuedLine, nil
}

// VerifyWalletLinkSignature checks an ed25519 signature over the link message.
func VerifyWalletLinkSignature(expectedUserID, pubkey, message, signatureB64 string, maxAge time.Duration) error {
	expectedUserID = strings.TrimSpace(expectedUserID)
	pubkey = strings.TrimSpace(pubkey)
	message = strings.TrimSpace(message)
	signatureB64 = strings.TrimSpace(signatureB64)
	if expectedUserID == "" || pubkey == "" || message == "" || signatureB64 == "" {
		return fmt.Errorf("missing wallet link fields")
	}

	msgUser, msgWallet, issuedLine, err := parseWalletLinkMessage(message)
	if err != nil {
		return err
	}
	if msgUser != expectedUserID {
		return fmt.Errorf("wallet link user mismatch")
	}
	if msgWallet != pubkey {
		return fmt.Errorf("wallet link pubkey mismatch")
	}
	issuedAt, err := time.Parse(time.RFC3339, issuedLine)
	if err != nil {
		return fmt.Errorf("invalid issued timestamp")
	}
	if time.Since(issuedAt) > maxAge || time.Until(issuedAt) > 2*time.Minute {
		return fmt.Errorf("wallet link message expired")
	}

	pk, err := solana.PublicKeyFromBase58(pubkey)
	if err != nil {
		return fmt.Errorf("invalid pubkey: %w", err)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length")
	}

	if !ed25519.Verify(pk.Bytes(), []byte(message), sigBytes) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}