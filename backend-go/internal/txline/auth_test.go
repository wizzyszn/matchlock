package txline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnsureGuestJWT(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/auth/guest/start" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(guestStartResponse{Token: "jwt-123"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	if err := client.EnsureGuestJWT(context.Background(), false); err != nil {
		t.Fatalf("EnsureGuestJWT: %v", err)
	}
	if client.JWT() != "jwt-123" {
		t.Fatalf("jwt = %q", client.JWT())
	}
	if err := client.EnsureGuestJWT(context.Background(), false); err != nil {
		t.Fatalf("second EnsureGuestJWT: %v", err)
	}
	if calls != 1 {
		t.Fatalf("guest start calls = %d, want 1", calls)
	}

	headers, err := client.AuthHeaders()
	if err != nil {
		t.Fatalf("AuthHeaders: %v", err)
	}
	if headers.Get("Authorization") != "Bearer jwt-123" {
		t.Fatalf("authorization = %q", headers.Get("Authorization"))
	}
	if headers.Get("X-Api-Token") != "api-token" {
		t.Fatalf("api token header = %q", headers.Get("X-Api-Token"))
	}
}

func TestDoAuthenticatedRefreshesOn401(t *testing.T) {
	var guestCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/guest/start":
			guestCalls++
			token := "jwt-1"
			if guestCalls > 1 {
				token = "jwt-2"
			}
			_ = json.NewEncoder(w).Encode(guestStartResponse{Token: token})
		case "/probe":
			if r.Header.Get("Authorization") == "Bearer jwt-1" {
				http.Error(w, "expired", http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/probe", nil)
	resp, err := client.DoAuthenticated(context.Background(), req)
	if err != nil {
		t.Fatalf("DoAuthenticated: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if guestCalls != 2 {
		t.Fatalf("guest calls = %d, want 2", guestCalls)
	}
}

func TestNewClientDefaultsHTTPClient(t *testing.T) {
	client := NewClient("http://example.com", "http://example.com/auth", "token", nil)
	if client.httpClient == nil {
		t.Fatal("expected default http client")
	}
	if client.APIToken() != "token" {
		t.Fatalf("token = %q", client.APIToken())
	}
}

func TestAuthHeadersRequiresJWT(t *testing.T) {
	client := NewClient("http://example.com", "http://example.com/auth", "token", nil)
	if _, err := client.AuthHeaders(); err == nil {
		t.Fatal("expected missing jwt error")
	}
}

func TestRefreshGuestJWTErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadRequest)
	}))
	defer srv.Close()
	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "token", srv.Client())
	if err := client.EnsureGuestJWT(context.Background(), true); err == nil {
		t.Fatal("expected auth failure")
	}

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"token":""}`))
	}))
	defer empty.Close()
	client2 := NewClient(empty.URL, empty.URL+"/auth/guest/start", "token", empty.Client())
	if err := client2.EnsureGuestJWT(context.Background(), true); err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestRefreshGuestJWTDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not-json`))
	}))
	defer srv.Close()
	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "token", srv.Client())
	if err := client.EnsureGuestJWT(context.Background(), true); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestDoAuthenticatedRefreshFailureOn401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/guest/start":
			http.Error(w, "auth down", http.StatusUnauthorized)
		case "/probe":
			http.Error(w, "expired", http.StatusUnauthorized)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/probe", nil)
	if _, err := client.DoAuthenticated(context.Background(), req); err == nil {
		t.Fatal("expected refresh failure")
	}
}

func TestPing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"token":"jwt"}`))
	}))
	defer srv.Close()
	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "token", srv.Client())
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}