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

func TestSettleMatchValidationMappingError(t *testing.T) {
	w := &Worker{
		Cache:      newMemCache(),
		Txline:     badValidationTxline{},
		Solana:     &fakeSolana{},
		KeeperKey:  solana.NewWallet().PrivateKey,
		StatKey:    1002,
		AutoSettle: true,
	}
	if err := w.SettleMatch(context.Background(), finalScoreUpdate()); err == nil {
		t.Fatal("expected validation mapping error")
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