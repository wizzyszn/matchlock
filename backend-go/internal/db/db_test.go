package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := Open("postgres://matchlock:matchlock@127.0.0.1:5432/matchlock?sslmode=disable")
	if err != nil {
		t.Fatalf("Open test db: %v", err)
	}
	t.Cleanup(func() {
		gdb.Unscoped().Where("1 = 1").Delete(&MagicLinkToken{})
	})
	return gdb
}

func seedToken(gdb *gorm.DB, email string, expiresAt time.Time, used bool) {
	token := MagicLinkToken{
		Email:     email,
		TokenHash: uuid.New().String(),
		ExpiresAt: expiresAt,
	}
	if used {
		now := time.Now().UTC()
		token.UsedAt = &now
	}
	if err := gdb.Create(&token).Error; err != nil {
		panic(err)
	}
}

func TestPurgeExpiredTokens_DeletesUsed(t *testing.T) {
	gdb := testDB(t)
	seedToken(gdb, "used@test.com", time.Now().UTC().Add(time.Hour), true)

	n, err := PurgeExpiredTokens(gdb)
	if err != nil {
		t.Fatalf("PurgeExpiredTokens: %v", err)
	}
	if n != 1 {
		t.Fatalf("deleted %d, want 1", n)
	}
}

func TestPurgeExpiredTokens_DeletesExpired(t *testing.T) {
	gdb := testDB(t)
	seedToken(gdb, "expired@test.com", time.Now().UTC().Add(-time.Hour), false)

	n, err := PurgeExpiredTokens(gdb)
	if err != nil {
		t.Fatalf("PurgeExpiredTokens: %v", err)
	}
	if n != 1 {
		t.Fatalf("deleted %d, want 1", n)
	}
}

func TestPurgeExpiredTokens_KeepsActive(t *testing.T) {
	gdb := testDB(t)
	seedToken(gdb, "active@test.com", time.Now().UTC().Add(time.Hour), false)

	n, err := PurgeExpiredTokens(gdb)
	if err != nil {
		t.Fatalf("PurgeExpiredTokens: %v", err)
	}
	if n != 0 {
		t.Fatalf("deleted %d, want 0", n)
	}
}

func TestPurgeExpiredTokens_OnlyTargetsTokens(t *testing.T) {
	gdb := testDB(t)
	seedToken(gdb, "purge-me@test.com", time.Now().UTC().Add(-time.Hour), false)
	seedToken(gdb, "keep-me@test.com", time.Now().UTC().Add(time.Hour), false)

	n, err := PurgeExpiredTokens(gdb)
	if err != nil {
		t.Fatalf("PurgeExpiredTokens: %v", err)
	}
	if n != 1 {
		t.Fatalf("deleted %d, want 1", n)
	}

	var remaining int64
	gdb.Model(&MagicLinkToken{}).Count(&remaining)
	if remaining != 1 {
		t.Fatalf("remaining %d, want 1", remaining)
	}
}
