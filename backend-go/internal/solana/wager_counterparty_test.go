package solana

import (
	"testing"

	"github.com/gagliardetto/solana-go"
)

func TestHasCounterpartyOpenWager(t *testing.T) {
	w := Wager{
		Status: WagerStatusOpen,
		Taker:  SystemProgramID,
	}
	if w.HasCounterparty() {
		t.Fatal("open wager with default taker should not have counterparty")
	}
}

func TestHasCounterpartyMatchedWager(t *testing.T) {
	w := Wager{
		Status: WagerStatusMatched,
		Taker:  solana.NewWallet().PublicKey(),
	}
	if !w.HasCounterparty() {
		t.Fatal("matched wager should have counterparty")
	}
}