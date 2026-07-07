package txline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNextBackoff(t *testing.T) {
	got := nextBackoff(time.Second, 10*time.Second)
	if got < time.Second || got > 12*time.Second {
		t.Fatalf("backoff = %s", got)
	}
	if nextBackoff(8*time.Second, 10*time.Second) > 12*time.Second {
		t.Fatal("expected capped backoff")
	}
}

func TestStreamScoresHonorsCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
			return
		}
		http.Error(w, "gone", http.StatusGone)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- StreamScores(ctx, client, StreamConfig{
			StreamURL:   srv.URL + "/api/scores/stream",
			InitialWait: time.Millisecond,
			MaxWait:     2 * time.Millisecond,
		}, make(chan ScoreUpdate, 1))
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Fatalf("StreamScores: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cancel")
	}
}

func TestStreamScoresDefaultsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
		}
	}))
	defer srv.Close()
	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "token", srv.Client())
	err := StreamScores(ctx, client, StreamConfig{}, make(chan ScoreUpdate, 1))
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestStreamOnceRejectsNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	events := make(chan ScoreUpdate, 1)
	err := streamOnce(context.Background(), client, srv.URL+"/api/scores/stream", events)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("err = %v", err)
	}
}

