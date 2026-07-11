package db

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open connects to Postgres and runs migrations.
func Open(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("postgres pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := migrate(gdb); err != nil {
		return nil, err
	}
	return gdb, nil
}

func migrate(gdb *gorm.DB) error {
	if err := gdb.AutoMigrate(
		&User{},
		&MagicLinkToken{},
		&Session{},
		&WalletLink{},
		&WalletLinkChallenge{},
		&WagerInvite{},
		&LeaderboardEntry{},
		&LeaderboardSettlement{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	if err := migrateWalletLinkUniqueness(gdb); err != nil {
		return fmt.Errorf("wallet link uniqueness: %w", err)
	}
	if err := dedupeWalletLinks(gdb); err != nil {
		return fmt.Errorf("dedupe wallet links: %w", err)
	}
	return nil
}

// migrateWalletLinkUniqueness enforces one pubkey → one Matchlock account.
func migrateWalletLinkUniqueness(gdb *gorm.DB) error {
	if err := gdb.Exec(`DROP INDEX IF EXISTS idx_wallet_user_pubkey`).Error; err != nil {
		return err
	}
	if err := gdb.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_links_pubkey ON wallet_links (pubkey)`).Error; err != nil {
		return err
	}
	if err := gdb.Exec(`CREATE INDEX IF NOT EXISTS idx_wallet_links_user_id ON wallet_links (user_id)`).Error; err != nil {
		return err
	}
	return nil
}

// dedupeWalletLinks keeps the earliest link per pubkey (one wallet → one account).
func dedupeWalletLinks(gdb *gorm.DB) error {
	return gdb.Exec(`
		DELETE FROM wallet_links AS newer
		USING wallet_links AS older
		WHERE newer.pubkey = older.pubkey
		  AND newer.created_at > older.created_at
	`).Error
}

// PurgeExpiredTokens deletes used and expired magic-link tokens.
func PurgeExpiredTokens(gdb *gorm.DB) (int64, error) {
	res := gdb.Unscoped().
		Where("used_at IS NOT NULL OR expires_at < ?", time.Now().UTC()).
		Delete(&MagicLinkToken{})
	return res.RowsAffected, res.Error
}

// Ping verifies database connectivity for readiness probes.
func Ping(ctx context.Context, gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
