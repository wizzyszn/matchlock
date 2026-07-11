package email

import "testing"

func TestParseFromWithDisplayName(t *testing.T) {
	name, email := parseFrom("Matchlock <noreply@matchlock.dev>")
	if name != "Matchlock" {
		t.Fatalf("name = %q", name)
	}
	if email != "noreply@matchlock.dev" {
		t.Fatalf("email = %q", email)
	}
}

func TestParseFromPlainEmail(t *testing.T) {
	name, email := parseFrom("noreply@matchlock.dev")
	if name != "" {
		t.Fatalf("name = %q", name)
	}
	if email != "noreply@matchlock.dev" {
		t.Fatalf("email = %q", email)
	}
}
