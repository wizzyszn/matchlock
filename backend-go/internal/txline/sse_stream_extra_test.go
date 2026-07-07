package txline

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func expectStreamOnceDone(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("streamOnce: %v", err)
	}
}

func TestStreamOnceContextCancelWhilePublishing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: score\ndata: {\"fixtureId\":1,\"gameState\":\"HT\",\"seq\":1}\n\n")
		<-r.Context().Done()
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	events := make(chan ScoreUpdate)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- streamOnce(ctx, client, srv.URL+"/api/scores/stream", events)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled && !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("streamOnce: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestStreamOnceDeliversAndSkipsMalformed(t *testing.T) {
	sseBody := "data: {\"gameState\":\"HT\"}\n\nevent: score\ndata: {\"fixtureId\":99,\"gameState\":\"HT\",\"seq\":1}\n\n"
	srv := newSSEServer(t, sseBody)
	defer srv.Close()
	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	events := make(chan ScoreUpdate, 2)
	expectStreamOnceDone(t, streamOnce(context.Background(), client, srv.URL+"/api/scores/stream", events))
	update := <-events
	if update.MatchID() != "99" {
		t.Fatalf("match_id = %q", update.MatchID())
	}
}

func TestStreamOnceDeliversSingleEvent(t *testing.T) {
	srv := newSSEServer(t, "event: score\ndata: {\"fixtureId\":42,\"gameState\":\"HT\",\"seq\":1}\n\n")
	defer srv.Close()
	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	events := make(chan ScoreUpdate, 1)
	expectStreamOnceDone(t, streamOnce(context.Background(), client, srv.URL+"/api/scores/stream", events))
	if (<-events).MatchID() != "42" {
		t.Fatal("unexpected match")
	}
}

func newSSEServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, body)
	}))
}

func TestStreamOnceReadSSEError(t *testing.T) {
	client := NewClient("http://example.com", "http://example.com/auth/guest/start", "api-token", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodPost {
				return jsonResponse(`{"token":"jwt-1"}`), nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       &failReader{},
				Header:     http.Header{"Content-Type": {"text/event-stream"}},
			}, nil
		}),
	})
	err := streamOnce(context.Background(), client, "http://example.com/api/scores/stream", make(chan ScoreUpdate, 1))
	if err == nil || !strings.Contains(err.Error(), "read sse") {
		t.Fatalf("err = %v", err)
	}
}

func TestStreamScoresReconnectsAfterEOF(t *testing.T) {
	var streamCalls int
	client := NewClient("http://test", "http://test/auth/guest/start", "api-token", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/auth/guest/start") {
				return jsonResponse(`{"token":"jwt-1"}`), nil
			}
			streamCalls++
			if streamCalls == 1 {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("event: score\ndata: {\"fixtureId\":1,\"gameState\":\"HT\",\"seq\":1}\n\n")),
					Header:     http.Header{"Content-Type": {"text/event-stream"}},
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusGone,
				Body:       io.NopCloser(strings.NewReader("gone")),
			}, nil
		}),
	})

	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan ScoreUpdate, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- StreamScores(ctx, client, StreamConfig{
			StreamURL:   "http://test/api/scores/stream",
			InitialWait: time.Millisecond,
			MaxWait:     2 * time.Millisecond,
		}, events)
	}()

	select {
	case update := <-events:
		if update.MatchID() != "1" {
			t.Fatalf("match_id = %q", update.MatchID())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
	cancel()
	<-errCh
}

func TestStreamOnceEnsureJWTFailure(t *testing.T) {
	client := NewClient("http://test", "http://test/auth/guest/start", "token", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader("no"))}, nil
		}),
	})
	err := streamOnce(context.Background(), client, "http://test/stream", make(chan ScoreUpdate, 1))
	if err == nil || !strings.Contains(err.Error(), "ensure guest jwt") {
		t.Fatalf("err = %v", err)
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": {"application/json"}},
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}