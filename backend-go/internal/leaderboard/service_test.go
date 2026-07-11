package leaderboard

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/matchlock/backend-go/internal/db"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.Open("postgres://matchlock:matchlock@127.0.0.1:5432/matchlock?sslmode=disable")
	if err != nil {
		t.Fatalf("Open test db: %v", err)
	}
	cleanup := func() {
		gdb.Unscoped().Where("1 = 1").Delete(&db.LeaderboardSettlement{})
		gdb.Unscoped().Where("1 = 1").Delete(&db.LeaderboardEntry{})
		gdb.Unscoped().Where("1 = 1").Delete(&db.WalletLink{})
		gdb.Unscoped().Where("1 = 1").Delete(&db.User{})
	}
	cleanup()
	t.Cleanup(cleanup)
	return gdb
}

func seedUser(gdb *gorm.DB, id uuid.UUID, email, displayName string) {
	u := db.User{ID: id, Email: email, DisplayName: displayName}
	if err := gdb.Create(&u).Error; err != nil {
		panic(err)
	}
}

func seedWalletLink(gdb *gorm.DB, userID uuid.UUID, pubkey string) {
	w := db.WalletLink{
		UserID:     userID,
		Pubkey:     pubkey,
		VerifiedAt: time.Now().UTC(),
	}
	if err := gdb.Create(&w).Error; err != nil {
		panic(err)
	}
}

func TestRecordSettlement_CreatesEntries(t *testing.T) {
	gdb := testDB(t)
	svc := NewService(gdb)
	ctx := context.Background()

	winnerID := uuid.New()
	loserID := uuid.New()
	seedUser(gdb, winnerID, "winner@test.com", "Winner")
	seedUser(gdb, loserID, "loser@test.com", "Loser")
	seedWalletLink(gdb, winnerID, "winnerpubkey123")
	seedWalletLink(gdb, loserID, "loserpubkey456")

	err := svc.RecordSettlement(ctx, SettlementEvent{
		WagerPubkey:  "wager-1",
		WinnerPubkey: "winnerpubkey123",
		LoserPubkey:  "loserpubkey456",
		Stake:        100,
		MatchID:      "match-1",
	})
	if err != nil {
		t.Fatalf("RecordSettlement: %v", err)
	}

	var winnerEntry, loserEntry db.LeaderboardEntry
	if err := gdb.Where("user_id = ?", winnerID).First(&winnerEntry).Error; err != nil {
		t.Fatalf("find winner entry: %v", err)
	}
	if err := gdb.Where("user_id = ?", loserID).First(&loserEntry).Error; err != nil {
		t.Fatalf("find loser entry: %v", err)
	}

	if winnerEntry.Wins != 1 || winnerEntry.Losses != 0 {
		t.Fatalf("winner: wins=%d losses=%d, want wins=1 losses=0", winnerEntry.Wins, winnerEntry.Losses)
	}
	if winnerEntry.NetPnL != 100 {
		t.Fatalf("winner net_pnl=%d, want 100", winnerEntry.NetPnL)
	}
	if winnerEntry.TotalVolume != 200 {
		t.Fatalf("winner total_volume=%d, want 200", winnerEntry.TotalVolume)
	}
	if winnerEntry.TotalWagers != 1 {
		t.Fatalf("winner total_wagers=%d, want 1", winnerEntry.TotalWagers)
	}

	if loserEntry.Losses != 1 || loserEntry.Wins != 0 {
		t.Fatalf("loser: losses=%d wins=%d, want losses=1 wins=0", loserEntry.Losses, loserEntry.Wins)
	}
	if loserEntry.NetPnL != -100 {
		t.Fatalf("loser net_pnl=%d, want -100", loserEntry.NetPnL)
	}
	if loserEntry.TotalVolume != 200 {
		t.Fatalf("loser total_volume=%d, want 200", loserEntry.TotalVolume)
	}
}

func TestRecordSettlement_UpdatesExisting(t *testing.T) {
	gdb := testDB(t)
	svc := NewService(gdb)
	ctx := context.Background()

	userID := uuid.New()
	seedUser(gdb, userID, "player@test.com", "Player")
	seedWalletLink(gdb, userID, "pubkey1")

	gdb.Create(&db.LeaderboardEntry{
		UserID:      userID,
		Email:       "player@test.com",
		DisplayName: "Player",
		TotalWagers: 5,
		Wins:        3,
		Losses:      2,
		TotalVolume: 1000,
		NetPnL:      100,
	})

	err := svc.RecordSettlement(ctx, SettlementEvent{
		WagerPubkey:  "wager-2",
		WinnerPubkey: "pubkey1",
		LoserPubkey:  "unknownpubkey",
		Stake:        200,
		MatchID:      "match-2",
	})
	if err != nil {
		t.Fatalf("RecordSettlement: %v", err)
	}

	var entry db.LeaderboardEntry
	if err := gdb.Where("user_id = ?", userID).First(&entry).Error; err != nil {
		t.Fatalf("find entry: %v", err)
	}

	if entry.Wins != 4 || entry.Losses != 2 {
		t.Fatalf("wins=%d losses=%d, want wins=4 losses=2", entry.Wins, entry.Losses)
	}
	if entry.NetPnL != 300 {
		t.Fatalf("net_pnl=%d, want 300", entry.NetPnL)
	}
	if entry.TotalVolume != 1400 {
		t.Fatalf("total_volume=%d, want 1400", entry.TotalVolume)
	}
	if entry.TotalWagers != 6 {
		t.Fatalf("total_wagers=%d, want 6", entry.TotalWagers)
	}
}

func TestRecordSettlement_IsIdempotentPerWager(t *testing.T) {
	gdb := testDB(t)
	svc := NewService(gdb)
	ctx := context.Background()

	winnerID := uuid.New()
	loserID := uuid.New()
	seedUser(gdb, winnerID, "winner@test.com", "Winner")
	seedUser(gdb, loserID, "loser@test.com", "Loser")
	seedWalletLink(gdb, winnerID, "winnerpubkey123")
	seedWalletLink(gdb, loserID, "loserpubkey456")

	ev := SettlementEvent{
		WagerPubkey:  "wager-dup",
		WinnerPubkey: "winnerpubkey123",
		LoserPubkey:  "loserpubkey456",
		Stake:        100,
		MatchID:      "match-dup",
	}
	if err := svc.RecordSettlement(ctx, ev); err != nil {
		t.Fatalf("first RecordSettlement: %v", err)
	}
	if err := svc.RecordSettlement(ctx, ev); err != nil {
		t.Fatalf("second RecordSettlement: %v", err)
	}

	var winnerEntry db.LeaderboardEntry
	if err := gdb.Where("user_id = ?", winnerID).First(&winnerEntry).Error; err != nil {
		t.Fatalf("find winner entry: %v", err)
	}
	if winnerEntry.TotalWagers != 1 || winnerEntry.Wins != 1 || winnerEntry.NetPnL != 100 {
		t.Fatalf("winner entry = %+v", winnerEntry)
	}

	var synced int64
	gdb.Model(&db.LeaderboardSettlement{}).Where("wager_pubkey = ?", "wager-dup").Count(&synced)
	if synced != 1 {
		t.Fatalf("settlement sync rows = %d, want 1", synced)
	}
}

func TestGetLeaderboard_ReturnsRanked(t *testing.T) {
	gdb := testDB(t)
	svc := NewService(gdb)
	ctx := context.Background()

	topID := uuid.New()
	midID := uuid.New()
	botID := uuid.New()
	for _, u := range []struct {
		id    uuid.UUID
		email string
		name  string
		pnl   int64
	}{
		{topID, "top@test.com", "Top", 500},
		{midID, "mid@test.com", "Mid", 200},
		{botID, "bot@test.com", "Bot", -100},
	} {
		seedUser(gdb, u.id, u.email, u.name)
		gdb.Create(&db.LeaderboardEntry{
			UserID:      u.id,
			Email:       u.email,
			DisplayName: u.name,
			TotalWagers: 10,
			Wins:        5,
			Losses:      5,
			TotalVolume: 1000,
			NetPnL:      u.pnl,
		})
	}

	page, err := svc.GetLeaderboard(ctx, 0, 10)
	if err != nil {
		t.Fatalf("GetLeaderboard: %v", err)
	}

	entries := page.Entries
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	if page.Total != 3 {
		t.Fatalf("total=%d, want 3", page.Total)
	}
	if page.HasMore {
		t.Fatal("has_more should be false")
	}
	if entries[0].NetPnL != 500 || entries[0].Rank != 1 {
		t.Fatalf("rank 1: pnl=%d rank=%d, want pnl=500 rank=1", entries[0].NetPnL, entries[0].Rank)
	}
	if entries[1].NetPnL != 200 || entries[1].Rank != 2 {
		t.Fatalf("rank 2: pnl=%d rank=%d, want pnl=200 rank=2", entries[1].NetPnL, entries[1].Rank)
	}
	if entries[2].NetPnL != -100 || entries[2].Rank != 3 {
		t.Fatalf("rank 3: pnl=%d rank=%d, want pnl=-100 rank=3", entries[2].NetPnL, entries[2].Rank)
	}
}

func TestGetLeaderboard_RespectsLimit(t *testing.T) {
	gdb := testDB(t)
	svc := NewService(gdb)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		uid := uuid.New()
		email := string(rune('a'+i)) + "@test.com"
		seedUser(gdb, uid, email, "User")
		gdb.Create(&db.LeaderboardEntry{
			UserID: uid,
			Email:  email,
			NetPnL: int64(100 - i*10),
		})
	}

	t.Run("limit 3 offset 0", func(t *testing.T) {
		page, err := svc.GetLeaderboard(ctx, 0, 3)
		if err != nil {
			t.Fatalf("GetLeaderboard: %v", err)
		}
		if len(page.Entries) != 3 {
			t.Fatalf("got %d entries, want 3", len(page.Entries))
		}
		if !page.HasMore {
			t.Fatal("has_more should be true")
		}
	})

	t.Run("offset skips first entries", func(t *testing.T) {
		page, err := svc.GetLeaderboard(ctx, 3, 3)
		if err != nil {
			t.Fatalf("GetLeaderboard: %v", err)
		}
		if len(page.Entries) != 2 {
			t.Fatalf("got %d entries, want 2", len(page.Entries))
		}
		if page.Offset != 3 {
			t.Fatalf("offset=%d, want 3", page.Offset)
		}
	})

	t.Run("zero limit uses default 20", func(t *testing.T) {
		page, err := svc.GetLeaderboard(ctx, 0, 0)
		if err != nil {
			t.Fatalf("GetLeaderboard: %v", err)
		}
		if len(page.Entries) != 5 {
			t.Fatalf("got %d entries, want 5", len(page.Entries))
		}
	})
}

func TestGetRank_ReturnsCorrectRank(t *testing.T) {
	gdb := testDB(t)
	svc := NewService(gdb)
	ctx := context.Background()

	userID := uuid.New()
	seedUser(gdb, userID, "me@test.com", "Me")
	gdb.Create(&db.LeaderboardEntry{
		UserID:      userID,
		Email:       "me@test.com",
		DisplayName: "Me",
		TotalWagers: 10,
		Wins:        6,
		Losses:      4,
		TotalVolume: 2000,
		NetPnL:      300,
	})

	aboveID := uuid.New()
	seedUser(gdb, aboveID, "above@test.com", "Above")
	gdb.Create(&db.LeaderboardEntry{
		UserID: aboveID,
		Email:  "above@test.com",
		NetPnL: 500,
	})

	entry, err := svc.GetRank(ctx, userID)
	if err != nil {
		t.Fatalf("GetRank: %v", err)
	}
	if entry == nil {
		t.Fatal("GetRank returned nil")
	}
	if entry.Rank != 2 {
		t.Fatalf("rank=%d, want 2", entry.Rank)
	}
	if entry.NetPnL != 300 {
		t.Fatalf("net_pnl=%d, want 300", entry.NetPnL)
	}
}

func TestGetRank_UsesSameTiebreakersAsLeaderboard(t *testing.T) {
	gdb := testDB(t)
	svc := NewService(gdb)
	ctx := context.Background()

	firstID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	for _, row := range []db.LeaderboardEntry{
		{
			UserID:      secondID,
			Email:       "second@test.com",
			DisplayName: "Second",
			TotalWagers: 5,
			Wins:        3,
			Losses:      2,
			TotalVolume: 2_000,
			NetPnL:      500,
		},
		{
			UserID:      firstID,
			Email:       "first@test.com",
			DisplayName: "First",
			TotalWagers: 5,
			Wins:        3,
			Losses:      2,
			TotalVolume: 2_000,
			NetPnL:      500,
		},
	} {
		seedUser(gdb, row.UserID, row.Email, row.DisplayName)
		if err := gdb.Create(&row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}

	page, err := svc.GetLeaderboard(ctx, 0, 10)
	if err != nil {
		t.Fatalf("GetLeaderboard: %v", err)
	}
	if len(page.Entries) != 2 {
		t.Fatalf("entries=%d, want 2", len(page.Entries))
	}
	if page.Entries[0].UserID != firstID.String() {
		t.Fatalf("leaderboard first user=%s, want %s", page.Entries[0].UserID, firstID.String())
	}

	entry, err := svc.GetRank(ctx, firstID)
	if err != nil {
		t.Fatalf("GetRank: %v", err)
	}
	if entry == nil || entry.Rank != 1 {
		t.Fatalf("rank entry = %+v, want rank 1", entry)
	}
}

func TestGetRank_ReturnsNilForNoEntry(t *testing.T) {
	gdb := testDB(t)
	svc := NewService(gdb)
	ctx := context.Background()

	entry, err := svc.GetRank(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetRank: %v", err)
	}
	if entry != nil {
		t.Fatal("expected nil entry for unknown user")
	}
}

func TestRecordSettlement_SkipsUnlinkedPubkey(t *testing.T) {
	gdb := testDB(t)
	svc := NewService(gdb)
	ctx := context.Background()

	err := svc.RecordSettlement(ctx, SettlementEvent{
		WagerPubkey:  "wager-3",
		WinnerPubkey: "nobody",
		LoserPubkey:  "noone",
		Stake:        100,
		MatchID:      "match-3",
	})
	if err != nil {
		t.Fatalf("RecordSettlement: %v", err)
	}

	var count int64
	gdb.Model(&db.LeaderboardEntry{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 entries, got %d", count)
	}
}

func TestEntry_WinRate(t *testing.T) {
	gdb := testDB(t)
	svc := NewService(gdb)
	ctx := context.Background()

	userID := uuid.New()
	seedUser(gdb, userID, "wr@test.com", "WR")
	seedWalletLink(gdb, userID, "wrpubkey")

	gdb.Create(&db.LeaderboardEntry{
		UserID:      userID,
		Email:       "wr@test.com",
		DisplayName: "WR",
		TotalWagers: 10,
		Wins:        7,
		Losses:      3,
		TotalVolume: 2000,
		NetPnL:      400,
	})

	page, err := svc.GetLeaderboard(ctx, 0, 1)
	if err != nil {
		t.Fatalf("GetLeaderboard: %v", err)
	}
	if len(page.Entries) == 0 {
		t.Fatal("no entries")
	}
	if page.Entries[0].WinRate != 70.0 {
		t.Fatalf("win_rate=%f, want 70.0", page.Entries[0].WinRate)
	}

	entry, err := svc.GetRank(ctx, userID)
	if err != nil {
		t.Fatalf("GetRank: %v", err)
	}
	if entry == nil {
		t.Fatal("GetRank returned nil")
	}
	if entry.WinRate != 70.0 {
		t.Fatalf("win_rate=%f, want 70.0", entry.WinRate)
	}
}

func TestGetStats_AggregatesAllRows(t *testing.T) {
	gdb := testDB(t)
	svc := NewService(gdb)
	ctx := context.Background()

	rows := []db.LeaderboardEntry{
		{
			UserID:      uuid.New(),
			Email:       "a@test.com",
			DisplayName: "A",
			TotalWagers: 10,
			Wins:        7,
			Losses:      3,
			TotalVolume: 2_000,
			NetPnL:      400,
		},
		{
			UserID:      uuid.New(),
			Email:       "b@test.com",
			DisplayName: "B",
			TotalWagers: 4,
			Wins:        1,
			Losses:      3,
			TotalVolume: 800,
			NetPnL:      -100,
		},
	}
	for _, row := range rows {
		seedUser(gdb, row.UserID, row.Email, row.DisplayName)
		if err := gdb.Create(&row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}

	stats, err := svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalUsers != 2 {
		t.Fatalf("total_users=%d, want 2", stats.TotalUsers)
	}
	if stats.TotalWagers != 14 {
		t.Fatalf("total_wagers=%d, want 14", stats.TotalWagers)
	}
	if stats.TotalVolume != 2_800 {
		t.Fatalf("total_volume=%d, want 2800", stats.TotalVolume)
	}
	if stats.AvgWinRate != 47.5 {
		t.Fatalf("avg_win_rate=%f, want 47.5", stats.AvgWinRate)
	}
}
