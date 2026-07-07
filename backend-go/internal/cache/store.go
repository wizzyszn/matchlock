package cache

import (
	"context"
	"time"
)

// Store persists match state and settlement idempotency markers.
type Store interface {
	Ping(ctx context.Context) error

	UpsertMatch(ctx context.Context, match Match) error
	GetMatch(ctx context.Context, matchID string) (Match, error)
	ListMatches(ctx context.Context) ([]Match, error)

	// MarkFinalOnce records that a final TxLINE event was observed (telemetry only).
	MarkFinalOnce(ctx context.Context, matchID string) (bool, error)

	// MarkSettled records a successful settlement; false if already settled.
	MarkSettled(ctx context.Context, rec SettlementRecord) (bool, error)
	IsSettled(ctx context.Context, matchID, wagerPubkey string) (bool, error)
	GetSettlement(ctx context.Context, matchID, wagerPubkey string) (SettlementRecord, error)

	EnqueuePendingSettlement(ctx context.Context, item PendingSettlement) error
	GetPendingSettlement(ctx context.Context, matchID, wagerPubkey string) (PendingSettlement, error)
	UpdatePendingSettlement(ctx context.Context, item PendingSettlement) error
	RemovePendingSettlement(ctx context.Context, matchID, wagerPubkey string) error
	ListDuePendingSettlements(ctx context.Context, dueBefore time.Time, limit int) ([]PendingSettlement, error)
	CountPendingSettlements(ctx context.Context) (int64, error)

	// PublishMatchUpdate broadcasts a match update via Pub/Sub (best-effort).
	PublishMatchUpdate(ctx context.Context, match Match) error
}