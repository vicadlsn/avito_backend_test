package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"avito_backend_task/internal/config"
	"avito_backend_task/internal/repository"
	pullrequest "avito_backend_task/internal/service/pullrequest"
	team "avito_backend_task/internal/service/team"
	user "avito_backend_task/internal/service/user"
	transport "avito_backend_task/internal/transport/http"
	"avito_backend_task/pkg/db"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: .env file not found: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("error loading configuration: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.ParseLogLevel(),
	}))

	pool, err := connectDB(&cfg.Database)
	if err != nil {
		logger.Error("error connecting to db", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	dbInstance := db.NewDB(pool)
	txManager, err := db.NewTransactionManager(pool)
	if err != nil {
		logger.Error("error creating transaction manager", slog.Any("error", err))
		os.Exit(1)
	}

	teamRepo := repository.NewTeamRepository(dbInstance)
	userRepo := repository.NewUserRepository(dbInstance)
	prRepo := repository.NewPullRequestRepository(dbInstance)

	teamService := team.NewTeamService(teamRepo, userRepo, txManager, logger)
	userService := user.NewUserService(userRepo, prRepo, logger)
	prService := pullrequest.NewPullRequestService(prRepo, userRepo, txManager, logger)

	services := transport.Services{
		TeamService:        teamService,
		UserService:        userService,
		PullRequestService: prService,
	}

	validate := validator.New()

	router := transport.NewRouter(services, logger, validate)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		logger.Info("service started", slog.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("failed to start service", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("service forced to shutdown", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("service stopped")
}

func connectDB(cfg *config.DatabaseConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
	)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return pool, nil
}
