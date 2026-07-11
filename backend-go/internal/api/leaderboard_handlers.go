package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/matchlock/backend-go/internal/auth"
	chainsol "github.com/matchlock/backend-go/internal/solana"
)

func (h *handler) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 && n <= 100 {
		limit = n
	}
	offset := 0
	if n, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && n >= 0 {
		offset = n
	}

	page, err := h.leaderboard.GetLeaderboard(r.Context(), offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LEADERBOARD_ERROR", "failed to load leaderboard")
		return
	}

	writeJSON(w, http.StatusOK, page)
}

func (h *handler) getMyLeaderboardRank(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}

	entry, err := h.leaderboard.GetRank(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LEADERBOARD_ERROR", "failed to load rank")
		return
	}

	if entry == nil {
		writeJSON(w, http.StatusOK, map[string]any{"rank": nil, "total_wagers": 0})
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

func (h *handler) getLeaderboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.leaderboard.GetStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LEADERBOARD_ERROR", "failed to load stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *handler) syncLeaderboardSettlement(w http.ResponseWriter, r *http.Request) {
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
			writeError(w, http.StatusNotFound, "WAGER_NOT_FOUND", "wager not found")
			return
		}
		writeError(w, http.StatusBadGateway, "RPC_ERROR", "failed to load wager")
		return
	}
	if wager.Status != chainsol.WagerStatusSettled {
		writeError(w, http.StatusConflict, "INVALID_STATUS", "wager is not settled yet")
		return
	}
	match, ok := h.matchForLeaderboardSync(r.Context(), wager)
	if !ok {
		writeError(w, http.StatusConflict, "MATCH_NOT_FINAL", "final score is not available yet")
		return
	}
	winningSide, ok := winningSideFromMatch(match)
	if !ok {
		writeError(w, http.StatusConflict, "MATCH_NOT_FINAL", "final score is not available yet")
		return
	}
	txSignature := strings.TrimSpace(r.URL.Query().Get("tx_signature"))
	if err := h.leaderboard.SyncSettledWager(r.Context(), wager, winningSide, txSignature); err != nil {
		writeError(w, http.StatusInternalServerError, "LEADERBOARD_ERROR", "failed to sync settlement")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"synced": true})
}
