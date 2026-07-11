package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func setTestAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://matchlock:matchlock@127.0.0.1:5432/matchlock?sslmode=disable")
	t.Setenv("JWT_ACCESS_SECRET", "test-jwt-access-secret-at-least-32-chars-long")
	t.Setenv("BREVO_API_KEY", "test-brevo-api-key")
}

func TestLoadDevnetDefaults(t *testing.T) {
	t.Setenv("SOLANA_CLUSTER", "devnet")
	t.Setenv("TXLINE_API_TOKEN", "test-token")
	t.Setenv("KEEPER_KEYPAIR_PATH", "/tmp/keeper.json")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	setTestAuthEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.TxlineAPIOrigin != DevnetTxlineOrigin {
		t.Fatalf("origin = %q, want %q", cfg.TxlineAPIOrigin, DevnetTxlineOrigin)
	}
	if cfg.StablecoinMint != DevnetStablecoinMint {
		t.Fatalf("mint = %q", cfg.StablecoinMint)
	}
	if cfg.ScoresStreamURL() != DevnetTxlineOrigin+"/api/scores/stream" {
		t.Fatalf("stream url = %q", cfg.ScoresStreamURL())
	}
}

func TestLoadRejectsDevnetMainnetOriginMismatch(t *testing.T) {
	t.Setenv("SOLANA_CLUSTER", "devnet")
	t.Setenv("TXLINE_API_ORIGIN", MainnetTxlineOrigin)
	t.Setenv("TXLINE_API_TOKEN", "test-token")
	t.Setenv("KEEPER_KEYPAIR_PATH", "/tmp/keeper.json")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	setTestAuthEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatal("expected network mismatch error")
	}
}

func TestLoadRejectsMissingAPIToken(t *testing.T) {
	t.Setenv("SOLANA_CLUSTER", "devnet")
	t.Setenv("TXLINE_API_TOKEN", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestLoadRejectsMissingKeeperKeypair(t *testing.T) {
	t.Setenv("SOLANA_CLUSTER", "devnet")
	t.Setenv("TXLINE_API_TOKEN", "test-token")
	t.Setenv("KEEPER_KEYPAIR_PATH", "")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	setTestAuthEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatal("expected missing keeper keypair error")
	}
}

func TestLoadMainnetDefaults(t *testing.T) {
	t.Setenv("SOLANA_CLUSTER", "mainnet-beta")
	t.Setenv("SOLANA_RPC_URL", "https://api.mainnet-beta.solana.com")
	t.Setenv("TXLINE_API_ORIGIN", MainnetTxlineOrigin)
	t.Setenv("TXLINE_API_TOKEN", "test-token")
	t.Setenv("KEEPER_KEYPAIR_PATH", "/tmp/keeper.json")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	setTestAuthEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TxlineAPIOrigin != MainnetTxlineOrigin {
		t.Fatalf("origin = %q", cfg.TxlineAPIOrigin)
	}
	if cfg.GuestAuthURL() != MainnetTxlineOrigin+"/auth/guest/start" {
		t.Fatalf("guest url = %q", cfg.GuestAuthURL())
	}
}

func TestLoadRejectsInvalidClusterAndBackoff(t *testing.T) {
	t.Setenv("SOLANA_CLUSTER", "localnet")
	t.Setenv("TXLINE_API_TOKEN", "test-token")
	t.Setenv("KEEPER_KEYPAIR_PATH", "/tmp/keeper.json")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	setTestAuthEnv(t)
	if _, err := Load(); err == nil {
		t.Fatal("expected unsupported cluster error")
	}

	t.Setenv("SOLANA_CLUSTER", "devnet")
	t.Setenv("SSE_INITIAL_BACKOFF", "0s")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid SSE backoff error")
	}
}

func TestLoadRejectsDevnetMainnetMint(t *testing.T) {
	t.Setenv("SOLANA_CLUSTER", "devnet")
	t.Setenv("STABLECOIN_MINT", MainnetStablecoinMint)
	t.Setenv("TXLINE_API_TOKEN", "test-token")
	t.Setenv("KEEPER_KEYPAIR_PATH", "/tmp/keeper.json")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	setTestAuthEnv(t)
	if _, err := Load(); err == nil {
		t.Fatal("expected mint mismatch error")
	}
}

func TestLoadRejectsMainnetDevnetOrigin(t *testing.T) {
	t.Setenv("SOLANA_CLUSTER", "mainnet-beta")
	t.Setenv("TXLINE_API_ORIGIN", DevnetTxlineOrigin)
	t.Setenv("TXLINE_API_TOKEN", "test-token")
	t.Setenv("KEEPER_KEYPAIR_PATH", "/tmp/keeper.json")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	setTestAuthEnv(t)
	if _, err := Load(); err == nil {
		t.Fatal("expected origin mismatch error")
	}
}

func TestEnvHelpers(t *testing.T) {
	t.Setenv("TEST_INT", "not-a-number")
	if intEnv("TEST_INT", 7) != 7 {
		t.Fatal("intEnv fallback failed")
	}
	t.Setenv("TEST_INT", "42")
	if intEnv("TEST_INT", 7) != 42 {
		t.Fatal("intEnv parse failed")
	}
	t.Setenv("TEST_DUR", "bad")
	if durationEnv("TEST_DUR", time.Second) != time.Second {
		t.Fatal("durationEnv fallback failed")
	}
	t.Setenv("TEST_DUR", "2s")
	if durationEnv("TEST_DUR", time.Second) != 2*time.Second {
		t.Fatal("durationEnv parse failed")
	}
	if len(parseCORSOrigins(" http://a.com , ,http://b.com ")) != 2 {
		t.Fatal("parseCORSOrigins failed")
	}
}

func TestLoadRejectsInvalidURLsAndZeroStatKey(t *testing.T) {
	t.Setenv("SOLANA_CLUSTER", "devnet")
	t.Setenv("SOLANA_RPC_URL", "not-a-url")
	t.Setenv("TXLINE_API_TOKEN", "test-token")
	t.Setenv("KEEPER_KEYPAIR_PATH", "/tmp/keeper.json")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	setTestAuthEnv(t)
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid rpc url error")
	}

	t.Setenv("SOLANA_RPC_URL", "")
	t.Setenv("TXLINE_API_ORIGIN", "not-a-url")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid txline origin error")
	}

	t.Setenv("TXLINE_API_ORIGIN", "")
	t.Setenv("TXLINE_STAT_KEY", "0")
	if _, err := Load(); err == nil {
		t.Fatal("expected zero stat key error")
	}
}

func TestBuildDatabaseURLFromDATABASE_URL(t *testing.T) {
	v := viper.New()
	v.Set("DATABASE_URL", "postgres://custom:pass@localhost:5555/mydb?sslmode=require")
	got := buildDatabaseURL(v)
	want := "postgres://custom:pass@localhost:5555/mydb?sslmode=require"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildDatabaseURLFromIndividualFields(t *testing.T) {
	v := viper.New()
	v.Set("DB_HOST", "db.example.com")
	v.Set("DB_PORT", 9999)
	v.Set("DB_NAME", "matchlock")
	v.Set("DB_USER", "appuser")
	v.Set("DB_PASS", "secret123")
	v.Set("DB_SSLMODE", "require")
	got := buildDatabaseURL(v)
	want := "postgres://appuser:secret123@db.example.com:9999/matchlock?sslmode=require"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildDatabaseURLWithSpecialCharsInPassword(t *testing.T) {
	v := viper.New()
	v.Set("DB_NAME", "matchlock")
	v.Set("DB_USER", "appuser")
	v.Set("DB_PASS", "p@ss:w rd!")
	got := buildDatabaseURL(v)
	want := "postgres://appuser:p%40ss%3Aw%20rd%21@127.0.0.1:5432/matchlock?sslmode=disable"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildDatabaseURLDefaultsHostPortSSLMode(t *testing.T) {
	v := viper.New()
	v.Set("DB_NAME", "matchlock")
	v.Set("DB_USER", "appuser")
	v.Set("DB_PASS", "pass")
	got := buildDatabaseURL(v)
	want := "postgres://appuser:pass@127.0.0.1:5432/matchlock?sslmode=disable"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildDatabaseURLMissingUserOrName(t *testing.T) {
	v := viper.New()
	v.Set("DB_NAME", "matchlock")
	if got := buildDatabaseURL(v); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	v.Set("DB_USER", "appuser")
	v.Set("DB_NAME", "")
	if got := buildDatabaseURL(v); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestLoadRejectsMainnetDevnetRPC(t *testing.T) {
	t.Setenv("SOLANA_CLUSTER", "mainnet-beta")
	t.Setenv("SOLANA_RPC_URL", "https://api.devnet.solana.com")
	t.Setenv("TXLINE_API_TOKEN", "test-token")
	t.Setenv("KEEPER_KEYPAIR_PATH", "/tmp/keeper.json")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	setTestAuthEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatal("expected RPC mismatch error")
	}
}
