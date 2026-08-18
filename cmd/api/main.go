package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/adnlv/lowbud/internal/config"
	"github.com/adnlv/lowbud/internal/delivery"
	"github.com/adnlv/lowbud/internal/domain"
	"github.com/adnlv/lowbud/internal/infrastructure"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func mustReadConfig() *config.Config {
	err := config.InitEnv()
	if err != nil {
		panic(err)
	}

	envConfig, err := config.ReadEnv()
	if err != nil {
		panic(err)
	}

	return envConfig
}

func mustInitLogger() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))
	slog.SetDefault(logger)
}

func mustConnectToPostgres(cfg *config.Config) *pgxpool.Pool {
	pgxConfig, err := pgxpool.ParseConfig(cfg.Postgres.URL)
	if err != nil {
		slog.Error("Failed to parse postgres config: %s", err)
		os.Exit(1)
	}

	dbpool, err := pgxpool.NewWithConfig(context.Background(), pgxConfig)
	if err != nil {
		slog.Error("Failed to connect to postgres: %s", err)
		os.Exit(1)
	}

	if err = dbpool.Ping(context.Background()); err != nil {
		slog.Error("Failed to ping postgres: %s", err)
		os.Exit(1)
	}
	slog.Info("Connected to postgres")

	return dbpool
}

func mustListenAndServe(cfg *config.Config, db *pgxpool.Pool) {
	mux := http.NewServeMux()

	uuidProvider := infrastructure.NewGoogleUUIDV7Provider()
	passwordHasher := infrastructure.NewBcryptPasswordHasher(bcrypt.DefaultCost)
	accessTokenProvider := infrastructure.NewJwtProvider([]byte(cfg.Jwt.Secret), cfg.Jwt.AccessTokenDuration)

	authService := domain.NewAuthService(db, uuidProvider, passwordHasher, accessTokenProvider)
	accountService := domain.NewAccountService(db)

	authHandler := delivery.NewAuthHandler(authService)
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.RegisterAccount)
	mux.HandleFunc("POST /api/v1/auth/login/basic", authHandler.BasicLogin)

	accountHandler := delivery.NewAccountHandler(accountService)
	mux.HandleFunc("GET /api/v1/account", authHandler.DemandAccessTokenMiddleware(accountHandler.GetAccountInfo))
	mux.HandleFunc("DELETE /api/v1/account", authHandler.DemandAccessTokenMiddleware(accountHandler.CloseAccount))

	ledgerHandler := delivery.NewLedgerHandler(db, uuidProvider)
	mux.HandleFunc("POST /api/v1/ledger", authHandler.DemandAccessTokenMiddleware(ledgerHandler.CreateTransaction))

	server := &http.Server{
		Addr:         net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		Handler:      mux,
		IdleTimeout:  cfg.Server.IdleTimeout,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Listen and serve: %s", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Debug("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown: %s", err)
		os.Exit(1)
	}
	slog.Debug("Server shut down")
}

func main() {
	cfg := mustReadConfig()
	mustInitLogger()

	dbpool := mustConnectToPostgres(cfg)
	defer dbpool.Close()

	mustListenAndServe(cfg, dbpool)
}
