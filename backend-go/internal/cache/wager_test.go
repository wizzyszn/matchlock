package cache

import (
	"context"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func TestWagerSetAndGet(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	w := WagerCacheItem{
		Pubkey:    "ABC123",
		Maker:     "DEF456",
		Taker:     "GHI789",
		MatchID:   "100001",
		MakerSide: 0,
		TakerSide: 1,
		Stake:     1_000_000,
		Status:    1,
	}
	if err := store.SetWager(ctx, w); err != nil {
		t.Fatalf("SetWager: %v", err)
	}

	got, err := store.GetWager(ctx, "ABC123")
	if err != nil {
		t.Fatalf("GetWager: %v", err)
	}
	if got.Pubkey != "ABC123" || got.MatchID != "100001" || got.Status != 1 {
		t.Fatalf("got = %#v", got)
	}
}

func TestWagerGetNotFound(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	_, err = store.GetWager(context.Background(), "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for missing wager")
	}
}

func TestWagerList(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	wagers := []WagerCacheItem{
		{Pubkey: "W1", Maker: "M1", MatchID: "1001", Stake: 1_000_000, Status: 0},
		{Pubkey: "W2", Maker: "M2", MatchID: "1001", Stake: 2_000_000, Status: 1},
		{Pubkey: "W3", Maker: "M3", MatchID: "1002", Stake: 3_000_000, Status: 0},
	}
	for _, w := range wagers {
		if err := store.SetWager(ctx, w); err != nil {
			t.Fatalf("SetWager: %v", err)
		}
	}

	list, err := store.ListWagers(ctx)
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("len = %d, want 3", len(list))
	}

	found := map[string]bool{}
	for _, w := range list {
		found[w.Pubkey] = true
	}
	for _, pk := range []string{"W1", "W2", "W3"} {
		if !found[pk] {
			t.Fatalf("missing wager %s", pk)
		}
	}
}

func TestWagerDelete(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	w := WagerCacheItem{Pubkey: "DEL1", Maker: "M1", MatchID: "1001", Stake: 500_000, Status: 0}
	if err := store.SetWager(ctx, w); err != nil {
		t.Fatalf("SetWager: %v", err)
	}
	if err := store.DeleteWager(ctx, "DEL1"); err != nil {
		t.Fatalf("DeleteWager: %v", err)
	}

	_, err = store.GetWager(ctx, "DEL1")
	if err == nil {
		t.Fatal("expected error after delete")
	}

	list, err := store.ListWagers(ctx)
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("len = %d, want 0 after delete", len(list))
	}
}

func TestWagerUpdateOverwrites(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	w1 := WagerCacheItem{Pubkey: "UPD1", Maker: "M1", MatchID: "1001", Stake: 1_000_000, Status: 0}
	if err := store.SetWager(ctx, w1); err != nil {
		t.Fatalf("SetWager: %v", err)
	}

	w2 := w1
	w2.Status = 2
	w2.Stake = 2_000_000
	if err := store.SetWager(ctx, w2); err != nil {
		t.Fatalf("SetWager (update): %v", err)
	}

	got, err := store.GetWager(ctx, "UPD1")
	if err != nil {
		t.Fatalf("GetWager: %v", err)
	}
	if got.Status != 2 || got.Stake != 2_000_000 {
		t.Fatalf("got = %#v, want status=2 stake=2000000", got)
	}

	list, err := store.ListWagers(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %#v err=%v", list, err)
	}
}

func TestWagerFullFields(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	w := WagerCacheItem{
		Pubkey:       "FULL1",
		Maker:        "MAKER1",
		InvitedTaker: "INVITED1",
		Taker:        "TAKER1",
		MatchID:      "99999",
		MakerSide:    0,
		TakerSide:    1,
		Stake:        5_000_000,
		Status:       1,
		Bump:         255,
		VaultBump:    254,
	}
	if err := store.SetWager(ctx, w); err != nil {
		t.Fatalf("SetWager: %v", err)
	}

	got, err := store.GetWager(ctx, "FULL1")
	if err != nil {
		t.Fatalf("GetWager: %v", err)
	}
	if got.Pubkey != "FULL1" || got.Maker != "MAKER1" || got.InvitedTaker != "INVITED1" {
		t.Fatalf("pubkey/maker/invited: got=%#v", got)
	}
	if got.Taker != "TAKER1" || got.MakerSide != 0 || got.TakerSide != 1 {
		t.Fatalf("taker/sides: got=%#v", got)
	}
	if got.Stake != 5_000_000 || got.Status != 1 {
		t.Fatalf("stake/status: got=%#v", got)
	}
	if got.Bump != 255 || got.VaultBump != 254 {
		t.Fatalf("bumps: got=%#v", got)
	}
}

func TestWagerEmptyInvitedTaker(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	w := WagerCacheItem{
		Pubkey:  "NOINV",
		Maker:   "M1",
		MatchID: "1001",
		Stake:   100,
		Status:  0,
	}
	if err := store.SetWager(ctx, w); err != nil {
		t.Fatalf("SetWager: %v", err)
	}

	got, err := store.GetWager(ctx, "NOINV")
	if err != nil {
		t.Fatalf("GetWager: %v", err)
	}
	if got.InvitedTaker != "" {
		t.Fatalf("expected empty InvitedTaker, got %q", got.InvitedTaker)
	}
}

func TestWagerListEmpty(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	list, err := store.ListWagers(context.Background())
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("len = %d, want 0", len(list))
	}
}

func TestWagerGetDecodeError(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "matchlock:wager:bad"
	if err := store.client.Set(ctx, key, "not-json", 0).Err(); err != nil {
		t.Fatalf("seed bad json: %v", err)
	}
	if err := store.client.SAdd(ctx, wagerIndexKey, "bad").Err(); err != nil {
		t.Fatalf("seed index: %v", err)
	}
	if _, err := store.GetWager(ctx, "bad"); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestWagerListSkipsOrphans(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.client.SAdd(ctx, wagerIndexKey, "orphan").Err(); err != nil {
		t.Fatalf("seed orphan: %v", err)
	}
	list, err := store.ListWagers(ctx)
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}

func TestWagerConcurrentReadWrite(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	seed := WagerCacheItem{Pubkey: "CONCUR", Maker: "M1", MatchID: "1001", Stake: 1_000_000, Status: 0}
	if err := store.SetWager(ctx, seed); err != nil {
		t.Fatalf("seed SetWager: %v", err)
	}

	const workers = 16
	var wg sync.WaitGroup
	wg.Add(workers * 2)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			w := seed
			w.Stake = 500_000
			if err := store.SetWager(ctx, w); err != nil {
				t.Errorf("SetWager: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			if _, err := store.GetWager(ctx, "CONCUR"); err != nil {
				t.Errorf("GetWager: %v", err)
			}
			if _, err := store.ListWagers(ctx); err != nil {
				t.Errorf("ListWagers: %v", err)
			}
		}()
	}
	wg.Wait()

	got, err := store.GetWager(ctx, "CONCUR")
	if err != nil {
		t.Fatalf("GetWager: %v", err)
	}
	if got.Pubkey != "CONCUR" {
		t.Fatalf("got = %#v", got)
	}
}

func TestWagerSetError(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	mr.Close()

	if err := store.SetWager(context.Background(), WagerCacheItem{Pubkey: "ERR"}); err == nil {
		t.Fatal("expected set error on closed redis")
	}
	_ = store.Close()
}

func TestWagerGetError(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	mr.Close()

	if _, err := store.GetWager(context.Background(), "ERR"); err == nil {
		t.Fatal("expected get error on closed redis")
	}
	_ = store.Close()
}

func TestWagerDeleteError(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	mr.Close()

	if err := store.DeleteWager(context.Background(), "ERR"); err == nil {
		t.Fatal("expected delete error on closed redis")
	}
	_ = store.Close()
}

func TestWagerListSMembersError(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	mr.Close()

	if _, err := store.ListWagers(context.Background()); err == nil {
		t.Fatal("expected list error on closed redis")
	}
	_ = store.Close()
}

func TestWagerListMultipleWithDelete(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	for _, pk := range []string{"A1", "A2", "A3", "A4", "A5"} {
		if err := store.SetWager(ctx, WagerCacheItem{Pubkey: pk, Maker: "M", MatchID: "1001", Stake: 100, Status: 0}); err != nil {
			t.Fatalf("SetWager %s: %v", pk, err)
		}
	}

	if err := store.DeleteWager(ctx, "A2"); err != nil {
		t.Fatalf("DeleteWager A2: %v", err)
	}
	if err := store.DeleteWager(ctx, "A4"); err != nil {
		t.Fatalf("DeleteWager A4: %v", err)
	}

	list, err := store.ListWagers(ctx)
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("len = %d, want 3", len(list))
	}

	found := map[string]bool{}
	for _, w := range list {
		found[w.Pubkey] = true
	}
	for _, pk := range []string{"A1", "A3", "A5"} {
		if !found[pk] {
			t.Fatalf("missing %s after deletes", pk)
		}
	}
	if found["A2"] || found["A4"] {
		t.Fatal("deleted wagers still appear in list")
	}
}

func TestWagerSetMultipleAndIndexCount(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	for i := 0; i < 100; i++ {
		pk := "W" + string(rune('A'+i%26)) + string(rune('0'+i/10)) + string(rune('0'+i%10))
		if err := store.SetWager(ctx, WagerCacheItem{Pubkey: pk, Maker: "M", MatchID: "1001", Stake: uint64(i * 100), Status: 0}); err != nil {
			t.Fatalf("SetWager %s: %v", pk, err)
		}
	}

	list, err := store.ListWagers(ctx)
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(list) != 100 {
		t.Fatalf("len = %d, want 100", len(list))
	}
}

func TestWagerSetAndGetWithInvitedTakerEmpty(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	w := WagerCacheItem{
		Pubkey:  "V1WAGER",
		Maker:   "MAKER_V1",
		Taker:   "TAKER_V1",
		MatchID: "200001",
		Stake:   999_999_999,
		Status:  0,
	}
	if err := store.SetWager(ctx, w); err != nil {
		t.Fatalf("SetWager: %v", err)
	}

	got, err := store.GetWager(ctx, "V1WAGER")
	if err != nil {
		t.Fatalf("GetWager: %v", err)
	}
	if got.Pubkey != "V1WAGER" || got.InvitedTaker != "" || got.Taker != "TAKER_V1" {
		t.Fatalf("v1 wager round-trip failed: got=%#v", got)
	}
}
