package keeper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/matchlock/backend-go/internal/cache"
	"github.com/matchlock/backend-go/internal/txline"
)

// ReconcileWorker retries settlement for final fixtures and drains the pending queue.
type ReconcileWorker struct {
	Worker   *Worker
	Interval time.Duration
	Batch    int
}

// Run executes reconciliation on startup and on every interval until ctx is cancelled.
func (r *ReconcileWorker) Run(ctx context.Context) error {
	if r.Worker == nil {
		return fmt.Errorf("reconcile worker: nil keeper worker")
	}
	interval := r.Interval
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	batch := r.Batch
	if batch <= 0 {
		batch = 50
	}

	r.reconcileOnce(ctx, batch)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			r.reconcileOnce(ctx, batch)
		}
	}
}

func (r *ReconcileWorker) reconcileOnce(ctx context.Context, batch int) {
	if err := r.Worker.ReconcileFinalMatches(ctx); err != nil {
		slog.Error("reconcile final matches failed", "err", err)
	}
	if r.Worker.AutoSettle {
		if err := r.Worker.ProcessPendingQueue(ctx, batch); err != nil {
			slog.Error("process pending settlement queue failed", "err", err)
		}
	}
}

// ReconcileFinalMatches verifies overdue fixtures, closes wagering, and settles
// any remaining matched wagers.
func (w *Worker) ReconcileFinalMatches(ctx context.Context) error {
	matches, err := w.Cache.ListMatches(ctx)
	if err != nil {
		return fmt.Errorf("list matches: %w", err)
	}

	candidates := make(map[string]cache.Match, len(matches))
	for _, match := range matches {
		candidates[match.MatchID] = match
	}

	// Redis and fixture schedules are projections, not durable settlement sources.
	// Rebuild missing match candidates from on-chain active wagers after cache loss
	// or when an old fixture has fallen out of the schedule snapshot window. Open
	// wagers are included so a finished fixture is closed before a late acceptance.
	if w.Solana != nil && w.Txline != nil {
		wagers, listErr := w.Solana.ListActiveWagers(ctx)
		if listErr != nil {
			slog.Warn("reconcile active wagers scan failed", "err", listErr)
		} else {
			for _, wager := range wagers {
				matchID := strings.TrimSpace(wager.MatchIDString())
				if matchID == "" {
					continue
				}
				if _, ok := candidates[matchID]; ok {
					continue
				}
				match, hydrateErr := w.HydrateMatchFromSnapshot(ctx, matchID)
				if hydrateErr != nil {
					slog.Debug("reconcile match hydrate failed",
						"match_id", matchID,
						"wager", wager.Pubkey.String(),
						"err", hydrateErr,
					)
					continue
				}
				candidates[matchID] = match
			}
		}
	}

	now := time.Now().UTC()
	for _, match := range candidates {
		if !match.IsFinal && !cache.FinalVerificationEligible(match, now) {
			continue
		}
		_, update, err := w.RefreshVerifiedFinal(ctx, match)
		if err != nil {
			if cache.LiveStatusExpired(match, now) {
				slog.Warn("match final verification overdue",
					"match_id", match.MatchID,
					"fixture_id", match.FixtureID,
					"start_time", match.StartTime,
					"err", err,
				)
			} else {
				slog.Debug("skip reconcile match", "match_id", match.MatchID, "err", err)
			}
			continue
		}
		if err := w.closeMatchOnChain(ctx, update.MatchID()); err != nil {
			slog.Error("reconcile close match failed", "match_id", match.MatchID, "err", err)
		}
		if w.AutoSettle {
			if err := w.SettleMatch(ctx, update); err != nil {
				slog.Error("reconcile settle match failed", "match_id", match.MatchID, "err", err)
			}
		}
	}
	return nil
}

// HydrateMatchFromSnapshot rebuilds a match projection directly from TxLINE.
// It is used by startup reconciliation and API read-repair after Redis loss.
func (w *Worker) HydrateMatchFromSnapshot(ctx context.Context, matchID string) (cache.Match, error) {
	if w.Cache == nil || w.Txline == nil {
		return cache.Match{}, fmt.Errorf("match hydration dependencies unavailable")
	}

	fixtureID, err := strconv.ParseInt(strings.TrimSpace(matchID), 10, 64)
	if err != nil {
		return cache.Match{}, fmt.Errorf("parse fixture id %q: %w", matchID, err)
	}
	if fixtureID <= 0 {
		return cache.Match{}, fmt.Errorf("fixture id must be positive: %q", matchID)
	}

	existing, err := w.Cache.GetMatch(ctx, matchID)
	if err != nil && !isCacheMiss(err) {
		return cache.Match{}, fmt.Errorf("load cached match %s: %w", matchID, err)
	}

	rows, err := w.Txline.FetchScoreSnapshot(ctx, fixtureID)
	if err != nil {
		return cache.Match{}, fmt.Errorf("fetch score snapshot for fixture %d: %w", fixtureID, err)
	}
	row, err := latestSettlementSnapshot(rows)
	if err != nil {
		return cache.Match{}, fmt.Errorf("latest score snapshot for fixture %d: %w", fixtureID, err)
	}
	update, err := row.ToScoreUpdate()
	if err != nil {
		return cache.Match{}, fmt.Errorf("map score snapshot for fixture %d: %w", fixtureID, err)
	}
	if update.FixtureID != fixtureID {
		return cache.Match{}, fmt.Errorf("snapshot fixture mismatch: got %d want %d", update.FixtureID, fixtureID)
	}

	match := cache.ApplyScoreUpdate(existing, update)
	if err := w.Cache.UpsertMatch(ctx, match); err != nil {
		return cache.Match{}, fmt.Errorf("upsert hydrated match %s: %w", matchID, err)
	}
	if err := w.Cache.PublishMatchUpdate(ctx, match); err != nil {
		slog.Debug("publish hydrated match failed", "match_id", matchID, "err", err)
	}
	return match, nil
}

// RefreshVerifiedFinal upgrades a cached final match to a TxLINE-verified final
// by fetching final score snapshots when the SSE final event was missed.
func (w *Worker) RefreshVerifiedFinal(ctx context.Context, match cache.Match) (cache.Match, txline.ScoreUpdate, error) {
	update, err := w.resolveFinalUpdate(ctx, match)
	if err != nil {
		return match, txline.ScoreUpdate{}, err
	}
	refreshed := cache.ApplyScoreUpdate(match, update)
	if shouldPersistVerifiedFinal(match, refreshed) {
		if err := w.Cache.UpsertMatch(ctx, refreshed); err != nil {
			return match, txline.ScoreUpdate{}, fmt.Errorf("upsert verified final match: %w", err)
		}
		if err := w.Cache.PublishMatchUpdate(ctx, refreshed); err != nil {
			slog.Debug("publish verified final match failed", "match_id", match.MatchID, "err", err)
		}
	}
	return refreshed, update, nil
}

func shouldPersistVerifiedFinal(before, after cache.Match) bool {
	if after.FinalSource != cache.FinalSourceTxline {
		return false
	}
	if before.FinalSource != after.FinalSource ||
		before.IsFinal != after.IsFinal ||
		before.GameState != after.GameState ||
		before.Seq != after.Seq {
		return true
	}
	if goalValue(before.HomeGoals) != goalValue(after.HomeGoals) ||
		goalValue(before.AwayGoals) != goalValue(after.AwayGoals) {
		return true
	}
	return false
}

func goalValue(v *int32) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(*v)
}

func (w *Worker) resolveFinalUpdate(ctx context.Context, match cache.Match) (txline.ScoreUpdate, error) {
	if match.FinalSource == cache.FinalSourceTxline && match.Seq > 0 {
		update, err := cache.ScoreUpdateFromMatch(match)
		if err == nil && update.IsFinal() {
			return update, nil
		}
	}

	if w.Txline == nil || match.FixtureID == 0 {
		return txline.ScoreUpdate{}, fmt.Errorf("cannot resolve final update for match %s", match.MatchID)
	}

	rows, err := w.Txline.FetchScoreSnapshot(ctx, match.FixtureID)
	if err != nil {
		return txline.ScoreUpdate{}, fmt.Errorf("fetch score snapshot: %w", err)
	}
	row, err := txline.LatestFinalSnapshot(rows)
	if err != nil {
		return txline.ScoreUpdate{}, err
	}
	update, err := row.ToScoreUpdate()
	if err != nil {
		return txline.ScoreUpdate{}, err
	}
	if !update.IsFinal() {
		return txline.ScoreUpdate{}, errors.New("snapshot update not final")
	}
	return update, nil
}

func isWagerAccountMissing(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}
