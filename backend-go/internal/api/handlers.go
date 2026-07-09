package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/auth"
	"github.com/matchlock/backend-go/internal/cache"
	"github.com/matchlock/backend-go/internal/leaderboard"
	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

// MatchStore serves cached match data to the API.
type MatchStore interface {
	ListMatches(ctx context.Context) ([]cache.Match, error)
	GetMatch(ctx context.Context, matchID string) (cache.Match, error)
}

// WagerIndex reads wager accounts from chain.
type WagerIndex interface {
	ListWagers(ctx context.Context, filter chainsol.WagerFilter) ([]chainsol.Wager, error)
	GetWager(ctx context.Context, pubkey solana.PublicKey) (chainsol.Wager, error)
}

// ReadinessProbe checks external dependencies for /readyz.
type ReadinessProbe interface {
	Ping(ctx context.Context) error
}

type handler struct {
	cache       cache.Store
	wagers      WagerIndex
	redis       ReadinessProbe
	rpc         ReadinessProbe
	txline      ReadinessProbe
	postgres    ReadinessProbe
	txlineData  settlementProofTxline
	solana      settlementProofSolana
	auth        *auth.Service
	tokenCfg    auth.TokenConfig
	matchSub    MatchUpdateSubscriber
	leaderboard *leaderboard.Service
}

var _ = &leaderboard.Service{}

type settlementProofTxline interface {
	FetchStatValidation(ctx context.Context, fixtureID int64, seq int32, statKey uint32) (txline.StatValidation, error)
	FetchScoreSnapshot(ctx context.Context, fixtureID int64) ([]txline.ScoreSnapshotRow, error)
}

type settlementProofSolana interface {
	TxlineProgramID() solana.PublicKey
}

func (h *handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) readyz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	checks := map[string]string{
		"redis":  "ok",
		"rpc":    "ok",
		"txline": "ok",
	}
	if h.postgres != nil {
		checks["postgres"] = "ok"
	}
	var failed []string

	if err := h.redis.Ping(ctx); err != nil {
		checks["redis"] = err.Error()
		failed = append(failed, "redis")
	}
	if err := h.rpc.Ping(ctx); err != nil {
		checks["rpc"] = err.Error()
		failed = append(failed, "rpc")
	}
	if err := h.txline.Ping(ctx); err != nil {
		checks["txline"] = err.Error()
		failed = append(failed, "txline")
	}
	if h.postgres != nil {
		if err := h.postgres.Ping(ctx); err != nil {
			checks["postgres"] = err.Error()
			failed = append(failed, "postgres")
		}
	}

	if len(failed) > 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "not_ready",
			"checks": checks,
			"error":  "dependencies unavailable",
			"code":   "NOT_READY",
		})
		return
	}

	pendingSettlements := int64(0)
	if count, err := h.cache.CountPendingSettlements(ctx); err != nil {
		checks["settlement_queue"] = err.Error()
	} else {
		pendingSettlements = count
		checks["settlement_queue"] = "ok"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":              "ready",
		"checks":              checks,
		"pending_settlements": pendingSettlements,
	})
}

func (h *handler) listMatches(w http.ResponseWriter, r *http.Request) {
	matches, err := h.cache.ListMatches(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CACHE_ERROR", "failed to list matches")
		return
	}

	out := make([]MatchView, 0, len(matches))
	for _, match := range matches {
		out = append(out, matchViewFromCache(match))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) getMatch(w http.ResponseWriter, r *http.Request) {
	matchID := strings.TrimSpace(r.PathValue("id"))
	if matchID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_MATCH_ID", "match id is required")
		return
	}

	match, err := h.cache.GetMatch(r.Context(), matchID)
	if err != nil {
		if isRedisMiss(err) {
			writeError(w, http.StatusNotFound, "MATCH_NOT_FOUND", "match not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "CACHE_ERROR", "failed to load match")
		return
	}
	writeJSON(w, http.StatusOK, matchViewFromCache(match))
}

func (h *handler) listWagers(w http.ResponseWriter, r *http.Request) {
	filter, err := parseWagerFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	wagers, err := h.wagers.ListWagers(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusBadGateway, "RPC_ERROR", "failed to list wagers")
		return
	}

	out := make([]WagerView, 0, len(wagers))
	for _, wager := range wagers {
		// When no explicit status filter and no wallet filter, only list open/matched (backward compat for markets).
		// Wallet-filtered queries (history/profile) return all statuses.
		if filter.Status == nil && filter.Wallet == "" && !isListableWagerStatus(wager.Status) {
			continue
		}
		out = append(out, wagerViewFromChain(wager))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) getWager(w http.ResponseWriter, r *http.Request) {
	pubkeyRaw := strings.TrimSpace(r.PathValue("pubkey"))
	if pubkeyRaw == "" {
		writeError(w, http.StatusBadRequest, "INVALID_PUBKEY", "wager pubkey is required")
		return
	}
	pubkey, err := solana.PublicKeyFromBase58(pubkeyRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PUBKEY", "invalid wager pubkey")
		return
	}

	wager, err := h.wagers.GetWager(r.Context(), pubkey)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "WAGER_NOT_FOUND", "wager not found")
			return
		}
		writeError(w, http.StatusBadGateway, "RPC_ERROR", "failed to load wager")
		return
	}
	writeJSON(w, http.StatusOK, wagerViewFromChain(wager))
}

func (h *handler) getWagerSettlement(w http.ResponseWriter, r *http.Request) {
	pubkeyRaw := strings.TrimSpace(r.PathValue("pubkey"))
	if pubkeyRaw == "" {
		writeError(w, http.StatusBadRequest, "INVALID_PUBKEY", "wager pubkey is required")
		return
	}
	pubkey, err := parseWagerPubkey(pubkeyRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PUBKEY", "invalid wager pubkey")
		return
	}

	wager, err := h.wagers.GetWager(r.Context(), pubkey)
	if err != nil {
		if isWagerMissing(err) {
			rec := cache.WagerSettlementView{State: settlementStateSettled}
			writeJSON(w, http.StatusOK, settlementViewFromCache(rec))
			return
		}
		writeError(w, http.StatusBadGateway, "RPC_ERROR", "failed to load wager")
		return
	}

	view := resolveWagerSettlement(r.Context(), h.cache, wager)
	writeJSON(w, http.StatusOK, settlementViewFromCache(view))
}

func parseWagerFilter(r *http.Request) (chainsol.WagerFilter, error) {
	filter := chainsol.WagerFilter{
		MatchID: strings.TrimSpace(r.URL.Query().Get("match_id")),
		Wallet:  strings.TrimSpace(r.URL.Query().Get("wallet")),
	}
	statusRaw := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	switch statusRaw {
	case "":
		return filter, nil
	case "open":
		status := chainsol.WagerStatusOpen
		filter.Status = &status
	case "matched":
		status := chainsol.WagerStatusMatched
		filter.Status = &status
	case "settled":
		status := chainsol.WagerStatusSettled
		filter.Status = &status
	case "cancelled":
		status := chainsol.WagerStatusCancelled
		filter.Status = &status
	default:
		return chainsol.WagerFilter{}, errors.New("status must be open, matched, settled, or cancelled")
	}
	return filter, nil
}

func isListableWagerStatus(status uint8) bool {
	return status == chainsol.WagerStatusOpen || status == chainsol.WagerStatusMatched
}

func isRedisMiss(err error) bool {
	return errors.Is(err, cache.ErrMatchNotFound) || strings.Contains(err.Error(), "redis: nil")
}
