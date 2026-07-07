// Command place-matched-wager funds wallets, creates and accepts a wager (no settlement).
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/config"
	chainsol "github.com/matchlock/backend-go/internal/solana"
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

	matchID := strings.TrimSpace(os.Getenv("MATCH_ID"))
	if matchID == "" {
		fixture := envInt64("FIXTURE_ID", 17952170)
		matchID = strconv.FormatInt(fixture, 10)
	}
	stake := envUint64("WAGER_STAKE", defaultStake)
	makerSide := chainsol.SideHome
	if strings.EqualFold(os.Getenv("MAKER_SIDE"), "away") {
		makerSide = chainsol.SideAway
	} else if strings.EqualFold(os.Getenv("MAKER_SIDE"), "draw") {
		makerSide = chainsol.SideDraw
	}
	takerSide := chainsol.SideAway
	switch makerSide {
	case chainsol.SideAway:
		takerSide = chainsol.SideHome
	case chainsol.SideDraw:
		takerSide = chainsol.SideHome
	}
	if env := os.Getenv("TAKER_SIDE"); env != "" {
		switch strings.ToLower(env) {
		case "home":
			takerSide = chainsol.SideHome
		case "away":
			takerSide = chainsol.SideAway
		case "draw":
			takerSide = chainsol.SideDraw
		}
	}

	maker := solana.NewWallet()
	taker := solana.NewWallet()
	for _, w := range []struct {
		name string
		key  solana.PrivateKey
	}{
		{"maker", maker.PrivateKey},
		{"taker", taker.PrivateKey},
	} {
		if err := fundWallet(ctx, client, keeper, w.key); err != nil {
			exitErr("fund %s: %w", w.name, err)
		}
		if _, err := client.RequestDevnetFaucet(ctx, w.key); err != nil {
			exitErr("faucet %s: %w", w.name, err)
		}
	}

	wagerPDA, makeSig, err := client.MakeWager(ctx, chainsol.MakeWagerParams{
		Maker:     maker.PrivateKey,
		MatchID:   matchID,
		Stake:     stake,
		MakerSide: makerSide,
	})
	if err != nil {
		exitErr("make_wager: %w", err)
	}
	acceptSig, err := client.AcceptWager(ctx, chainsol.AcceptWagerParams{
		Taker:     taker.PrivateKey,
		Wager:     wagerPDA,
		Maker:     maker.PublicKey(),
		TakerSide: takerSide,
	})
	if err != nil {
		exitErr("accept_wager: %w", err)
	}

	fmt.Printf("matched wager ready for keeper\n")
	fmt.Printf("match_id: %s\n", matchID)
	fmt.Printf("wager pda: %s\n", wagerPDA)
	fmt.Printf("maker: %s\n", maker.PublicKey())
	fmt.Printf("taker: %s\n", taker.PublicKey())
	fmt.Printf("maker_side: %s\n", chainsol.SideName(makerSide))
	fmt.Printf("stake: %d\n", stake)
	fmt.Printf("make tx: %s\n", makeSig)
	fmt.Printf("accept tx: %s\n", acceptSig)
}

func fundWallet(ctx context.Context, client *chainsol.Client, keeper solana.PrivateKey, dest solana.PrivateKey) error {
	if _, err := client.TransferSOL(ctx, keeper, dest.PublicKey(), solTopUpLamports); err != nil {
		return err
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		bal, err := client.SOLBalance(ctx, dest.PublicKey())
		if err == nil && bal > 0 {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for SOL on %s", dest.PublicKey())
}

func envInt64(key string, fallback int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	var out int64
	if _, err := fmt.Sscan(v, &out); err != nil {
		return fallback
	}
	return out
}

func envUint64(key string, fallback uint64) uint64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	var out uint64
	if _, err := fmt.Sscan(v, &out); err != nil {
		return fallback
	}
	return out
}

func exitErr(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}