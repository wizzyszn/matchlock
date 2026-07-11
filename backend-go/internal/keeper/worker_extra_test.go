package keeper

import (
	"context"
	"errors"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	solanapkg "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

type badValidationTxline struct{}

func (badValidationTxline) FetchScoreSnapshot(ctx context.Context, fixtureID int64) ([]txline.ScoreSnapshotRow, error) {
	return nil, errors.New("snapshot unavailable")
}

func (badValidationTxline) FetchStatValidation(ctx context.Context, fixtureID int64, seq int32, statKey uint32) (txline.StatValidation, error) {
	return txline.StatValidation{
		Summary: txline.StatValidationSummary{
			FixtureID:             fixtureID,
			EventStatsSubTreeRoot: "not-valid-hash",
		},
		EventStatRoot: "also-bad",
		StatToProve:   txline.ScoreStatResponse{Key: statKey, Value: 1, Period: 0},
	}, nil
}

type recordFailCache struct {
	memCache
}

func (r *recordFailCache) MarkSettled(ctx context.Context, rec cache.SettlementRecord) (bool, error) {
	if rec.TxSignature != "already-settled" {
		return false, errors.New("redis down")
	}
	return r.memCache.MarkSettled(ctx, rec)
}

func TestSettleMatchValidationMappingErrorQueuesWager(t *testing.T) {
	c := newMemCache()
	sc := &fakeSolana{}
	w := &Worker{
		Cache:      c,
		Txline:     badValidationTxline{},
		Solana:     sc,
		KeeperKey:  solana.NewWallet().PrivateKey,
		StatKey:    1002,
		AutoSettle: true,
	}
	update := finalScoreUpdate()
	if err := w.SettleMatch(context.Background(), update); err != nil {
		t.Fatalf("mapping error should queue per wager: %v", err)
	}
	if _, err := c.GetPendingSettlement(context.Background(), update.MatchID(), sc.storedWager.Pubkey.String()); err != nil {
		t.Fatalf("expected mapping failure to be queued: %v", err)
	}
}

func TestSettleOneRecordSettlementError(t *testing.T) {
	c := &recordFailCache{memCache: *newMemCache()}
	sc := &fakeSolana{}
	w := &Worker{Cache: c, Solana: sc, KeeperKey: solana.NewWallet().PrivateKey}
	wager := solanapkg.Wager{
		Pubkey: solana.NewWallet().PublicKey(),
		Status: solanapkg.WagerStatusMatched,
	}
	if err := w.settleOne(context.Background(), "17952170", wager, solanapkg.ValidateStatArgs{}, [32]byte{}, solanapkg.SideHome); err == nil {
		t.Fatal("expected record settlement error")
	}
}

func TestSettleOneIsSettledLookupError(t *testing.T) {
	c := &failLookupCache{memCache: *newMemCache()}
	w := &Worker{Cache: c, Solana: &fakeSolana{}, KeeperKey: solana.NewWallet().PrivateKey}
	if err := w.settleOne(context.Background(), "1", solanapkg.Wager{Pubkey: solana.NewWallet().PublicKey()}, solanapkg.ValidateStatArgs{}, [32]byte{}, solanapkg.SideHome); err == nil {
		t.Fatal("expected lookup error")
	}
}

type failLookupCache struct{ memCache }

func (f *failLookupCache) IsSettled(ctx context.Context, matchID, wagerPubkey string) (bool, error) {
	return false, errors.New("lookup failed")
}
