package leaderboard

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/matchlock/backend-go/internal/db"
	"gorm.io/gorm"
)

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

func (s *Service) GetLeaderboard(ctx context.Context, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows []db.LeaderboardEntry
	if err := s.gdb.WithContext(ctx).
		Order("net_pnl DESC, wins DESC, total_volume DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("leaderboard query: %w", err)
	}

	entries := make([]Entry, 0, len(rows))
	for i, row := range rows {
		entries = append(entries, Entry{
			Rank:        i + 1,
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
	return entries, nil
}

type SettlementEvent struct {
	WinnerPubkey string
	LoserPubkey  string
	Stake        uint64
	MatchID      string
}

func (s *Service) RecordSettlement(ctx context.Context, ev SettlementEvent) error {
	now := time.Now().UTC()
	stakeVolume := ev.Stake * 2

	for _, pubkey := range []string{ev.WinnerPubkey, ev.LoserPubkey} {
		if pubkey == "" {
			continue
		}
		var link db.WalletLink
		if err := s.gdb.WithContext(ctx).
			Where("pubkey = ?", pubkey).
			Preload("User").
			First(&link).Error; err != nil {
			continue
		}

		var entry db.LeaderboardEntry
		err := s.gdb.WithContext(ctx).
			Where("user_id = ?", link.UserID).
			First(&entry).Error

		isNew := err == gorm.ErrRecordNotFound
		if isNew {
			entry = db.LeaderboardEntry{
				UserID: link.UserID,
				Email:  link.User.Email,
			}
		}
		entry.DisplayName = link.User.DisplayName
		entry.Email = link.User.Email
		entry.TotalVolume += stakeVolume
		entry.TotalWagers++
		entry.UpdatedAt = now

		if pubkey == ev.WinnerPubkey {
			entry.Wins++
			entry.NetPnL += int64(ev.Stake)
		} else {
			entry.Losses++
			entry.NetPnL -= int64(ev.Stake)
		}

		if isNew {
			if err := s.gdb.WithContext(ctx).Create(&entry).Error; err != nil {
				return fmt.Errorf("create leaderboard entry: %w", err)
			}
		} else {
			if err := s.gdb.WithContext(ctx).Save(&entry).Error; err != nil {
				return fmt.Errorf("update leaderboard entry: %w", err)
			}
		}
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
		Where("net_pnl > ? OR (net_pnl = ? AND wins > ?)", entry.NetPnL, entry.NetPnL, entry.Wins).
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
