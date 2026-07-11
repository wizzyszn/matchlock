package keeper

import (
	"context"
	"fmt"

	"github.com/matchlock/backend-go/internal/txline"
)

func (w *Worker) fetchDeclaredWinStatValidation(
	ctx context.Context,
	fixtureID int64,
	seq int32,
	winningSide uint8,
	participant1IsHome bool,
) (txline.StatValidation, uint32, error) {
	key := StatKeyForWinningSide(winningSide, participant1IsHome)
	v, err := w.Txline.FetchStatValidation(ctx, fixtureID, seq, key)
	if err != nil {
		return txline.StatValidation{}, 0, fmt.Errorf("fetch declared outcome stat %d for fixture %d seq %d: %w", key, fixtureID, seq, err)
	}
	if v.StatToProve.Value == 0 {
		return txline.StatValidation{}, 0, fmt.Errorf("declared outcome stat %d is zero at fixture %d seq %d", key, fixtureID, seq)
	}
	return v, key, nil
}
