package keeper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	if !r.Worker.AutoSettle {
		return
	}
	if err := r.Worker.ReconcileFinalMatches(ctx); err != nil {
		slog.Error("reconcile final matches failed", "err", err)
	}
	if err := r.Worker.ProcessPendingQueue(ctx, batch); err != nil {
		slog.Error("process pending settlement queue failed", "err", err)
	}
}

// ReconcileFinalMatches scans cached final fixtures and settles any remaining matched wagers.
func (w *Worker) ReconcileFinalMatches(ctx context.Context) error {
	matches, err := w.Cache.ListMatches(ctx)
	if err != nil {
		return fmt.Errorf("list matches: %w", err)
	}

	for _, match := range matches {
		if !match.IsFinal {
			continue
		}
		update, err := w.resolveFinalUpdate(ctx, match)
		if err != nil {
			slog.Debug("skip reconcile match", "match_id", match.MatchID, "err", err)
			continue
		}
		if err := w.SettleMatch(ctx, update); err != nil {
			slog.Error("reconcile settle match failed", "match_id", match.MatchID, "err", err)
		}
	}
	return nil
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