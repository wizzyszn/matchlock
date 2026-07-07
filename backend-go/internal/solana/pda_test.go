package solana

import (
	"testing"

	"github.com/gagliardetto/solana-go"
)

func TestPDADerivation(t *testing.T) {
	program, err := solana.PublicKeyFromBase58(testProgramID)
	if err != nil {
		t.Fatalf("program: %v", err)
	}
	txline, err := solana.PublicKeyFromBase58(testTxlineID)
	if err != nil {
		t.Fatalf("txline: %v", err)
	}
	wager := solana.NewWallet().PublicKey()

	if _, _, err := FindConfigPDA(program); err != nil {
		t.Fatalf("FindConfigPDA: %v", err)
	}
	if _, _, err := FindVaultPDA(program, wager); err != nil {
		t.Fatalf("FindVaultPDA: %v", err)
	}
	if _, _, err := FindDailyScoresRootsPDA(txline, EpochDayFromMillis(1700000000000)); err != nil {
		t.Fatalf("FindDailyScoresRootsPDA: %v", err)
	}
}

func TestEpochDayFromMillis(t *testing.T) {
	const dayMs = 24 * 60 * 60 * 1000
	if got := EpochDayFromMillis(int64(2*dayMs + 1)); got != 2 {
		t.Fatalf("epoch day = %d", got)
	}
}