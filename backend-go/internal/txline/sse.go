package txline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// StreamConfig controls SSE reconnect behavior.
type StreamConfig struct {
	StreamURL   string
	InitialWait time.Duration
	MaxWait     time.Duration
}

// StreamScores connects to the TxLINE scores SSE feed and publishes parsed updates.
// It reconnects with exponential backoff until ctx is cancelled.
func StreamScores(ctx context.Context, client *Client, cfg StreamConfig, events chan<- ScoreUpdate) error {
	if cfg.InitialWait <= 0 {
		cfg.InitialWait = time.Second
	}
	if cfg.MaxWait < cfg.InitialWait {
		cfg.MaxWait = 30 * time.Second
	}

	backoff := cfg.InitialWait
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := streamOnce(ctx, client, cfg.StreamURL, events)
		if err == nil {
			backoff = cfg.InitialWait
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		slog.Warn("txline sse disconnected", "err", err, "retry_in", backoff)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		backoff = nextBackoff(backoff, cfg.MaxWait)
	}
}

func streamOnce(ctx context.Context, client *Client, streamURL string, events chan<- ScoreUpdate) error {
	if err := client.EnsureGuestJWT(ctx, false); err != nil {
		return fmt.Errorf("ensure guest jwt: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return fmt.Errorf("build sse request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.DoAuthenticated(ctx, req)
	if err != nil {
		return fmt.Errorf("open sse stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("sse stream status=%d body=%s", resp.StatusCode, truncate(body, 256))
	}

	raw := make(chan SSEMessage, 64)
	readDone := make(chan error, 1)
	go func() {
		readDone <- ReadSSE(resp.Body, raw)
		close(raw)
	}()

	for msg := range raw {
		if err := ctx.Err(); err != nil {
			return err
		}
		if strings.EqualFold(msg.Event, "heartbeat") {
			continue
		}
		update, err := parseScoreUpdate(msg)
		if err != nil {
			slog.Warn("skip malformed sse payload", "err", err, "event", msg.Event)
			continue
		}
		select {
		case events <- update:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := <-readDone; err != nil {
		return fmt.Errorf("read sse: %w", err)
	}
	return nil
}

func parseScoreUpdate(msg SSEMessage) (ScoreUpdate, error) {
	if msg.Data == "" {
		return ScoreUpdate{}, fmt.Errorf("empty data field")
	}
	var update ScoreUpdate
	if err := json.Unmarshal([]byte(msg.Data), &update); err != nil {
		return ScoreUpdate{}, fmt.Errorf("decode score update: %w", err)
	}
	if update.FixtureID == 0 {
		return ScoreUpdate{}, fmt.Errorf("missing fixtureId")
	}
	update.RawEvent = msg.Event
	update.ReceivedAt = time.Now().UTC()
	return update, nil
}

func nextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		next = max
	}
	jitter := time.Duration(rand.Int63n(int64(next / 5)))
	return next + jitter
}