package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	adhkarHandler "github.com/ilmnafi/backend/internal/adhkar/handler"
	adhkarRepo "github.com/ilmnafi/backend/internal/adhkar/repository"
	adhkarService "github.com/ilmnafi/backend/internal/adhkar/service"
	authHandler "github.com/ilmnafi/backend/internal/auth/handler"
	authRepo "github.com/ilmnafi/backend/internal/auth/repository"
	sessionRepo "github.com/ilmnafi/backend/internal/auth/repository"
	authService "github.com/ilmnafi/backend/internal/auth/service"
	"github.com/ilmnafi/backend/internal/config"
	"github.com/ilmnafi/backend/internal/database"
	"github.com/ilmnafi/backend/internal/email"
	"github.com/ilmnafi/backend/internal/health"
	"github.com/ilmnafi/backend/internal/middleware"
	quranHandler "github.com/ilmnafi/backend/internal/quran/handler"
	quranProvider "github.com/ilmnafi/backend/internal/quran/provider"
	quranRepo "github.com/ilmnafi/backend/internal/quran/repository"
	quranServicePkg "github.com/ilmnafi/backend/internal/quran/service"
	"github.com/ilmnafi/backend/internal/routes"
	"github.com/ilmnafi/backend/internal/security"
	userHandler "github.com/ilmnafi/backend/internal/user/handler"
	userRepo "github.com/ilmnafi/backend/internal/user/repository"
	userService "github.com/ilmnafi/backend/internal/user/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db, cfg); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	authRepo := authRepo.NewPostgresRepository(db)
	sessionRepo := sessionRepo.NewPostgresSessionRepository(db)
	userRepo := userRepo.NewPostgresUserRepository(db)
	quranRepo := quranRepo.NewPostgresQuranRepository(db)
	adhkarRepo := adhkarRepo.NewPostgresAdhkarRepository(db)

	passwordSvc := security.NewPasswordService()
	tokenSvc := security.NewTokenService()
	emailSvc := email.NewEmailService(cfg)

	authService := authService.NewAuthService(authRepo, sessionRepo, emailSvc, passwordSvc, tokenSvc, cfg)
	userService := userService.NewUserService(userRepo, authRepo)
	quranService := quranServicePkg.NewQuranService(quranRepo)
	quranContentService := quranServicePkg.NewContentService(quranProvider.NewQuranProvider(cfg.Quran))
	adhkarService := adhkarService.NewAdhkarService(adhkarRepo)

	authHdl := authHandler.NewAuthHandler(authService)
	userHdl := userHandler.NewUserHandler(userService)
	quranHdl := quranHandler.NewQuranHandler(quranService)
	quranContentHdl := quranHandler.NewContentHandler(quranContentService)
	adhkarHdl := adhkarHandler.NewAdhkarHandler(adhkarService)

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit.Requests, cfg.RateLimit.Window)
	authMiddleware := middleware.AuthMiddleware(cfg)

	healthChecker := health.NewHealthChecker(db.DB, cfg)

	router := routes.SetupRoutes(authHdl, userHdl, quranHdl, quranContentHdl, adhkarHdl, authMiddleware, rateLimiter, healthChecker)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
