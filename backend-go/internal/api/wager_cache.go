package api

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	chainsol "github.com/matchlock/backend-go/internal/solana"
)

type CachedWagerIndex struct {
	store *cache.RedisStore
}

func NewCachedWagerIndex(store *cache.RedisStore) *CachedWagerIndex {
	return &CachedWagerIndex{store: store}
}

func (c *CachedWagerIndex) ListWagers(ctx context.Context, filter chainsol.WagerFilter) ([]chainsol.Wager, error) {
	items, err := c.store.ListWagers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list cached wagers: %w", err)
	}

	out := make([]chainsol.Wager, 0, len(items))
	for _, item := range items {
		if filter.Status != nil && item.Status != *filter.Status {
			continue
		}
		if filter.MatchID != "" && item.MatchID != filter.MatchID {
			continue
		}
		if filter.Wallet != "" && item.Maker != filter.Wallet && item.Taker != filter.Wallet {
			continue
		}
		w, err := cacheItemToWager(item)
		if err != nil {
			continue
		}
		out = append(out, w)
	}
	return out, nil
}

func (c *CachedWagerIndex) GetWager(ctx context.Context, pubkey solana.PublicKey) (chainsol.Wager, error) {
	item, err := c.store.GetWager(ctx, pubkey.String())
	if err != nil {
		return chainsol.Wager{}, fmt.Errorf("get cached wager: %w", err)
	}
	return cacheItemToWager(item)
}

func cacheItemToWager(item cache.WagerCacheItem) (chainsol.Wager, error) {
	pk, err := solana.PublicKeyFromBase58(item.Pubkey)
	if err != nil {
		return chainsol.Wager{}, err
	}
	maker, err := solana.PublicKeyFromBase58(item.Maker)
	if err != nil {
		return chainsol.Wager{}, err
	}
	taker, err := solana.PublicKeyFromBase58(item.Taker)
	if err != nil {
		return chainsol.Wager{}, err
	}
	var invitedTaker solana.PublicKey
	if item.InvitedTaker != "" {
		invitedTaker, err = solana.PublicKeyFromBase58(item.InvitedTaker)
		if err != nil {
			invitedTaker = solana.PublicKey{}
		}
	}
	var matchID [32]byte
	copy(matchID[:], item.MatchID)

	return chainsol.Wager{
		Pubkey:       pk,
		Maker:        maker,
		InvitedTaker: invitedTaker,
		Taker:        taker,
		MatchID:      matchID,
		MatchIDLen:   uint8(len(item.MatchID)),
		MakerSide:    item.MakerSide,
		TakerSide:    item.TakerSide,
		Stake:        item.Stake,
		Status:       item.Status,
		Bump:         item.Bump,
		VaultBump:    item.VaultBump,
	}, nil
}
