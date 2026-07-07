package txline

import (
	"context"
	"time"
)

// OddsHydrator fetches odds for a fixture using TxLINE snapshot and updates endpoints.
type OddsHydrator interface {
	FetchOddsSnapshot(ctx context.Context, fixtureID int64) ([]OddsPayload, error)
	FetchOddsSnapshotAsOf(ctx context.Context, fixtureID int64, asOf int64) ([]OddsPayload, error)
	FetchOddsUpdates(ctx context.Context, fixtureID int64) ([]OddsPayload, error)
}

// HydrateMatchOdds resolves the best available 1X2 line for a fixture.
// It preserves prior odds when no new line is available.
func HydrateMatchOdds(
	ctx context.Context,
	client OddsHydrator,
	fixtureID int64,
	startTime int64,
	existing *MatchOdds,
) (MatchOdds, bool) {
	candidates := make([][]OddsPayload, 0, 4)
	appendRows := func(rows []OddsPayload, err error) {
		if err == nil && len(rows) > 0 {
			candidates = append(candidates, rows)
		}
	}

	appendRows(client.FetchOddsSnapshot(ctx, fixtureID))
	appendRows(client.FetchOddsUpdates(ctx, fixtureID))

	now := time.Now().UTC().UnixMilli()
	if startTime > 0 && startTime <= now {
		appendRows(client.FetchOddsSnapshotAsOf(ctx, fixtureID, startTime-60*60*1000))
		appendRows(client.FetchOddsSnapshotAsOf(ctx, fixtureID, startTime-5*60*1000))
	}

	for _, rows := range candidates {
		if odds, ok := Parse1X2Odds(rows); ok {
			return odds, true
		}
	}
	if existing != nil && existing.Home > 0 {
		return *existing, true
	}
	return MatchOdds{}, false
}