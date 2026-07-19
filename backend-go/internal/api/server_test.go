package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

type fakeCache struct {
	matches map[string]cache.Match
}

func (f *fakeCache) Ping(ctx context.Context) error { return nil }

func (f *fakeCache) UpsertMatch(ctx context.Context, match cache.Match) error {
	f.matches[match.MatchID] = match
	return nil
}

func (f *fakeCache) GetMatch(ctx context.Context, matchID string) (cache.Match, error) {
	match, ok := f.matches[matchID]
	if !ok {
		return cache.Match{}, errors.New("get match " + matchID + ": redis: nil")
	}
	return match, nil
}

func (f *fakeCache) ListMatches(ctx context.Context) ([]cache.Match, error) {
	out := make([]cache.Match, 0, len(f.matches))
	for _, match := range f.matches {
		out = append(out, match)
	}
	return out, nil
}

func (f *fakeCache) MarkFinalOnce(ctx context.Context, matchID string) (bool, error) {
	return true, nil
}

func (f *fakeCache) MarkSettled(ctx context.Context, rec cache.SettlementRecord) (bool, error) {
	return true, nil
}

func (f *fakeCache) IsSettled(ctx context.Context, matchID, wagerPubkey string) (bool, error) {
	return false, nil
}

func (f *fakeCache) GetSettlement(ctx context.Context, matchID, wagerPubkey string) (cache.SettlementRecord, error) {
	return cache.SettlementRecord{}, cache.ErrSettlementNotFound
}

func (f *fakeCache) EnqueuePendingSettlement(ctx context.Context, item cache.PendingSettlement) error {
	return nil
}

func (f *fakeCache) GetPendingSettlement(ctx context.Context, matchID, wagerPubkey string) (cache.PendingSettlement, error) {
	return cache.PendingSettlement{}, cache.ErrPendingSettlementNotFound
}

func (f *fakeCache) UpdatePendingSettlement(ctx context.Context, item cache.PendingSettlement) error {
	return nil
}

func (f *fakeCache) RemovePendingSettlement(ctx context.Context, matchID, wagerPubkey string) error {
	return nil
}

func (f *fakeCache) ListDuePendingSettlements(ctx context.Context, dueBefore time.Time, limit int) ([]cache.PendingSettlement, error) {
	return nil, nil
}

func (f *fakeCache) CountPendingSettlements(ctx context.Context) (int64, error) {
	return 0, nil
}

func (f *fakeCache) PublishMatchUpdate(ctx context.Context, match cache.Match) error {
	return nil
}

type fakeWagers struct {
	wagers []chainsol.Wager
}

func (f *fakeWagers) ListWagers(ctx context.Context, filter chainsol.WagerFilter) ([]chainsol.Wager, error) {
	out := make([]chainsol.Wager, 0, len(f.wagers))
	for _, wager := range f.wagers {
		if filter.Status != nil && wager.Status != *filter.Status {
			continue
		}
		if filter.MatchID != "" && wager.MatchIDString() != filter.MatchID {
			continue
		}
		if filter.Wallet != "" &&
			wager.Maker.String() != filter.Wallet &&
			wager.Taker.String() != filter.Wallet {
			continue
		}
		out = append(out, wager)
	}
	return out, nil
}

func (f *fakeWagers) GetWager(ctx context.Context, pubkey solana.PublicKey) (chainsol.Wager, error) {
	for _, wager := range f.wagers {
		if wager.Pubkey.Equals(pubkey) {
			return wager, nil
		}
	}
	return chainsol.Wager{}, errors.New("wager account " + pubkey.String() + " not found")
}

type fakeProbe struct {
	err error
}

func (f fakeProbe) Ping(ctx context.Context) error { return f.err }

type settlementSnapshotTxline struct {
	rows []txline.ScoreSnapshotRow
}

func (s settlementSnapshotTxline) FetchScoreSnapshot(ctx context.Context, fixtureID int64) ([]txline.ScoreSnapshotRow, error) {
	return s.rows, nil
}

func (s settlementSnapshotTxline) FetchStatValidation(ctx context.Context, fixtureID int64, seq int32, statKey uint32) (txline.StatValidation, error) {
	return txline.StatValidation{}, nil
}

func newTestHandler(t *testing.T, cacheStore cache.Store, wagers WagerIndex, probes ...fakeProbe) http.Handler {
	t.Helper()
	h := &handler{
		cache:  cacheStore,
		wagers: wagers,
		redis:  fakeProbe{},
		rpc:    fakeProbe{},
		txline: fakeProbe{},
	}
	if len(probes) > 0 {
		h.redis = probes[0]
	}
	return corsMiddleware([]string{"http://localhost:5173"})(newMux(h))
}

func TestHealthz(t *testing.T) {
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %#v", body)
	}
}

func TestListMatches(t *testing.T) {
	cacheStore := &fakeCache{matches: map[string]cache.Match{
		"17952170": {
			MatchID:   "17952170",
			FixtureID: 17952170,
			GameState: "HT",
			IsFinal:   false,
			UpdatedAt: time.Now().UTC(),
		},
	}}
	handler := newTestHandler(t, cacheStore, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/matches", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body []MatchView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 || body[0].MatchID != "17952170" {
		t.Fatalf("body = %#v", body)
	}
}

func TestGetMatchNotFound(t *testing.T) {
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/matches/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestListWagersFiltersSettled(t *testing.T) {
	openWager := chainsol.Wager{
		Pubkey:     solana.NewWallet().PublicKey(),
		Maker:      solana.NewWallet().PublicKey(),
		Taker:      solana.NewWallet().PublicKey(),
		MatchIDLen: 8,
		MakerSide:  chainsol.SideHome,
		Stake:      1_000_000,
		Status:     chainsol.WagerStatusOpen,
	}
	copy(openWager.MatchID[:], []byte("17952170"))
	wagers := &fakeWagers{
		wagers: []chainsol.Wager{
			openWager,
			{
				Pubkey:     solana.NewWallet().PublicKey(),
				Status:     chainsol.WagerStatusSettled,
				MatchIDLen: 1,
			},
		},
	}
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, wagers)
	req := httptest.NewRequest(http.MethodGet, "/wagers", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var body []WagerView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("len(body) = %d, want 1 open wager", len(body))
	}
}

func TestGetWager(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	wager := chainsol.Wager{
		Pubkey:     pubkey,
		Maker:      solana.NewWallet().PublicKey(),
		Taker:      solana.NewWallet().PublicKey(),
		MatchIDLen: 8,
		MakerSide:  chainsol.SideAway,
		Stake:      2_000_000,
		Status:     chainsol.WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte("17952170"))
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &fakeWagers{wagers: []chainsol.Wager{wager}})

	req := httptest.NewRequest(http.MethodGet, "/wagers/"+pubkey.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body WagerView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Pubkey != pubkey.String() || body.Status != "matched" {
		t.Fatalf("body = %#v", body)
	}
}

func TestListWagerHistoryFiltersOnBackend(t *testing.T) {
	wallet := solana.NewWallet().PublicKey()
	opponent := solana.NewWallet().PublicKey()
	matchID := "17952170"
	homeGoals, awayGoals := int32(3), int32(1)

	wonWager := chainsol.Wager{
		Pubkey:     solana.NewWallet().PublicKey(),
		Maker:      wallet,
		Taker:      opponent,
		MatchIDLen: uint8(len(matchID)),
		MakerSide:  chainsol.SideHome,
		TakerSide:  chainsol.SideAway,
		Stake:      1_000_000,
		Status:     chainsol.WagerStatusSettled,
	}
	copy(wonWager.MatchID[:], []byte(matchID))

	openWager := chainsol.Wager{
		Pubkey:     solana.NewWallet().PublicKey(),
		Maker:      wallet,
		Taker:      chainsol.SystemProgramID,
		MatchIDLen: uint8(len(matchID)),
		MakerSide:  chainsol.SideDraw,
		Stake:      500_000,
		Status:     chainsol.WagerStatusOpen,
	}
	copy(openWager.MatchID[:], []byte(matchID))

	cacheStore := &fakeCache{matches: map[string]cache.Match{
		matchID: {
			MatchID:   matchID,
			FixtureID: 17952170,
			GameState: "FT",
			IsFinal:   true,
			HomeGoals: &homeGoals,
			AwayGoals: &awayGoals,
			StartTime: 1_720_000_000_000,
			UpdatedAt: time.Now().UTC(),
		},
	}}
	handler := newTestHandler(t, cacheStore, &fakeWagers{wagers: []chainsol.Wager{wonWager, openWager}})

	req := httptest.NewRequest(
		http.MethodGet,
		"/wagers/history?wallet="+wallet.String()+"&settlement_status=settled&outcome=won&from=1719999999999&to=1720000000000",
		nil,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var body WagerHistoryPageView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(body.Entries))
	}
	if body.Total != 1 || body.Offset != 0 || body.Limit != 25 || body.HasMore {
		t.Fatalf("page = %#v", body)
	}
	if body.Entries[0].Outcome != historyOutcomeWon || body.Entries[0].SettlementStatus != historySettlementSettled {
		t.Fatalf("history entry = %#v", body.Entries[0])
	}
	if body.Entries[0].Match == nil || body.Entries[0].Match.MatchID != matchID {
		t.Fatalf("match = %#v", body.Entries[0].Match)
	}
}

func TestListWagerHistoryRequiresWallet(t *testing.T) {
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &fakeWagers{})
	req := httptest.NewRequest(http.MethodGet, "/wagers/history", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestListWagerHistoryPaginates(t *testing.T) {
	wallet := solana.NewWallet().PublicKey()
	opponent := solana.NewWallet().PublicKey()
	matchID := "2001"
	homeGoals, awayGoals := int32(2), int32(0)

	first := chainsol.Wager{
		Pubkey:     solana.NewWallet().PublicKey(),
		Maker:      wallet,
		Taker:      opponent,
		MatchIDLen: uint8(len(matchID)),
		MakerSide:  chainsol.SideHome,
		TakerSide:  chainsol.SideAway,
		Stake:      1_000_000,
		Status:     chainsol.WagerStatusSettled,
	}
	copy(first.MatchID[:], []byte(matchID))
	second := chainsol.Wager{
		Pubkey:     solana.NewWallet().PublicKey(),
		Maker:      wallet,
		Taker:      opponent,
		MatchIDLen: uint8(len(matchID)),
		MakerSide:  chainsol.SideHome,
		TakerSide:  chainsol.SideAway,
		Stake:      2_000_000,
		Status:     chainsol.WagerStatusSettled,
	}
	copy(second.MatchID[:], []byte(matchID))

	cacheStore := &fakeCache{matches: map[string]cache.Match{
		matchID: {
			MatchID:   matchID,
			FixtureID: 2001,
			GameState: "FT",
			IsFinal:   true,
			HomeGoals: &homeGoals,
			AwayGoals: &awayGoals,
			StartTime: 1_720_000_000_000,
			UpdatedAt: time.Now().UTC(),
		},
	}}
	handler := newTestHandler(t, cacheStore, &fakeWagers{wagers: []chainsol.Wager{first, second}})

	req := httptest.NewRequest(
		http.MethodGet,
		"/wagers/history?wallet="+wallet.String()+"&limit=1&offset=1",
		nil,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var body WagerHistoryPageView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Entries) != 1 || body.Total != 2 || body.Offset != 1 || body.Limit != 1 || body.HasMore {
		t.Fatalf("page = %#v", body)
	}
}

func TestReadyzFailure(t *testing.T) {
	h := &handler{
		cache:  &fakeCache{matches: map[string]cache.Match{}},
		wagers: &fakeWagers{},
		redis:  fakeProbe{err: errors.New("redis down")},
		rpc:    fakeProbe{},
		txline: fakeProbe{},
	}
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	h.readyz(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestGetWagerSettlementMatchedLive(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	wager := chainsol.Wager{
		Pubkey:     pubkey,
		Maker:      solana.NewWallet().PublicKey(),
		Taker:      solana.NewWallet().PublicKey(),
		MatchIDLen: 8,
		MakerSide:  chainsol.SideHome,
		Stake:      1_000_000,
		Status:     chainsol.WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte("17952170"))
	cacheStore := &fakeCache{matches: map[string]cache.Match{
		"17952170": {
			MatchID:   "17952170",
			FixtureID: 17952170,
			GameState: "HT",
			IsFinal:   false,
			UpdatedAt: time.Now().UTC(),
		},
	}}
	handler := newTestHandler(t, cacheStore, &fakeWagers{wagers: []chainsol.Wager{wager}})

	req := httptest.NewRequest(http.MethodGet, "/wagers/"+pubkey.String()+"/settlement", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body SettlementStatusView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.State != settlementStateMatchLive {
		t.Fatalf("state = %q, want %q", body.State, settlementStateMatchLive)
	}
	if body.Message == "" {
		t.Fatal("expected user-facing settlement message")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	for _, key := range []string{"last_error", "pending_attempts", "next_retry_at"} {
		if _, ok := raw[key]; ok {
			t.Fatalf("internal field %q leaked to API", key)
		}
	}
}

func TestGetWagerSettlementUnknownFinalSourceIsUnverified(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	wager := chainsol.Wager{
		Pubkey:     pubkey,
		Maker:      solana.NewWallet().PublicKey(),
		Taker:      solana.NewWallet().PublicKey(),
		MatchIDLen: 8,
		MakerSide:  chainsol.SideHome,
		Stake:      1_000_000,
		Status:     chainsol.WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte("17952170"))
	home, away := int32(2), int32(1)
	cacheStore := &fakeCache{matches: map[string]cache.Match{
		"17952170": {
			MatchID:   "17952170",
			FixtureID: 17952170,
			GameState: "FT",
			IsFinal:   true,
			HomeGoals: &home,
			AwayGoals: &away,
			Seq:       100,
			UpdatedAt: time.Now().UTC(),
		},
	}}
	handler := newTestHandler(t, cacheStore, &fakeWagers{wagers: []chainsol.Wager{wager}})

	req := httptest.NewRequest(http.MethodGet, "/wagers/"+pubkey.String()+"/settlement", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body SettlementStatusView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.State != settlementStateMatchEndedUnverified {
		t.Fatalf("state = %q, want %q", body.State, settlementStateMatchEndedUnverified)
	}
}

func TestGetWagerSettlementRefreshesVerifiedFinal(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	wager := chainsol.Wager{
		Pubkey:     pubkey,
		Maker:      solana.NewWallet().PublicKey(),
		Taker:      solana.NewWallet().PublicKey(),
		MatchIDLen: 8,
		MakerSide:  chainsol.SideHome,
		TakerSide:  chainsol.SideAway,
		Stake:      1_000_000,
		Status:     chainsol.WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte("17952170"))
	home, away := int32(0), int32(0)
	cacheStore := &fakeCache{matches: map[string]cache.Match{
		"17952170": {
			MatchID:     "17952170",
			FixtureID:   17952170,
			GameState:   "FT",
			IsFinal:     true,
			FinalSource: cache.FinalSourceInferred,
			HomeGoals:   &home,
			AwayGoals:   &away,
			Seq:         1,
			UpdatedAt:   time.Now().UTC(),
		},
	}}
	h := &handler{
		cache:  cacheStore,
		wagers: &fakeWagers{wagers: []chainsol.Wager{wager}},
		redis:  fakeProbe{},
		rpc:    fakeProbe{},
		txline: fakeProbe{},
		txlineData: settlementSnapshotTxline{rows: []txline.ScoreSnapshotRow{{
			FixtureID:          17952170,
			GameState:          "F2",
			Seq:                42,
			Participant1IsHome: true,
			ScoreSoccer: &txline.SoccerFixtureScore{
				Participant1: txline.SoccerTotalScore{Goals: 2},
				Participant2: txline.SoccerTotalScore{Goals: 0},
			},
		}}},
	}
	handler := corsMiddleware([]string{"http://localhost:5173"})(newMux(h))

	req := httptest.NewRequest(http.MethodGet, "/wagers/"+pubkey.String()+"/settlement", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body SettlementStatusView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.State != settlementStateClaimable {
		t.Fatalf("state = %q, want %q body=%s", body.State, settlementStateClaimable, rec.Body.String())
	}
	got, err := cacheStore.GetMatch(context.Background(), "17952170")
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if got.FinalSource != cache.FinalSourceTxline || got.Seq != 42 {
		t.Fatalf("refreshed match = %#v", got)
	}
}

func TestGetWagerSettlementHydratesFinalAfterCacheLoss(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	wager := chainsol.Wager{
		Pubkey:     pubkey,
		Maker:      solana.NewWallet().PublicKey(),
		Taker:      solana.NewWallet().PublicKey(),
		MatchIDLen: 8,
		MakerSide:  chainsol.SideHome,
		TakerSide:  chainsol.SideAway,
		Stake:      1_000_000,
		Status:     chainsol.WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte("18237038"))

	cacheStore := &fakeCache{matches: map[string]cache.Match{}}
	h := &handler{
		cache:  cacheStore,
		wagers: &fakeWagers{wagers: []chainsol.Wager{wager}},
		redis:  fakeProbe{},
		rpc:    fakeProbe{},
		txline: fakeProbe{},
		txlineData: settlementSnapshotTxline{rows: []txline.ScoreSnapshotRow{{
			FixtureIDAlt:     18237038,
			GameStateAlt:     "scheduled",
			ActionAlt:        "game_finalised",
			StatusIDAlt:      json.RawMessage("100"),
			SeqAlt:           1026,
			Participant1Home: true,
			Score: &txline.SnapshotScore{
				Participant1: txline.SoccerTotalScore{Total: &txline.SoccerScore{Goals: 0}},
				Participant2: txline.SoccerTotalScore{Total: &txline.SoccerScore{Goals: 2}},
			},
		}}},
	}
	handler := corsMiddleware([]string{"http://localhost:5173"})(newMux(h))

	req := httptest.NewRequest(http.MethodGet, "/wagers/"+pubkey.String()+"/settlement", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body SettlementStatusView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.State != settlementStateClaimable {
		t.Fatalf("state = %q, want %q body=%s", body.State, settlementStateClaimable, rec.Body.String())
	}
	got, err := cacheStore.GetMatch(context.Background(), "18237038")
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if !got.IsFinal || got.FinalSource != cache.FinalSourceTxline || got.Seq != 1026 {
		t.Fatalf("hydrated match = %#v", got)
	}
}

func TestGetWagerSettlementMarksUnselectedOutcomeRefundable(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	wager := chainsol.Wager{
		Pubkey:     pubkey,
		Maker:      solana.NewWallet().PublicKey(),
		Taker:      solana.NewWallet().PublicKey(),
		MatchIDLen: 8,
		MakerSide:  chainsol.SideHome,
		TakerSide:  chainsol.SideDraw,
		Stake:      1_000_000,
		Status:     chainsol.WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte("18241006"))
	home, away := int32(1), int32(2)
	cacheStore := &fakeCache{matches: map[string]cache.Match{
		"18241006": {
			MatchID:     "18241006",
			FixtureID:   18241006,
			GameState:   "FT",
			IsFinal:     true,
			FinalSource: cache.FinalSourceTxline,
			HomeGoals:   &home,
			AwayGoals:   &away,
			Seq:         962,
			UpdatedAt:   time.Now().UTC(),
		},
	}}
	handler := newTestHandler(t, cacheStore, &fakeWagers{wagers: []chainsol.Wager{wager}})

	req := httptest.NewRequest(http.MethodGet, "/wagers/"+pubkey.String()+"/settlement", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body SettlementStatusView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.State != settlementStateRefundable {
		t.Fatalf("state = %q, want %q body=%s", body.State, settlementStateRefundable, rec.Body.String())
	}
}

func TestGetWagerSettlementKeepsEligibleLiveMatchLive(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	wager := chainsol.Wager{
		Pubkey:     pubkey,
		Maker:      solana.NewWallet().PublicKey(),
		Taker:      solana.NewWallet().PublicKey(),
		MatchIDLen: 8,
		MakerSide:  chainsol.SideHome,
		TakerSide:  chainsol.SideAway,
		Stake:      1_000_000,
		Status:     chainsol.WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte("18213979"))
	cacheStore := &fakeCache{matches: map[string]cache.Match{
		"18213979": {
			MatchID:   "18213979",
			FixtureID: 18213979,
			GameState: "HT",
			StartTime: time.Now().Add(-2 * time.Hour).UnixMilli(),
			Seq:       77,
			UpdatedAt: time.Now().UTC(),
		},
	}}
	h := &handler{
		cache:  cacheStore,
		wagers: &fakeWagers{wagers: []chainsol.Wager{wager}},
		redis:  fakeProbe{},
		rpc:    fakeProbe{},
		txline: fakeProbe{},
		txlineData: settlementSnapshotTxline{rows: []txline.ScoreSnapshotRow{{
			FixtureID:          18213979,
			GameState:          "HT",
			Seq:                77,
			Participant1IsHome: true,
			ScoreSoccer: &txline.SoccerFixtureScore{
				Participant1: txline.SoccerTotalScore{Goals: 1},
				Participant2: txline.SoccerTotalScore{Goals: 1},
			},
		}}},
	}
	handler := corsMiddleware([]string{"http://localhost:5173"})(newMux(h))

	req := httptest.NewRequest(http.MethodGet, "/wagers/"+pubkey.String()+"/settlement", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body SettlementStatusView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.State != settlementStateMatchLive {
		t.Fatalf("state = %q, want %q body=%s", body.State, settlementStateMatchLive, rec.Body.String())
	}
	got, err := cacheStore.GetMatch(context.Background(), "18213979")
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if got.IsFinal {
		t.Fatalf("live match was incorrectly finalized: %#v", got)
	}
}

func TestGetWagerSettlementDoesNotPresentExpiredMatchAsLive(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	wager := chainsol.Wager{
		Pubkey:     pubkey,
		Maker:      solana.NewWallet().PublicKey(),
		Taker:      solana.NewWallet().PublicKey(),
		MatchIDLen: 8,
		MakerSide:  chainsol.SideHome,
		TakerSide:  chainsol.SideAway,
		Stake:      1_000_000,
		Status:     chainsol.WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte("18237038"))
	cacheStore := &fakeCache{matches: map[string]cache.Match{
		"18237038": {
			MatchID:   "18237038",
			FixtureID: 18237038,
			GameState: "live",
			StartTime: time.Now().Add(-5 * time.Hour).UnixMilli(),
			Seq:       1026,
			UpdatedAt: time.Now().UTC(),
		},
	}}
	handler := newTestHandler(t, cacheStore, &fakeWagers{wagers: []chainsol.Wager{wager}})

	req := httptest.NewRequest(http.MethodGet, "/wagers/"+pubkey.String()+"/settlement", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body SettlementStatusView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.State != settlementStateMatchEndedUnverified {
		t.Fatalf("state = %q, want %q body=%s", body.State, settlementStateMatchEndedUnverified, rec.Body.String())
	}
}

func TestCORSAllowsFrontendOrigin(t *testing.T) {
	handler := newTestHandler(t, &fakeCache{matches: map[string]cache.Match{}}, &fakeWagers{})
	req := httptest.NewRequest(http.MethodOptions, "/matches", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("cors origin = %q", got)
	}
}
