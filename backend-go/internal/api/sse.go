package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/matchlock/backend-go/internal/cache"
)

// MatchUpdateSubscriber provides a stream of match updates for SSE fan-out.
type MatchUpdateSubscriber interface {
	SubscribeMatchUpdates(ctx context.Context) (<-chan cache.Match, error)
}

func (h *handler) matchStream(w http.ResponseWriter, r *http.Request) {
	if h.matchSub == nil {
		writeError(w, http.StatusInternalServerError, "SSE_UNAVAILABLE", "match streaming not configured")
		return
	}

	ctx := r.Context()
	updates, err := h.matchSub.SubscribeMatchUpdates(ctx)
	if err != nil {
		slog.Error("sse subscribe failed", "err", err)
		writeError(w, http.StatusInternalServerError, "SSE_SUBSCRIBE_ERROR", "failed to subscribe")
		return
	}

	rc := http.NewResponseController(w)

	// Set headers before flushing
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	
	if err := rc.Flush(); err != nil {
		slog.Warn("sse stream missing flusher support", "err", err)
		return
	}

	// Disable the server-wide WriteTimeout for this specific SSE connection
	// so the stream isn't forcefully terminated with ERR_INCOMPLETE_CHUNKED_ENCODING.
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		slog.Warn("sse stream failed to clear write deadline", "err", err)
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case match, ok := <-updates:
			if !ok {
				return
			}
			data, err := json.Marshal(matchViewFromCache(match))
			if err != nil {
				slog.Warn("sse marshal failed", "match_id", match.MatchID, "err", err)
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			rc.Flush()
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ":\n\n"); err != nil {
				return
			}
			rc.Flush()
		}
	}
}
