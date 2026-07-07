package keeper

import (
	"context"
	"fmt"

	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

// fetchWinStatValidation loads the TxLINE proof for the match outcome at seq.
// Outcome stats (1001 draw, 1002 P1 win, 1003 P2 win) are authoritative; snapshot
// participant1IsHome metadata can disagree with them on some fixtures.
func (w *Worker) fetchWinStatValidation(
	ctx context.Context,
	fixtureID int64,
	seq int32,
	winningSide uint8,
	participant1IsHome bool,
) (txline.StatValidation, uint32, error) {
	if winningSide == chainsol.SideDraw {
		v, err := w.Txline.FetchStatValidation(ctx, fixtureID, seq, statKeyDraw)
		return v, statKeyDraw, err
	}

	for _, key := range []uint32{statKeyP1Win, statKeyP2Win} {
		v, err := w.Txline.FetchStatValidation(ctx, fixtureID, seq, key)
		if err != nil {
			continue
		}
		if v.StatToProve.Value > 0 {
			return v, key, nil
		}
	}

	key := StatKeyForWinningSide(winningSide, participant1IsHome)
	v, err := w.Txline.FetchStatValidation(ctx, fixtureID, seq, key)
	if err != nil {
		return txline.StatValidation{}, 0, fmt.Errorf("resolve outcome stat for fixture %d seq %d: %w", fixtureID, seq, err)
	}
	if v.StatToProve.Value == 0 {
		return txline.StatValidation{}, 0, fmt.Errorf("outcome stat %d is zero at fixture %d seq %d", key, fixtureID, seq)
	}
	return v, key, nil
}