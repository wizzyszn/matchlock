// Command request-faucet calls TxLINE request_devnet_faucet for a wallet keypair.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/matchlock/backend-go/internal/config"
	chainsol "github.com/matchlock/backend-go/internal/solana"
)

const minSolLamports = 5_000_000 

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		exitErr("config: %w", err)
	}

	keyPath := strings.TrimSpace(os.Getenv("KEYPAIR_PATH"))
	if keyPath == "" {
		keyPath = cfg.KeeperKeypairPath
	}

	key, err := chainsol.LoadKeeperKeypairFromFile(keyPath)
	if err != nil {
		exitErr("load keypair: %w", err)
	}

	wallet := key.PublicKey()
	fmt.Fprintf(os.Stderr, "wallet: %s\n", wallet)
	fmt.Fprintf(os.Stderr, "mint:   %s\n", cfg.StablecoinMint)

	client, err := chainsol.NewClient(
		cfg.SolanaRPCURL,
		cfg.MatchlockProgram,
		cfg.StablecoinMint,
		cfg.TxlineProgram,
	)
	if err != nil {
		exitErr("solana client: %w", err)
	}

	solBal, err := client.SOLBalance(ctx, wallet)
	if err != nil {
		exitErr("sol balance: %w", err)
	}
	fmt.Fprintf(os.Stderr, "sol before: %d lamports\n", solBal)

	if solBal < minSolLamports {
		fmt.Fprintf(os.Stderr,
			"request-faucet: wallet needs ≥0.005 SOL for fees (has %d lamports); fund via https://faucet.solana.com\n",
			solBal,
		)
		os.Exit(1)
	}

	before, err := client.TokenBalance(ctx, wallet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "usdt before: 0 (no ATA yet)\n")
		before = 0
	} else {
		fmt.Fprintf(os.Stderr, "usdt before: %d base units\n", before)
	}

	sig, err := client.RequestDevnetFaucet(ctx, key)
	if err != nil {
		if strings.Contains(err.Error(), "already") || before > 0 {
			after, balErr := client.TokenBalance(ctx, wallet)
			if balErr == nil && after > 0 {
				fmt.Printf("Faucet already claimed or unavailable; current balance: %d base units (%.6f USDT)\n",
					after, float64(after)/1_000_000)
				return
			}
		}
		exitErr("faucet: %w", err)
	}

	after, err := client.TokenBalance(ctx, wallet)
	if err != nil {
		exitErr("balance after faucet: %w", err)
	}

	fmt.Printf("Faucet success\n")
	fmt.Printf("wallet: %s\n", wallet)
	fmt.Printf("tx: %s\n", sig)
	fmt.Printf("usdt: %d base units (%.6f USDT)\n", after, float64(after)/1_000_000)
}

func exitErr(format string, err error) {
	fmt.Fprintf(os.Stderr, "request-faucet: "+format+"\n", err)
	os.Exit(1)
}