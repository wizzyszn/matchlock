package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	ClusterDevnet        = "devnet"
	ClusterMainnet       = "mainnet-beta"
	DevnetTxlineOrigin   = "https://txline-dev.txodds.com"
	MainnetTxlineOrigin  = "https://txline.txodds.com"
	DevnetStablecoinMint = "ELWTKspHKCnCfCiCiqYw1EDH77k8VCP74dK9qytG2Ujh"
	DevnetTxlineProgram  = "6pW64gN1s2uqjHkn1unFeEjAwJkPGHoppGvS715wyP2J"
	DefaultProgramID     = "6a1hkAgtuewKaB6B4vt1bMymcFVtK85mGVbVBJkURaZ8"
)

// Config holds validated runtime settings for the keeper service.
type Config struct {
	Cluster                 string
	SolanaRPCURL            string
	TxlineAPIOrigin         string
	TxlineAPIToken          string
	MatchlockProgram        string
	StablecoinMint          string
	TxlineProgram           string
	HTTPAddr                string
	HTTPReadTimeout         time.Duration
	HTTPWriteTimeout        time.Duration
	SSEInitialDelay         time.Duration
	SSEMaxDelay             time.Duration
	ScheduleRefreshInterval time.Duration
	OddsRefreshInterval     time.Duration
	ReconcileInterval       time.Duration
	SettlementRetryBase     time.Duration
	MaxSettlementAttempts   int
	RedisURL                string
	KeeperKeypairPath       string
	KeeperAutoSettle        bool
	TxlineStatKey           uint32
	CORSAllowedOrigins      []string

	DatabaseURL     string
	JWTAccessSecret string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	MagicLinkTTL    time.Duration
	CookieSecure    bool
	CookieDomain    string
	FrontendURL     string
	SMTPHost        string
	SMTPPort        int
	SMTPUser        string
	SMTPPass        string
	SMTPFrom        string
	LogLevel        string
	LogEncoding     string
}

// Load reads configuration with Viper and validates network consistency.
func Load() (Config, error) {
	v := newViper()
	if err := readConfig(v); err != nil {
		return Config{}, err
	}
	return loadFromViper(v)
}

func loadFromViper(v *viper.Viper) (Config, error) {
	cluster := strings.TrimSpace(v.GetString("SOLANA_CLUSTER"))
	rpcURL := strings.TrimSpace(v.GetString("SOLANA_RPC_URL"))
	if rpcURL == "" {
		rpcURL = defaultRPCURL(cluster)
	}

	txlineOrigin := strings.TrimRight(strings.TrimSpace(v.GetString("TXLINE_API_ORIGIN")), "/")
	if txlineOrigin == "" {
		txlineOrigin = defaultTxlineOrigin(cluster)
	}

	cfg := Config{
		Cluster:                 cluster,
		SolanaRPCURL:            rpcURL,
		TxlineAPIOrigin:         txlineOrigin,
		TxlineAPIToken:          strings.TrimSpace(v.GetString("TXLINE_API_TOKEN")),
		MatchlockProgram:        stringValue(v, "MATCHLOCK_PROGRAM_ID", DefaultProgramID),
		StablecoinMint:          stringValue(v, "STABLECOIN_MINT", DevnetStablecoinMint),
		TxlineProgram:           stringValue(v, "TXLINE_PROGRAM_ID", DevnetTxlineProgram),
		HTTPAddr:                stringValue(v, "HTTP_ADDR", ":8080"),
		HTTPReadTimeout:         durationValue(v, "HTTP_READ_TIMEOUT", 15*time.Second),
		HTTPWriteTimeout:        durationValue(v, "HTTP_WRITE_TIMEOUT", 15*time.Second),
		SSEInitialDelay:         durationValue(v, "SSE_INITIAL_BACKOFF", time.Second),
		SSEMaxDelay:             durationValue(v, "SSE_MAX_BACKOFF", 30*time.Second),
		ScheduleRefreshInterval: durationValue(v, "SCHEDULE_REFRESH_INTERVAL", 15*time.Minute),
		OddsRefreshInterval:     durationValue(v, "ODDS_REFRESH_INTERVAL", 60*time.Second),
		ReconcileInterval:       durationValue(v, "RECONCILE_INTERVAL", 2*time.Minute),
		SettlementRetryBase:     durationValue(v, "SETTLEMENT_RETRY_BASE", 30*time.Second),
		MaxSettlementAttempts:   intValue(v, "MAX_SETTLEMENT_ATTEMPTS", 12),
		RedisURL:                stringValue(v, "REDIS_URL", "redis://127.0.0.1:6379/0"),
		KeeperKeypairPath:       strings.TrimSpace(v.GetString("KEEPER_KEYPAIR_PATH")),
		KeeperAutoSettle:        boolValue(v, "KEEPER_AUTO_SETTLE", false),
		TxlineStatKey:           uint32(intValue(v, "TXLINE_STAT_KEY", 1002)),
		CORSAllowedOrigins:      parseCORSOrigins(stringValue(v, "CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:3000,http://127.0.0.1:3000")),
		DatabaseURL:             buildDatabaseURL(v),
		JWTAccessSecret:         strings.TrimSpace(v.GetString("JWT_ACCESS_SECRET")),
		AccessTokenTTL:          durationValue(v, "ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:         durationValue(v, "REFRESH_TOKEN_TTL", 168*time.Hour),
		MagicLinkTTL:            durationValue(v, "MAGIC_LINK_TTL", 15*time.Minute),
		CookieSecure:            boolValue(v, "COOKIE_SECURE", false),
		CookieDomain:            strings.TrimSpace(v.GetString("COOKIE_DOMAIN")),
		FrontendURL:             stringValue(v, "FRONTEND_URL", "http://localhost:3000"),
		SMTPHost:                stringValue(v, "SMTP_HOST", "smtp.mail.yahoo.com"),
		SMTPPort:                intValue(v, "SMTP_PORT", 587),
		SMTPUser:                firstNonEmpty(v.GetString("SMTP_USER"), v.GetString("SMTP_USERNAME")),
		SMTPPass:                firstNonEmpty(v.GetString("SMTP_PASS"), v.GetString("SMTP_PASSWORD")),
		SMTPFrom:                stringValue(v, "SMTP_FROM", "Matchlock <noreply@matchlock.dev>"),
		LogLevel:                strings.ToLower(stringValue(v, "LOG_LEVEL", "info")),
		LogEncoding:             strings.ToLower(stringValue(v, "LOG_ENCODING", "json")),
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func newViper() *viper.Viper {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/matchlock")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	for key, value := range map[string]any{
		"SOLANA_CLUSTER":            ClusterDevnet,
		"MATCHLOCK_PROGRAM_ID":      DefaultProgramID,
		"STABLECOIN_MINT":           DevnetStablecoinMint,
		"TXLINE_PROGRAM_ID":         DevnetTxlineProgram,
		"HTTP_ADDR":                 ":8080",
		"HTTP_READ_TIMEOUT":         "15s",
		"HTTP_WRITE_TIMEOUT":        "15s",
		"SSE_INITIAL_BACKOFF":       "1s",
		"SSE_MAX_BACKOFF":           "30s",
		"SCHEDULE_REFRESH_INTERVAL": "15m",
		"ODDS_REFRESH_INTERVAL":     "60s",
		"RECONCILE_INTERVAL":        "2m",
		"SETTLEMENT_RETRY_BASE":     "30s",
		"MAX_SETTLEMENT_ATTEMPTS":   12,
		"REDIS_URL":                 "redis://127.0.0.1:6379/0",
		"KEEPER_AUTO_SETTLE":        false,
		"TXLINE_STAT_KEY":           1002,
		"CORS_ALLOWED_ORIGINS":      "http://localhost:5173,http://127.0.0.1:5173,http://localhost:3000,http://127.0.0.1:3000",
		"ACCESS_TOKEN_TTL":          "15m",
		"REFRESH_TOKEN_TTL":         "168h",
		"MAGIC_LINK_TTL":            "15m",
		"COOKIE_SECURE":             false,
		"FRONTEND_URL":              "http://localhost:3000",
		"SMTP_HOST":                 "smtp.mail.yahoo.com",
		"SMTP_PORT":                 587,
		"SMTP_FROM":                 "Matchlock <noreply@matchlock.dev>",
		"LOG_LEVEL":                 "info",
		"LOG_ENCODING":              "json",
	} {
		v.SetDefault(key, value)
	}

	return v
}

func readConfig(v *viper.Viper) error {
	if configFile := strings.TrimSpace(os.Getenv("MATCHLOCK_CONFIG_FILE")); configFile != "" {
		v.SetConfigFile(configFile)
		return v.ReadInConfig()
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("read config: %w", err)
		}
	}

	if _, err := os.Stat(".env"); err == nil {
		v.SetConfigFile(".env")
		v.SetConfigType("env")
		if err := v.MergeInConfig(); err != nil {
			return fmt.Errorf("read .env: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat .env: %w", err)
	}

	return nil
}

func buildDatabaseURL(v *viper.Viper) string {
	if url := strings.TrimSpace(v.GetString("DATABASE_URL")); url != "" {
		return url
	}
	host := stringValue(v, "DB_HOST", "127.0.0.1")
	port := intValue(v, "DB_PORT", 5432)
	name := strings.TrimSpace(v.GetString("DB_NAME"))
	user := strings.TrimSpace(v.GetString("DB_USER"))
	pass := strings.TrimSpace(v.GetString("DB_PASS"))
	sslmode := stringValue(v, "DB_SSLMODE", "disable")

	if name == "" || user == "" {
		return ""
	}

	userinfo := url.User(user)
	if pass != "" {
		userinfo = url.UserPassword(user, pass)
	}
	return fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=%s", userinfo.String(), host, port, name, sslmode)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func stringValue(v *viper.Viper, key, fallback string) string {
	value := strings.TrimSpace(v.GetString(key))
	if value == "" {
		return fallback
	}
	return value
}

func durationValue(v *viper.Viper, key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(v.GetString(key))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

func intValue(v *viper.Viper, key string, fallback int) int {
	raw := strings.TrimSpace(fmt.Sprint(v.Get(key)))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func boolValue(v *viper.Viper, key string, fallback bool) bool {
	rawValue := v.Get(key)
	if value, ok := rawValue.(bool); ok {
		return value
	}
	raw := strings.TrimSpace(strings.ToLower(fmt.Sprint(rawValue)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func (c Config) validate() error {
	switch c.Cluster {
	case ClusterDevnet, ClusterMainnet:
	default:
		return fmt.Errorf("unsupported SOLANA_CLUSTER %q (want %q or %q)", c.Cluster, ClusterDevnet, ClusterMainnet)
	}

	if _, err := url.ParseRequestURI(c.SolanaRPCURL); err != nil {
		return fmt.Errorf("invalid SOLANA_RPC_URL: %w", err)
	}
	if _, err := url.ParseRequestURI(c.TxlineAPIOrigin); err != nil {
		return fmt.Errorf("invalid TXLINE_API_ORIGIN: %w", err)
	}

	if err := validateNetworkConsistency(c); err != nil {
		return err
	}

	if c.TxlineAPIToken == "" {
		return fmt.Errorf("TXLINE_API_TOKEN is required (activate via /api/token/activate after on-chain subscribe)")
	}

	for name, value := range map[string]string{
		"MATCHLOCK_PROGRAM_ID": c.MatchlockProgram,
		"STABLECOIN_MINT":      c.StablecoinMint,
		"TXLINE_PROGRAM_ID":    c.TxlineProgram,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
	}

	if c.SSEInitialDelay <= 0 || c.SSEMaxDelay < c.SSEInitialDelay {
		return fmt.Errorf("invalid SSE backoff: initial=%s max=%s", c.SSEInitialDelay, c.SSEMaxDelay)
	}
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL must not be empty")
	}
	if c.KeeperKeypairPath == "" {
		return fmt.Errorf("KEEPER_KEYPAIR_PATH is required for settlement")
	}
	if c.TxlineStatKey == 0 {
		return fmt.Errorf("TXLINE_STAT_KEY must not be zero")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.JWTAccessSecret) < 32 {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least 32 characters")
	}
	if c.AccessTokenTTL <= 0 || c.RefreshTokenTTL <= 0 || c.MagicLinkTTL <= 0 {
		return fmt.Errorf("auth token TTLs must be positive")
	}
	if _, err := url.ParseRequestURI(c.FrontendURL); err != nil {
		return fmt.Errorf("invalid FRONTEND_URL: %w", err)
	}
	if c.SMTPUser == "" || c.SMTPPass == "" {
		return fmt.Errorf("SMTP_USER and SMTP_PASS are required")
	}

	return nil
}

// func boolEnv(key string, fallback bool) bool {
// 	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
// 	if raw == "" {
// 		return fallback
// 	}
// 	switch raw {
// 	case "1", "true", "yes", "on":
// 		return true
// 	case "0", "false", "no", "off":
// 		return false
// 	default:
// 		return fallback
// 	}
// }

func intEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	var v int
	if _, err := fmt.Sscanf(raw, "%d", &v); err != nil {
		return fallback
	}
	return v
}

func validateNetworkConsistency(c Config) error {
	origin := strings.ToLower(c.TxlineAPIOrigin)
	rpc := strings.ToLower(c.SolanaRPCURL)

	switch c.Cluster {
	case ClusterDevnet:
		if !strings.Contains(origin, "txline-dev") {
			return fmt.Errorf("devnet requires TXLINE_API_ORIGIN on txline-dev (got %s)", c.TxlineAPIOrigin)
		}
		if strings.Contains(rpc, "mainnet") {
			return fmt.Errorf("devnet cluster cannot use mainnet RPC URL")
		}
		if c.StablecoinMint == MainnetStablecoinMint {
			return fmt.Errorf("devnet cluster cannot use mainnet stablecoin mint %s", MainnetStablecoinMint)
		}
	case ClusterMainnet:
		if strings.Contains(origin, "txline-dev") {
			return fmt.Errorf("mainnet-beta cannot use devnet TXLINE_API_ORIGIN")
		}
		if strings.Contains(rpc, "devnet") {
			return fmt.Errorf("mainnet-beta cluster cannot use devnet RPC URL")
		}
	}
	return nil
}

const MainnetStablecoinMint = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"

func defaultRPCURL(cluster string) string {
	switch cluster {
	case ClusterMainnet:
		return "https://api.mainnet-beta.solana.com"
	default:
		return "https://api.devnet.solana.com"
	}
}

func defaultTxlineOrigin(cluster string) string {
	switch cluster {
	case ClusterMainnet:
		return MainnetTxlineOrigin
	default:
		return DevnetTxlineOrigin
	}
}

// func envOr(key, fallback string) string {
// 	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
// 		return v
// 	}
// 	return fallback
// }

func parseCORSOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		out = append(out, origin)
	}
	return out
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

// ScoresStreamURL returns the TxLINE scores SSE endpoint for this config.
func (c Config) ScoresStreamURL() string {
	return c.TxlineAPIOrigin + "/api/scores/stream"
}

// FixturesSnapshotURL returns the TxLINE fixtures schedule snapshot endpoint.
func (c Config) FixturesSnapshotURL() string {
	return c.TxlineAPIOrigin + "/api/fixtures/snapshot"
}

// GuestAuthURL returns the guest JWT bootstrap endpoint.
func (c Config) GuestAuthURL() string {
	return c.TxlineAPIOrigin + "/auth/guest/start"
}
