package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/matchlock/backend-go/internal/db"
)

var names = []string{
	"WhaleWatcher", "BetKing", "CryptoKid", "LuckyAce", "SolTrader",
	"MoonShot", "RiskTaker", "TopGunner", "DegenPlaya", "WagerWizard",
	"FlipMaster", "OddsMaker", "StackSniper", "PnlPirate", "PoolShark",
	"EdgeLord", "ValueVet", "SpreadSheikh", "HedgeHog", "ZenBet",
	"Bailiff", "RakeRunner", "VigSlayer", "ArbDawg", "LineStepper",
	"BookieBreaker",
}

func main() {
	gdb, err := db.Open("postgres://matchlock:matchlock@127.0.0.1:5432/matchlock?sslmode=disable")
	if err != nil {
		log.Fatalf("db open: %v", err)
	}

	gdb.Unscoped().Where("1 = 1").Delete(&db.LeaderboardEntry{})
	gdb.Unscoped().Where("1 = 1").Delete(&db.WalletLink{})
	gdb.Unscoped().Where("1 = 1").Delete(&db.User{})

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, name := range names {
		uid := uuid.New()
		email := fmt.Sprintf("%s@matchlock.io", name)
		u := db.User{ID: uid, Email: email, DisplayName: name}
		if err := gdb.Create(&u).Error; err != nil {
			log.Fatalf("create user: %v", err)
		}

		wagers := int64(rng.Intn(50) + 5)
		wins := int64(rng.Intn(int(wagers)))
		losses := wagers - wins
		volume := uint64(rng.Int63n(50000) + 1000)
		pnl := int64(wins*int64(rng.Intn(200)+50)) - int64(losses*int64(rng.Intn(150)+30))

		entry := db.LeaderboardEntry{
			UserID:      uid,
			Email:       email,
			DisplayName: name,
			TotalWagers: wagers,
			Wins:        wins,
			Losses:      losses,
			TotalVolume: volume,
			NetPnL:      pnl,
		}
		if err := gdb.Create(&entry).Error; err != nil {
			log.Fatalf("create leaderboard entry: %v", err)
		}
	}

	fmt.Printf("seeded %d leaderboard entries\n", len(names))
}
