package keeper

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	solanapkg "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

type memCache struct {
	matches map[string]cache.Match
	finals  map[string]bool
	settled map[string]cache.SettlementRecord
	pending map[string]cache.PendingSettlement
}

func newMemCache() *memCache {
	return &memCache{
		matches: make(map[string]cache.Match),
		finals:  make(map[string]bool),
		settled: make(map[string]cache.SettlementRecord),
		pending: make(map[string]cache.PendingSettlement),
	}
}

func (m *memCache) Ping(ctx context.Context) error { return nil }

func (m *memCache) UpsertMatch(ctx context.Context, match cache.Match) error {
	m.matches[match.MatchID] = match
	return nil
}
func (m *memCache) GetMatch(ctx context.Context, matchID string) (cache.Match, error) {
	return m.matches[matchID], nil
}
func (m *memCache) ListMatches(ctx context.Context) ([]cache.Match, error) {
	out := make([]cache.Match, 0, len(m.matches))
	for _, v := range m.matches {
		out = append(out, v)
	}
	return out, nil
}
func (m *memCache) MarkFinalOnce(ctx context.Context, matchID string) (bool, error) {
	if m.finals[matchID] {
		return false, nil
	}
	m.finals[matchID] = true
	return true, nil
}
func (m *memCache) MarkSettled(ctx context.Context, rec cache.SettlementRecord) (bool, error) {
	key := rec.MatchID + ":" + rec.WagerPubkey
	if _, ok := m.settled[key]; ok {
		return false, nil
	}
	m.settled[key] = rec
	return true, nil
}
func (m *memCache) IsSettled(ctx context.Context, matchID, wagerPubkey string) (bool, error) {
	_, ok := m.settled[matchID+":"+wagerPubkey]
	return ok, nil
}
func (m *memCache) GetSettlement(ctx context.Context, matchID, wagerPubkey string) (cache.SettlementRecord, error) {
	rec, ok := m.settled[matchID+":"+wagerPubkey]
	if !ok {
		return cache.SettlementRecord{}, cache.ErrSettlementNotFound
	}
	return rec, nil
}
func (m *memCache) EnqueuePendingSettlement(ctx context.Context, item cache.PendingSettlement) error {
	m.pending[item.MatchID+":"+item.WagerPubkey] = item
	return nil
}
func (m *memCache) GetPendingSettlement(ctx context.Context, matchID, wagerPubkey string) (cache.PendingSettlement, error) {
	item, ok := m.pending[matchID+":"+wagerPubkey]
	if !ok {
		return cache.PendingSettlement{}, cache.ErrPendingSettlementNotFound
	}
	return item, nil
}
func (m *memCache) UpdatePendingSettlement(ctx context.Context, item cache.PendingSettlement) error {
	m.pending[item.MatchID+":"+item.WagerPubkey] = item
	return nil
}
func (m *memCache) RemovePendingSettlement(ctx context.Context, matchID, wagerPubkey string) error {
	delete(m.pending, matchID+":"+wagerPubkey)
	return nil
}
func (m *memCache) ListDuePendingSettlements(ctx context.Context, dueBefore time.Time, limit int) ([]cache.PendingSettlement, error) {
	out := make([]cache.PendingSettlement, 0)
	for _, item := range m.pending {
		if !item.NextRetryAt.After(dueBefore) {
			out = append(out, item)
		}
	}
	return out, nil
}
func (m *memCache) CountPendingSettlements(ctx context.Context) (int64, error) {
	return int64(len(m.pending)), nil
}
func (m *memCache) PublishMatchUpdate(ctx context.Context, match cache.Match) error {
	return nil
}

type fakeTxline struct {
	calls         int
	err           error
	lastFixtureID int64
	lastSeq       int32
	lastStatKey   uint32
}

func (f *fakeTxline) FetchScoreSnapshot(ctx context.Context, fixtureID int64) ([]txline.ScoreSnapshotRow, error) {
	return nil, errors.New("snapshot unavailable")
}

func (f *fakeTxline) FetchStatValidation(ctx context.Context, fixtureID int64, seq int32, statKey uint32) (txline.StatValidation, error) {
	f.calls++
	f.lastFixtureID = fixtureID
	f.lastSeq = seq
	f.lastStatKey = statKey
	if f.err != nil {
		return txline.StatValidation{}, f.err
	}
	return txline.StatValidation{
		Summary: txline.StatValidationSummary{
			FixtureID: fixtureID,
			UpdateStats: txline.ScoresUpdateStatsResp{
				UpdateCount:  1,
				MinTimestamp: 1700000000000,
				MaxTimestamp: 1700000000000,
			},
			EventStatsSubTreeRoot: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		},
		EventStatRoot: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		StatToProve:   txline.ScoreStatResponse{Key: statKey, Value: 1, Period: 0},
	}, nil
}

type snapshotTxline struct {
	fakeTxline
	rows []txline.ScoreSnapshotRow
}

func (s *snapshotTxline) FetchScoreSnapshot(ctx context.Context, fixtureID int64) ([]txline.ScoreSnapshotRow, error) {
	return s.rows, nil
}

type queueAwareTxline struct {
	fakeTxline
	cache         *memCache
	matchID       string
	wagerPubkey   string
	queuedAtFetch bool
}

func (q *queueAwareTxline) FetchStatValidation(ctx context.Context, fixtureID int64, seq int32, statKey uint32) (txline.StatValidation, error) {
	item, err := q.cache.GetPendingSettlement(ctx, q.matchID, q.wagerPubkey)
	q.queuedAtFetch = err == nil && item.Attempts == 0
	return q.fakeTxline.FetchStatValidation(ctx, fixtureID, seq, statKey)
}

type fakeSolana struct {
	closeCalls      int
	closeErr        error
	settleCalls     int
	voidCalls       int
	lastWinningSide uint8
	lastWagerPubkey string
	settleErr       error
	listErr         error
	storedWager     solanapkg.Wager
}

func (f *fakeSolana) GetWager(ctx context.Context, pubkey solana.PublicKey) (solanapkg.Wager, error) {
	wagers, err := f.ListMatchedWagers(ctx, "")
	if err != nil {
		return solanapkg.Wager{}, err
	}
	for _, wager := range wagers {
		if wager.Pubkey.Equals(pubkey) {
			return wager, nil
		}
	}
	return solanapkg.Wager{}, errors.New("wager account " + pubkey.String() + " not found")
}

func (f *fakeSolana) ListMatchedWagers(ctx context.Context, matchID string) ([]solanapkg.Wager, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	wager := f.activeWager(matchID)
	if wager.Status != solanapkg.WagerStatusMatched {
		return nil, nil
	}
	return []solanapkg.Wager{wager}, nil
}

func (f *fakeSolana) ListActiveWagers(ctx context.Context) ([]solanapkg.Wager, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.storedWager.Pubkey.IsZero() {
		return nil, nil
	}
	wager := f.activeWager("")
	if wager.Status != solanapkg.WagerStatusOpen && wager.Status != solanapkg.WagerStatusMatched {
		return nil, nil
	}
	return []solanapkg.Wager{wager}, nil
}

func (f *fakeSolana) activeWager(matchID string) solanapkg.Wager {
	var matchBytes [32]byte
	copy(matchBytes[:], []byte(matchID))
	if f.storedWager.Pubkey.IsZero() {
		f.storedWager = solanapkg.Wager{
			Pubkey:             solana.NewWallet().PublicKey(),
			Maker:              solana.NewWallet().PublicKey(),
			Taker:              solana.NewWallet().PublicKey(),
			MatchID:            matchBytes,
			MatchIDLen:         uint8(len(matchID)),
			Participant1IsHome: true,
			MakerSide:          solanapkg.SideHome,
			TakerSide:          solanapkg.SideAway,
			Status:             solanapkg.WagerStatusMatched,
		}
	}
	return f.storedWager
}

func (f *fakeSolana) SettleWager(ctx context.Context, p solanapkg.SettleParams) (solana.Signature, error) {
	f.settleCalls++
	f.lastWinningSide = p.WinningSide
	f.lastWagerPubkey = p.Wager.Pubkey.String()
	if f.settleErr != nil {
		return solana.Signature{}, f.settleErr
	}
	var sig solana.Signature
	sig[0] = 1
	return sig, nil
}

func (f *fakeSolana) VoidWager(ctx context.Context, p solanapkg.VoidParams) (solana.Signature, error) {
	f.voidCalls++
	f.lastWinningSide = p.WinningSide
	f.lastWagerPubkey = p.Wager.Pubkey.String()
	if f.settleErr != nil {
		return solana.Signature{}, f.settleErr
	}
	var sig solana.Signature
	sig[0] = 3
	return sig, nil
}

func (f *fakeSolana) CloseMatch(ctx context.Context, keeperKey solana.PrivateKey, matchID string) (solana.Signature, error) {
	f.closeCalls++
	if f.closeErr != nil {
		return solana.Signature{}, f.closeErr
	}
	var sig solana.Signature
	sig[0] = 2
	return sig, nil
}

func TestCloseMatchOnChainTreatsAlreadyClosedAsSuccess(t *testing.T) {
	sc := &fakeSolana{closeErr: solanapkg.ErrMatchAlreadyClosed}
	w := &Worker{Solana: sc, KeeperKey: solana.NewWallet().PrivateKey}

	if err := w.closeMatchOnChain(context.Background(), "18237038"); err != nil {
		t.Fatalf("closeMatchOnChain: %v", err)
	}
	if sc.closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", sc.closeCalls)
	}
}

func finalScoreUpdate() txline.ScoreUpdate {
	return txline.ScoreUpdate{
		FixtureID:          17952170,
		GameState:          "F2",
		Seq:                10,
		Participant1IsHome: true,
		ScoreSoccer: &txline.SoccerFixtureScore{
			Participant1: txline.SoccerTotalScore{Goals: 2},
			Participant2: txline.SoccerTotalScore{Goals: 1},
		},
	}
}

func TestWorkerSettleMatchIdempotent(t *testing.T) {
	c := newMemCache()
	tx := &fakeTxline{}
	sc := &fakeSolana{}
	w := &Worker{
		Cache:      c,
		Txline:     tx,
		Solana:     sc,
		KeeperKey:  solana.NewWallet().PrivateKey,
		StatKey:    1002,
		AutoSettle: true,
	}

	update := finalScoreUpdate()
	ctx := context.Background()
	if err := w.SettleMatch(ctx, update); err != nil {
		t.Fatalf("first settle: %v", err)
	}
	if err := w.SettleMatch(ctx, update); err != nil {
		t.Fatalf("second settle: %v", err)
	}
	if sc.settleCalls != 1 {
		t.Fatalf("settle calls = %d, want 1", sc.settleCalls)
	}
}

func TestSettleMatchQueuesBeforeFetchingProof(t *testing.T) {
	c := newMemCache()
	sc := &fakeSolana{}
	update := finalScoreUpdate()
	wager := sc.activeWager(update.MatchID())
	tx := &queueAwareTxline{
		cache:       c,
		matchID:     update.MatchID(),
		wagerPubkey: wager.Pubkey.String(),
	}
	w := &Worker{
		Cache:      c,
		Txline:     tx,
		Solana:     sc,
		KeeperKey:  solana.NewWallet().PrivateKey,
		StatKey:    1002,
		AutoSettle: true,
	}

	if err := w.SettleMatch(context.Background(), update); err != nil {
		t.Fatalf("SettleMatch: %v", err)
	}
	if !tx.queuedAtFetch {
		t.Fatal("settlement was not durably queued before proof fetch")
	}
	if _, err := c.GetPendingSettlement(context.Background(), update.MatchID(), wager.Pubkey.String()); !errors.Is(err, cache.ErrPendingSettlementNotFound) {
		t.Fatalf("pending settlement remained after success: %v", err)
	}
}

func TestSettleMatchLeavesExistingQueueForRetryWorker(t *testing.T) {
	c := newMemCache()
	sc := &fakeSolana{}
	update := finalScoreUpdate()
	wager := sc.activeWager(update.MatchID())
	now := time.Now().UTC()
	c.pending[update.MatchID()+":"+wager.Pubkey.String()] = cache.PendingSettlement{
		MatchID:     update.MatchID(),
		WagerPubkey: wager.Pubkey.String(),
		FixtureID:   update.FixtureID,
		Seq:         update.Seq,
		GameState:   update.GameState,
		Attempts:    2,
		NextRetryAt: now.Add(time.Minute),
		CreatedAt:   now.Add(-time.Minute),
		UpdatedAt:   now,
	}
	tx := &fakeTxline{}
	w := &Worker{
		Cache:      c,
		Txline:     tx,
		Solana:     sc,
		KeeperKey:  solana.NewWallet().PrivateKey,
		StatKey:    1002,
		AutoSettle: true,
	}

	if err := w.SettleMatch(context.Background(), update); err != nil {
		t.Fatalf("SettleMatch: %v", err)
	}
	if tx.calls != 0 || sc.settleCalls != 0 {
		t.Fatalf("existing retry was attempted early: proof_calls=%d settle_calls=%d", tx.calls, sc.settleCalls)
	}
}

func TestHandleUpdateCachesGoals(t *testing.T) {
	c := newMemCache()
	w := &Worker{Cache: c, StatKey: 1002}
	update := txline.ScoreUpdate{
		FixtureID:          2,
		GameState:          "HT",
		Seq:                3,
		Participant1IsHome: false,
		ScoreSoccer: &txline.SoccerFixtureScore{
			Participant1: txline.SoccerTotalScore{Goals: 1},
			Participant2: txline.SoccerTotalScore{Goals: 2},
		},
	}
	if err := w.HandleUpdate(context.Background(), update); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	got, _ := c.GetMatch(context.Background(), update.MatchID())
	if got.HomeGoals == nil || got.AwayGoals == nil || *got.HomeGoals != 2 || *got.AwayGoals != 1 {
		t.Fatalf("got = %#v", got)
	}
}

func TestHandleUpdateCachesNonFinal(t *testing.T) {
	c := newMemCache()
	w := &Worker{Cache: c, StatKey: 1002}
	update := txline.ScoreUpdate{FixtureID: 1, GameState: "HT", Seq: 1}
	if err := w.HandleUpdate(context.Background(), update); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	got, _ := c.GetMatch(context.Background(), update.MatchID())
	if got.IsFinal {
		t.Fatal("expected non-final cached match")
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("expected updated_at")
	}
}

func TestWorkerRunDuplicateFinalEvents(t *testing.T) {
	c := newMemCache()
	tx := &fakeTxline{}
	sc := &fakeSolana{}
	w := &Worker{
		Cache:      c,
		Txline:     tx,
		Solana:     sc,
		KeeperKey:  solana.NewWallet().PrivateKey,
		StatKey:    1002,
		AutoSettle: true,
	}

	events := make(chan txline.ScoreUpdate, 2)
	events <- finalScoreUpdate()
	events <- finalScoreUpdate()
	close(events)

	if err := w.Run(context.Background(), events); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sc.settleCalls != 1 {
		t.Fatalf("settle calls = %d, want 1", sc.settleCalls)
	}
}

func TestSettleOneSkipsWhenAlreadyInCache(t *testing.T) {
	c := newMemCache()
	wagerPubkey := solana.NewWallet().PublicKey()
	matchID := "17952170"
	if _, err := c.MarkSettled(context.Background(), cache.SettlementRecord{
		MatchID:     matchID,
		WagerPubkey: wagerPubkey.String(),
	}); err != nil {
		t.Fatalf("MarkSettled: %v", err)
	}

	sc := &fakeSolana{}
	w := &Worker{Cache: c, Solana: sc, KeeperKey: solana.NewWallet().PrivateKey}
	wager := solanapkg.Wager{Pubkey: wagerPubkey, Status: solanapkg.WagerStatusMatched}

	if err := w.settleOne(context.Background(), matchID, wager, solanapkg.ValidateStatArgs{}, [32]byte{}, solanapkg.SideHome); err != nil {
		t.Fatalf("settleOne: %v", err)
	}
	if sc.settleCalls != 0 {
		t.Fatalf("settle calls = %d, want 0", sc.settleCalls)
	}
}

func TestSettleOneAlreadySettledOnChain(t *testing.T) {
	c := newMemCache()
	sc := &fakeSolana{settleErr: solanapkg.ErrAlreadySettled}
	w := &Worker{Cache: c, Solana: sc, KeeperKey: solana.NewWallet().PrivateKey}

	wager := solanapkg.Wager{
		Pubkey: solana.NewWallet().PublicKey(),
		Status: solanapkg.WagerStatusMatched,
	}
	matchID := "17952170"

	if err := w.settleOne(context.Background(), matchID, wager, solanapkg.ValidateStatArgs{}, [32]byte{}, solanapkg.SideHome); err != nil {
		t.Fatalf("settleOne: %v", err)
	}
	settled, err := c.IsSettled(context.Background(), matchID, wager.Pubkey.String())
	if err != nil || !settled {
		t.Fatalf("cache settled = %v err=%v", settled, err)
	}
}

func TestSettleWagerFailureDoesNotMarkSettled(t *testing.T) {
	c := newMemCache()
	sc := &fakeSolana{settleErr: errors.New("simulation failed")}
	w := &Worker{Cache: c, Solana: sc, KeeperKey: solana.NewWallet().PrivateKey}

	wager := solanapkg.Wager{
		Pubkey: solana.NewWallet().PublicKey(),
		Status: solanapkg.WagerStatusMatched,
	}
	matchID := "17952170"

	err := w.settleOne(context.Background(), matchID, wager, solanapkg.ValidateStatArgs{}, [32]byte{}, solanapkg.SideHome)
	if err == nil {
		t.Fatal("expected settle error")
	}
	settled, _ := c.IsSettled(context.Background(), matchID, wager.Pubkey.String())
	if settled {
		t.Fatal("wager should not be marked settled after failure")
	}
}

func TestSettleMatchSettlesDrawScore(t *testing.T) {
	c := newMemCache()
	sc := &fakeSolana{}
	w := &Worker{
		Cache:      c,
		Txline:     &fakeTxline{},
		Solana:     sc,
		KeeperKey:  solana.NewWallet().PrivateKey,
		StatKey:    1002,
		AutoSettle: true,
	}
	update := txline.ScoreUpdate{
		FixtureID: 17952170,
		GameState: "F2",
		Seq:       10,
		ScoreSoccer: &txline.SoccerFixtureScore{
			Participant1: txline.SoccerTotalScore{Goals: 1},
			Participant2: txline.SoccerTotalScore{Goals: 1},
		},
	}
	if err := w.SettleMatch(context.Background(), update); err != nil {
		t.Fatalf("SettleMatch draw: %v", err)
	}
	if sc.lastWinningSide != solanapkg.SideDraw {
		t.Fatalf("winning_side = %d want draw", sc.lastWinningSide)
	}
}

func TestSettleOneVoidsWhenNeitherSelectedSideWon(t *testing.T) {
	c := newMemCache()
	sc := &fakeSolana{}
	w := &Worker{
		Cache:     c,
		Solana:    sc,
		KeeperKey: solana.NewWallet().PrivateKey,
	}
	wager := solanapkg.Wager{
		Pubkey:    solana.NewWallet().PublicKey(),
		Maker:     solana.NewWallet().PublicKey(),
		Taker:     solana.NewWallet().PublicKey(),
		MakerSide: solanapkg.SideHome,
		TakerSide: solanapkg.SideDraw,
		Status:    solanapkg.WagerStatusMatched,
	}

	if err := w.settleOne(
		context.Background(),
		"18241006",
		wager,
		solanapkg.ValidateStatArgs{},
		[32]byte{},
		solanapkg.SideAway,
	); err != nil {
		t.Fatalf("settleOne: %v", err)
	}
	if sc.voidCalls != 1 || sc.settleCalls != 0 {
		t.Fatalf("void calls = %d settle calls = %d", sc.voidCalls, sc.settleCalls)
	}
}

func TestHandleUpdateFinalTriggersSettlement(t *testing.T) {
	c := newMemCache()
	tx := &fakeTxline{}
	sc := &fakeSolana{}
	w := &Worker{
		Cache:      c,
		Txline:     tx,
		Solana:     sc,
		KeeperKey:  solana.NewWallet().PrivateKey,
		StatKey:    1002,
		AutoSettle: true,
	}
	if err := w.HandleUpdate(context.Background(), finalScoreUpdate()); err != nil {
		t.Fatalf("HandleUpdate: %v", err)
	}
	if sc.settleCalls != 1 {
		t.Fatalf("settle calls = %d, want 1", sc.settleCalls)
	}
	got, _ := c.GetMatch(context.Background(), "17952170")
	if !got.IsFinal || got.FinalizedAt == nil {
		t.Fatalf("cached match = %#v", got)
	}
}

func TestReconcileRefreshesInferredFinalWithoutAutoSettle(t *testing.T) {
	c := newMemCache()
	home, away := int32(0), int32(0)
	c.matches["17952170"] = cache.Match{
		MatchID:     "17952170",
		FixtureID:   17952170,
		GameState:   "FT",
		IsFinal:     true,
		FinalSource: cache.FinalSourceInferred,
		HomeGoals:   &home,
		AwayGoals:   &away,
		Seq:         1,
	}
	tx := &snapshotTxline{rows: []txline.ScoreSnapshotRow{{
		FixtureID:          17952170,
		GameState:          "F2",
		Seq:                42,
		Participant1IsHome: true,
		ScoreSoccer: &txline.SoccerFixtureScore{
			Participant1: txline.SoccerTotalScore{Goals: 2},
			Participant2: txline.SoccerTotalScore{Goals: 0},
		},
	}}}
	sc := &fakeSolana{}
	w := &Worker{Cache: c, Txline: tx, Solana: sc, AutoSettle: false}

	if err := w.ReconcileFinalMatches(context.Background()); err != nil {
		t.Fatalf("ReconcileFinalMatches: %v", err)
	}
	got, err := c.GetMatch(context.Background(), "17952170")
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if got.FinalSource != cache.FinalSourceTxline || got.Seq != 42 {
		t.Fatalf("refreshed match = %#v", got)
	}
	if got.HomeGoals == nil || *got.HomeGoals != 2 {
		t.Fatalf("home goals = %#v", got.HomeGoals)
	}
	if sc.settleCalls != 0 {
		t.Fatalf("settle calls = %d, want 0", sc.settleCalls)
	}
}

func TestReconcileRefreshesEligibleNonFinalWithoutAutoSettle(t *testing.T) {
	c := newMemCache()
	kickoff := time.Now().Add(-2 * time.Hour).UnixMilli()
	c.matches["18213979"] = cache.Match{
		MatchID:   "18213979",
		FixtureID: 18213979,
		GameState: "HT",
		StartTime: kickoff,
		Seq:       77,
	}
	tx := &snapshotTxline{rows: []txline.ScoreSnapshotRow{{
		FixtureID:          18213979,
		GameState:          "HT",
		Seq:                77,
		Participant1IsHome: true,
		ScoreSoccer: &txline.SoccerFixtureScore{
			Participant1: txline.SoccerTotalScore{Goals: 1},
			Participant2: txline.SoccerTotalScore{Goals: 1},
		},
	}}}
	sc := &fakeSolana{}
	w := &Worker{Cache: c, Txline: tx, Solana: sc, AutoSettle: false}

	if err := w.ReconcileFinalMatches(context.Background()); err != nil {
		t.Fatalf("ReconcileFinalMatches: %v", err)
	}
	got, err := c.GetMatch(context.Background(), "18213979")
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if got.IsFinal {
		t.Fatalf("live match was incorrectly finalized: %#v", got)
	}
	if got.GameState != "HT" || got.Seq != 77 {
		t.Fatalf("live state changed unexpectedly: %#v", got)
	}
	if sc.settleCalls != 0 {
		t.Fatalf("settle calls = %d, want 0", sc.settleCalls)
	}
}

func TestReconcileHydratesMissingFinalFromMatchedWager(t *testing.T) {
	const matchID = "18237038"
	var matchBytes [32]byte
	copy(matchBytes[:], matchID)

	c := newMemCache()
	sc := &fakeSolana{storedWager: solanapkg.Wager{
		Pubkey:             solana.NewWallet().PublicKey(),
		Maker:              solana.NewWallet().PublicKey(),
		Taker:              solana.NewWallet().PublicKey(),
		MatchID:            matchBytes,
		MatchIDLen:         uint8(len(matchID)),
		Participant1IsHome: true,
		MakerSide:          solanapkg.SideHome,
		TakerSide:          solanapkg.SideAway,
		Status:             solanapkg.WagerStatusMatched,
	}}
	tx := &snapshotTxline{rows: []txline.ScoreSnapshotRow{{
		FixtureIDAlt:     18237038,
		GameStateAlt:     "scheduled",
		StartTimeAlt:     1784055600000,
		ActionAlt:        "game_finalised",
		StatusIDAlt:      json.RawMessage("100"),
		TSAlt:            1784063054751,
		SeqAlt:           1026,
		Participant1Home: true,
		Score: &txline.SnapshotScore{
			Participant1: txline.SoccerTotalScore{Total: &txline.SoccerScore{Goals: 0}},
			Participant2: txline.SoccerTotalScore{Total: &txline.SoccerScore{Goals: 2}},
		},
	}}}
	w := &Worker{
		Cache:      c,
		Txline:     tx,
		Solana:     sc,
		KeeperKey:  solana.NewWallet().PrivateKey,
		AutoSettle: false,
	}

	if err := w.ReconcileFinalMatches(context.Background()); err != nil {
		t.Fatalf("ReconcileFinalMatches: %v", err)
	}
	got, err := c.GetMatch(context.Background(), matchID)
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if !got.IsFinal || got.FinalSource != cache.FinalSourceTxline || got.Seq != 1026 {
		t.Fatalf("hydrated match = %#v", got)
	}
	if got.HomeGoals == nil || got.AwayGoals == nil || *got.HomeGoals != 0 || *got.AwayGoals != 2 {
		t.Fatalf("hydrated score = %#v-%#v", got.HomeGoals, got.AwayGoals)
	}
	if sc.closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", sc.closeCalls)
	}
	if sc.settleCalls != 0 {
		t.Fatalf("settle calls = %d, want 0", sc.settleCalls)
	}
}

func TestReconcileHydratesAndClosesMissingFinalFromOpenWager(t *testing.T) {
	const matchID = "18237038"
	var matchBytes [32]byte
	copy(matchBytes[:], matchID)

	c := newMemCache()
	sc := &fakeSolana{storedWager: solanapkg.Wager{
		Pubkey:     solana.NewWallet().PublicKey(),
		Maker:      solana.NewWallet().PublicKey(),
		MatchID:    matchBytes,
		MatchIDLen: uint8(len(matchID)),
		MakerSide:  solanapkg.SideHome,
		TakerSide:  solanapkg.SideUnset,
		Status:     solanapkg.WagerStatusOpen,
	}}
	tx := &snapshotTxline{rows: []txline.ScoreSnapshotRow{{
		FixtureIDAlt:     18237038,
		GameStateAlt:     "scheduled",
		StartTimeAlt:     1784055600000,
		ActionAlt:        "game_finalised",
		StatusIDAlt:      json.RawMessage("100"),
		TSAlt:            1784063054751,
		SeqAlt:           1026,
		Participant1Home: true,
		Score: &txline.SnapshotScore{
			Participant1: txline.SoccerTotalScore{Total: &txline.SoccerScore{Goals: 0}},
			Participant2: txline.SoccerTotalScore{Total: &txline.SoccerScore{Goals: 2}},
		},
	}}}
	w := &Worker{
		Cache:      c,
		Txline:     tx,
		Solana:     sc,
		KeeperKey:  solana.NewWallet().PrivateKey,
		AutoSettle: true,
	}

	if err := w.ReconcileFinalMatches(context.Background()); err != nil {
		t.Fatalf("ReconcileFinalMatches: %v", err)
	}
	got, err := c.GetMatch(context.Background(), matchID)
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if !got.IsFinal || got.FinalSource != cache.FinalSourceTxline || got.Seq != 1026 {
		t.Fatalf("hydrated match = %#v", got)
	}
	if sc.closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", sc.closeCalls)
	}
	if sc.settleCalls != 0 {
		t.Fatalf("settle calls = %d, want 0 for an open wager", sc.settleCalls)
	}
}

func TestProcessPendingItemUsesHydratedFinalSeq(t *testing.T) {
	c := newMemCache()
	matchID := "17952170"
	wagerPubkey := solana.NewWallet().PublicKey()
	maker := solana.NewWallet().PublicKey()
	taker := solana.NewWallet().PublicKey()
	var matchBytes [32]byte
	copy(matchBytes[:], []byte(matchID))
	sc := &fakeSolana{storedWager: solanapkg.Wager{
		Pubkey:             wagerPubkey,
		Maker:              maker,
		Taker:              taker,
		MatchID:            matchBytes,
		MatchIDLen:         uint8(len(matchID)),
		Participant1IsHome: true,
		MakerSide:          solanapkg.SideHome,
		TakerSide:          solanapkg.SideAway,
		Status:             solanapkg.WagerStatusMatched,
	}}
	tx := &snapshotTxline{rows: []txline.ScoreSnapshotRow{{
		FixtureID:          17952170,
		GameState:          "F2",
		Seq:                42,
		Participant1IsHome: true,
		ScoreSoccer: &txline.SoccerFixtureScore{
			Participant1: txline.SoccerTotalScore{Goals: 2},
			Participant2: txline.SoccerTotalScore{Goals: 0},
		},
	}}}
	w := &Worker{Cache: c, Txline: tx, Solana: sc, KeeperKey: solana.NewWallet().PrivateKey}

	err := w.processPendingItem(context.Background(), cache.PendingSettlement{
		MatchID:     matchID,
		WagerPubkey: wagerPubkey.String(),
		FixtureID:   17952170,
		Seq:         1,
		GameState:   "FT",
		Attempts:    1,
	})
	if err != nil {
		t.Fatalf("processPendingItem: %v", err)
	}
	if tx.lastSeq != 42 {
		t.Fatalf("proof seq = %d, want hydrated final seq 42", tx.lastSeq)
	}
	if sc.settleCalls != 1 {
		t.Fatalf("settle calls = %d, want 1", sc.settleCalls)
	}
}

func TestRunRespectsContextCancel(t *testing.T) {
	w := &Worker{Cache: newMemCache(), StatKey: 1002}
	events := make(chan txline.ScoreUpdate)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := w.Run(ctx, events); err == nil {
		t.Fatal("expected context error")
	}
	close(events)
	if err := w.Run(context.Background(), events); err != nil {
		t.Fatalf("closed channel: %v", err)
	}
}

func TestSettleMatchErrorBranches(t *testing.T) {
	update := finalScoreUpdate()
	ctx := context.Background()

	c := newMemCache()
	sc := &fakeSolana{settleErr: errors.New("rpc down")}
	w := &Worker{Cache: c, Txline: &fakeTxline{}, Solana: sc, KeeperKey: solana.NewWallet().PrivateKey, StatKey: 1002, AutoSettle: true}
	if _, err := c.MarkFinalOnce(ctx, update.MatchID()); err != nil {
		t.Fatalf("mark final: %v", err)
	}
	if err := w.SettleMatch(ctx, update); err != nil {
		t.Fatalf("first settle should not fail aggregate: %v", err)
	}
	if sc.settleCalls != 1 {
		t.Fatalf("settle calls = %d, want 1", sc.settleCalls)
	}
	pending, err := c.GetPendingSettlement(ctx, update.MatchID(), sc.lastWagerPubkey)
	if err != nil {
		t.Fatalf("expected pending settlement: %v", err)
	}
	if pending.Attempts != 1 {
		t.Fatalf("pending attempts = %d, want 1", pending.Attempts)
	}

	sc.settleErr = nil
	if err := c.UpsertMatch(ctx, cache.ApplyScoreUpdate(cache.Match{}, update)); err != nil {
		t.Fatalf("cache final match for retry: %v", err)
	}
	pending.NextRetryAt = time.Now().Add(-time.Second)
	if err := c.UpdatePendingSettlement(ctx, pending); err != nil {
		t.Fatalf("make pending settlement due: %v", err)
	}
	if err := w.ProcessPendingQueue(ctx, 10); err != nil {
		t.Fatalf("process pending queue: %v", err)
	}
	if sc.settleCalls != 2 {
		t.Fatalf("settle calls = %d, want 2 after queued retry", sc.settleCalls)
	}

	w2 := &Worker{
		Cache:      c,
		Txline:     &fakeTxline{},
		Solana:     &fakeSolana{listErr: errors.New("rpc")},
		KeeperKey:  solana.NewWallet().PrivateKey,
		StatKey:    1002,
		AutoSettle: true,
	}
	if err := w2.SettleMatch(ctx, txline.ScoreUpdate{FixtureID: 99, GameState: "F2", Seq: 1, ScoreSoccer: update.ScoreSoccer}); err == nil {
		t.Fatal("expected list error")
	}

	c3 := newMemCache()
	sc3 := &fakeSolana{}
	w3 := &Worker{
		Cache:      c3,
		Txline:     &fakeTxline{err: errors.New("proof")},
		Solana:     sc3,
		KeeperKey:  solana.NewWallet().PrivateKey,
		StatKey:    1002,
		AutoSettle: true,
	}
	if err := w3.SettleMatch(ctx, update); err != nil {
		t.Fatalf("proof error should queue per wager: %v", err)
	}
	if _, err := c3.GetPendingSettlement(ctx, update.MatchID(), sc3.storedWager.Pubkey.String()); err != nil {
		t.Fatalf("expected proof failure to be queued: %v", err)
	}

	empty := &Worker{Cache: newMemCache(), Txline: &fakeTxline{}, Solana: &fakeSolana{}, KeeperKey: solana.NewWallet().PrivateKey, StatKey: 1002, AutoSettle: true}
	if err := empty.SettleMatch(ctx, update); err != nil {
		t.Fatalf("no wagers: %v", err)
	}
}

func TestHandleUpdateUpsertError(t *testing.T) {
	w := &Worker{Cache: &failingCache{}, StatKey: 1002}
	if err := w.HandleUpdate(context.Background(), txline.ScoreUpdate{FixtureID: 1, GameState: "HT", Seq: 1}); err == nil {
		t.Fatal("expected cache error")
	}
}

type failingCache struct{ memCache }

func (f *failingCache) UpsertMatch(ctx context.Context, match cache.Match) error {
	return errors.New("cache down")
}

func TestSettleMatchLogsPerWagerFailure(t *testing.T) {
	c := newMemCache()
	sc := &fakeSolana{settleErr: errors.New("sim failed")}
	w := &Worker{
		Cache:     c,
		Txline:    &fakeTxline{},
		Solana:    sc,
		KeeperKey: solana.NewWallet().PrivateKey,
		StatKey:   1002,
	}
	if err := w.SettleMatch(context.Background(), finalScoreUpdate()); err != nil {
		t.Fatalf("SettleMatch should not fail aggregate: %v", err)
	}
}

func TestRunLogsHandleUpdateError(t *testing.T) {
	w := &Worker{Cache: &failingCache{}, StatKey: 1002}
	events := make(chan txline.ScoreUpdate, 1)
	events <- txline.ScoreUpdate{FixtureID: 1, GameState: "HT", Seq: 1}
	close(events)
	if err := w.Run(context.Background(), events); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestSettleOneSuccess(t *testing.T) {
	c := newMemCache()
	sc := &fakeSolana{}
	w := &Worker{Cache: c, Solana: sc, KeeperKey: solana.NewWallet().PrivateKey}
	wager := solanapkg.Wager{
		Pubkey: solana.NewWallet().PublicKey(),
		Status: solanapkg.WagerStatusMatched,
	}
	if err := w.settleOne(context.Background(), "17952170", wager, solanapkg.ValidateStatArgs{}, [32]byte{}, solanapkg.SideHome); err != nil {
		t.Fatalf("settleOne: %v", err)
	}
	settled, _ := c.IsSettled(context.Background(), "17952170", wager.Pubkey.String())
	if !settled {
		t.Fatal("expected settled record")
	}
}

func TestWinningSideFromScore(t *testing.T) {
	homeWin := finalScoreUpdate()
	side, ok := winningSideFromScore(homeWin)
	if !ok || side != solanapkg.SideHome {
		t.Fatalf("side = %d ok=%v", side, ok)
	}

	awayWin := homeWin
	awayWin.ScoreSoccer.Participant2.Goals = 3
	side, ok = winningSideFromScore(awayWin)
	if !ok || side != solanapkg.SideAway {
		t.Fatalf("side = %d ok=%v", side, ok)
	}

	if _, ok := winningSideFromScore(txline.ScoreUpdate{FixtureID: 1}); ok {
		t.Fatal("expected false without scores")
	}

	draw := homeWin
	draw.ScoreSoccer.Participant1.Goals = 1
	draw.ScoreSoccer.Participant2.Goals = 1
	side, ok = winningSideFromScore(draw)
	if !ok || side != solanapkg.SideDraw {
		t.Fatalf("draw side = %d ok=%v", side, ok)
	}
}
