package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	solanago "github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/api"
	"github.com/matchlock/backend-go/internal/auth"
	"github.com/matchlock/backend-go/internal/cache"
	"github.com/matchlock/backend-go/internal/config"
	"github.com/matchlock/backend-go/internal/db"
	"github.com/matchlock/backend-go/internal/email"
	"github.com/matchlock/backend-go/internal/keeper"
	"github.com/matchlock/backend-go/internal/leaderboard"
	applogger "github.com/matchlock/backend-go/internal/logger"
	"github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
	"go.uber.org/zap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		fallbackLogger, _ := zap.NewProduction()
		fallbackLogger.Error("config load failed", zap.Error(err))
		os.Exit(1)
	}
	zapLogger, err := applogger.Configure(applogger.Config{
		Level:    cfg.LogLevel,
		Encoding: cfg.LogEncoding,
	})
	if err != nil {
		fallbackLogger, _ := zap.NewProduction()
		fallbackLogger.Error("logger init failed", zap.Error(err))
		os.Exit(1)
	}
	defer zapLogger.Sync()

	redisStore, err := cache.NewRedisStore(ctx, cfg.RedisURL)
	if err != nil {
		slog.Error("redis connect failed", "err", err)
		os.Exit(1)
	}
	defer redisStore.Close()

	slog.Info("matchlock keeper starting",
		"cluster", cfg.Cluster,
		"txline_origin", cfg.TxlineAPIOrigin,
		"program", cfg.MatchlockProgram,
		"redis", cfg.RedisURL,
	)

	txClient := txline.NewClient(cfg.TxlineAPIOrigin, cfg.GuestAuthURL(), cfg.TxlineAPIToken, nil)
	if err := txClient.EnsureGuestJWT(ctx, false); err != nil {
		slog.Error("txline guest auth failed", "err", err)
		os.Exit(1)
	}
	slog.Info("txline guest jwt acquired")

	gdb, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	sqlDB, err := gdb.DB()
	if err == nil {
		defer sqlDB.Close()
	}

	mailer := email.NewMailer(email.Config{
		APIKey: cfg.BrevoAPIKey,
		From:   cfg.BrevoFrom,
	})
	emailQueue := email.NewQueue(mailer, 128)
	go emailQueue.Start(ctx)

	tokenCfg := auth.TokenConfig{
		AccessSecret: []byte(cfg.JWTAccessSecret),
		AccessTTL:    cfg.AccessTokenTTL,
		RefreshTTL:   cfg.RefreshTokenTTL,
		MagicLinkTTL: cfg.MagicLinkTTL,
		CookieSecure: cfg.CookieSecure,
		CookieDomain: cfg.CookieDomain,
		FrontendURL:  cfg.FrontendURL,
	}
	authSvc := auth.NewService(gdb, mailer, tokenCfg)
	authSvc.SetEmailQueue(emailQueue)
	slog.Info("auth service ready", "frontend", cfg.FrontendURL)

	lbService := leaderboard.NewService(gdb)

	keeperKey, err := solana.LoadKeeperKeypairFromFile(cfg.KeeperKeypairPath)
	if err != nil {
		slog.Error("keeper keypair load failed", "err", err)
		os.Exit(1)
	}

	solClient, err := solana.NewClient(cfg.SolanaRPCURL, cfg.MatchlockProgram, cfg.StablecoinMint, cfg.TxlineProgram)
	if err != nil {
		slog.Error("solana client init failed", "err", err)
		os.Exit(1)
	}

	// Enable on-chain wallet registration during LinkWallet.
	authSvc.SetWalletRegistrar(solClient, keeperKey)

	worker := &keeper.Worker{
		Cache:                 redisStore,
		Txline:                txClient,
		Solana:                solClient,
		KeeperKey:             keeperKey,
		StatKey:               cfg.TxlineStatKey,
		AutoSettle:            cfg.KeeperAutoSettle,
		MaxSettlementAttempts: cfg.MaxSettlementAttempts,
		SettlementRetryBase:   cfg.SettlementRetryBase,
		Leaderboard:           lbService,
	}
	if cfg.KeeperAutoSettle {
		slog.Info("keeper auto-settle enabled")
	} else {
		slog.Info("keeper auto-settle disabled; winners must claim via wallet")
	}

	programPubkey := solanago.MustPublicKeyFromBase58(cfg.MatchlockProgram)
	wagerIndexer := keeper.NewWagerIndexer(
		cfg.SolanaRPCURL,
		keeper.WSURLFromRPC(cfg.SolanaRPCURL),
		programPubkey,
		redisStore,
	)
	go func() {
		if err := wagerIndexer.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("wager indexer exited", "err", err)
		}
	}()

	reconcileWorker := &keeper.ReconcileWorker{
		Worker:   worker,
		Interval: cfg.ReconcileInterval,
	}
	go func() {
		if err := reconcileWorker.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("reconcile worker exited", "err", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(4 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := db.PurgeExpiredTokens(gdb)
				if err != nil {
					slog.Error("token cleanup failed", "err", err)
				} else if n > 0 {
					slog.Info("token cleanup complete", "deleted", n)
				}
			}
		}
	}()

	apiServer := api.NewServer(api.ServerConfig{
		Addr:         cfg.HTTPAddr,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		CORSOrigins:  cfg.CORSAllowedOrigins,
		Logger:       zapLogger,
	}, api.Dependencies{
		Cache:       redisStore,
		Solana:      solClient,
		Leaderboard: lbService,
		Wagers:      api.NewCachedWagerIndex(redisStore),
		Txline:      txClient,
		Auth:        authSvc,
		Postgres:    db.Probe{DB: gdb},
		TokenCfg:    tokenCfg,
		MatchSub:    redisStore,
	})

	scheduleWorker := &keeper.ScheduleWorker{
		Cache:    redisStore,
		Txline:   txClient,
		Interval: cfg.ScheduleRefreshInterval,
	}
	go func() {
		if err := scheduleWorker.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("schedule prefetch exited", "err", err)
		}
	}()

	oddsWorker := &keeper.OddsWorker{
		Cache:    redisStore,
		Txline:   txClient,
		Interval: cfg.OddsRefreshInterval,
	}
	go func() {
		if err := oddsWorker.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("odds refresh exited", "err", err)
		}
	}()

	events := make(chan txline.ScoreUpdate, 256)
	go func() {
		err := txline.StreamScores(ctx, txClient, txline.StreamConfig{
			StreamURL:   cfg.ScoresStreamURL(),
			InitialWait: cfg.SSEInitialDelay,
			MaxWait:     cfg.SSEMaxDelay,
		}, events)
		if err != nil && ctx.Err() == nil {
			slog.Error("sse stream exited", "err", err)
		}
		close(events)
	}()

	go func() {
		if err := apiServer.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("http api exited", "err", err)
			stop()
		}
	}()

	if err := worker.Run(ctx, events); err != nil && ctx.Err() == nil {
		slog.Error("keeper worker exited", "err", err)
		os.Exit(1)
	}
	slog.Info("shutting down")
}
