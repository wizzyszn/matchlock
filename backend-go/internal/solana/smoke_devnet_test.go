//go:build devnet_smoke

package solana

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
)

func TestDevnetFaucetFreshWallet(t *testing.T) {
	ctx := context.Background()
	keeperPath := filepath.Join("..", "..", "keys", "keeper.json")
	keeper, err := LoadKeeperKeypairFromFile(keeperPath)
	if err != nil {
		t.Fatalf("keeper: %v", err)
	}
	client, err := NewClient(devnetRPC(), testProgramID, testMint, testTxlineProgramID)
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	maker := solana.NewWallet()
	if _, err := client.TransferSOL(ctx, keeper, maker.PublicKey(), 50_000_000); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		bal, err := client.SOLBalance(ctx, maker.PublicKey())
		if err == nil && bal > 0 {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if _, err := client.RequestDevnetFaucet(ctx, maker.PrivateKey); err != nil {
		t.Fatalf("faucet: %v", err)
	}
	usdt, err := client.TokenBalance(ctx, maker.PublicKey())
	if err != nil {
		t.Fatalf("token balance: %v", err)
	}
	if usdt == 0 {
		t.Fatal("expected USDT from faucet")
	}
}

const testTxlineProgramID = "6pW64gN1s2uqjHkn1unFeEjAwJkPGHoppGvS715wyP2J"

func devnetRPC() string { return "https://api.devnet.solana.com" }