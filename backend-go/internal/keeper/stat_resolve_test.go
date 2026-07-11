package keeper

import (
	"context"
	"testing"

	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

type probeTxline struct {
	stats map[uint32]uint32
}

func (p probeTxline) FetchStatValidation(ctx context.Context, fixtureID int64, seq int32, statKey uint32) (txline.StatValidation, error) {
	return txline.StatValidation{
		StatToProve: txline.ScoreStatResponse{Key: statKey, Value: int32(p.stats[statKey]), Period: 0},
	}, nil
}

func (p probeTxline) FetchScoreSnapshot(ctx context.Context, fixtureID int64) ([]txline.ScoreSnapshotRow, error) {
	return nil, nil
}

func TestFetchDeclaredWinStatValidationUsesWagerOrientation(t *testing.T) {
	w := &Worker{Txline: probeTxline{stats: map[uint32]uint32{
		statKeyP1Win: 1,
		statKeyP2Win: 0,
	}}}
	v, key, err := w.fetchDeclaredWinStatValidation(context.Background(), 18179763, 941, chainsol.SideHome, true)
	if err != nil {
		t.Fatalf("fetchDeclaredWinStatValidation: %v", err)
	}
	if key != statKeyP1Win {
		t.Fatalf("stat key = %d, want %d", key, statKeyP1Win)
	}
	if v.StatToProve.Value != 1 {
		t.Fatalf("stat value = %d", v.StatToProve.Value)
	}
}

func TestFetchDeclaredWinStatValidationRejectsWrongOrientationStat(t *testing.T) {
	w := &Worker{Txline: probeTxline{stats: map[uint32]uint32{
		statKeyP1Win: 0,
		statKeyP2Win: 1,
	}}}
	if _, _, err := w.fetchDeclaredWinStatValidation(context.Background(), 18179763, 941, chainsol.SideHome, true); err == nil {
		t.Fatal("expected declared stat mismatch to fail")
	}
}
