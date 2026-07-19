package keeper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	solanago "github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	"github.com/matchlock/backend-go/internal/leaderboard"
	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

// TxlineClient fetches proofs and manages auth for settlement.
type TxlineClient interface {
	FetchStatValidation(ctx context.Context, fixtureID int64, seq int32, statKey uint32) (txline.StatValidation, error)
	FetchScoreSnapshot(ctx context.Context, fixtureID int64) ([]txline.ScoreSnapshotRow, error)
}

// SolanaClient lists wagers and submits settlement transactions.
type SolanaClient interface {
	ListActiveWagers(ctx context.Context) ([]chainsol.Wager, error)
	ListMatchedWagers(ctx context.Context, matchID string) ([]chainsol.Wager, error)
	GetWager(ctx context.Context, pubkey solanago.PublicKey) (chainsol.Wager, error)
	CloseMatch(ctx context.Context, keeperKey solanago.PrivateKey, matchID string) (solanago.Signature, error)
	SettleWager(ctx context.Context, p chainsol.SettleParams) (solanago.Signature, error)
	VoidWager(ctx context.Context, p chainsol.VoidParams) (solanago.Signature, error)
}

// Worker consumes score events and settles matched wagers when fixtures finalize.
type Worker struct {
	Cache                 cache.Store
	Txline                TxlineClient
	Solana                SolanaClient
	KeeperKey             solanago.PrivateKey
	StatKey               uint32
	AutoSettle            bool
	MaxSettlementAttempts int
	SettlementRetryBase   time.Duration
	Leaderboard           *leaderboard.Service
}

// Run processes score updates until ctx is cancelled.
func (w *Worker) Run(ctx context.Context, events <-chan txline.ScoreUpdate) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, ok := <-events:
			if !ok {
				return nil
			}
			if err := w.HandleUpdate(ctx, update); err != nil {
				slog.Error("keeper handle update failed", "match_id", update.MatchID(), "err", err)
			}
		}
	}
}

// HandleUpdate caches the match and triggers settlement when final.
func (w *Worker) HandleUpdate(ctx context.Context, update txline.ScoreUpdate) error {
	existing, err := w.Cache.GetMatch(ctx, update.MatchID())
	if err != nil && !isCacheMiss(err) {
		return fmt.Errorf("load cached match: %w", err)
	}
	match := cache.ApplyScoreUpdate(existing, update)
	if err := w.Cache.UpsertMatch(ctx, match); err != nil {
		return fmt.Errorf("cache match: %w", err)
	}
	if err := w.Cache.PublishMatchUpdate(ctx, match); err != nil {
		slog.Debug("publish match update failed", "match_id", update.MatchID(), "err", err)
	}
	if !update.IsFinal() {
		return nil
	}
	if err := w.closeMatchOnChain(ctx, update.MatchID()); err != nil {
		slog.Error("close match on-chain failed", "match_id", update.MatchID(), "err", err)
	}
	if !w.AutoSettle {
		slog.Info("match final; winner claim required (keeper auto-settle disabled)",
			"match_id", update.MatchID(),
		)
		return nil
	}
	return w.SettleMatch(ctx, update)
}

func (w *Worker) SettleMatch(ctx context.Context, update txline.ScoreUpdate) error {
	if !w.AutoSettle {
		return nil
	}
	matchID := update.MatchID()
	if _, err := w.Cache.MarkFinalOnce(ctx, matchID); err != nil {
		slog.Warn("record final observation failed", "match_id", matchID, "err", err)
	}

	winningSide, ok := winningSideFromScore(update)
	if !ok {
		return fmt.Errorf("cannot determine winner for match %s", matchID)
	}

	wagers, err := w.Solana.ListMatchedWagers(ctx, matchID)
	if err != nil {
		return fmt.Errorf("list matched wagers: %w", err)
	}
	if len(wagers) == 0 {
		slog.Info("no matched wagers for final match", "match_id", matchID)
		return nil
	}

	for _, wager := range wagers {
		attemptNow, err := w.schedulePendingSettlement(ctx, update, wager)
		if err != nil {
			slog.Error("schedule settlement failed",
				"match_id", matchID,
				"wager", wager.Pubkey.String(),
				"err", err,
			)
			continue
		}
		if !attemptNow {
			continue
		}

		validation, statKey, err := w.fetchDeclaredWinStatValidation(ctx, update.FixtureID, update.Seq, winningSide, wager.Participant1IsHome)
		if err != nil {
			slog.Error("fetch settlement proof failed",
				"match_id", matchID,
				"wager", wager.Pubkey.String(),
				"err", err,
			)
			w.enqueuePendingSettlement(ctx, update, wager, err)
			continue
		}
		slog.Debug("resolved outcome stat",
			"match_id", matchID,
			"fixture_id", update.FixtureID,
			"seq", update.Seq,
			"winning_side", winningSide,
			"stat_key", statKey,
			"wager", wager.Pubkey.String(),
		)
		args, merkleRoot, err := chainsol.ValidationFromAPI(validation)
		if err != nil {
			slog.Error("map settlement proof failed",
				"match_id", matchID,
				"wager", wager.Pubkey.String(),
				"err", err,
			)
			w.enqueuePendingSettlement(ctx, update, wager, err)
			continue
		}
		if err := w.settleOne(ctx, matchID, wager, args, merkleRoot, winningSide); err != nil {
			slog.Error("settle wager failed",
				"match_id", matchID,
				"wager", wager.Pubkey.String(),
				"err", err,
			)
			w.enqueuePendingSettlement(ctx, update, wager, err)
		}
	}
	return nil
}

func (w *Worker) closeMatchOnChain(ctx context.Context, matchID string) error {
	if w.Solana == nil || len(w.KeeperKey) == 0 {
		return nil
	}
	sig, err := w.Solana.CloseMatch(ctx, w.KeeperKey, matchID)
	if err != nil {
		if errors.Is(err, chainsol.ErrMatchAlreadyClosed) {
			return nil
		}
		return err
	}
	slog.Info("match closed for wagering on-chain", "match_id", matchID, "tx_sig", sig.String())
	return nil
}

func (w *Worker) settleOne(
	ctx context.Context,
	matchID string,
	wager chainsol.Wager,
	validation chainsol.ValidateStatArgs,
	merkleRoot [32]byte,
	winningSide uint8,
) error {
	settled, err := w.Cache.IsSettled(ctx, matchID, wager.Pubkey.String())
	if err != nil {
		return err
	}
	if settled {
		return nil
	}

	params := chainsol.SettleParams{
		Settler:     w.KeeperKey,
		Wager:       wager,
		Validation:  validation,
		MerkleRoot:  merkleRoot,
		WinningSide: winningSide,
	}
	resolution := "payout"
	var sig solanago.Signature
	if _, winnerErr := wager.WinnerPubkey(winningSide); winnerErr != nil {
		resolution = "refund"
		sig, err = w.Solana.VoidWager(ctx, params)
	} else {
		sig, err = w.Solana.SettleWager(ctx, params)
	}
	if err != nil {
		if errors.Is(err, chainsol.ErrAlreadySettled) {
			_, _ = w.Cache.MarkSettled(ctx, cache.SettlementRecord{
				MatchID:     matchID,
				WagerPubkey: wager.Pubkey.String(),
				TxSignature: "already-settled",
				SettledAt:   time.Now().UTC(),
			})
			w.clearPendingSettlement(ctx, matchID, wager.Pubkey.String())
			return nil
		}
		return err
	}

	_, err = w.Cache.MarkSettled(ctx, cache.SettlementRecord{
		MatchID:     matchID,
		WagerPubkey: wager.Pubkey.String(),
		TxSignature: sig.String(),
		SettledAt:   time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("record settlement: %w", err)
	}
	w.clearPendingSettlement(ctx, matchID, wager.Pubkey.String())

	slog.Info("wager settled",
		"match_id", matchID,
		"wager_pubkey", wager.Pubkey.String(),
		"tx_sig", sig.String(),
		"winning_side", winningSide,
		"resolution", resolution,
	)

	if w.Leaderboard != nil && resolution == "payout" {
		winnerPubkey, winnerErr := wager.WinnerPubkey(winningSide)
		if winnerErr != nil {
			return winnerErr
		}
		loserPubkey := wager.Maker
		if winnerPubkey.Equals(wager.Maker) && wager.HasCounterparty() {
			loserPubkey = wager.Taker
		}
		if err := w.Leaderboard.RecordSettlement(ctx, leaderboard.SettlementEvent{
			WagerPubkey:  wager.Pubkey.String(),
			WinnerPubkey: winnerPubkey.String(),
			LoserPubkey:  loserPubkey.String(),
			Stake:        wager.Stake,
			MatchID:      matchID,
			TxSignature:  sig.String(),
			WinningSide:  winningSide,
			SettledAt:    time.Now().UTC(),
		}); err != nil {
			slog.Warn("leaderboard record failed",
				"match_id", matchID,
				"wager_pubkey", wager.Pubkey.String(),
				"err", err,
			)
		}
	}

	return nil
}

func winningSideFromScore(update txline.ScoreUpdate) (uint8, bool) {
	home, okHome := update.HomeGoals()
	away, okAway := update.AwayGoals()
	if !okHome || !okAway {
		return 0, false
	}
	switch {
	case home > away:
		return chainsol.SideHome, true
	case away > home:
		return chainsol.SideAway, true
	default:
		return chainsol.SideDraw, true
	}
}
