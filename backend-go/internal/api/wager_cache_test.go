package api

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	chainsol "github.com/matchlock/backend-go/internal/solana"
)

func newTestCache(t *testing.T) (*cache.RedisStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	store, err := cache.NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	return store, mr
}

func pkStr() string {
	return solana.NewWallet().PublicKey().String()
}

func wagerCacheItem(matchID string, status uint8) cache.WagerCacheItem {
	maker := solana.NewWallet().PublicKey()
	return cache.WagerCacheItem{
		Pubkey:    solana.NewWallet().PublicKey().String(),
		Maker:     maker.String(),
		Taker:     maker.String(),
		MatchID:   matchID,
		MakerSide: 0,
		TakerSide: 1,
		Stake:     1_000_000,
		Status:    status,
	}
}

func TestCachedWagerIndexListAll(t *testing.T) {
	redisStore, _ := newTestCache(t)
	defer redisStore.Close()

	ctx := context.Background()
	for _, w := range []cache.WagerCacheItem{
		wagerCacheItem("1001", chainsol.WagerStatusOpen),
		wagerCacheItem("1001", chainsol.WagerStatusMatched),
		wagerCacheItem("1002", chainsol.WagerStatusOpen),
	} {
		if err := redisStore.SetWager(ctx, w); err != nil {
			t.Fatalf("SetWager: %v", err)
		}
	}

	idx := NewCachedWagerIndex(redisStore)
	wagers, err := idx.ListWagers(ctx, chainsol.WagerFilter{})
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(wagers) != 3 {
		t.Fatalf("len = %d, want 3", len(wagers))
	}
}

func TestCachedWagerIndexFilterByStatus(t *testing.T) {
	redisStore, _ := newTestCache(t)
	defer redisStore.Close()

	ctx := context.Background()
	for _, w := range []cache.WagerCacheItem{
		wagerCacheItem("1001", chainsol.WagerStatusOpen),
		wagerCacheItem("1001", chainsol.WagerStatusMatched),
		wagerCacheItem("1001", chainsol.WagerStatusSettled),
		wagerCacheItem("1002", chainsol.WagerStatusOpen),
	} {
		if err := redisStore.SetWager(ctx, w); err != nil {
			t.Fatalf("SetWager: %v", err)
		}
	}

	idx := NewCachedWagerIndex(redisStore)

	open := chainsol.WagerStatusOpen
	openWagers, err := idx.ListWagers(ctx, chainsol.WagerFilter{Status: &open})
	if err != nil {
		t.Fatalf("ListWagers open: %v", err)
	}
	if len(openWagers) != 2 {
		t.Fatalf("open count = %d, want 2", len(openWagers))
	}

	matched := chainsol.WagerStatusMatched
	matchedWagers, err := idx.ListWagers(ctx, chainsol.WagerFilter{Status: &matched})
	if err != nil {
		t.Fatalf("ListWagers matched: %v", err)
	}
	if len(matchedWagers) != 1 {
		t.Fatalf("matched count = %d, want 1", len(matchedWagers))
	}

	settled := chainsol.WagerStatusSettled
	settledWagers, err := idx.ListWagers(ctx, chainsol.WagerFilter{Status: &settled})
	if err != nil {
		t.Fatalf("ListWagers settled: %v", err)
	}
	if len(settledWagers) != 1 {
		t.Fatalf("settled count = %d, want 1", len(settledWagers))
	}

	cancelled := chainsol.WagerStatusCancelled
	cancelledWagers, err := idx.ListWagers(ctx, chainsol.WagerFilter{Status: &cancelled})
	if err != nil {
		t.Fatalf("ListWagers cancelled: %v", err)
	}
	if len(cancelledWagers) != 0 {
		t.Fatalf("cancelled count = %d, want 0", len(cancelledWagers))
	}
}

func TestCachedWagerIndexFilterByMatchID(t *testing.T) {
	redisStore, _ := newTestCache(t)
	defer redisStore.Close()

	ctx := context.Background()
	for _, w := range []cache.WagerCacheItem{
		wagerCacheItem("1001", chainsol.WagerStatusOpen),
		wagerCacheItem("1001", chainsol.WagerStatusOpen),
		wagerCacheItem("1002", chainsol.WagerStatusOpen),
	} {
		if err := redisStore.SetWager(ctx, w); err != nil {
			t.Fatalf("SetWager: %v", err)
		}
	}

	idx := NewCachedWagerIndex(redisStore)
	wagers, err := idx.ListWagers(ctx, chainsol.WagerFilter{MatchID: "1001"})
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(wagers) != 2 {
		t.Fatalf("len = %d, want 2", len(wagers))
	}

	wagers, err = idx.ListWagers(ctx, chainsol.WagerFilter{MatchID: "9999"})
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(wagers) != 0 {
		t.Fatalf("len = %d, want 0", len(wagers))
	}
}

func TestCachedWagerIndexCombinedFilter(t *testing.T) {
	redisStore, _ := newTestCache(t)
	defer redisStore.Close()

	ctx := context.Background()
	pks := make([]string, 3)
	for i, w := range []cache.WagerCacheItem{
		wagerCacheItem("1001", chainsol.WagerStatusOpen),
		wagerCacheItem("1001", chainsol.WagerStatusMatched),
		wagerCacheItem("1002", chainsol.WagerStatusOpen),
	} {
		if err := redisStore.SetWager(ctx, w); err != nil {
			t.Fatalf("SetWager: %v", err)
		}
		pks[i] = w.Pubkey
	}

	idx := NewCachedWagerIndex(redisStore)

	open := chainsol.WagerStatusOpen
	wagers, err := idx.ListWagers(ctx, chainsol.WagerFilter{Status: &open, MatchID: "1001"})
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(wagers) != 1 {
		t.Fatalf("len = %d, want 1 (open + match 1001)", len(wagers))
	}
	if wagers[0].Pubkey.String() != pks[0] {
		t.Fatalf("unexpected wager: %s, want %s", wagers[0].Pubkey.String(), pks[0])
	}
}

func TestCachedWagerIndexGetByPubkey(t *testing.T) {
	redisStore, _ := newTestCache(t)
	defer redisStore.Close()

	ctx := context.Background()
	targetPK := solana.NewWallet().PublicKey()
	makerPK := solana.NewWallet().PublicKey()
	takerPK := solana.NewWallet().PublicKey()

	item := cache.WagerCacheItem{
		Pubkey:    targetPK.String(),
		Maker:     makerPK.String(),
		Taker:     takerPK.String(),
		MatchID:   "1001",
		MakerSide: chainsol.SideHome,
		TakerSide: chainsol.SideAway,
		Stake:     2_500_000,
		Status:    chainsol.WagerStatusMatched,
		Bump:      200,
		VaultBump: 201,
	}
	if err := redisStore.SetWager(ctx, item); err != nil {
		t.Fatalf("SetWager: %v", err)
	}

	idx := NewCachedWagerIndex(redisStore)
	w, err := idx.GetWager(ctx, targetPK)
	if err != nil {
		t.Fatalf("GetWager: %v", err)
	}
	if w.Pubkey.String() != targetPK.String() || w.Maker.String() != makerPK.String() {
		t.Fatalf("pubkey/maker: got maker=%s want=%s", w.Maker.String(), makerPK.String())
	}
	if w.Taker.String() != takerPK.String() || w.MatchIDString() != "1001" {
		t.Fatalf("taker/match: got=%#v", w)
	}
	if w.MakerSide != chainsol.SideHome || w.TakerSide != chainsol.SideAway {
		t.Fatalf("sides: got=%#v", w)
	}
	if w.Stake != 2_500_000 || w.Status != chainsol.WagerStatusMatched {
		t.Fatalf("stake/status: got=%#v", w)
	}
	if w.Bump != 200 || w.VaultBump != 201 {
		t.Fatalf("bumps: got=%#v", w)
	}
}

func TestCachedWagerIndexGetNotFound(t *testing.T) {
	redisStore, _ := newTestCache(t)
	defer redisStore.Close()

	idx := NewCachedWagerIndex(redisStore)
	pk := solana.NewWallet().PublicKey()
	_, err := idx.GetWager(context.Background(), pk)
	if err == nil {
		t.Fatal("expected error for missing wager")
	}
}

func TestCachedWagerIndexRoundTripWithInvitedTaker(t *testing.T) {
	redisStore, _ := newTestCache(t)
	defer redisStore.Close()

	ctx := context.Background()
	targetPK := solana.NewWallet().PublicKey()
	makerPK := solana.NewWallet().PublicKey()
	invitedPK := solana.NewWallet().PublicKey()
	takerPK := solana.NewWallet().PublicKey()

	item := cache.WagerCacheItem{
		Pubkey:       targetPK.String(),
		Maker:        makerPK.String(),
		InvitedTaker: invitedPK.String(),
		Taker:        takerPK.String(),
		MatchID:      "3001",
		MakerSide:    chainsol.SideHome,
		TakerSide:    chainsol.SideAway,
		Stake:        1_000_000,
		Status:       chainsol.WagerStatusMatched,
		Bump:         10,
		VaultBump:    11,
	}
	if err := redisStore.SetWager(ctx, item); err != nil {
		t.Fatalf("SetWager: %v", err)
	}

	idx := NewCachedWagerIndex(redisStore)
	w, err := idx.GetWager(ctx, targetPK)
	if err != nil {
		t.Fatalf("GetWager: %v", err)
	}
	if w.InvitedTaker.String() != invitedPK.String() {
		t.Fatalf("InvitedTaker = %s, want %s", w.InvitedTaker.String(), invitedPK.String())
	}
	if !w.HasCounterparty() {
		t.Fatal("expected HasCounterparty=true for matched wager with taker")
	}
}

func TestCachedWagerIndexRoundTripNoInvitedTaker(t *testing.T) {
	redisStore, _ := newTestCache(t)
	defer redisStore.Close()

	ctx := context.Background()
	targetPK := solana.NewWallet().PublicKey()

	item := cache.WagerCacheItem{
		Pubkey:    targetPK.String(),
		Maker:     solana.NewWallet().PublicKey().String(),
		Taker:     solana.NewWallet().PublicKey().String(),
		MatchID:   "4001",
		MakerSide: chainsol.SideHome,
		TakerSide: chainsol.SideAway,
		Stake:     500_000,
		Status:    chainsol.WagerStatusOpen,
		Bump:      5,
		VaultBump: 6,
	}
	if err := redisStore.SetWager(ctx, item); err != nil {
		t.Fatalf("SetWager: %v", err)
	}

	idx := NewCachedWagerIndex(redisStore)
	w, err := idx.GetWager(ctx, targetPK)
	if err != nil {
		t.Fatalf("GetWager: %v", err)
	}
	if !w.InvitedTaker.IsZero() {
		t.Fatalf("expected zero InvitedTaker for V1 wager, got %s", w.InvitedTaker.String())
	}
}

func TestCachedWagerIndexEmptyList(t *testing.T) {
	redisStore, _ := newTestCache(t)
	defer redisStore.Close()

	idx := NewCachedWagerIndex(redisStore)
	wagers, err := idx.ListWagers(context.Background(), chainsol.WagerFilter{})
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(wagers) != 0 {
		t.Fatalf("len = %d, want 0", len(wagers))
	}
}

func TestCachedWagerIndexRedisDown(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := cache.NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	mr.Close()

	idx := NewCachedWagerIndex(store)
	_, err = idx.ListWagers(context.Background(), chainsol.WagerFilter{})
	if err == nil {
		t.Fatal("expected error on closed redis")
	}

	pk := solana.NewWallet().PublicKey()
	_, err = idx.GetWager(context.Background(), pk)
	if err == nil {
		t.Fatal("expected error on closed redis")
	}
	_ = store.Close()
}
