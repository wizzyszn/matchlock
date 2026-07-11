package api

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	"github.com/matchlock/backend-go/internal/keeper"
	chainsol "github.com/matchlock/backend-go/internal/solana"
)

const (
	settlementStateMatchLive            = "match_live"
	settlementStateMatchEndedUnverified = "match_ended_unverified"
	settlementStateQueued               = "queued"
	settlementStateRetrying             = "retrying"
	settlementStateSettled              = "settled"
	settlementStateFailed               = "failed"
	settlementStateNotApplicable        = "not_applicable"
)

// SettlementStore reads settlement queue and history from cache.
type SettlementStore interface {
	GetMatch(ctx context.Context, matchID string) (cache.Match, error)
	IsSettled(ctx context.Context, matchID, wagerPubkey string) (bool, error)
	GetSettlement(ctx context.Context, matchID, wagerPubkey string) (cache.SettlementRecord, error)
	GetPendingSettlement(ctx context.Context, matchID, wagerPubkey string) (cache.PendingSettlement, error)
}

func resolveWagerSettlement(
	ctx context.Context,
	store SettlementStore,
	wager chainsol.Wager,
) cache.WagerSettlementView {
	pubkey := wager.Pubkey.String()
	matchID := wager.MatchIDString()

	if wager.Status != chainsol.WagerStatusMatched {
		return cache.WagerSettlementView{State: settlementStateNotApplicable}
	}

	if settled, err := store.IsSettled(ctx, matchID, pubkey); err == nil && settled {
		view := cache.WagerSettlementView{State: settlementStateSettled}
		if rec, err := store.GetSettlement(ctx, matchID, pubkey); err == nil {
			view.SettledAt = &rec.SettledAt
			view.TxSignature = rec.TxSignature
		}
		return view
	}

	match, matchErr := store.GetMatch(ctx, matchID)
	if matchErr != nil {
		return cache.WagerSettlementView{State: settlementStateQueued}
	}

	view := cache.WagerSettlementView{
		MatchFinal:  match.IsFinal,
		FinalSource: match.FinalSource,
	}

	if !match.IsFinal {
		view.State = settlementStateMatchLive
		return view
	}
	if match.FinalSource != cache.FinalSourceTxline {
		view.State = settlementStateMatchEndedUnverified
	}

	if pending, err := store.GetPendingSettlement(ctx, matchID, pubkey); err == nil {
		view.PendingAttempts = pending.Attempts
		view.LastError = pending.LastError
		view.NextRetryAt = &pending.NextRetryAt
		if pending.Attempts >= 12 {
			view.State = settlementStateFailed
		} else if pending.Attempts > 1 {
			view.State = settlementStateRetrying
		} else {
			view.State = settlementStateQueued
		}
		return view
	} else if !errors.Is(err, cache.ErrPendingSettlementNotFound) {
		view.State = settlementStateQueued
		return view
	}

	if view.State == "" {
		view.State = settlementStateQueued
	}
	return view
}

func isWagerMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found")
}

func parseWagerPubkey(raw string) (solana.PublicKey, error) {
	pubkey, err := solana.PublicKeyFromBase58(strings.TrimSpace(raw))
	if err != nil {
		return solana.PublicKey{}, err
	}
	return pubkey, nil
}

func settlementUserMessage(state string) string {
	switch state {
	case settlementStateMatchLive:
		return "The match is still in progress. We'll settle your wager once the final result is confirmed."
	case settlementStateMatchEndedUnverified:
		return "The match has ended. We're confirming the official final score before paying out."
	case settlementStateQueued, settlementStateRetrying:
		return "The final result is verified. Settlement is ready or already being processed on-chain."
	case settlementStateFailed:
		return "Settlement is taking longer than usual. Our system is still working on it."
	case settlementStateSettled:
		return "Your wager has been settled and winnings were sent to the winner's wallet."
	default:
		return ""
	}
}

func (h *handler) refreshVerifiedFinalForWager(ctx context.Context, wager chainsol.Wager) {
	if h.txlineData == nil || h.cache == nil {
		return
	}
	match, err := h.cache.GetMatch(ctx, wager.MatchIDString())
	if err != nil {
		return
	}
	candidate := cache.InferFinalState(match, time.Now().UTC())
	if !candidate.IsFinal || candidate.FinalSource == cache.FinalSourceTxline {
		return
	}

	worker := &keeper.Worker{Cache: h.cache, Txline: h.txlineData}
	if _, _, err := worker.RefreshVerifiedFinal(ctx, candidate); err != nil {
		slog.Debug("settlement final verification unavailable",
			"match_id", candidate.MatchID,
			"err", err,
		)
	}
}

func (h *handler) matchForLeaderboardSync(ctx context.Context, wager chainsol.Wager) (cache.Match, bool) {
	h.refreshVerifiedFinalForWager(ctx, wager)
	match, err := h.cache.GetMatch(ctx, wager.MatchIDString())
	if err != nil {
		return cache.Match{}, false
	}
	if !match.IsFinal || match.HomeGoals == nil || match.AwayGoals == nil {
		return cache.Match{}, false
	}
	return match, true
}

func settlementViewFromCache(view cache.WagerSettlementView) SettlementStatusView {
	return SettlementStatusView{
		State:       view.State,
		Message:     settlementUserMessage(view.State),
		MatchFinal:  view.MatchFinal,
		SettledAt:   view.SettledAt,
		TxSignature: view.TxSignature,
		UpdatedAt:   time.Now().UTC(),
	}
}
