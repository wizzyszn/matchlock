package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	chainsol "github.com/matchlock/backend-go/internal/solana"
)

type errCache struct {
	fakeCache
	listErr error
	getErr  error
}

func (e *errCache) ListMatches(ctx context.Context) ([]cache.Match, error) {
	if e.listErr != nil {
		return nil, e.listErr
	}
	return e.fakeCache.ListMatches(ctx)
}

func (e *errCache) GetMatch(ctx context.Context, matchID string) (cache.Match, error) {
	if e.getErr != nil {
		return cache.Match{}, e.getErr
	}
	return e.fakeCache.GetMatch(ctx, matchID)
}

type errWagers struct {
	fakeWagers
	listErr error
	getErr  error
}

func (e *errWagers) ListWagers(ctx context.Context, filter chainsol.WagerFilter) ([]chainsol.Wager, error) {
	if e.listErr != nil {
		return nil, e.listErr
	}
	return e.fakeWagers.ListWagers(ctx, filter)
}

func (e *errWagers) GetWager(ctx context.Context, pubkey solana.PublicKey) (chainsol.Wager, error) {
	if e.getErr != nil {
		return chainsol.Wager{}, e.getErr
	}
	return e.fakeWagers.GetWager(ctx, pubkey)
}

func assertErrorJSON(t *testing.T, rec *httptest.ResponseRecorder, code string) {
	t.Helper()
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v raw=%s", err, rec.Body.String())
	}
	if body.Code != code {
		t.Fatalf("code = %q, want %q body=%s", body.Code, code, rec.Body.String())
	}
}

func TestReadyzSuccess(t *testing.T) {
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ready" {
		t.Fatalf("body = %#v", body)
	}
}

func TestReadyzReportsStaleMatches(t *testing.T) {
	cacheStore := &fakeCache{matches: map[string]cache.Match{
		"18237038": {
			MatchID:   "18237038",
			FixtureID: 18237038,
			GameState: "live",
			StartTime: time.Now().Add(-5 * time.Hour).UnixMilli(),
		},
	}}
	handler := newTestHandler(t, cacheStore, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		StaleMatches int               `json:"stale_matches"`
		Checks       map[string]string `json:"checks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.StaleMatches != 1 || body.Checks["match_freshness"] != "degraded" {
		t.Fatalf("body = %#v", body)
	}
}

func TestGetMatchOK(t *testing.T) {
	now := time.Now().UTC()
	cacheStore := &fakeCache{matches: map[string]cache.Match{
		"17952170": {
			MatchID:   "17952170",
			FixtureID: 17952170,
			GameState: "F2",
			IsFinal:   true,
			UpdatedAt: now,
		},
	}}
	handler := newTestHandler(t, cacheStore, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/matches/17952170", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body MatchView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.MatchID != "17952170" || !body.IsFinal {
		t.Fatalf("body = %#v", body)
	}
}

func TestListMatchesCacheError(t *testing.T) {
	handler := newTestHandler(t, &errCache{
		fakeCache: fakeCache{matches: map[string]cache.Match{}},
		listErr:   errors.New("redis unavailable"),
	}, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/matches", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorJSON(t, rec, "CACHE_ERROR")
}

func TestListWagersInvalidStatus(t *testing.T) {
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/wagers?status=unknown", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorJSON(t, rec, "INVALID_QUERY")
}

func TestListWagersRPCError(t *testing.T) {
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &errWagers{
		fakeWagers: fakeWagers{},
		listErr:    errors.New("rpc down"),
	})
	req := httptest.NewRequest(http.MethodGet, "/wagers", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorJSON(t, rec, "RPC_ERROR")
}

func TestGetWagerNotFound(t *testing.T) {
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &fakeWagers{})
	pubkey := solana.NewWallet().PublicKey()
	req := httptest.NewRequest(http.MethodGet, "/wagers/"+pubkey.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorJSON(t, rec, "WAGER_NOT_FOUND")
}

func TestGetWagerInvalidPubkey(t *testing.T) {
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/wagers/not-a-pubkey", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorJSON(t, rec, "INVALID_PUBKEY")
}

func TestReadyzAllChecksFail(t *testing.T) {
	h := &handler{
		cache:  &fakeCache{matches: map[string]cache.Match{}},
		wagers: &fakeWagers{},
		redis:  fakeProbe{err: errors.New("redis down")},
		rpc:    fakeProbe{err: errors.New("rpc down")},
		txline: fakeProbe{err: errors.New("txline down")},
	}
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	h.readyz(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetMatchCacheError(t *testing.T) {
	handler := newTestHandler(t, &errCache{
		fakeCache: fakeCache{matches: map[string]cache.Match{}},
		getErr:    errors.New("redis timeout"),
	}, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/matches/17952170", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorJSON(t, rec, "CACHE_ERROR")
}

func TestListWagersStatusFilters(t *testing.T) {
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &fakeWagers{})
	for _, status := range []string{"open", "matched", "settled", "cancelled"} {
		req := httptest.NewRequest(http.MethodGet, "/wagers?status="+status, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %s code = %d", status, rec.Code)
		}
	}
}

func TestGetWagerRPCError(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &errWagers{
		getErr: errors.New("rpc down"),
	})
	req := httptest.NewRequest(http.MethodGet, "/wagers/"+pubkey.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rec.Code)
	}
	assertErrorJSON(t, rec, "RPC_ERROR")
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/matches", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unexpected cors header for unknown origin")
	}
}

func TestListWagersViaCachedWagerIndex(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := cache.NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	maker := solana.NewWallet().PublicKey().String()
	for _, w := range []cache.WagerCacheItem{
		{Pubkey: solana.NewWallet().PublicKey().String(), Maker: maker, Taker: maker, MatchID: "1001", MakerSide: 0, TakerSide: 1, Stake: 1_000_000, Status: chainsol.WagerStatusOpen},
		{Pubkey: solana.NewWallet().PublicKey().String(), Maker: maker, Taker: maker, MatchID: "1002", MakerSide: 0, TakerSide: 1, Stake: 2_000_000, Status: chainsol.WagerStatusMatched},
	} {
		if err := store.SetWager(ctx, w); err != nil {
			t.Fatalf("SetWager: %v", err)
		}
	}

	cachedIdx := NewCachedWagerIndex(store)
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, cachedIdx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wagers", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}
	var body []WagerView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if len(body) != 2 {
		t.Fatalf("len = %d, want 2", len(body))
	}
}

func TestGetWagerViaCachedWagerIndex(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := cache.NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	targetPK := solana.NewWallet().PublicKey().String()
	maker := solana.NewWallet().PublicKey().String()
	if err := store.SetWager(ctx, cache.WagerCacheItem{Pubkey: targetPK, Maker: maker, Taker: maker, MatchID: "1001", MakerSide: 1, TakerSide: 0, Stake: 3_000_000, Status: chainsol.WagerStatusMatched}); err != nil {
		t.Fatalf("SetWager: %v", err)
	}

	cachedIdx := NewCachedWagerIndex(store)
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, cachedIdx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wagers/"+targetPK, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}
	var body WagerView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if body.Pubkey != targetPK || body.Status != "matched" {
		t.Fatalf("body = %#v", body)
	}
}

func TestGetWagerViaCachedIndexNotFound(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := cache.NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	cachedIdx := NewCachedWagerIndex(store)
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, cachedIdx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wagers/"+solana.NewWallet().PublicKey().String(), nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 body=%s", rec.Code, rec.Body.String())
	}
	assertErrorJSON(t, rec, "WAGER_NOT_FOUND")
}

func TestListWagersViaCachedIndexWithStatusFilter(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := cache.NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	openPK := solana.NewWallet().PublicKey().String()
	maker := solana.NewWallet().PublicKey().String()
	for _, w := range []cache.WagerCacheItem{
		{Pubkey: openPK, Maker: maker, Taker: maker, MatchID: "1001", MakerSide: 0, TakerSide: 1, Stake: 100, Status: chainsol.WagerStatusOpen},
		{Pubkey: solana.NewWallet().PublicKey().String(), Maker: maker, Taker: maker, MatchID: "1001", MakerSide: 0, TakerSide: 1, Stake: 200, Status: chainsol.WagerStatusMatched},
		{Pubkey: solana.NewWallet().PublicKey().String(), Maker: maker, Taker: maker, MatchID: "1001", MakerSide: 0, TakerSide: 1, Stake: 300, Status: chainsol.WagerStatusSettled},
	} {
		if err := store.SetWager(ctx, w); err != nil {
			t.Fatalf("SetWager: %v", err)
		}
	}

	cachedIdx := NewCachedWagerIndex(store)
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, cachedIdx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wagers?status=open", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body []WagerView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("open count = %d, want 1", len(body))
	}
	if body[0].Pubkey != openPK {
		t.Fatalf("unexpected wager: %s, want %s", body[0].Pubkey, openPK)
	}
}

func TestListWagersViaCachedIndexWithMatchIDFilter(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := cache.NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	maker := solana.NewWallet().PublicKey().String()
	for _, w := range []cache.WagerCacheItem{
		{Pubkey: solana.NewWallet().PublicKey().String(), Maker: maker, Taker: maker, MatchID: "2001", MakerSide: 0, TakerSide: 1, Stake: 100, Status: chainsol.WagerStatusOpen},
		{Pubkey: solana.NewWallet().PublicKey().String(), Maker: maker, Taker: maker, MatchID: "2001", MakerSide: 0, TakerSide: 1, Stake: 200, Status: chainsol.WagerStatusMatched},
		{Pubkey: solana.NewWallet().PublicKey().String(), Maker: maker, Taker: maker, MatchID: "3001", MakerSide: 0, TakerSide: 1, Stake: 300, Status: chainsol.WagerStatusOpen},
	} {
		if err := store.SetWager(ctx, w); err != nil {
			t.Fatalf("SetWager: %v", err)
		}
	}

	cachedIdx := NewCachedWagerIndex(store)
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, cachedIdx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wagers?match_id=2001", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body []WagerView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("count = %d, want 2", len(body))
	}
}

func TestListWagersViaCachedIndexIsListableOnly(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := cache.NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	maker := solana.NewWallet().PublicKey().String()
	for _, w := range []cache.WagerCacheItem{
		{Pubkey: solana.NewWallet().PublicKey().String(), Maker: maker, Taker: maker, MatchID: "1001", MakerSide: 0, TakerSide: 1, Stake: 100, Status: chainsol.WagerStatusOpen},
		{Pubkey: solana.NewWallet().PublicKey().String(), Maker: maker, Taker: maker, MatchID: "1001", MakerSide: 0, TakerSide: 1, Stake: 200, Status: chainsol.WagerStatusMatched},
		{Pubkey: solana.NewWallet().PublicKey().String(), Maker: maker, Taker: maker, MatchID: "1001", MakerSide: 0, TakerSide: 1, Stake: 300, Status: chainsol.WagerStatusSettled},
		{Pubkey: solana.NewWallet().PublicKey().String(), Maker: maker, Taker: maker, MatchID: "1001", MakerSide: 0, TakerSide: 1, Stake: 400, Status: chainsol.WagerStatusCancelled},
	} {
		if err := store.SetWager(ctx, w); err != nil {
			t.Fatalf("SetWager: %v", err)
		}
	}

	cachedIdx := NewCachedWagerIndex(store)
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, cachedIdx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wagers", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body []WagerView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("listable count = %d, want 2 (only open+matched should be returned)", len(body))
	}
	for _, w := range body {
		if w.Status != "open" && w.Status != "matched" {
			t.Fatalf("unexpected non-listable wager in response: status=%s", w.Status)
		}
	}
}

func TestListWagersViaCachedIndexRPCErrorFallback(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := cache.NewRedisStore(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	mr.Close()

	cachedIdx := NewCachedWagerIndex(store)
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, cachedIdx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wagers", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 body=%s", rec.Code, rec.Body.String())
	}
	assertErrorJSON(t, rec, "RPC_ERROR")
	_ = store.Close()
}
