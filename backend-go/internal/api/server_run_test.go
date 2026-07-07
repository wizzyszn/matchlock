package api

import (
	"context"
	"testing"
	"time"

	"github.com/matchlock/backend-go/internal/cache"
)

func TestServerRunGracefulShutdown(t *testing.T) {
	srv := NewServer(ServerConfig{
		Addr:         "127.0.0.1:0",
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		CORSOrigins:  []string{"http://localhost:5173"},
	}, Dependencies{
		Cache: &fakeCache{matches: map[string]cache.Match{}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for shutdown")
	}
}

func TestServerRunListenError(t *testing.T) {
	srv := NewServer(ServerConfig{Addr: "\x00"}, Dependencies{
		Cache: &fakeCache{matches: map[string]cache.Match{}},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Run(ctx); err == nil {
		t.Fatal("expected listen error")
	}
}

func TestNewHandlerWiring(t *testing.T) {
	h := newHandler(Dependencies{Cache: &fakeCache{matches: map[string]cache.Match{}}})
	if h == nil || newMux(h) == nil {
		t.Fatal("expected handler and mux")
	}
}