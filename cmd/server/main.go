package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mini-crm/internal/config"
	httphandler "mini-crm/internal/handler/http"
	mysqlrepo "mini-crm/internal/repository/mysql"
	redisrepo "mini-crm/internal/repository/redis"
	"mini-crm/internal/service"
)

func main() {
	cfg := config.Load()

	var logHandler slog.Handler
	if cfg.Server.Env == "production" {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(logHandler))

	slog.Info("starting mini-crm server", "port", cfg.Server.Port, "env", cfg.Server.Env)

	db, err := mysqlrepo.NewDB(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.RunMigrations("migrations/001_init.sql"); err != nil {
		slog.Error("failed to run database migrations", "err", err)
		os.Exit(1)
	}

	redisClient, err := redisrepo.NewClient(cfg.Redis)
	if err != nil {
		slog.Warn("redis initialization error", "err", err)
	}
	defer redisClient.Close()

	taskCacheRepo := redisrepo.NewTaskCacheRepository(redisClient, cfg.Redis.CacheTTL)

	userRepo := mysqlrepo.NewUserRepository(db)
	teamRepo := mysqlrepo.NewTeamRepository(db)
	taskRepo := mysqlrepo.NewTaskRepository(db)
	historyRepo := mysqlrepo.NewHistoryRepository(db)
	commentRepo := mysqlrepo.NewCommentRepository(db)
	statsRepo := mysqlrepo.NewStatsRepository(db)

	authService := service.NewAuthService(userRepo, cfg.JWT)
	teamService := service.NewTeamService(teamRepo, userRepo)
	taskService := service.NewTaskService(taskRepo, teamRepo, historyRepo, taskCacheRepo)
	commentService := service.NewCommentService(commentRepo, taskRepo, teamRepo)
	statsService := service.NewStatsService(statsRepo, teamRepo)

	authHandler := httphandler.NewAuthHandler(authService)
	teamHandler := httphandler.NewTeamHandler(teamService)
	taskHandler := httphandler.NewTaskHandler(taskService)
	commentHandler := httphandler.NewCommentHandler(commentService)
	statsHandler := httphandler.NewStatsHandler(statsService)

	router := httphandler.NewRouter(httphandler.RouterParams{
		Config:         cfg,
		AuthService:    authService,
		AuthHandler:    authHandler,
		TeamHandler:    teamHandler,
		TaskHandler:    taskHandler,
		CommentHandler: commentHandler,
		StatsHandler:   statsHandler,
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "err", err)
	}

	slog.Info("server stopped gracefully")
}
