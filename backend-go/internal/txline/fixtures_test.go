package txline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchFixturesSnapshot(t *testing.T) {
	body := loadFixture(t, "fixtures_snapshot.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/fixtures/snapshot" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL+"/auth/guest/start", "api-token", srv.Client())
	client.jwt = "jwt"

	fixtures, err := client.FetchFixturesSnapshot(context.Background(), nil)
	if err != nil {
		t.Fatalf("FetchFixturesSnapshot: %v", err)
	}
	if len(fixtures) != 1 {
		t.Fatalf("fixtures = %#v", fixtures)
	}
	f := fixtures[0]
	if f.FixtureID != 18172379 {
		t.Fatalf("fixture_id = %d", f.FixtureID)
	}
	if f.HomeTeam() != "USA" || f.AwayTeam() != "Bosnia & Herzegovina" {
		t.Fatalf("teams = %q vs %q", f.HomeTeam(), f.AwayTeam())
	}
}

func TestFixtureMatchID(t *testing.T) {
	f := Fixture{FixtureID: 18172379}
	if f.MatchID() != "18172379" {
		t.Fatalf("match_id = %q", f.MatchID())
	}
}