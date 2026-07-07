// Command smoke-wager runs a devnet E2E wager lifecycle: faucet → make → accept → settle.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/config"
	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

const (
	defaultFixtureID int64 = 17952170
	defaultSeq       int32 = 941
	defaultStake     uint64 = 100_000 // 0.1 USDT (6 decimals)
	solTopUpLamports uint64 = 50_000_000
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

	maker := solana.NewWallet()
	taker := solana.NewWallet()
	fmt.Fprintf(os.Stderr, "maker wallet: %s\n", maker.PublicKey())
	fmt.Fprintf(os.Stderr, "taker wallet: %s\n", taker.PublicKey())

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
		faucetSig, err := client.RequestDevnetFaucet(ctx, w.key)
		if err != nil {
			exitErr("faucet %s: %w", w.name, err)
		}
		bal, err := client.TokenBalance(ctx, w.key.PublicKey())
		if err != nil {
			exitErr("balance %s: %w", w.name, err)
		}
		fmt.Fprintf(os.Stderr, "%s faucet tx: %s (usdt=%d)\n", w.name, faucetSig, bal)
		if bal < defaultStake {
			exitErr("%s USDT balance %d < stake %d", w.name, bal, defaultStake)
		}
	}

	matchID := fmt.Sprintf("%d", envInt64("SMOKE_FIXTURE_ID", defaultFixtureID))
	makerSide := chainsol.SideHome
	wagerPDA, makeSig, err := client.MakeWager(ctx, chainsol.MakeWagerParams{
		Maker:     maker.PrivateKey,
		MatchID:   matchID,
		Stake:     defaultStake,
		MakerSide: makerSide,
	})
	if err != nil {
		exitErr("make_wager: %w", err)
	}
	fmt.Fprintf(os.Stderr, "make_wager tx: %s wager=%s\n", makeSig, wagerPDA)

	acceptSig, err := client.AcceptWager(ctx, chainsol.AcceptWagerParams{
		Taker:     taker.PrivateKey,
		Wager:     wagerPDA,
		Maker:     maker.PublicKey(),
		TakerSide: chainsol.SideAway,
	})
	if err != nil {
		exitErr("accept_wager: %w", err)
	}
	fmt.Fprintf(os.Stderr, "accept_wager tx: %s\n", acceptSig)

	wager, err := waitForWager(ctx, client, wagerPDA, chainsol.WagerStatusMatched)
	if err != nil {
		exitErr("get wager: %w", err)
	}
	if wager.Status != chainsol.WagerStatusMatched {
		exitErr("wager status = %d, want matched", wager.Status)
	}

	txlineClient := txline.NewClient(cfg.TxlineAPIOrigin, cfg.GuestAuthURL(), cfg.TxlineAPIToken, nil)
	if err := txlineClient.EnsureGuestJWT(ctx, false); err != nil {
		exitErr("txline guest jwt: %w", err)
	}
	fixtureID := envInt64("SMOKE_FIXTURE_ID", defaultFixtureID)
	seq := int32(envInt("SMOKE_SEQ", int(defaultSeq)))
	statKey := cfg.TxlineStatKey
	if statKey == 0 {
		statKey = 1002
	}

	validation, err := txlineClient.FetchStatValidation(ctx, fixtureID, seq, statKey)
	if err != nil {
		exitErr("fetch validation: %w", err)
	}
	args, merkleRoot, err := chainsol.ValidationFromAPI(validation)
	if err != nil {
		exitErr("map validation: %w", err)
	}

	makerBalBefore, _ := client.TokenBalance(ctx, maker.PublicKey())
	settleSig, err := client.SettleWager(ctx, chainsol.SettleParams{
		Settler:     keeper,
		Wager:       wager,
		Validation:  args,
		MerkleRoot:  merkleRoot,
		WinningSide: makerSide,
	})
	if err != nil {
		exitErr("settle_wager: %w", err)
	}
	makerBalAfter, _ := client.TokenBalance(ctx, maker.PublicKey())

	wager, err = client.GetWager(ctx, wagerPDA)
	if err == nil && wager.Status == chainsol.WagerStatusSettled {
		fmt.Fprintf(os.Stderr, "wager settled on-chain\n")
	}

	fmt.Printf("smoke-wager devnet E2E OK\n")
	fmt.Printf("match_id: %s\n", matchID)
	fmt.Printf("wager pda: %s\n", wagerPDA)
	fmt.Printf("make tx: %s\n", makeSig)
	fmt.Printf("accept tx: %s\n", acceptSig)
	fmt.Printf("settle tx: %s\n", settleSig)
	fmt.Printf("maker usdt before=%d after=%d payout_expected=%d\n", makerBalBefore, makerBalAfter, defaultStake*2)
}

func waitForWager(ctx context.Context, client *chainsol.Client, pubkey solana.PublicKey, wantStatus uint8) (chainsol.Wager, error) {
	deadline := time.Now().Add(45 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		w, err := client.GetWager(ctx, pubkey)
		if err == nil && w.Status == wantStatus {
			return w, nil
		}
		lastErr = err
		time.Sleep(2 * time.Second)
	}
	if lastErr != nil {
		return chainsol.Wager{}, lastErr
	}
	return chainsol.Wager{}, fmt.Errorf("timeout waiting for wager %s status %d", pubkey, wantStatus)
}

func fundWallet(ctx context.Context, client *chainsol.Client, keeper solana.PrivateKey, dest solana.PrivateKey) error {
	sig, err := client.TransferSOL(ctx, keeper, dest.PublicKey(), solTopUpLamports)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "sol top-up %s tx: %s\n", dest.PublicKey(), sig)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		bal, err := client.SOLBalance(ctx, dest.PublicKey())
		if err == nil && bal >= solTopUpLamports/2 {
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

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	var out int
	if _, err := fmt.Sscan(v, &out); err != nil {
		return fallback
	}
	return out
}

func exitErr(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

