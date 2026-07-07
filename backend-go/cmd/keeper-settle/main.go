// Command keeper-settle drives the keeper worker to settle matched wagers for a final fixture.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/matchlock/backend-go/internal/cache"
	"github.com/matchlock/backend-go/internal/config"
	"github.com/matchlock/backend-go/internal/keeper"
	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		exitErr("config: %w", err)
	}

	fixtureID := envInt64("FIXTURE_ID", 17952170)
	matchID := strings.TrimSpace(os.Getenv("MATCH_ID"))
	if matchID == "" {
		matchID = txline.MatchIDFromFixture(fixtureID)
	}

	redisStore, err := cache.NewRedisStore(ctx, cfg.RedisURL)
	if err != nil {
		exitErr("redis: %w", err)
	}
	defer redisStore.Close()

	txClient := txline.NewClient(cfg.TxlineAPIOrigin, cfg.GuestAuthURL(), cfg.TxlineAPIToken, nil)
	if err := txClient.EnsureGuestJWT(ctx, false); err != nil {
		exitErr("txline jwt: %w", err)
	}

	keeperKey, err := chainsol.LoadKeeperKeypairFromFile(cfg.KeeperKeypairPath)
	if err != nil {
		exitErr("keeper keypair: %w", err)
	}
	solClient, err := chainsol.NewClient(cfg.SolanaRPCURL, cfg.MatchlockProgram, cfg.StablecoinMint, cfg.TxlineProgram)
	if err != nil {
		exitErr("solana client: %w", err)
	}

	rows, err := txClient.FetchScoreSnapshot(ctx, fixtureID)
	if err != nil {
		exitErr("snapshot: %w", err)
	}
	update, err := buildFinalUpdate(rows, fixtureID)
	if err != nil {
		exitErr("final update: %w", err)
	}
	home, _ := update.HomeGoals()
	away, _ := update.AwayGoals()
	fmt.Fprintf(os.Stderr, "final update fixture=%d seq=%d state=%s score=%d-%d\n",
		update.FixtureID, update.Seq, update.GameState, home, away)

	wagers, err := solClient.ListMatchedWagers(ctx, matchID)
	if err != nil {
		exitErr("list wagers: %w", err)
	}
	if len(wagers) == 0 {
		exitErr("no matched wagers for match_id %s", matchID)
	}
	fmt.Fprintf(os.Stderr, "matched wagers: %d\n", len(wagers))

	worker := &keeper.Worker{
		Cache:     redisStore,
		Txline:    txClient,
		Solana:    solClient,
		KeeperKey: keeperKey,
		StatKey:   cfg.TxlineStatKey,
	}
	if err := worker.HandleUpdate(ctx, update); err != nil {
		exitErr("keeper handle update: %w", err)
	}

	for _, w := range wagers {
		w2, err := solClient.GetWager(ctx, w.Pubkey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wager %s status check: %v\n", w.Pubkey, err)
			continue
		}
		fmt.Printf("wager %s status=%s\n", w.Pubkey, chainsol.StatusName(w2.Status))
	}
	fmt.Printf("keeper settlement triggered for match_id=%s\n", matchID)
}

func buildFinalUpdate(rows []txline.ScoreSnapshotRow, fixtureID int64) (txline.ScoreUpdate, error) {
	if row, err := txline.LatestFinalSnapshot(rows); err == nil {
		return row.ToScoreUpdate()
	}
	seq := int32(envInt("SETTLE_SEQ", 941))
	state := envOr("SETTLE_GAME_STATE", "F2")
	home := int32(envInt("SETTLE_HOME_GOALS", 2))
	away := int32(envInt("SETTLE_AWAY_GOALS", 1))
	p1Home, _ := keeper.Participant1IsHomeFromRows(rows)
	if v := strings.TrimSpace(os.Getenv("SETTLE_P1_IS_HOME")); v != "" {
		p1Home = v == "1" || strings.EqualFold(v, "true")
	}
	p1Goals, p2Goals := home, away
	if !p1Home {
		p1Goals, p2Goals = away, home
	}
	update := txline.ScoreUpdate{
		FixtureID:          fixtureID,
		GameState:          state,
		Seq:                seq,
		Participant1IsHome: p1Home,
		ScoreSoccer: &txline.SoccerFixtureScore{
			Participant1: txline.SoccerTotalScore{Goals: p1Goals},
			Participant2: txline.SoccerTotalScore{Goals: p2Goals},
		},
	}
	if !update.IsFinal() {
		return txline.ScoreUpdate{}, fmt.Errorf("constructed update not final (state=%s)", state)
	}
	if home == away {
		return txline.ScoreUpdate{}, fmt.Errorf("draw score %d-%d cannot determine winner", home, away)
	}
	return update, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
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

func exitErr(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}