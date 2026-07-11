package leaderboard

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/matchlock/backend-go/internal/db"
	chainsol "github.com/matchlock/backend-go/internal/solana"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const leaderboardOrder = "net_pnl DESC, wins DESC, total_volume DESC, user_id ASC"

type Service struct {
	gdb *gorm.DB
}

func NewService(gdb *gorm.DB) *Service {
	return &Service{gdb: gdb}
}

type Entry struct {
	Rank        int     `json:"rank"`
	UserID      string  `json:"user_id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name,omitempty"`
	TotalWagers int64   `json:"total_wagers"`
	Wins        int64   `json:"wins"`
	Losses      int64   `json:"losses"`
	WinRate     float64 `json:"win_rate"`
	TotalVolume uint64  `json:"total_volume"`
	NetPnL      int64   `json:"net_pnl"`
}

type LeaderboardPage struct {
	Entries []Entry `json:"entries"`
	Total   int64   `json:"total"`
	Offset  int     `json:"offset"`
	Limit   int     `json:"limit"`
	HasMore bool    `json:"has_more"`
}

type Stats struct {
	TotalUsers  int64   `json:"total_users"`
	TotalWagers int64   `json:"total_wagers"`
	TotalVolume uint64  `json:"total_volume"`
	AvgWinRate  float64 `json:"avg_win_rate"`
}

type SettlementEvent struct {
	WagerPubkey  string
	WinnerPubkey string
	LoserPubkey  string
	Stake        uint64
	MatchID      string
	TxSignature  string
	WinningSide  uint8
	SettledAt    time.Time
}

func (s *Service) GetLeaderboard(ctx context.Context, offset, limit int) (*LeaderboardPage, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var total int64
	if err := s.gdb.WithContext(ctx).Model(&db.LeaderboardEntry{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("leaderboard count: %w", err)
	}

	var rows []db.LeaderboardEntry
	if err := s.gdb.WithContext(ctx).
		Order(leaderboardOrder).
		Offset(offset).
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("leaderboard query: %w", err)
	}

	entries := make([]Entry, 0, len(rows))
	for i, row := range rows {
		entries = append(entries, Entry{
			Rank:        offset + i + 1,
			UserID:      row.UserID.String(),
			Email:       row.Email,
			DisplayName: row.DisplayName,
			TotalWagers: row.TotalWagers,
			Wins:        row.Wins,
			Losses:      row.Losses,
			WinRate:     row.WinRate(),
			TotalVolume: row.TotalVolume,
			NetPnL:      row.NetPnL,
		})
	}

	nextOffset := offset + len(entries)
	return &LeaderboardPage{
		Entries: entries,
		Total:   total,
		Offset:  offset,
		Limit:   limit,
		HasMore: nextOffset < int(total),
	}, nil
}

func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	var stats Stats
	row := s.gdb.WithContext(ctx).Model(&db.LeaderboardEntry{}).Select(`
		COUNT(*) AS total_users,
		COALESCE(SUM(total_wagers), 0) AS total_wagers,
		COALESCE(SUM(total_volume), 0) AS total_volume,
		COALESCE(AVG(CASE
			WHEN total_wagers > 0 THEN (wins::double precision / total_wagers::double precision) * 100
			ELSE 0
		END), 0) AS avg_win_rate
	`).Row()
	if err := row.Scan(&stats.TotalUsers, &stats.TotalWagers, &stats.TotalVolume, &stats.AvgWinRate); err != nil {
		return nil, fmt.Errorf("leaderboard stats: %w", err)
	}
	return &stats, nil
}

func (s *Service) RecordSettlement(ctx context.Context, ev SettlementEvent) error {
	if ev.WagerPubkey == "" {
		return fmt.Errorf("wager pubkey is required")
	}
	if ev.WinnerPubkey == "" || ev.LoserPubkey == "" {
		return fmt.Errorf("winner and loser pubkeys are required")
	}
	if ev.SettledAt.IsZero() {
		ev.SettledAt = time.Now().UTC()
	}
	return s.recordSettlement(ctx, ev)
}

func (s *Service) SyncSettledWager(
	ctx context.Context,
	wager chainsol.Wager,
	winningSide uint8,
	txSignature string,
) error {
	if wager.Status != chainsol.WagerStatusSettled {
		return fmt.Errorf("wager is not settled")
	}
	winnerPubkey, err := wager.WinnerPubkey(winningSide)
	if err != nil {
		return err
	}
	loserPubkey := wager.Maker
	if winnerPubkey.Equals(wager.Maker) && wager.HasCounterparty() {
		loserPubkey = wager.Taker
	}
	if !wager.HasCounterparty() || loserPubkey.IsZero() || loserPubkey.Equals(chainsol.SystemProgramID) {
		return fmt.Errorf("wager has no matched counterparty")
	}
	return s.recordSettlement(ctx, SettlementEvent{
		WagerPubkey:  wager.Pubkey.String(),
		WinnerPubkey: winnerPubkey.String(),
		LoserPubkey:  loserPubkey.String(),
		Stake:        wager.Stake,
		MatchID:      wager.MatchIDString(),
		TxSignature:  txSignature,
		WinningSide:  winningSide,
		SettledAt:    time.Now().UTC(),
	})
}

func (s *Service) recordSettlement(ctx context.Context, ev SettlementEvent) error {
	now := time.Now().UTC()
	return s.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "wager_pubkey"}},
			DoNothing: true,
		}).Create(&db.LeaderboardSettlement{
			WagerPubkey:  ev.WagerPubkey,
			MatchID:      ev.MatchID,
			WinnerPubkey: ev.WinnerPubkey,
			LoserPubkey:  ev.LoserPubkey,
			Stake:        ev.Stake,
			SettledAt:    ev.SettledAt,
			SyncedAt:     now,
			TxSignature:  ev.TxSignature,
			WinningSide:  ev.WinningSide,
		})
		if res.Error != nil {
			return fmt.Errorf("record settlement sync state: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return nil
		}

		if err := s.applyParticipantResult(tx, now, ev.WinnerPubkey, ev.Stake, true); err != nil {
			return err
		}
		if err := s.applyParticipantResult(tx, now, ev.LoserPubkey, ev.Stake, false); err != nil {
			return err
		}
		return nil
	})
}

func (s *Service) applyParticipantResult(
	tx *gorm.DB,
	now time.Time,
	pubkey string,
	stake uint64,
	won bool,
) error {
	if pubkey == "" {
		return nil
	}
	var link db.WalletLink
	if err := tx.
		Where("pubkey = ?", pubkey).
		Preload("User").
		First(&link).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return fmt.Errorf("lookup wallet link: %w", err)
	}

	wins := int64(0)
	losses := int64(1)
	netPnL := -int64(stake)
	if won {
		wins = 1
		losses = 0
		netPnL = int64(stake)
	}
	stakeVolume := stake * 2

	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"email":        link.User.Email,
			"display_name": link.User.DisplayName,
			"total_wagers": gorm.Expr("leaderboard_entries.total_wagers + EXCLUDED.total_wagers"),
			"wins":         gorm.Expr("leaderboard_entries.wins + EXCLUDED.wins"),
			"losses":       gorm.Expr("leaderboard_entries.losses + EXCLUDED.losses"),
			"total_volume": gorm.Expr("leaderboard_entries.total_volume + EXCLUDED.total_volume"),
			"net_pnl":      gorm.Expr("leaderboard_entries.net_pnl + EXCLUDED.net_pnl"),
			"updated_at":   now,
		}),
	}).Create(&db.LeaderboardEntry{
		UserID:      link.UserID,
		Email:       link.User.Email,
		DisplayName: link.User.DisplayName,
		TotalWagers: 1,
		Wins:        wins,
		Losses:      losses,
		TotalVolume: stakeVolume,
		NetPnL:      netPnL,
		UpdatedAt:   now,
	}).Error; err != nil {
		return fmt.Errorf("upsert leaderboard entry: %w", err)
	}
	return nil
}

func (s *Service) GetRank(ctx context.Context, userID uuid.UUID) (*Entry, error) {
	var entry db.LeaderboardEntry
	if err := s.gdb.WithContext(ctx).Where("user_id = ?", userID).First(&entry).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	var rank int64
	if err := s.gdb.WithContext(ctx).
		Model(&db.LeaderboardEntry{}).
		Where(`
			net_pnl > ? OR
			(net_pnl = ? AND wins > ?) OR
			(net_pnl = ? AND wins = ? AND total_volume > ?) OR
			(net_pnl = ? AND wins = ? AND total_volume = ? AND user_id < ?)
		`,
			entry.NetPnL,
			entry.NetPnL, entry.Wins,
			entry.NetPnL, entry.Wins, entry.TotalVolume,
			entry.NetPnL, entry.Wins, entry.TotalVolume, entry.UserID,
		).
		Count(&rank).Error; err != nil {
		return nil, err
	}

	return &Entry{
		Rank:        int(rank) + 1,
		UserID:      entry.UserID.String(),
		Email:       entry.Email,
		DisplayName: entry.DisplayName,
		TotalWagers: entry.TotalWagers,
		Wins:        entry.Wins,
		Losses:      entry.Losses,
		WinRate:     entry.WinRate(),
		TotalVolume: entry.TotalVolume,
		NetPnL:      entry.NetPnL,
	}, nil
}
