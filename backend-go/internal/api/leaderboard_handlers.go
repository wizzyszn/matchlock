package api

import (
	"net/http"
	"strconv"

	"github.com/matchlock/backend-go/internal/auth"
)

func (h *handler) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 100 {
		limit = n
	}

	entries, err := h.leaderboard.GetLeaderboard(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LEADERBOARD_ERROR", "failed to load leaderboard")
		return
	}

	writeJSON(w, http.StatusOK, entries)
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
	entries, err := h.leaderboard.GetLeaderboard(r.Context(), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LEADERBOARD_ERROR", "failed to load stats")
		return
	}

	var totalWagers, totalVolume int64
	for _, e := range entries {
		totalWagers += e.TotalWagers
		totalVolume += int64(e.TotalVolume)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_wagers": totalWagers,
		"total_volume": totalVolume,
		"total_users":  len(entries),
	})
}
