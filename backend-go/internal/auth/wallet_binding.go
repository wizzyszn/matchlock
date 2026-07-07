package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/matchlock/backend-go/internal/db"
)

var (
	ErrWalletOwnedByOther = errors.New("wallet owned by another account")
	ErrWagerMakerMismatch = errors.New("wager maker does not match your linked wallet")
)

// WalletBindingStatus describes how a pubkey relates to the current user.
type WalletBindingStatus string

const (
	WalletUnlinked      WalletBindingStatus = "unlinked"
	WalletLinkedToYou   WalletBindingStatus = "linked_to_you"
	WalletLinkedToOther WalletBindingStatus = "linked_to_other"
)

type WalletBindingView struct {
	Pubkey        string              `json:"pubkey"`
	Status        WalletBindingStatus `json:"status"`
	OwnerLabel    string              `json:"owner_label,omitempty"`
	OwnerUserID   string              `json:"owner_user_id,omitempty"`
	LinkedToYou   bool                `json:"linked_to_you"`
	OwnedByOther  bool                `json:"owned_by_other"`
}

// CheckWalletBinding reports whether a pubkey is free, linked to user, or taken.
func (s *Service) CheckWalletBinding(ctx context.Context, userID uuid.UUID, pubkey string) (WalletBindingView, error) {
	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return WalletBindingView{}, fmt.Errorf("pubkey required")
	}
	view := WalletBindingView{Pubkey: pubkey, Status: WalletUnlinked}

	var link db.WalletLink
	result := s.gdb.WithContext(ctx).Preload("User").Where("pubkey = ?", pubkey).Limit(1).Find(&link)
	if result.Error != nil {
		return WalletBindingView{}, result.Error
	}
	if result.RowsAffected == 0 {
		return view, nil
	}

	if link.UserID == userID {
		view.Status = WalletLinkedToYou
		view.LinkedToYou = true
		return view, nil
	}

	label := link.User.Email
	if name := strings.TrimSpace(link.User.DisplayName); name != "" {
		label = name
	}
	view.Status = WalletLinkedToOther
	view.OwnedByOther = true
	view.OwnerUserID = link.UserID.String()
	view.OwnerLabel = label
	return view, nil
}

// AssertUserOwnsWallet ensures the pubkey is linked to userID.
func (s *Service) AssertUserOwnsWallet(ctx context.Context, userID uuid.UUID, pubkey string) error {
	view, err := s.CheckWalletBinding(ctx, userID, pubkey)
	if err != nil {
		return err
	}
	switch view.Status {
	case WalletLinkedToYou:
		return nil
	case WalletLinkedToOther:
		return ErrWalletOwnedByOther
	default:
		return ErrWalletNotLinked
	}
}

// UserLinkedPubkeys returns all pubkeys linked to a user.
func (s *Service) UserLinkedPubkeys(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var links []db.WalletLink
	if err := s.gdb.WithContext(ctx).Where("user_id = ?", userID).Find(&links).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(links))
	for _, link := range links {
		out = append(out, link.Pubkey)
	}
	return out, nil
}