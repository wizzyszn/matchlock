package keeper

import (
	"context"
	"log/slog"
	"time"

	"github.com/matchlock/backend-go/internal/cache"
	"github.com/matchlock/backend-go/internal/txline"
)

// OddsWorker refreshes 1X2 odds for all cached fixtures on a short interval.
type OddsWorker struct {
	Cache    cache.Store
	Txline   txline.OddsHydrator
	Interval time.Duration
}

// Run polls TxLINE odds until ctx is cancelled.
func (w *OddsWorker) Run(ctx context.Context) error {
	if w.Interval <= 0 {
		w.Interval = 60 * time.Second
	}

	if err := w.syncOnce(ctx); err != nil && ctx.Err() == nil {
		slog.Warn("initial odds refresh failed", "err", err)
	}

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.syncOnce(ctx); err != nil && ctx.Err() == nil {
				slog.Warn("odds refresh failed", "err", err)
			}
		}
	}
}

func (w *OddsWorker) syncOnce(ctx context.Context) error {
	matches, err := w.Cache.ListMatches(ctx)
	if err != nil {
		return err
	}

	var refreshed int
	for _, match := range matches {
		var existing *txline.MatchOdds
		if match.Odds != nil {
			existing = &txline.MatchOdds{
				Home: match.Odds.Home,
				Draw: match.Odds.Draw,
				Away: match.Odds.Away,
			}
		}

		odds, ok := txline.HydrateMatchOdds(ctx, w.Txline, match.FixtureID, match.StartTime, existing)
		if !ok {
			continue
		}

		hadOdds := match.Odds != nil
		same := hadOdds &&
			match.Odds.Home == odds.Home &&
			match.Odds.Draw == odds.Draw &&
			match.Odds.Away == odds.Away
		if same {
			continue
		}

		match = cache.ApplyMatchOdds(match, cache.MatchOdds{
			Home: odds.Home,
			Draw: odds.Draw,
			Away: odds.Away,
		})
		if err := w.Cache.UpsertMatch(ctx, match); err != nil {
			return err
		}
		refreshed++
	}

	if refreshed > 0 {
		slog.Info("odds refresh complete", "matches", len(matches), "updated", refreshed)
	}
	return nil
}