package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matchlock/backend-go/internal/cache"
)

type pingFailCache struct {
	fakeCache
}

func (p *pingFailCache) Ping(ctx context.Context) error {
	return errors.New("redis down")
}

func TestNewServerIntegration(t *testing.T) {
	h := &handler{
		cache:  &fakeCache{matches: map[string]cache.Match{}},
		wagers: &fakeWagers{},
		redis:  fakeProbe{},
		rpc:    fakeProbe{},
		txline: fakeProbe{},
	}
	handler := corsMiddleware([]string{"http://localhost:5173"})(newMux(h))
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetMatchEmptyID(t *testing.T) {
	h := &handler{cache: &fakeCache{matches: map[string]cache.Match{}}}
	req := httptest.NewRequest(http.MethodGet, "/matches/", nil)
	req.SetPathValue("id", "")
	rec := httptest.NewRecorder()
	h.getMatch(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}