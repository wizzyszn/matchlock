package cache

import (
	"time"
)

// Final source labels for match settlement eligibility.
const (
	FinalSourceTxline   = "txline"
	FinalSourceInferred = "inferred"
)

// PendingSettlement is a durable retry item for failed keeper settlements.
type PendingSettlement struct {
	MatchID     string    `json:"match_id"`
	WagerPubkey string    `json:"wager_pubkey"`
	FixtureID   int64     `json:"fixture_id"`
	Seq         int32     `json:"seq"`
	GameState   string    `json:"game_state"`
	Attempts    int       `json:"attempts"`
	LastError   string    `json:"last_error,omitempty"`
	NextRetryAt time.Time `json:"next_retry_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WagerSettlementView is the API-facing settlement state for a wager.
type WagerSettlementView struct {
	State           string     `json:"state"`
	MatchFinal      bool       `json:"match_final"`
	FinalSource     string     `json:"final_source,omitempty"`
	PendingAttempts int        `json:"pending_attempts,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
	NextRetryAt     *time.Time `json:"next_retry_at,omitempty"`
	SettledAt       *time.Time `json:"settled_at,omitempty"`
	TxSignature     string     `json:"tx_signature,omitempty"`
}