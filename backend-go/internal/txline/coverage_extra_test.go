package txline

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRefreshGuestJWTRequestError(t *testing.T) {
	client := NewClient("http://example.com", "http://example.com/auth", "token", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		}),
	})
	if err := client.EnsureGuestJWT(context.Background(), true); err == nil {
		t.Fatal("expected request error")
	}
}

func TestDoAuthenticatedTransportError(t *testing.T) {
	client := NewClient("http://example.com", "http://example.com/auth", "token", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/auth/guest/start") {
				return jsonResponse(`{"token":"jwt-1"}`), nil
			}
			return nil, errors.New("stream unavailable")
		}),
	})
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/data", nil)
	if _, err := client.DoAuthenticated(context.Background(), req); err == nil {
		t.Fatal("expected transport error")
	}
}

func TestStreamOnceContextCanceledDuringRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flush", http.StatusInternalServerError)
			return
		}
		for i := 0; i < 5; i++ {
			_, _ = io.WriteString(w, "event: score\ndata: {\"fixtureId\":1,\"gameState\":\"HT\",\"seq\":1}\n\n")
			flusher.Flush()
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan ScoreUpdate, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- streamOnce(ctx, client, srv.URL+"/api/scores/stream", events)
	}()

	time.Sleep(15 * time.Millisecond)
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

func TestStreamScoresBackoffTimerCancel(t *testing.T) {
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
			InitialWait: 500 * time.Millisecond,
			MaxWait:     time.Second,
		}, make(chan ScoreUpdate, 1))
	}()

	time.Sleep(30 * time.Millisecond)
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

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errReadCloser) Close() error             { return nil }

func TestStreamScoresMaxWaitDefaults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := NewClient("http://example.com", "http://example.com/auth", "token", nil)
	err := StreamScores(ctx, client, StreamConfig{
		InitialWait: 10 * time.Millisecond,
		MaxWait:     time.Millisecond,
	}, make(chan ScoreUpdate, 1))
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestStreamOnceBlockedPublishCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: score\ndata: {\"fixtureId\":7,\"gameState\":\"HT\",\"seq\":1}\n\n")
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan ScoreUpdate)
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

func TestRefreshGuestJWTReadError(t *testing.T) {
	client := NewClient("http://example.com", "http://example.com/auth", "token", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       errReadCloser{},
			}, nil
		}),
	})
	if err := client.EnsureGuestJWT(context.Background(), true); err == nil {
		t.Fatal("expected read error")
	}
}

func TestDoAuthenticated401RefreshAuthFailure(t *testing.T) {
	var guestCalls int
	client := NewClient("http://example.com", "http://example.com/auth", "token", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/auth/guest/start") {
				guestCalls++
				if guestCalls == 1 {
					return jsonResponse(`{"token":"jwt-1"}`), nil
				}
				return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader("down"))}, nil
			}
			return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader("expired"))}, nil
		}),
	})
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/probe", nil)
	if _, err := client.DoAuthenticated(context.Background(), req); err == nil {
		t.Fatal("expected refresh auth failure")
	}
}

func TestDoAuthenticatedRetryTransportError(t *testing.T) {
	var probeCalls int
	client := NewClient("http://example.com", "http://example.com/auth", "token", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/auth/guest/start") {
				return jsonResponse(`{"token":"jwt-1"}`), nil
			}
			probeCalls++
			if probeCalls == 1 {
				return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader("expired"))}, nil
			}
			return nil, errors.New("retry failed")
		}),
	})
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/probe", nil)
	if _, err := client.DoAuthenticated(context.Background(), req); err == nil {
		t.Fatal("expected retry transport error")
	}
}

func TestFetchStatValidationReadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/auth/guest/start") {
				return jsonResponse(`{"token":"jwt-1"}`), nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       errReadCloser{},
				Header:     http.Header{"Content-Type": {"application/json"}},
			}, nil
		}),
	})
	if _, err := client.FetchStatValidation(context.Background(), 1, 1, 1002); err == nil {
		t.Fatal("expected read error")
	}
}