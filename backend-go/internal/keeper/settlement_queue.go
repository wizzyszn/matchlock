package keeper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	solanago "github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

const (
	defaultMaxSettlementAttempts = 12
	defaultSettlementRetryBase   = 30 * time.Second
	defaultSettlementRetryMax    = 30 * time.Minute
)

func (w *Worker) maxSettlementAttempts() int {
	if w.MaxSettlementAttempts > 0 {
		return w.MaxSettlementAttempts
	}
	return defaultMaxSettlementAttempts
}

func (w *Worker) settlementRetryBase() time.Duration {
	if w.SettlementRetryBase > 0 {
		return w.SettlementRetryBase
	}
	return defaultSettlementRetryBase
}

func settlementRetryDelay(base time.Duration, attempts int) time.Duration {
	if attempts <= 0 {
		return base
	}
	delay := base
	for i := 1; i < attempts; i++ {
		delay *= 2
		if delay >= defaultSettlementRetryMax {
			return defaultSettlementRetryMax
		}
	}
	return delay
}

// schedulePendingSettlement persists work before proof fetching or transaction
// submission. The boolean is true only when a new item was created and should
// be attempted immediately; existing items retain their retry schedule.
func (w *Worker) schedulePendingSettlement(
	ctx context.Context,
	update txline.ScoreUpdate,
	wager chainsol.Wager,
) (bool, error) {
	matchID := update.MatchID()
	wagerPubkey := wager.Pubkey.String()
	settled, err := w.Cache.IsSettled(ctx, matchID, wagerPubkey)
	if err != nil {
		return false, fmt.Errorf("check settled marker: %w", err)
	}
	if settled {
		w.clearPendingSettlement(ctx, matchID, wagerPubkey)
		return false, nil
	}
	if _, err := w.Cache.GetPendingSettlement(ctx, matchID, wagerPubkey); err == nil {
		return false, nil
	} else if !errors.Is(err, cache.ErrPendingSettlementNotFound) {
		return false, fmt.Errorf("load pending settlement: %w", err)
	}

	now := time.Now().UTC()
	item := cache.PendingSettlement{
		MatchID:     matchID,
		WagerPubkey: wagerPubkey,
		FixtureID:   update.FixtureID,
		Seq:         update.Seq,
		GameState:   update.GameState,
		Attempts:    0,
		NextRetryAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := w.Cache.EnqueuePendingSettlement(ctx, item); err != nil {
		return false, fmt.Errorf("enqueue pending settlement: %w", err)
	}
	return true, nil
}

func (w *Worker) enqueuePendingSettlement(
	ctx context.Context,
	update txline.ScoreUpdate,
	wager chainsol.Wager,
	err error,
) {
	now := time.Now().UTC()
	matchID := update.MatchID()
	wagerPubkey := wager.Pubkey.String()

	attempts := 1
	createdAt := now
	if existing, getErr := w.Cache.GetPendingSettlement(ctx, matchID, wagerPubkey); getErr == nil {
		attempts = existing.Attempts + 1
		createdAt = existing.CreatedAt
	}

	item := cache.PendingSettlement{
		MatchID:     matchID,
		WagerPubkey: wagerPubkey,
		FixtureID:   update.FixtureID,
		Seq:         update.Seq,
		GameState:   update.GameState,
		Attempts:    attempts,
		LastError:   err.Error(),
		NextRetryAt: now.Add(settlementRetryDelay(w.settlementRetryBase(), attempts)),
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}

	if enqueueErr := w.Cache.EnqueuePendingSettlement(ctx, item); enqueueErr != nil {
		slog.Error("enqueue pending settlement failed",
			"match_id", matchID,
			"wager", wagerPubkey,
			"err", enqueueErr,
		)
	}
}

func (w *Worker) clearPendingSettlement(ctx context.Context, matchID, wagerPubkey string) {
	_ = w.Cache.RemovePendingSettlement(ctx, matchID, wagerPubkey)
}

func (w *Worker) ProcessPendingQueue(ctx context.Context, limit int) error {
	due, err := w.Cache.ListDuePendingSettlements(ctx, time.Now().UTC(), limit)
	if err != nil {
		return fmt.Errorf("list pending settlements: %w", err)
	}
	for _, item := range due {
		if err := w.processPendingItem(ctx, item); err != nil {
			slog.Warn("pending settlement retry failed",
				"match_id", item.MatchID,
				"wager", item.WagerPubkey,
				"attempts", item.Attempts,
				"err", err,
			)
		}
	}
	return nil
}

func (w *Worker) processPendingItem(ctx context.Context, item cache.PendingSettlement) error {
	if item.Attempts >= w.maxSettlementAttempts() {
		slog.Warn("pending settlement exceeded retry alert threshold",
			"match_id", item.MatchID,
			"wager", item.WagerPubkey,
			"attempts", item.Attempts,
			"last_error", item.LastError,
		)
	}

	pubkey, err := solanago.PublicKeyFromBase58(item.WagerPubkey)
	if err != nil {
		return fmt.Errorf("parse wager pubkey: %w", err)
	}

	wager, err := w.Solana.GetWager(ctx, pubkey)
	if err != nil {
		if isWagerAccountMissing(err) {
			w.clearPendingSettlement(ctx, item.MatchID, item.WagerPubkey)
			return nil
		}
		return err
	}
	if wager.Status != chainsol.WagerStatusMatched {
		w.clearPendingSettlement(ctx, item.MatchID, item.WagerPubkey)
		return nil
	}

	update := w.hydratePendingScoreUpdate(ctx, item.FixtureID, item.GameState, item.Seq, func() txline.ScoreUpdate {
		fallback := txline.ScoreUpdate{
			FixtureID: item.FixtureID,
			GameState: item.GameState,
			Seq:       item.Seq,
		}
		if match, err := w.Cache.GetMatch(ctx, item.MatchID); err == nil {
			fallback.Participant1IsHome = match.Participant1IsHome
			if match.HomeGoals != nil && match.AwayGoals != nil {
				p1Goals, p2Goals := *match.HomeGoals, *match.AwayGoals
				if !match.Participant1IsHome {
					p1Goals, p2Goals = *match.AwayGoals, *match.HomeGoals
				}
				fallback.ScoreSoccer = &txline.SoccerFixtureScore{
					Participant1: txline.SoccerTotalScore{Goals: p1Goals},
					Participant2: txline.SoccerTotalScore{Goals: p2Goals},
				}
			}
		} else {
			fallback.Participant1IsHome = true
		}
		return fallback
	})

	winningSide, ok := winningSideFromScore(update)
	if !ok {
		return fmt.Errorf("cannot determine winner for pending match %s", item.MatchID)
	}

	validation, _, err := w.fetchDeclaredWinStatValidation(ctx, update.FixtureID, update.Seq, winningSide, wager.Participant1IsHome)
	if err != nil {
		w.enqueuePendingSettlement(ctx, update, wager, err)
		return err
	}
	args, merkleRoot, err := chainsol.ValidationFromAPI(validation)
	if err != nil {
		w.enqueuePendingSettlement(ctx, update, wager, err)
		return err
	}

	if err := w.settleOne(ctx, item.MatchID, wager, args, merkleRoot, winningSide); err != nil {
		w.enqueuePendingSettlement(ctx, update, wager, err)
		return err
	}
	return nil
}
