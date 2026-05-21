package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/novudesk/novudesk/config"
	authapp "github.com/novudesk/novudesk/internal/application/auth"
	catapp "github.com/novudesk/novudesk/internal/application/category"
	orgapp "github.com/novudesk/novudesk/internal/application/organization"
	teamapp "github.com/novudesk/novudesk/internal/application/team"
	ticketapp "github.com/novudesk/novudesk/internal/application/ticket"
	userapp "github.com/novudesk/novudesk/internal/application/user"
	"github.com/novudesk/novudesk/internal/infrastructure/email"
	"github.com/novudesk/novudesk/internal/infrastructure/postgres"
	redisinfra "github.com/novudesk/novudesk/internal/infrastructure/redis"
	domainstorage "github.com/novudesk/novudesk/internal/domain/storage"
	"github.com/novudesk/novudesk/internal/infrastructure/storage"
	"github.com/novudesk/novudesk/internal/interfaces/http/handlers"
	httpserver "github.com/novudesk/novudesk/internal/interfaces/http"
	"github.com/novudesk/novudesk/internal/interfaces/sse"
	"github.com/novudesk/novudesk/pkg/logger"
)

func main() {
	log := logger.New(os.Getenv("APP_ENV"))

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// ─── Database ─────────────────────────────────────────────
	db, err := postgres.Connect(cfg.DB)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("connected to postgres")

	// ─── Redis ────────────────────────────────────────────────
	rdb, err := redisinfra.Connect(cfg.Redis)
	if err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	log.Info("connected to redis")

	// ─── Infrastructure ───────────────────────────────────────
	orgRepo        := postgres.NewOrgRepo(db)
	userRepo       := postgres.NewUserRepo(db)
	roleRepo       := postgres.NewRoleRepo(db)
	teamRepo       := postgres.NewTeamRepo(db)
	categoryRepo   := postgres.NewCategoryRepo(db)
	commentRepo    := postgres.NewCommentRepo(db)
	ticketRepo     := postgres.NewTicketRepo(db)
	auditRepo      := postgres.NewAuditRepo(db)
	attachmentRepo := postgres.NewAttachmentRepo(db)

	eventBus := redisinfra.NewEventBus(rdb, log)

	var storageProvider domainstorage.Provider
	if cfg.Storage.Driver == "s3" {
		storageProvider, err = storage.NewS3Provider(cfg.Storage.S3)
		if err != nil {
			log.Error("failed to init S3 storage", "error", err)
			os.Exit(1)
		}
	} else {
		storageProvider, err = storage.NewLocalProvider(cfg.Storage.LocalPath, fmt.Sprintf("http://localhost:%s", cfg.App.Port))
		if err != nil {
			log.Error("failed to init local storage", "error", err)
			os.Exit(1)
		}
	}

	smtpSender, err := email.NewSMTPSender(cfg.SMTP)
	if err != nil {
		log.Error("failed to init SMTP sender", "error", err)
		os.Exit(1)
	}
	_ = smtpSender

	// ─── Application services ─────────────────────────────────
	authService, err := authapp.NewService(userRepo, roleRepo, cfg.JWT)
	if err != nil {
		log.Error("failed to init auth service", "error", err)
		os.Exit(1)
	}
	authService.WithOrgRepo(orgRepo)
	authService.WithTeamRepo(teamRepo)

	_ = orgapp.NewService(orgRepo, userRepo, roleRepo)

	userService  := userapp.NewService(userRepo, roleRepo)
	teamService  := teamapp.NewService(teamRepo)
	catService   := catapp.NewService(categoryRepo)

	ticketService := ticketapp.NewService(ticketRepo, nil, auditRepo, eventBus)

	// ─── HTTP handlers ────────────────────────────────────────
	authHandler       := handlers.NewAuthHandler(authService, userService)
	ticketHandler     := handlers.NewTicketHandler(ticketService, teamRepo)
	memberHandler     := handlers.NewMemberHandler(userService, teamService, roleRepo)
	teamHandler       := handlers.NewTeamHandler(teamService, catService)
	categoryHandler   := handlers.NewCategoryHandler(catService)
	commentHandler    := handlers.NewCommentHandler(commentRepo, auditRepo)
	attachmentHandler := handlers.NewAttachmentHandler(attachmentRepo, storageProvider)

	// ─── SSE manager ──────────────────────────────────────────
	sseManager := sse.NewManager(eventBus, log)

	// ─── HTTP server ──────────────────────────────────────────
	handler := httpserver.NewRouter(
		authHandler,
		ticketHandler,
		memberHandler,
		teamHandler,
		categoryHandler,
		commentHandler,
		attachmentHandler,
		sseManager,
		authService,
		cfg.CORS.AllowedOrigins,
	)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// ─── Graceful shutdown ────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("server starting", "port", cfg.App.Port, "env", cfg.App.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	log.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}

	log.Info("server stopped")
}
