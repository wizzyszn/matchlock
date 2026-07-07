package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestRedisStoreMatchLifecycle(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	match := Match{
		MatchID:   "17952170",
		FixtureID: 17952170,
		GameState: "F2",
		IsFinal:   true,
		UpdatedAt: now,
	}
	if err := store.UpsertMatch(ctx, match); err != nil {
		t.Fatalf("UpsertMatch: %v", err)
	}

	got, err := store.GetMatch(ctx, "17952170")
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if !got.IsFinal || got.MatchID != "17952170" {
		t.Fatalf("got = %#v", got)
	}

	list, err := store.ListMatches(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListMatches = %#v err=%v", list, err)
	}
}

func TestRedisStoreFinalAndSettledIdempotency(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	first, err := store.MarkFinalOnce(ctx, "m1")
	if err != nil || !first {
		t.Fatalf("first final = %v err=%v", first, err)
	}
	second, err := store.MarkFinalOnce(ctx, "m1")
	if err != nil || second {
		t.Fatalf("second final = %v err=%v", second, err)
	}

	rec := SettlementRecord{
		MatchID:     "m1",
		WagerPubkey: "wager1",
		TxSignature: "sig1",
		SettledAt:   time.Now().UTC(),
	}
	ok, err := store.MarkSettled(ctx, rec)
	if err != nil || !ok {
		t.Fatalf("first settled = %v err=%v", ok, err)
	}
	ok, err = store.MarkSettled(ctx, rec)
	if err != nil || ok {
		t.Fatalf("second settled = %v err=%v", ok, err)
	}
	settled, err := store.IsSettled(ctx, "m1", "wager1")
	if err != nil || !settled {
		t.Fatalf("is settled = %v err=%v", settled, err)
	}
}

func TestRedisStoreConcurrentReadWrite(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	seed := Match{
		MatchID:   "concurrent",
		FixtureID: 17952170,
		GameState: "HT",
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.UpsertMatch(ctx, seed); err != nil {
		t.Fatalf("seed UpsertMatch: %v", err)
	}

	const workers = 16
	var wg sync.WaitGroup
	wg.Add(workers * 2)

	for i := 0; i < workers; i++ {
		id := i
		go func() {
			defer wg.Done()
			match := seed
			match.Seq = int32(id)
			match.UpdatedAt = time.Now().UTC()
			if err := store.UpsertMatch(ctx, match); err != nil {
				t.Errorf("UpsertMatch: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			if _, err := store.GetMatch(ctx, "concurrent"); err != nil {
				t.Errorf("GetMatch: %v", err)
			}
			if _, err := store.ListMatches(ctx); err != nil {
				t.Errorf("ListMatches: %v", err)
			}
		}()
	}
	wg.Wait()

	got, err := store.GetMatch(ctx, "concurrent")
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if got.MatchID != "concurrent" {
		t.Fatalf("got = %#v", got)
	}
}

func TestNewRedisStoreErrors(t *testing.T) {
	if _, err := NewRedisStore(context.Background(), "not-a-redis-url"); err == nil {
		t.Fatal("expected parse error")
	}
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()
	if _, err := NewRedisStore(context.Background(), "redis://"+addr); err == nil {
		t.Fatal("expected ping error")
	}
}

func TestRedisStoreGetMatchDecodeError(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	key := "matchlock:match:bad"
	if err := store.client.Set(ctx, key, "not-json", 0).Err(); err != nil {
		t.Fatalf("seed bad json: %v", err)
	}
	if err := store.client.SAdd(ctx, matchIndexKey, "bad").Err(); err != nil {
		t.Fatalf("seed index: %v", err)
	}
	if _, err := store.GetMatch(ctx, "bad"); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestRedisStoreListMatchesPropagatesDecodeError(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.client.SAdd(ctx, matchIndexKey, "bad-decode").Err(); err != nil {
		t.Fatalf("seed index: %v", err)
	}
	if err := store.client.Set(ctx, "matchlock:match:bad-decode", "not-json", 0).Err(); err != nil {
		t.Fatalf("seed value: %v", err)
	}
	if _, err := store.ListMatches(ctx); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestRedisStoreMarkFinalAndSettledErrors(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	mr.Close()
	ctx := context.Background()
	if _, err := store.MarkFinalOnce(ctx, "m1"); err == nil {
		t.Fatal("expected mark final error")
	}
	if _, err := store.MarkSettled(ctx, SettlementRecord{MatchID: "m1", WagerPubkey: "w1"}); err == nil {
		t.Fatal("expected mark settled error")
	}
	if _, err := store.IsSettled(ctx, "m1", "w1"); err == nil {
		t.Fatal("expected is settled error")
	}
}

func TestRedisStoreListMatchesSkipsMissing(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.client.SAdd(ctx, matchIndexKey, "orphan").Err(); err != nil {
		t.Fatalf("seed orphan: %v", err)
	}
	list, err := store.ListMatches(ctx)
	if err != nil {
		t.Fatalf("ListMatches: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %#v", list)
	}
}

func TestRedisStorePingTimeout(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	mr.Close()
	if err := store.Ping(context.Background()); err == nil {
		t.Fatal("expected ping error on closed redis")
	}
	_ = store.Close()
}

func TestRedisStoreUpsertMatchError(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	mr.Close()
	if err := store.UpsertMatch(context.Background(), Match{MatchID: "x"}); err == nil {
		t.Fatal("expected upsert error")
	}
	_ = store.Close()
}

func TestRedisStoreIsSettledFalse(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()
	ok, err := store.IsSettled(context.Background(), "m1", "w1")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestRedisStoreListMatchesSMembersError(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	mr.Close()
	if _, err := store.ListMatches(context.Background()); err == nil {
		t.Fatal("expected list matches error")
	}
	_ = store.Close()
}

func TestRedisStorePing(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	if err := store.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
