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

// ScheduleClient fetches fixtures, score snapshots, and odds for market hydration.
type ScheduleClient interface {
	txline.OddsHydrator
	FetchFixturesSnapshot(ctx context.Context, startEpochDay *int) ([]txline.Fixture, error)
	FetchScoreSnapshot(ctx context.Context, fixtureID int64) ([]txline.ScoreSnapshotRow, error)
}

// ScheduleWorker prefetches upcoming fixtures and hydrates scores/odds into the match cache.
type ScheduleWorker struct {
	Cache    cache.Store
	Txline   ScheduleClient
	Interval time.Duration
}

// Run syncs fixtures on startup and on each interval until ctx is cancelled.
func (w *ScheduleWorker) Run(ctx context.Context) error {
	if w.Interval <= 0 {
		w.Interval = time.Hour
	}

	if err := w.syncOnce(ctx); err != nil && ctx.Err() == nil {
		slog.Warn("initial schedule prefetch failed", "err", err)
	}

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.syncOnce(ctx); err != nil && ctx.Err() == nil {
				slog.Warn("schedule prefetch failed", "err", err)
			}
		}
	}
}

func (w *ScheduleWorker) syncOnce(ctx context.Context) error {
	fixtures, err := w.Txline.FetchFixturesSnapshot(ctx, nil)
	if err != nil {
		return fmt.Errorf("fetch fixtures snapshot: %w", err)
	}
	if len(fixtures) == 0 {
		slog.Info("schedule prefetch returned no fixtures")
		return nil
	}

	now := time.Now().UTC()
	var upserted, hydrated int

	for _, fixture := range fixtures {
		if fixture.FixtureID == 0 {
			continue
		}
		existing, err := w.Cache.GetMatch(ctx, fixture.MatchID())
		if err != nil && !isCacheMiss(err) {
			return fmt.Errorf("load match %s: %w", fixture.MatchID(), err)
		}

		match := cache.ApplyFixtureSchedule(existing, fixture)

		if shouldHydrateScores(fixture.StartTime, now) {
			if rows, err := w.Txline.FetchScoreSnapshot(ctx, fixture.FixtureID); err == nil {
				if row, err := latestSettlementSnapshot(rows); err == nil {
					if update, err := row.ToScoreUpdate(); err == nil {
						match = cache.ApplyScoreUpdate(match, update)
						hydrated++
					}
				}
			} else {
				slog.Debug("score snapshot skipped", "fixture_id", fixture.FixtureID, "err", err)
			}
		}

		var existingOdds *txline.MatchOdds
		if match.Odds != nil {
			existingOdds = &txline.MatchOdds{
				Home: match.Odds.Home,
				Draw: match.Odds.Draw,
				Away: match.Odds.Away,
			}
		}
		if odds, ok := txline.HydrateMatchOdds(ctx, w.Txline, fixture.FixtureID, fixture.StartTime, existingOdds); ok {
			match = cache.ApplyMatchOdds(match, cache.MatchOdds{
				Home: odds.Home,
				Draw: odds.Draw,
				Away: odds.Away,
			})
		}

		match = cache.InferFinalState(match, now)

		if err := w.Cache.UpsertMatch(ctx, match); err != nil {
			return fmt.Errorf("upsert scheduled match %s: %w", fixture.MatchID(), err)
		}
		if err := w.Cache.PublishMatchUpdate(ctx, match); err != nil {
			slog.Debug("publish schedule update failed", "match_id", fixture.MatchID(), "err", err)
		}
		upserted++
	}

	slog.Info("schedule prefetch complete",
		"fixtures", len(fixtures),
		"upserted", upserted,
		"scores_hydrated", hydrated,
	)
	return nil
}

func shouldHydrateScores(startTime int64, now time.Time) bool {
	if startTime <= 0 {
		return false
	}
	return startTime <= now.UnixMilli()
}

func isCacheMiss(err error) bool {
	return strings.Contains(err.Error(), "redis: nil") || errors.Is(err, cache.ErrMatchNotFound)
}
