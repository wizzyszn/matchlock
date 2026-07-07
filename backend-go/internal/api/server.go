package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/matchlock/backend-go/internal/auth"
	"github.com/matchlock/backend-go/internal/cache"
	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
	"go.uber.org/zap"
)

type ServerConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	CORSOrigins  []string
	Logger       *zap.Logger
}

type Dependencies struct {
	Cache    cache.Store
	Solana   *chainsol.Client
	Wagers   WagerIndex
	Txline   *txline.Client
	Auth     *auth.Service
	Postgres ReadinessProbe
	TokenCfg auth.TokenConfig
	MatchSub MatchUpdateSubscriber
}

type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
}

func newHandler(deps Dependencies) *handler {
	wagerIndex := deps.Wagers
	if wagerIndex == nil {
		wagerIndex = deps.Solana
	}
	return &handler{
		cache:      deps.Cache,
		wagers:     wagerIndex,
		redis:      deps.Cache,
		rpc:        deps.Solana,
		txline:     deps.Txline,
		postgres:   deps.Postgres,
		txlineData: deps.Txline,
		solana:     deps.Solana,
		auth:       deps.Auth,
		tokenCfg:   deps.TokenCfg,
		matchSub:   deps.MatchSub,
	}
}

func newMux(h *handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
	mux.HandleFunc("GET /matches", h.listMatches)
	mux.HandleFunc("GET /matches/stream", h.matchStream)
	mux.HandleFunc("GET /matches/{id}", h.getMatch)
	mux.HandleFunc("GET /wagers", h.listWagers)
	mux.HandleFunc("GET /wagers/{pubkey}", h.getWager)
	mux.HandleFunc("GET /wagers/{pubkey}/settlement", h.getWagerSettlement)
	mux.HandleFunc("GET /wagers/{pubkey}/settlement-proof", h.getWagerSettlementProof)

	mux.HandleFunc("POST /auth/magic-link", h.postMagicLink)
	mux.HandleFunc("GET /auth/verify", h.getVerifyMagicLink)
	mux.HandleFunc("POST /auth/refresh", h.postRefresh)
	mux.HandleFunc("POST /auth/logout", h.postLogout)
	mux.HandleFunc("GET /auth/me", auth.RequireAuth(h.getMe))
	mux.HandleFunc("PATCH /auth/me", auth.RequireAuth(h.patchMe))
	mux.HandleFunc("GET /auth/wallets/check", auth.RequireAuth(h.getWalletCheck))
	mux.HandleFunc("POST /auth/wallets/challenge", auth.RequireAuth(h.postWalletLinkChallenge))
	mux.HandleFunc("POST /auth/wallets/link", auth.RequireAuth(h.postWalletLink))
	mux.HandleFunc("POST /auth/wallets/{pubkey}/primary", auth.RequireAuth(h.postWalletPrimary))
	mux.HandleFunc("DELETE /auth/wallets/{pubkey}", auth.RequireAuth(h.deleteWallet))
	mux.HandleFunc("GET /users/lookup", auth.RequireAuth(h.getUserLookup))

	mux.HandleFunc("POST /challenges/invites", auth.RequireAuth(h.postChallengeInvite))
	mux.HandleFunc("GET /challenges/invites", auth.RequireAuth(h.listChallengeInvites))
	mux.HandleFunc("GET /challenges/invites/{id}", auth.RequireAuth(h.getChallengeInvite))
	mux.HandleFunc("PATCH /challenges/invites/{id}", auth.RequireAuth(h.patchChallengeInvite))
	mux.HandleFunc("POST /challenges/invites/{id}/wager", auth.RequireAuth(h.postChallengeInviteWager))

	return mux
}

func NewServer(cfg ServerConfig, deps Dependencies) *Server {
	zapLogger := cfg.Logger
	if zapLogger == nil {
		zapLogger = zap.L()
	}
	var stack http.Handler = newMux(newHandler(deps))
	if deps.Auth != nil {
		stack = auth.Middleware(deps.Auth)(stack)
	}
	stack = corsMiddleware(cfg.CORSOrigins)(stack)
	stack = loggingMiddleware(zapLogger)(stack)
	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Addr,
			Handler:      stack,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			ErrorLog:     zap.NewStdLog(zapLogger.Named("http")),
		},
		logger: zapLogger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("http api listening", zap.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("http api shutdown failed", zap.Error(err))
			return fmt.Errorf("http shutdown: %w", err)
		}
		s.logger.Info("http api stopped")
		return ctx.Err()
	case err := <-errCh:
		if err != nil {
			s.logger.Error("http api listen failed", zap.Error(err))
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

func loggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = zap.L()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(recorder, r)
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}

			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", status),
				zap.Int("bytes", recorder.bytes),
				zap.Duration("duration", time.Since(start)),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
			}
			if requestID := r.Header.Get("X-Request-Id"); requestID != "" {
				fields = append(fields, zap.String("request_id", requestID))
			}

			if status >= http.StatusInternalServerError {
				logger.Error("http request completed", fields...)
				return
			}
			if status >= http.StatusBadRequest {
				logger.Warn("http request completed", fields...)
				return
			}
			logger.Info("http request completed", fields...)
		})
	}
}
