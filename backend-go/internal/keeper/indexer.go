package keeper

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/matchlock/backend-go/internal/cache"
	chainsol "github.com/matchlock/backend-go/internal/solana"
)

type WagerIndexer struct {
	rpcClient    *rpc.Client
	wsURL        string
	programID    solana.PublicKey
	cache        *cache.RedisStore
	pollInterval time.Duration
}

func NewWagerIndexer(rpcURL, wsURL string, programID solana.PublicKey, redisCache *cache.RedisStore) *WagerIndexer {
	return &WagerIndexer{
		rpcClient:    rpc.New(rpcURL),
		wsURL:        wsURL,
		programID:    programID,
		cache:        redisCache,
		pollInterval: 30 * time.Second,
	}
}

func WSURLFromRPC(rpcURL string) string {
	s := strings.Replace(rpcURL, "https://", "wss://", 1)
	s = strings.Replace(s, "http://", "ws://", 1)
	return s
}

func (idx *WagerIndexer) Run(ctx context.Context) error {
	if err := idx.backfill(ctx); err != nil {
		slog.Error("wager indexer backfill failed", "err", err)
	}

	var wg sync.WaitGroup
	wsCtx, wsCancel := context.WithCancel(ctx)
	defer wsCancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := idx.runWSSubscription(wsCtx); err != nil {
			slog.Warn("wager indexer ws subscription exited", "err", err)
		}
	}()

	ticker := time.NewTicker(idx.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := idx.backfill(ctx); err != nil {
				slog.Error("wager indexer poll failed", "err", err)
			}
		}
	}
}

func (idx *WagerIndexer) backfill(ctx context.Context) error {
	disc := [8]byte{3, 110, 53, 190, 113, 31, 230, 40}
	v1Filters := []rpc.RPCFilter{
		{DataSize: 118},
		{Memcmp: &rpc.RPCFilterMemcmp{Offset: 0, Bytes: disc[:]}},
	}
	v2Filters := []rpc.RPCFilter{
		{DataSize: 150},
		{Memcmp: &rpc.RPCFilterMemcmp{Offset: 0, Bytes: disc[:]}},
	}
	v3Filters := []rpc.RPCFilter{
		{DataSize: 151},
		{Memcmp: &rpc.RPCFilterMemcmp{Offset: 0, Bytes: disc[:]}},
	}
	v4Filters := []rpc.RPCFilter{
		{DataSize: 159},
		{Memcmp: &rpc.RPCFilterMemcmp{Offset: 0, Bytes: disc[:]}},
	}

	var accounts []*rpc.KeyedAccount
	var mu sync.Mutex
	var wg sync.WaitGroup
	var err1, err2, err3, err4 error

	wg.Add(4)
	go func() {
		defer wg.Done()
		v1, err := idx.rpcClient.GetProgramAccountsWithOpts(ctx, idx.programID, &rpc.GetProgramAccountsOpts{Filters: v1Filters})
		if err != nil {
			err1 = err
			return
		}
		mu.Lock()
		accounts = append(accounts, v1...)
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		v2, err := idx.rpcClient.GetProgramAccountsWithOpts(ctx, idx.programID, &rpc.GetProgramAccountsOpts{Filters: v2Filters})
		if err != nil {
			err2 = err
			return
		}
		mu.Lock()
		accounts = append(accounts, v2...)
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		v3, err := idx.rpcClient.GetProgramAccountsWithOpts(ctx, idx.programID, &rpc.GetProgramAccountsOpts{Filters: v3Filters})
		if err != nil {
			err3 = err
			return
		}
		mu.Lock()
		accounts = append(accounts, v3...)
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		v4, err := idx.rpcClient.GetProgramAccountsWithOpts(ctx, idx.programID, &rpc.GetProgramAccountsOpts{Filters: v4Filters})
		if err != nil {
			err4 = err
			return
		}
		mu.Lock()
		accounts = append(accounts, v4...)
		mu.Unlock()
	}()
	wg.Wait()

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	if err3 != nil {
		return err3
	}
	if err4 != nil {
		return err4
	}

	currentWagers, err := idx.cache.ListWagers(ctx)
	if err != nil {
		slog.Error("wager indexer backfill active wagers fetch failed", "err", err)
	}
	activeMap := make(map[string]bool)

	indexed := 0
	for _, acct := range accounts {
		pubkeyStr := acct.Pubkey.String()
		activeMap[pubkeyStr] = true

		w, err := chainsol.DecodeWager(acct.Pubkey, acct.Account.Data.GetBinary())
		if err != nil {
			continue
		}
		item := wagerToCacheItem(w)
		if err := idx.cache.SetWager(ctx, item); err != nil {
			slog.Error("wager indexer cache set failed", "pubkey", item.Pubkey, "err", err)
			continue
		}
		indexed++
	}

	deleted := 0
	if currentWagers != nil {
		for _, cw := range currentWagers {
			if !activeMap[cw.Pubkey] {
				if err := idx.cache.DeleteWager(ctx, cw.Pubkey); err != nil {
					slog.Error("wager indexer backfill delete failed", "pubkey", cw.Pubkey, "err", err)
				} else {
					deleted++
				}
			}
		}
	}

	slog.Info("wager indexer backfill complete", "indexed", indexed, "deleted", deleted, "total_accounts", len(accounts))
	return nil
}

func (idx *WagerIndexer) runWSSubscription(ctx context.Context) error {
	wsClient, err := ws.Connect(ctx, idx.wsURL)
	if err != nil {
		return err
	}
	defer wsClient.Close()

	sub, err := wsClient.ProgramSubscribeWithOpts(
		idx.programID,
		rpc.CommitmentConfirmed,
		solana.EncodingBase64,
		nil,
	)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	slog.Info("wager indexer ws subscription established")
	for {
		got, err := sub.Recv(ctx)
		if err != nil {
			return err
		}

		if got.Value.Account.Lamports == 0 || len(got.Value.Account.Data.GetBinary()) == 0 {
			if err := idx.cache.DeleteWager(ctx, got.Value.Pubkey.String()); err != nil {
				slog.Error("wager indexer ws delete failed", "pubkey", got.Value.Pubkey.String(), "err", err)
			} else {
				slog.Debug("wager indexer ws deleted", "pubkey", got.Value.Pubkey.String())
			}
			continue
		}

		w, err := chainsol.DecodeWager(got.Value.Pubkey, got.Value.Account.Data.GetBinary())
		if err != nil {
			continue
		}
		item := wagerToCacheItem(w)
		if err := idx.cache.SetWager(ctx, item); err != nil {
			slog.Error("wager indexer ws upsert failed", "pubkey", item.Pubkey, "err", err)
			continue
		}
		slog.Debug("wager indexer ws updated", "pubkey", item.Pubkey, "status", item.Status)
	}
}

func wagerToCacheItem(w chainsol.Wager) cache.WagerCacheItem {
	item := cache.WagerCacheItem{
		Pubkey:    w.Pubkey.String(),
		Maker:     w.Maker.String(),
		Taker:     w.Taker.String(),
		MatchID:   w.MatchIDString(),
		MakerSide: w.MakerSide,
		TakerSide: w.TakerSide,
		Stake:     w.Stake,
		Status:    w.Status,
		Bump:      w.Bump,
		VaultBump: w.VaultBump,
	}
	if !w.InvitedTaker.IsZero() && !w.InvitedTaker.Equals(chainsol.SystemProgramID) {
		item.InvitedTaker = w.InvitedTaker.String()
	}
	return item
}
