package api

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/matchlock/backend-go/internal/cache"
)

type fakeMatchSub struct {
	ch chan cache.Match
}

func newFakeMatchSub() *fakeMatchSub {
	return &fakeMatchSub{ch: make(chan cache.Match, 8)}
}

func (f *fakeMatchSub) SubscribeMatchUpdates(ctx context.Context) (<-chan cache.Match, error) {
	out := make(chan cache.Match, 8)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-f.ch:
				if !ok {
					return
				}
				select {
				case out <- m:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func TestMatchStreamSendsSSEEvent(t *testing.T) {
	sub := newFakeMatchSub()
	h := &handler{
		cache:    &fakeCache{matches: map[string]cache.Match{}},
		wagers:   &fakeWagers{},
		redis:    fakeProbe{},
		rpc:      fakeProbe{},
		txline:   fakeProbe{},
		matchSub: sub,
	}

	srv := httptest.NewServer(newMux(h))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/matches/stream", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q", ct)
	}

	// Push a match update.
	match := cache.Match{
		MatchID:   "17952170",
		FixtureID: 17952170,
		GameState: "HT",
		HomeTeam:  "USA",
		AwayTeam:  "Mexico",
	}
	sub.ch <- match

	// Read the SSE data line.
	scanner := bufio.NewScanner(resp.Body)
	var dataLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	if dataLine == "" {
		t.Fatal("no data line received")
	}

	var view MatchView
	if err := json.Unmarshal([]byte(dataLine), &view); err != nil {
		t.Fatalf("unmarshal: %v raw=%s", err, dataLine)
	}
	if view.MatchID != "17952170" {
		t.Fatalf("match_id = %q", view.MatchID)
	}
	if view.Status != "HT" {
		t.Fatalf("status = %q", view.Status)
	}
	if view.HomeTeam != "USA" || view.AwayTeam != "Mexico" {
		t.Fatalf("teams = %q vs %q", view.HomeTeam, view.AwayTeam)
	}
}

func TestMatchStreamNilSubscriber(t *testing.T) {
	h := &handler{
		cache:    &fakeCache{matches: map[string]cache.Match{}},
		wagers:   &fakeWagers{},
		redis:    fakeProbe{},
		rpc:      fakeProbe{},
		txline:   fakeProbe{},
		matchSub: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/matches/stream", nil)
	rec := httptest.NewRecorder()
	h.matchStream(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 for nil subscriber", rec.Code)
	}
}
