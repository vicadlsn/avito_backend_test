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
	"avito_backend_task/internal/service/pullrequests"
	"avito_backend_task/internal/service/teams"
	"avito_backend_task/internal/service/users"
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

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
	)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("error creating connection pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("error connecting to database: %v", err)
	}

	dbInstance := db.NewDB(pool)
	txManager, err := db.NewTransactionManager(pool)
	if err != nil {
		log.Fatalf("error creating transaction manager: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.ParseLogLevel(),
	}))
	slog.SetDefault(logger)

	teamRepo := repository.NewTeamRepository(dbInstance)
	userRepo := repository.NewUserRepository(dbInstance)
	prRepo := repository.NewPullRequestRepository(dbInstance)

	teamService := teams.NewTeamService(teamRepo, userRepo, txManager, logger)
	userService := users.NewUserService(userRepo, prRepo, logger)
	prService := pullrequests.NewPullRequestService(prRepo, userRepo, txManager, logger)

	services := transport.Services{
		TeamService:        teamService,
		UserService:        userService,
		PullRequestService: prService,
	}

	validate := validator.New()

	router := transport.NewRouter(services, logger, validate)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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
