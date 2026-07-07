package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type WagerCacheItem struct {
	Pubkey       string `json:"pubkey"`
	Maker        string `json:"maker"`
	InvitedTaker string `json:"invited_taker,omitempty"`
	Taker        string `json:"taker"`
	MatchID      string `json:"match_id"`
	MakerSide    uint8  `json:"maker_side"`
	TakerSide    uint8  `json:"taker_side"`
	Stake        uint64 `json:"stake"`
	Status       uint8  `json:"status"`
	Bump         uint8  `json:"bump"`
	VaultBump    uint8  `json:"vault_bump"`
}

const (
	wagerKeyFmt   = keyPrefix + "wager:%s"
	wagerIndexKey = keyPrefix + "wager_index"
)

var ErrWagerNotFound = errors.New("wager not found")

func (s *RedisStore) SetWager(ctx context.Context, w WagerCacheItem) error {
	payload, err := json.Marshal(w)
	if err != nil {
		return fmt.Errorf("marshal wager: %w", err)
	}
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, fmt.Sprintf(wagerKeyFmt, w.Pubkey), payload, 0)
	pipe.SAdd(ctx, wagerIndexKey, w.Pubkey)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("set wager: %w", err)
	}
	return nil
}

func (s *RedisStore) GetWager(ctx context.Context, pubkey string) (WagerCacheItem, error) {
	raw, err := s.client.Get(ctx, fmt.Sprintf(wagerKeyFmt, pubkey)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return WagerCacheItem{}, fmt.Errorf("wager %s: %w", pubkey, ErrWagerNotFound)
		}
		return WagerCacheItem{}, fmt.Errorf("get wager %s: %w", pubkey, err)
	}
	var w WagerCacheItem
	if err := json.Unmarshal(raw, &w); err != nil {
		return WagerCacheItem{}, fmt.Errorf("decode wager %s: %w", pubkey, err)
	}
	return w, nil
}

func (s *RedisStore) ListWagers(ctx context.Context) ([]WagerCacheItem, error) {
	pubkeys, err := s.client.SMembers(ctx, wagerIndexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("list wager ids: %w", err)
	}
	out := make([]WagerCacheItem, 0, len(pubkeys))
	for _, pk := range pubkeys {
		w, err := s.GetWager(ctx, pk)
		if err != nil {
			if errors.Is(err, ErrWagerNotFound) || errors.Is(err, redis.Nil) {
				continue
			}
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

func (s *RedisStore) DeleteWager(ctx context.Context, pubkey string) error {
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, fmt.Sprintf(wagerKeyFmt, pubkey))
	pipe.SRem(ctx, wagerIndexKey, pubkey)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete wager: %w", err)
	}
	return nil
}
