package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/matchlock/backend-go/internal/db"
	"github.com/matchlock/backend-go/internal/email"
	"gorm.io/gorm"
)

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidToken       = errors.New("invalid token")
	ErrRateLimited        = errors.New("rate limited")
	ErrUserNotFound       = errors.New("user not found")
	ErrWalletNotLinked    = errors.New("wallet not linked")
	ErrInviteNotFound     = errors.New("invite not found")
	ErrInvalidInvite      = errors.New("invalid invite")
	ErrInvalidDisplayName = errors.New("invalid display name")
)

const (
	magicLinkCooldown  = 5 * time.Second
	magicLinkHourlyMax = 100
)

// WalletRegistrar creates an on-chain WalletProfile PDA for a given wallet.
type WalletRegistrar interface {
	RegisterWallet(ctx context.Context, keeperKey solana.PrivateKey, wallet solana.PublicKey, userIDHash [32]byte) error
}

// Service implements magic-link auth, sessions, wallet links, and wager invites.
type Service struct {
	gdb             *gorm.DB
	mailer          *email.Mailer
	emailQueue      *email.Queue
	tokens          TokenConfig
	walletRegistrar WalletRegistrar
	keeperKey       solana.PrivateKey
}

func NewService(gdb *gorm.DB, mailer *email.Mailer, tokens TokenConfig) *Service {
	return &Service{gdb: gdb, mailer: mailer, tokens: tokens}
}

// SetWalletRegistrar enables on-chain wallet registration during LinkWallet.
func (s *Service) SetWalletRegistrar(registrar WalletRegistrar, keeperKey solana.PrivateKey) {
	s.walletRegistrar = registrar
	s.keeperKey = keeperKey
}

// SetEmailQueue enables asynchronous SMTP delivery via a background worker.
func (s *Service) SetEmailQueue(q *email.Queue) {
	s.emailQueue = q
}

func normalizeEmail(raw string) (string, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "", fmt.Errorf("email required")
	}
	addr, err := mail.ParseAddress(raw)
	if err != nil {
		return "", fmt.Errorf("invalid email")
	}
	return strings.ToLower(addr.Address), nil
}

// RequestMagicLink emails a single-use login link.
func (s *Service) RequestMagicLink(ctx context.Context, emailAddr string) error {
	emailAddr, err := normalizeEmail(emailAddr)
	if err != nil {
		return err
	}

	var last db.MagicLinkToken
	lastResult := s.gdb.WithContext(ctx).
		Where("email = ?", emailAddr).
		Order("created_at DESC").
		Limit(1).
		Find(&last)
	if lastResult.Error != nil {
		return fmt.Errorf("rate limit check: %w", lastResult.Error)
	}
	if lastResult.RowsAffected > 0 && time.Since(last.CreatedAt) < magicLinkCooldown {
		return ErrRateLimited
	}

	var recent int64
	cutoff := time.Now().UTC().Add(-time.Hour)
	if err := s.gdb.WithContext(ctx).Model(&db.MagicLinkToken{}).
		Where("email = ? AND created_at > ?", emailAddr, cutoff).
		Count(&recent).Error; err != nil {
		return fmt.Errorf("rate limit check: %w", err)
	}
	if recent >= magicLinkHourlyMax {
		return ErrRateLimited
	}

	raw, hash, err := NewOpaqueToken()
	if err != nil {
		return err
	}
	expires := time.Now().UTC().Add(s.tokens.MagicLinkTTL)
	row := db.MagicLinkToken{
		Email:     emailAddr,
		TokenHash: hash,
		ExpiresAt: expires,
	}
	if err := s.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("store magic link: %w", err)
	}

	link := strings.TrimRight(s.tokens.FrontendURL, "/") + "/auth/verify?token=" + raw
	if err := s.enqueueMagicLink(emailAddr, link); err != nil {
		return fmt.Errorf("queue magic link email: %w", err)
	}
	return nil
}

func (s *Service) enqueueMagicLink(to, link string) error {
	job := email.Job{Type: email.JobMagicLink, To: to, Link: link}
	if s.emailQueue != nil {
		return s.emailQueue.Enqueue(job)
	}
	return s.mailer.SendMagicLink(to, link)
}

func (s *Service) enqueueWagerInvite(to, makerEmail, matchLabel, inviteURL string) error {
	job := email.Job{
		Type:       email.JobWagerInvite,
		To:         to,
		MakerEmail: makerEmail,
		MatchLabel: matchLabel,
		InviteURL:  inviteURL,
	}
	if s.emailQueue != nil {
		return s.emailQueue.Enqueue(job)
	}
	return s.mailer.SendWagerInvite(to, makerEmail, matchLabel, inviteURL)
}

// AuthSession bundles tokens issued after login or refresh.
type AuthSession struct {
	User          db.User
	AccessToken   string
	AccessExpiry  time.Time
	RefreshRaw    string
	RefreshExpiry time.Time
}

// VerifyMagicLink consumes a magic link token and creates a new session.
func (s *Service) VerifyMagicLink(ctx context.Context, rawToken, userAgent, ip string) (AuthSession, error) {
	if strings.TrimSpace(rawToken) == "" {
		return AuthSession{}, ErrInvalidToken
	}
	hash := HashToken(rawToken)
	now := time.Now().UTC()

	var row db.MagicLinkToken
	result := s.gdb.WithContext(ctx).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, now).
		Limit(1).
		Find(&row)
	if result.Error != nil {
		return AuthSession{}, result.Error
	}
	if result.RowsAffected == 0 {
		return AuthSession{}, ErrInvalidToken
	}

	err := s.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&db.MagicLinkToken{}).
			Where("id = ? AND used_at IS NULL", row.ID).
			Update("used_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrInvalidToken
		}
		return nil
	})
	if err != nil {
		return AuthSession{}, err
	}

	var user db.User
	err = s.gdb.WithContext(ctx).Where("email = ?", row.Email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = db.User{Email: row.Email}
		if err := s.gdb.WithContext(ctx).Create(&user).Error; err != nil {
			return AuthSession{}, err
		}
	} else if err != nil {
		return AuthSession{}, err
	}

	return s.issueSession(ctx, user, userAgent, ip)
}

func (s *Service) issueSession(ctx context.Context, user db.User, userAgent, ip string) (AuthSession, error) {
	access, accessExp, err := NewAccessToken(s.tokens, user.ID, user.Email)
	if err != nil {
		return AuthSession{}, err
	}
	refreshRaw, refreshHash, err := NewOpaqueToken()
	if err != nil {
		return AuthSession{}, err
	}
	refreshExp := time.Now().UTC().Add(s.tokens.RefreshTTL)
	sess := db.Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshHash,
		ExpiresAt:        refreshExp,
		UserAgent:        truncate(userAgent, 512),
		IPAddress:        truncate(ip, 64),
	}
	if err := s.gdb.WithContext(ctx).Create(&sess).Error; err != nil {
		return AuthSession{}, err
	}
	return AuthSession{
		User:          user,
		AccessToken:   access,
		AccessExpiry:  accessExp,
		RefreshRaw:    refreshRaw,
		RefreshExpiry: refreshExp,
	}, nil
}

// RefreshSession rotates the refresh token and issues a new access token.
func (s *Service) RefreshSession(ctx context.Context, refreshRaw, userAgent, ip string) (AuthSession, error) {
	if strings.TrimSpace(refreshRaw) == "" {
		return AuthSession{}, ErrUnauthorized
	}
	hash := HashToken(refreshRaw)
	now := time.Now().UTC()

	var sess db.Session
	err := s.gdb.WithContext(ctx).
		Preload("User").
		Where("refresh_token_hash = ? AND expires_at > ?", hash, now).
		First(&sess).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthSession{}, ErrUnauthorized
		}
		return AuthSession{}, err
	}

	if err := s.gdb.WithContext(ctx).Delete(&sess).Error; err != nil {
		return AuthSession{}, err
	}
	return s.issueSession(ctx, sess.User, userAgent, ip)
}

// Logout revokes a refresh session.
func (s *Service) Logout(ctx context.Context, refreshRaw string) error {
	if strings.TrimSpace(refreshRaw) == "" {
		return nil
	}
	hash := HashToken(refreshRaw)
	return s.gdb.WithContext(ctx).
		Where("refresh_token_hash = ?", hash).
		Delete(&db.Session{}).Error
}

// UserFromAccessToken validates a JWT access token.
func (s *Service) UserFromAccessToken(ctx context.Context, accessRaw string) (db.User, error) {
	claims, err := ParseAccessToken(s.tokens, accessRaw)
	if err != nil {
		return db.User{}, ErrUnauthorized
	}
	var user db.User
	if err := s.gdb.WithContext(ctx).First(&user, "id = ?", claims.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return db.User{}, ErrUnauthorized
		}
		return db.User{}, err
	}
	return user, nil
}

type WalletView struct {
	Pubkey    string    `json:"pubkey"`
	Label     string    `json:"label,omitempty"`
	IsPrimary bool      `json:"is_primary"`
	LinkedAt  time.Time `json:"linked_at"`
}

type UserProfile struct {
	ID          uuid.UUID    `json:"id"`
	Email       string       `json:"email"`
	DisplayName string       `json:"display_name,omitempty"`
	Wallets     []WalletView `json:"wallets"`
}

var displayNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

func normalizeDisplayName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if !displayNamePattern.MatchString(name) {
		return "", ErrInvalidDisplayName
	}
	return name, nil
}

// UpdateDisplayName sets the user's public username.
func (s *Service) UpdateDisplayName(ctx context.Context, userID uuid.UUID, displayName string) (UserProfile, error) {
	name, err := normalizeDisplayName(displayName)
	if err != nil {
		return UserProfile{}, err
	}
	res := s.gdb.WithContext(ctx).Model(&db.User{}).
		Where("id = ?", userID).
		Update("display_name", name)
	if res.Error != nil {
		return UserProfile{}, res.Error
	}
	if res.RowsAffected == 0 {
		return UserProfile{}, ErrUserNotFound
	}
	return s.GetProfile(ctx, userID)
}

// GetProfile returns the user and linked wallets.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (UserProfile, error) {
	var user db.User
	if err := s.gdb.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserProfile{}, ErrUserNotFound
		}
		return UserProfile{}, err
	}
	var wallets []db.WalletLink
	if err := s.gdb.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_primary DESC, created_at ASC").
		Find(&wallets).Error; err != nil {
		return UserProfile{}, err
	}
	views := make([]WalletView, 0, len(wallets))
	for _, w := range wallets {
		views = append(views, WalletView{
			Pubkey:    w.Pubkey,
			Label:     w.Label,
			IsPrimary: w.IsPrimary,
			LinkedAt:  w.VerifiedAt,
		})
	}
	return UserProfile{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Wallets:     views,
	}, nil
}

// CreateWalletLinkChallenge stores a one-time nonce and returns the message to sign.
func (s *Service) CreateWalletLinkChallenge(ctx context.Context, userID uuid.UUID, pubkey string) (string, error) {
	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return "", fmt.Errorf("pubkey required")
	}
	binding, err := s.CheckWalletBinding(ctx, userID, pubkey)
	if err != nil {
		return "", err
	}
	if binding.OwnedByOther {
		return "", ErrWalletOwnedByOther
	}
	issued := time.Now().UTC()
	message := BuildWalletLinkMessage(userID.String(), pubkey, issued)
	hash := HashToken(message)
	row := db.WalletLinkChallenge{
		UserID:      userID,
		Pubkey:      pubkey,
		MessageHash: hash,
		ExpiresAt:   issued.Add(walletLinkTTL),
	}
	if err := s.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return "", fmt.Errorf("store wallet link challenge: %w", err)
	}
	return message, nil
}

// LinkWallet verifies a signature and stores the wallet link.
func (s *Service) LinkWallet(ctx context.Context, userID uuid.UUID, pubkey, message, signature string, label string) (WalletView, error) {
	pubkey = strings.TrimSpace(pubkey)
	if err := VerifyWalletLinkSignature(userID.String(), pubkey, message, signature, walletLinkTTL); err != nil {
		return WalletView{}, fmt.Errorf("wallet verification failed: %w", err)
	}

	msgHash := HashToken(message)
	var challenge db.WalletLinkChallenge
	chResult := s.gdb.WithContext(ctx).
		Where("user_id = ? AND pubkey = ? AND message_hash = ? AND used_at IS NULL AND expires_at > ?",
			userID, pubkey, msgHash, time.Now().UTC()).
		Limit(1).
		Find(&challenge)
	if chResult.Error != nil {
		return WalletView{}, chResult.Error
	}
	if chResult.RowsAffected == 0 {
		return WalletView{}, fmt.Errorf("wallet link challenge expired or already used")
	}
	if err := s.gdb.WithContext(ctx).Model(&db.WalletLinkChallenge{}).
		Where("id = ? AND used_at IS NULL", challenge.ID).
		Update("used_at", time.Now().UTC()).Error; err != nil {
		return WalletView{}, err
	}

	binding, err := s.CheckWalletBinding(ctx, userID, pubkey)
	if err != nil {
		return WalletView{}, err
	}
	if binding.OwnedByOther {
		return WalletView{}, ErrWalletOwnedByOther
	}
	if binding.LinkedToYou {
		var existing db.WalletLink
		if err := s.gdb.WithContext(ctx).
			Where("user_id = ? AND pubkey = ?", userID, pubkey).
			First(&existing).Error; err != nil {
			return WalletView{}, err
		}
		return WalletView{
			Pubkey:    existing.Pubkey,
			Label:     existing.Label,
			IsPrimary: existing.IsPrimary,
			LinkedAt:  existing.VerifiedAt,
		}, nil
	}

	var count int64
	if err := s.gdb.WithContext(ctx).Model(&db.WalletLink{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return WalletView{}, err
	}

	link := db.WalletLink{
		UserID:     userID,
		Pubkey:     pubkey,
		Label:      truncate(label, 64),
		IsPrimary:  count == 0,
		VerifiedAt: time.Now().UTC(),
	}
	if err := s.gdb.WithContext(ctx).Create(&link).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			if rebinding, berr := s.CheckWalletBinding(ctx, userID, pubkey); berr == nil {
				if rebinding.OwnedByOther {
					return WalletView{}, ErrWalletOwnedByOther
				}
				if rebinding.LinkedToYou {
					var existing db.WalletLink
					if err := s.gdb.WithContext(ctx).
						Where("user_id = ? AND pubkey = ?", userID, pubkey).
						First(&existing).Error; err != nil {
						return WalletView{}, err
					}
					return WalletView{
						Pubkey:    existing.Pubkey,
						Label:     existing.Label,
						IsPrimary: existing.IsPrimary,
						LinkedAt:  existing.VerifiedAt,
					}, nil
				}
			}
		}
		return WalletView{}, err
	}

	// Create the on-chain WalletProfile PDA so MakeWager can find it.
	if s.walletRegistrar != nil {
		walletPK, pkErr := solana.PublicKeyFromBase58(pubkey)
		if pkErr == nil {
			userHash := sha256.Sum256([]byte(userID.String()))
			if regErr := s.walletRegistrar.RegisterWallet(ctx, s.keeperKey, walletPK, userHash); regErr != nil {
				slog.Error("on-chain wallet registration failed", "pubkey", pubkey, "err", regErr)
				// Don't fail the link — the DB record is saved. Registration can be retried.
			}
		}
	}

	return WalletView{
		Pubkey:    link.Pubkey,
		Label:     link.Label,
		IsPrimary: link.IsPrimary,
		LinkedAt:  link.VerifiedAt,
	}, nil
}

// SetPrimaryWallet marks a linked wallet as primary.
func (s *Service) SetPrimaryWallet(ctx context.Context, userID uuid.UUID, pubkey string) error {
	pubkey = strings.TrimSpace(pubkey)
	return s.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&db.WalletLink{}).
			Where("user_id = ? AND pubkey = ?", userID, pubkey).
			Update("is_primary", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrWalletNotLinked
		}
		return tx.Model(&db.WalletLink{}).
			Where("user_id = ? AND pubkey <> ?", userID, pubkey).
			Update("is_primary", false).Error
	})
}

// UnlinkWallet removes a linked wallet.
func (s *Service) UnlinkWallet(ctx context.Context, userID uuid.UUID, pubkey string) error {
	res := s.gdb.WithContext(ctx).
		Where("user_id = ? AND pubkey = ?", userID, pubkey).
		Delete(&db.WalletLink{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrWalletNotLinked
	}
	return nil
}

type UserLookup struct {
	Email         string `json:"email"`
	UserID        string `json:"user_id,omitempty"`
	HasAccount    bool   `json:"has_account"`
	PrimaryWallet string `json:"primary_wallet,omitempty"`
}

// LookupUserByEmail resolves a friend for direct challenges.
func (s *Service) LookupUserByEmail(ctx context.Context, emailAddr string) (UserLookup, error) {
	emailAddr, err := normalizeEmail(emailAddr)
	if err != nil {
		return UserLookup{}, err
	}
	var user db.User
	err = s.gdb.WithContext(ctx).Where("email = ?", emailAddr).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return UserLookup{Email: emailAddr, HasAccount: false}, nil
	}
	if err != nil {
		return UserLookup{}, err
	}
	var wallet db.WalletLink
	err = s.gdb.WithContext(ctx).
		Where("user_id = ? AND is_primary = true", user.ID).
		First(&wallet).Error
	out := UserLookup{
		Email:      emailAddr,
		UserID:     user.ID.String(),
		HasAccount: true,
	}
	if err == nil {
		out.PrimaryWallet = wallet.Pubkey
	}
	return out, nil
}

type CreateInviteInput struct {
	RecipientEmail string
	WagerPubkey    string
	MatchID        string
	MakerSide      string
	Stake          uint64
	HomeTeam       string
	AwayTeam       string
}

type InviteView struct {
	ID             string    `json:"id"`
	MakerEmail     string    `json:"maker_email"`
	RecipientEmail string    `json:"recipient_email"`
	WagerPubkey    string    `json:"wager_pubkey,omitempty"`
	MatchID        string    `json:"match_id"`
	MakerSide      string    `json:"maker_side"`
	Stake          uint64    `json:"stake"`
	HomeTeam       string    `json:"home_team,omitempty"`
	AwayTeam       string    `json:"away_team,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateWagerInvite stores an invite and emails the recipient.
func (s *Service) CreateWagerInvite(ctx context.Context, maker db.User, in CreateInviteInput) (InviteView, error) {
	emailAddr, err := normalizeEmail(in.RecipientEmail)
	if err != nil {
		return InviteView{}, err
	}
	if emailAddr == maker.Email {
		return InviteView{}, ErrInvalidInvite
	}
	if strings.TrimSpace(in.MatchID) == "" || in.Stake == 0 {
		return InviteView{}, ErrInvalidInvite
	}

	var recipient db.User
	recipientErr := s.gdb.WithContext(ctx).Where("email = ?", emailAddr).First(&recipient).Error
	var recipientID *uuid.UUID
	if recipientErr == nil {
		recipientID = &recipient.ID
	} else if !errors.Is(recipientErr, gorm.ErrRecordNotFound) {
		return InviteView{}, recipientErr
	}

	invite := db.WagerInvite{
		MakerUserID:     maker.ID,
		RecipientEmail:  emailAddr,
		RecipientUserID: recipientID,
		WagerPubkey:     strings.TrimSpace(in.WagerPubkey),
		MatchID:         in.MatchID,
		MakerSide:       in.MakerSide,
		Stake:           in.Stake,
		HomeTeam:        in.HomeTeam,
		AwayTeam:        in.AwayTeam,
		Status:          db.InvitePending,
	}
	if err := s.gdb.WithContext(ctx).Create(&invite).Error; err != nil {
		return InviteView{}, err
	}

	matchLabel := strings.TrimSpace(in.HomeTeam + " vs " + in.AwayTeam)
	if matchLabel == "vs" {
		matchLabel = "Match " + in.MatchID
	}
	inviteURL := strings.TrimRight(s.tokens.FrontendURL, "/") + "/invites/" + invite.ID.String()
	if err := s.enqueueWagerInvite(emailAddr, maker.Email, matchLabel, inviteURL); err != nil {
		return InviteView{}, fmt.Errorf("queue invite email: %w", err)
	}

	return inviteView(maker.Email, invite), nil
}

// ListInvites returns inbox and outbox invites for a user.
func (s *Service) ListInvites(ctx context.Context, user db.User) ([]InviteView, error) {
	var invites []db.WagerInvite
	err := s.gdb.WithContext(ctx).
		Preload("Maker").
		Where("recipient_email = ? OR maker_user_id = ?", user.Email, user.ID).
		Order("created_at DESC").
		Limit(50).
		Find(&invites).Error
	if err != nil {
		return nil, err
	}
	out := make([]InviteView, 0, len(invites))
	for _, inv := range invites {
		out = append(out, inviteView(inv.Maker.Email, inv))
	}
	return out, nil
}

// GetInvite returns a single invite if the user is maker or recipient.
func (s *Service) GetInvite(ctx context.Context, user db.User, inviteID uuid.UUID) (InviteView, error) {
	var invite db.WagerInvite
	err := s.gdb.WithContext(ctx).
		Preload("Maker").
		Where("id = ?", inviteID).
		First(&invite).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return InviteView{}, ErrInviteNotFound
		}
		return InviteView{}, err
	}
	if invite.MakerUserID != user.ID && invite.RecipientEmail != user.Email {
		return InviteView{}, ErrUnauthorized
	}
	return inviteView(invite.Maker.Email, invite), nil
}

// UpdateInviteStatus updates invite status for the recipient.
func (s *Service) UpdateInviteStatus(ctx context.Context, user db.User, inviteID uuid.UUID, status db.WagerInviteStatus) (InviteView, error) {
	if status != db.InviteAccepted && status != db.InviteDeclined {
		return InviteView{}, ErrInvalidInvite
	}
	var invite db.WagerInvite
	err := s.gdb.WithContext(ctx).Preload("Maker").Where("id = ?", inviteID).First(&invite).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return InviteView{}, ErrInviteNotFound
		}
		return InviteView{}, err
	}
	if invite.RecipientEmail != user.Email {
		return InviteView{}, ErrUnauthorized
	}
	if invite.Status != db.InvitePending {
		return InviteView{}, ErrInvalidInvite
	}
	invite.Status = status
	if err := s.gdb.WithContext(ctx).Save(&invite).Error; err != nil {
		return InviteView{}, err
	}
	return inviteView(invite.Maker.Email, invite), nil
}

// AttachWagerToInvite sets the on-chain wager pubkey after maker creates the wager.
func (s *Service) AttachWagerToInvite(ctx context.Context, user db.User, inviteID uuid.UUID, wagerPubkey string) (InviteView, error) {
	wagerPubkey = strings.TrimSpace(wagerPubkey)
	if wagerPubkey == "" {
		return InviteView{}, ErrInvalidInvite
	}
	var invite db.WagerInvite
	err := s.gdb.WithContext(ctx).Preload("Maker").Where("id = ?", inviteID).First(&invite).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return InviteView{}, ErrInviteNotFound
		}
		return InviteView{}, err
	}
	if invite.MakerUserID != user.ID {
		return InviteView{}, ErrUnauthorized
	}
	invite.WagerPubkey = wagerPubkey
	if err := s.gdb.WithContext(ctx).Save(&invite).Error; err != nil {
		return InviteView{}, err
	}
	return inviteView(invite.Maker.Email, invite), nil
}

func inviteView(makerEmail string, inv db.WagerInvite) InviteView {
	return InviteView{
		ID:             inv.ID.String(),
		MakerEmail:     makerEmail,
		RecipientEmail: inv.RecipientEmail,
		WagerPubkey:    inv.WagerPubkey,
		MatchID:        inv.MatchID,
		MakerSide:      inv.MakerSide,
		Stake:          inv.Stake,
		HomeTeam:       inv.HomeTeam,
		AwayTeam:       inv.AwayTeam,
		Status:         string(inv.Status),
		CreatedAt:      inv.CreatedAt,
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
