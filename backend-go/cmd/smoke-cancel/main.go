// Command smoke-cancel verifies open wagers refund the maker on devnet.
// Set FIXTURE_ID or MATCH_ID to bind the wager to a live TxLINE fixture (recommended during match windows).
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/config"
	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

const (
	defaultStake     = uint64(100_000)
	solTopUpLamports = uint64(50_000_000)
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		exitErr("config: %w", err)
	}
	keeper, err := chainsol.LoadKeeperKeypairFromFile(cfg.KeeperKeypairPath)
	if err != nil {
		exitErr("keeper keypair: %w", err)
	}
	client, err := chainsol.NewClient(cfg.SolanaRPCURL, cfg.MatchlockProgram, cfg.StablecoinMint, cfg.TxlineProgram)
	if err != nil {
		exitErr("solana client: %w", err)
	}

	matchID, fixtureID := resolveMatchID()
	if fixtureID > 0 {
		txClient := txline.NewClient(cfg.TxlineAPIOrigin, cfg.GuestAuthURL(), cfg.TxlineAPIToken, &http.Client{Timeout: 30 * time.Second})
		rows, err := txClient.FetchScoreSnapshot(ctx, fixtureID)
		if err != nil {
			exitErr("txline snapshot fixture=%d: %w", fixtureID, err)
		}
		latest := rows[len(rows)-1]
		fmt.Fprintf(os.Stderr, "txline fixture=%d gameState=%s seq=%d (live snapshot rows=%d)\n",
			fixtureID, latest.State(), latest.Sequence(), len(rows))
	}

	maker := solana.NewWallet()

	if _, err := client.TransferSOL(ctx, keeper, maker.PublicKey(), solTopUpLamports); err != nil {
		exitErr("sol top-up: %w", err)
	}
	waitSOL(ctx, client, maker.PublicKey())
	if _, err := client.RequestDevnetFaucet(ctx, maker.PrivateKey); err != nil {
		exitErr("faucet: %w", err)
	}

	balBefore, err := client.TokenBalance(ctx, maker.PublicKey())
	if err != nil {
		exitErr("balance before: %w", err)
	}

	wagerPDA, makeSig, err := client.MakeWager(ctx, chainsol.MakeWagerParams{
		Maker:              maker.PrivateKey,
		MatchID:            matchID,
		Stake:              defaultStake,
		MakerSide:          chainsol.SideHome,
		Participant1IsHome: true,
		Nonce:              uint64(time.Now().UnixNano()),
	})
	if err != nil {
		exitErr("make_wager: %w", err)
	}
	balAfterMake, err := client.TokenBalance(ctx, maker.PublicKey())
	if err != nil {
		exitErr("balance after make: %w", err)
	}
	if balBefore < defaultStake || balAfterMake > balBefore-defaultStake {
		exitErr("unexpected balance after make: before=%d after=%d stake=%d", balBefore, balAfterMake, defaultStake)
	}

	waitWagerOpen(ctx, client, wagerPDA)

	cancelSig, err := client.CancelWager(ctx, chainsol.CancelWagerParams{
		Maker: maker.PrivateKey,
		Wager: wagerPDA,
	})
	if err != nil {
		exitErr("cancel_wager: %w", err)
	}
	balAfterCancel, err := client.TokenBalance(ctx, maker.PublicKey())
	if err != nil {
		exitErr("balance after cancel: %w", err)
	}
	if balAfterCancel != balBefore {
		exitErr("refund mismatch: before=%d after_cancel=%d", balBefore, balAfterCancel)
	}

	fmt.Printf("smoke-cancel devnet OK\n")
	fmt.Printf("match_id: %s\n", matchID)
	if fixtureID > 0 {
		fmt.Printf("fixture_id: %d\n", fixtureID)
	}
	fmt.Printf("maker: %s\n", maker.PublicKey())
	fmt.Printf("wager pda: %s\n", wagerPDA)
	fmt.Printf("make tx: %s\n", makeSig)
	fmt.Printf("cancel tx: %s\n", cancelSig)
	fmt.Printf("maker usdt before=%d after_make=%d after_cancel=%d\n", balBefore, balAfterMake, balAfterCancel)
}

func resolveMatchID() (matchID string, fixtureID int64) {
	matchID = strings.TrimSpace(os.Getenv("MATCH_ID"))
	if matchID != "" {
		if v := strings.TrimSpace(os.Getenv("FIXTURE_ID")); v != "" {
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				return matchID, id
			}
		}
		return matchID, 0
	}
	if v := strings.TrimSpace(os.Getenv("FIXTURE_ID")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			exitErr("invalid FIXTURE_ID %q: %v", v, err)
		}
		return strconv.FormatInt(id, 10), id
	}
	return fmt.Sprintf("cancel-%d", time.Now().Unix()), 0
}

func waitWagerOpen(ctx context.Context, client *chainsol.Client, wager solana.PublicKey) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		w, err := client.GetWager(ctx, wager)
		if err == nil && w.Status == chainsol.WagerStatusOpen {
			fmt.Fprintf(os.Stderr, "wager status=open pubkey=%s match_id=%s\n", wager, w.MatchID)
			return
		}
		time.Sleep(1 * time.Second)
	}
	exitErr("timeout waiting for open wager %s", wager)
}

func waitSOL(ctx context.Context, client *chainsol.Client, owner solana.PublicKey) {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		bal, err := client.SOLBalance(ctx, owner)
		if err == nil && bal > 0 {
			return
		}
		time.Sleep(2 * time.Second)
	}
	exitErr("timeout waiting for SOL on %s", owner)
}

func exitErr(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
