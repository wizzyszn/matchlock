package txline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchStatValidation(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "txline_proof_response.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/guest/start":
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
		case "/api/scores/stat-validation":
			if r.Header.Get("Authorization") == "" || r.Header.Get("X-Api-Token") == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write(raw)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	out, err := client.FetchStatValidation(context.Background(), 17952170, 10, 1002)
	if err != nil {
		t.Fatalf("FetchStatValidation: %v", err)
	}
	if out.Summary.FixtureID != 17952170 {
		t.Fatalf("fixture = %d", out.Summary.FixtureID)
	}
}

func TestFetchStatValidationErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
			return
		}
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	if _, err := client.FetchStatValidation(context.Background(), 1, 1, 1002); err == nil {
		t.Fatal("expected status error")
	}

	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"summary":{}}`))
	}))
	defer badJSON.Close()
	client2 := NewClient(badJSON.URL, badJSON.URL+"/auth/guest/start", "api-token", badJSON.Client())
	if _, err := client2.FetchStatValidation(context.Background(), 1, 1, 1002); err == nil {
		t.Fatal("expected missing fixture error")
	}

	invalid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
			return
		}
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer invalid.Close()
	client3 := NewClient("::://bad", invalid.URL+"/auth/guest/start", "api-token", invalid.Client())
	if _, err := client3.FetchStatValidation(context.Background(), 1, 1, 1002); err == nil {
		t.Fatal("expected url parse error")
	}
}

func TestFetchStatValidationDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/guest/start" {
			_, _ = w.Write([]byte(`{"token":"jwt-1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"summary":{"fixtureId":1},"not-valid-json`))
	}))
	defer srv.Close()
	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	if _, err := client.FetchStatValidation(context.Background(), 1, 1, 1002); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestTruncate(t *testing.T) {
	if truncate([]byte("short"), 10) != "short" {
		t.Fatal("short truncate changed value")
	}
	got := truncate([]byte(strings.Repeat("a", 20)), 5)
	if got != "aaaaa..." {
		t.Fatalf("truncate = %q", got)
	}
}
