package db

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User is a platform identity (email); signing stays wallet-based.
type User struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email       string    `gorm:"uniqueIndex;size:320;not null"`
	DisplayName string    `gorm:"size:120"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// MagicLinkToken is a single-use email login token (stored hashed).
type MagicLinkToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email     string    `gorm:"index;size:320;not null"`
	TokenHash string    `gorm:"uniqueIndex;size:64;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (m *MagicLinkToken) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// Session stores a refresh token hash for cookie-based auth rotation.
type Session struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID           uuid.UUID `gorm:"type:uuid;index;not null"`
	RefreshTokenHash string    `gorm:"uniqueIndex;size:64;not null"`
	ExpiresAt        time.Time `gorm:"index;not null"`
	UserAgent        string    `gorm:"size:512"`
	IPAddress        string    `gorm:"size:64"`
	CreatedAt        time.Time
	User             User `gorm:"constraint:OnDelete:CASCADE"`
}

func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// WalletLink associates a Solana pubkey with a user (verified via signed nonce).
// Pubkey is globally unique — one wallet, one Matchlock account.
type WalletLink struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;index;not null"`
	Pubkey     string    `gorm:"uniqueIndex;size:44;not null"`
	Label      string    `gorm:"size:64"`
	IsPrimary  bool      `gorm:"not null;default:false"`
	VerifiedAt time.Time `gorm:"not null"`
	CreatedAt  time.Time
	User       User `gorm:"constraint:OnDelete:CASCADE"`
}

func (w *WalletLink) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// WalletLinkChallenge is a one-time signed nonce for linking a wallet.
type WalletLinkChallenge struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID `gorm:"type:uuid;index;not null"`
	Pubkey      string    `gorm:"index;size:44;not null"`
	MessageHash string    `gorm:"uniqueIndex;size:64;not null"`
	ExpiresAt   time.Time `gorm:"index;not null"`
	UsedAt      *time.Time
	CreatedAt   time.Time
}

func (c *WalletLinkChallenge) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// WagerInviteStatus tracks off-chain invite lifecycle.
type WagerInviteStatus string

const (
	InvitePending  WagerInviteStatus = "pending"
	InviteAccepted WagerInviteStatus = "accepted"
	InviteDeclined WagerInviteStatus = "declined"
	InviteExpired  WagerInviteStatus = "expired"
)

// WagerInvite notifies a user about a direct challenge (optionally before on-chain wager).
type WagerInvite struct {
	ID              uuid.UUID         `gorm:"type:uuid;primaryKey"`
	MakerUserID     uuid.UUID         `gorm:"type:uuid;index;not null"`
	RecipientEmail  string            `gorm:"index;size:320"`
	RecipientUserID *uuid.UUID        `gorm:"type:uuid;index"`
	WagerPubkey     string            `gorm:"size:44;index"`
	MatchID         string            `gorm:"size:64;not null"`
	MakerSide       string            `gorm:"size:8;not null"`
	Stake           uint64            `gorm:"not null"`
	HomeTeam        string            `gorm:"size:120"`
	AwayTeam        string            `gorm:"size:120"`
	Status          WagerInviteStatus `gorm:"size:16;index;not null;default:pending"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Maker           User `gorm:"foreignKey:MakerUserID;constraint:OnDelete:CASCADE"`
}

func (w *WagerInvite) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

type LeaderboardEntry struct {
	UserID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email       string    `gorm:"size:320;not null"`
	DisplayName string    `gorm:"size:120"`
	TotalWagers int64     `gorm:"not null;default:0"`
	Wins        int64     `gorm:"not null;default:0"`
	Losses      int64     `gorm:"not null;default:0"`
	TotalVolume uint64    `gorm:"not null;default:0"`
	NetPnL      int64     `gorm:"column:net_pnl;not null;default:0"`
	UpdatedAt   time.Time
}

func (l *LeaderboardEntry) BeforeCreate(tx *gorm.DB) error {
	return nil
}

func (l LeaderboardEntry) WinRate() float64 {
	if l.TotalWagers == 0 {
		return 0
	}
	return float64(l.Wins) / float64(l.TotalWagers) * 100
}
