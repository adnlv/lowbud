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

	"github.com/adnlv/lowbud/internal/auth"
	"github.com/adnlv/lowbud/internal/config"
	"github.com/adnlv/lowbud/internal/delivery"
	"github.com/adnlv/lowbud/pkg/hash"
	"github.com/adnlv/lowbud/pkg/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxUUID "github.com/vgarvardt/pgx-google-uuid/v5"
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

	pgxConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxUUID.Register(conn.TypeMap())
		return nil
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

func mustListenAndServe(cfg *config.Config, dbpool *pgxpool.Pool) {
	mux := http.NewServeMux()

	tokenManager := auth.NewJwtManager(cfg.Jwt.Secret, cfg.Jwt.AccessTokenDuration, cfg.Jwt.RefreshTokenDuration)
	passwordHasher := hash.NewBcryptPasswordHasher(bcrypt.DefaultCost)
	uuidGenerator := uuid.NewV7Generator()

	authHandler := delivery.NewAuthHandler(dbpool, tokenManager, passwordHasher, uuidGenerator)
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)

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
