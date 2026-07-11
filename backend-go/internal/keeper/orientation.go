package keeper

import (
	"context"

	"github.com/matchlock/backend-go/internal/txline"
)

// Participant1IsHomeFromRows reads home/away mapping from TxLINE score snapshot rows.
func Participant1IsHomeFromRows(rows []txline.ScoreSnapshotRow) (bool, bool) {
	row, err := txline.LatestScoreSnapshot(rows)
	if err != nil {
		return true, false
	}
	update, err := row.ToScoreUpdate()
	if err != nil {
		return true, false
	}
	return update.Participant1IsHome, true
}

func (w *Worker) hydratePendingScoreUpdate(
	ctx context.Context,
	itemFixtureID int64,
	itemGameState string,
	itemSeq int32,
	matchFallback func() txline.ScoreUpdate,
) txline.ScoreUpdate {
	if w.Txline != nil && itemFixtureID != 0 {
		rows, err := w.Txline.FetchScoreSnapshot(ctx, itemFixtureID)
		if err == nil {
			if row, err := latestSettlementSnapshot(rows); err == nil {
				if update, err := row.ToScoreUpdate(); err == nil {
					return update
				}
			}
		}
	}
	return matchFallback()
}

func latestSettlementSnapshot(rows []txline.ScoreSnapshotRow) (txline.ScoreSnapshotRow, error) {
	if row, err := txline.LatestFinalSnapshot(rows); err == nil {
		return row, nil
	}
	return txline.LatestScoreSnapshot(rows)
}
