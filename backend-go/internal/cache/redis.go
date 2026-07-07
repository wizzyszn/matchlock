package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix             = "matchlock:"
	matchKeyFmt           = keyPrefix + "match:%s"
	finalKeyFmt           = keyPrefix + "final:%s"
	settledKeyFmt         = keyPrefix + "settled:%s:%s"
	pendingSettleKeyFmt   = keyPrefix + "pending_settle:%s:%s"
	pendingSettleIndexKey = keyPrefix + "pending_settle_due"
	matchIndexKey         = keyPrefix + "matches"
	matchUpdateChannel    = keyPrefix + "match_updates"
	finalTTL              = 7 * 24 * time.Hour
	settledTTL            = 30 * 24 * time.Hour
	pendingSettleTTL      = 30 * 24 * time.Hour
)

// RedisStore implements Store on Redis.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore connects to Redis and verifies connectivity.
func NewRedisStore(ctx context.Context, redisURL string) (*RedisStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &RedisStore{client: client}, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

func (s *RedisStore) UpsertMatch(ctx context.Context, match Match) error {
	payload, err := json.Marshal(match)
	if err != nil {
		return fmt.Errorf("marshal match: %w", err)
	}
	key := fmt.Sprintf(matchKeyFmt, match.MatchID)
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, key, payload, 0)
	pipe.SAdd(ctx, matchIndexKey, match.MatchID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("upsert match: %w", err)
	}
	return nil
}

func (s *RedisStore) GetMatch(ctx context.Context, matchID string) (Match, error) {
	raw, err := s.client.Get(ctx, fmt.Sprintf(matchKeyFmt, matchID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return Match{}, fmt.Errorf("get match %s: %w", matchID, ErrMatchNotFound)
		}
		return Match{}, fmt.Errorf("get match %s: %w", matchID, err)
	}
	var match Match
	if err := json.Unmarshal(raw, &match); err != nil {
		return Match{}, fmt.Errorf("decode match %s: %w", matchID, err)
	}
	return match, nil
}

func (s *RedisStore) ListMatches(ctx context.Context) ([]Match, error) {
	ids, err := s.client.SMembers(ctx, matchIndexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("list match ids: %w", err)
	}
	out := make([]Match, 0, len(ids))
	for _, id := range ids {
		match, err := s.GetMatch(ctx, id)
		if err != nil {
			if errors.Is(err, ErrMatchNotFound) || errors.Is(err, redis.Nil) {
				continue
			}
			return nil, err
		}
		out = append(out, match)
	}
	return out, nil
}

func (s *RedisStore) MarkFinalOnce(ctx context.Context, matchID string) (bool, error) {
	ok, err := s.client.SetNX(ctx, fmt.Sprintf(finalKeyFmt, matchID), time.Now().UTC().Format(time.RFC3339Nano), finalTTL).Result()
	if err != nil {
		return false, fmt.Errorf("mark final: %w", err)
	}
	return ok, nil
}

func (s *RedisStore) MarkSettled(ctx context.Context, rec SettlementRecord) (bool, error) {
	payload, err := json.Marshal(rec)
	if err != nil {
		return false, fmt.Errorf("marshal settlement: %w", err)
	}
	ok, err := s.client.SetNX(ctx, fmt.Sprintf(settledKeyFmt, rec.MatchID, rec.WagerPubkey), payload, settledTTL).Result()
	if err != nil {
		return false, fmt.Errorf("mark settled: %w", err)
	}
	return ok, nil
}

func (s *RedisStore) IsSettled(ctx context.Context, matchID, wagerPubkey string) (bool, error) {
	n, err := s.client.Exists(ctx, fmt.Sprintf(settledKeyFmt, matchID, wagerPubkey)).Result()
	if err != nil {
		return false, fmt.Errorf("is settled: %w", err)
	}
	return n > 0, nil
}

func (s *RedisStore) GetSettlement(ctx context.Context, matchID, wagerPubkey string) (SettlementRecord, error) {
	raw, err := s.client.Get(ctx, fmt.Sprintf(settledKeyFmt, matchID, wagerPubkey)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return SettlementRecord{}, fmt.Errorf("settlement %s/%s: %w", matchID, wagerPubkey, ErrSettlementNotFound)
		}
		return SettlementRecord{}, fmt.Errorf("get settlement: %w", err)
	}
	var rec SettlementRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return SettlementRecord{}, fmt.Errorf("decode settlement: %w", err)
	}
	return rec, nil
}

func pendingSettleMember(matchID, wagerPubkey string) string {
	return matchID + ":" + wagerPubkey
}

func parsePendingSettleMember(member string) (string, string, error) {
	parts := strings.SplitN(member, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid pending settlement member %q", member)
	}
	return parts[0], parts[1], nil
}

func (s *RedisStore) EnqueuePendingSettlement(ctx context.Context, item PendingSettlement) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal pending settlement: %w", err)
	}
	key := fmt.Sprintf(pendingSettleKeyFmt, item.MatchID, item.WagerPubkey)
	member := pendingSettleMember(item.MatchID, item.WagerPubkey)
	score := float64(item.NextRetryAt.UnixMilli())

	pipe := s.client.TxPipeline()
	pipe.Set(ctx, key, payload, pendingSettleTTL)
	pipe.ZAdd(ctx, pendingSettleIndexKey, redis.Z{Score: score, Member: member})
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("enqueue pending settlement: %w", err)
	}
	return nil
}

func (s *RedisStore) GetPendingSettlement(ctx context.Context, matchID, wagerPubkey string) (PendingSettlement, error) {
	raw, err := s.client.Get(ctx, fmt.Sprintf(pendingSettleKeyFmt, matchID, wagerPubkey)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return PendingSettlement{}, fmt.Errorf("pending settlement %s/%s: %w", matchID, wagerPubkey, ErrPendingSettlementNotFound)
		}
		return PendingSettlement{}, fmt.Errorf("get pending settlement: %w", err)
	}
	var item PendingSettlement
	if err := json.Unmarshal(raw, &item); err != nil {
		return PendingSettlement{}, fmt.Errorf("decode pending settlement: %w", err)
	}
	return item, nil
}

func (s *RedisStore) UpdatePendingSettlement(ctx context.Context, item PendingSettlement) error {
	return s.EnqueuePendingSettlement(ctx, item)
}

func (s *RedisStore) RemovePendingSettlement(ctx context.Context, matchID, wagerPubkey string) error {
	key := fmt.Sprintf(pendingSettleKeyFmt, matchID, wagerPubkey)
	member := pendingSettleMember(matchID, wagerPubkey)
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, key)
	pipe.ZRem(ctx, pendingSettleIndexKey, member)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("remove pending settlement: %w", err)
	}
	return nil
}

func (s *RedisStore) ListDuePendingSettlements(ctx context.Context, dueBefore time.Time, limit int) ([]PendingSettlement, error) {
	if limit <= 0 {
		limit = 50
	}
	members, err := s.client.ZRangeByScore(ctx, pendingSettleIndexKey, &redis.ZRangeBy{
		Min:   "0",
		Max:   fmt.Sprintf("%d", dueBefore.UnixMilli()),
		Count: int64(limit),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("list due pending settlements: %w", err)
	}

	out := make([]PendingSettlement, 0, len(members))
	for _, member := range members {
		matchID, wagerPubkey, err := parsePendingSettleMember(member)
		if err != nil {
			continue
		}
		item, err := s.GetPendingSettlement(ctx, matchID, wagerPubkey)
		if err != nil {
			if errors.Is(err, ErrPendingSettlementNotFound) {
				_, _ = s.client.ZRem(ctx, pendingSettleIndexKey, member).Result()
				continue
			}
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *RedisStore) CountPendingSettlements(ctx context.Context) (int64, error) {
	n, err := s.client.ZCard(ctx, pendingSettleIndexKey).Result()
	if err != nil {
		return 0, fmt.Errorf("count pending settlements: %w", err)
	}
	return n, nil
}

func (s *RedisStore) PublishMatchUpdate(ctx context.Context, match Match) error {
	payload, err := json.Marshal(match)
	if err != nil {
		return fmt.Errorf("marshal match update: %w", err)
	}
	return s.client.Publish(ctx, matchUpdateChannel, payload).Err()
}

// SubscribeMatchUpdates returns a channel that receives decoded match updates
// from Redis Pub/Sub. The channel is closed when ctx is cancelled.
func (s *RedisStore) SubscribeMatchUpdates(ctx context.Context) (<-chan Match, error) {
	sub := s.client.Subscribe(ctx, matchUpdateChannel)
	if _, err := sub.Receive(ctx); err != nil {
		return nil, fmt.Errorf("subscribe match updates: %w", err)
	}

	out := make(chan Match, 64)
	go func() {
		defer close(out)
		defer sub.Close()
		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var match Match
				if err := json.Unmarshal([]byte(msg.Payload), &match); err != nil {
					continue
				}
				select {
				case out <- match:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

// Client returns the underlying Redis client (used for SSE subscriptions in the API layer).
func (s *RedisStore) Client() *redis.Client {
	return s.client
}