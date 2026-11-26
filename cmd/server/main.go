package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"oge-ege/internal/admin"
	"oge-ege/internal/auth"
	"oge-ege/internal/content"
	"oge-ege/internal/help"
	"oge-ege/internal/notify"
	"oge-ege/internal/profile"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/config"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/httpserver"
	"oge-ege/internal/shared/logging"
	"oge-ege/internal/shared/mailer"
	"oge-ege/internal/shared/render"
	"oge-ege/internal/shared/storage"
	"oge-ege/migrations"
	webadmin "oge-ege/web/admin"
)

const notifyInterval = 10 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("загрузить конфигурацию: %w", err)
	}

	logger := logging.New(cfg.Log.Level)
	setGinMode(cfg.App.Env)

	pool, err := db.Open(context.Background(), cfg.DB.URL, cfg.DB.MinConns, cfg.DB.MaxConns)
	if err != nil {
		return fmt.Errorf("подключиться к базе данных: %w", err)
	}
	defer pool.Close()

	latestMigration, err := migrations.LatestVersion()
	if err != nil {
		return fmt.Errorf("определить версию миграций: %w", err)
	}

	clk := clock.New()
	txManager := db.NewTxManager(pool)

	mailerSender := mailer.NewSMTP(mailer.Options{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		User:     cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		TLS:      cfg.SMTP.TLS,
	})

	notifyRepo := notify.NewPgRepository()
	notifyService := notify.NewService(notifyRepo, txManager, mailerSender, clk, logger)
	notifyWorker := notify.NewWorker(notifyService, notifyInterval, logger)

	authRepo := auth.NewPgRepository()
	authService := auth.NewService(authRepo, pool, txManager, notifyService, clk, cfg.App.BaseURL, logger)

	contentRepo := content.NewPgRepository()
	contentService := content.NewService(contentRepo, pool, txManager, clk, nil, logger)

	profileRepo := profile.NewPgRepository()
	profileService := profile.NewService(profileRepo, pool, txManager, clk, authService, logger, contentService)

	storageStore, err := storage.NewMinIO(storage.Options{
		Endpoint:  cfg.S3.Endpoint,
		Bucket:    cfg.S3.Bucket,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
		Region:    cfg.S3.Region,
	})
	if err != nil {
		return fmt.Errorf("настроить объектное хранилище: %w", err)
	}

	adminRepo := admin.NewPgRepository()
	adminService := admin.NewService(contentService, authService, adminRepo, pool, txManager, clk, logger)
	adminHandler := admin.NewHandler(contentService, adminService, storageStore, logger)

	helpService := help.NewService()
	helpHandler := help.NewHandler(helpService)

	csrf := httpserver.NewCSRF(cfg.App.SessionSecret, cfg.App.Env == "production", auth.SessionTokenFromRequest)
	authHandler := auth.NewHandler(authService, csrf, profileService, logger)
	profileHandler := profile.NewHandler(profileService, logger)

	loginLimiter := httpserver.NewRateLimiter(cfg.RateLimit.LoginPerMin, time.Minute)
	apiLimiter := httpserver.NewRateLimiter(cfg.RateLimit.APIPerMin, time.Minute)

	router := httpserver.NewRouter(httpserver.Options{
		Env:              cfg.App.Env,
		Logger:           logger,
		MaxBodyBytes:     cfg.HTTP.MaxBodyBytes,
		Pinger:           pool,
		MigrationChecker: db.NewMigrationChecker(db.NewPgMigrationStateReader(pool), latestMigration),
		CSRF:             csrf.Middleware(),
		Routes: func(router *gin.Engine) {
			registerRoutes(router, authService, authHandler, profileHandler, adminHandler, helpHandler, storageStore, loginLimiter, apiLimiter)
		},
	})

	server := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	workerCtx, stopWorker := context.WithCancel(context.Background())
	defer stopWorker()
	go notifyWorker.Run(workerCtx)

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("HTTP-сервер запущен", "addr", cfg.HTTP.Addr, "env", cfg.App.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("запустить HTTP-сервер: %w", err)
	case sig := <-stop:
		logger.Info("получен сигнал остановки", "signal", sig.String())
	}

	stopWorker()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("остановить HTTP-сервер: %w", err)
	}

	logger.Info("HTTP-сервер остановлен")

	return nil
}

func registerRoutes(
	router *gin.Engine,
	authService *auth.Service,
	authHandler *auth.Handler,
	profileHandler *profile.Handler,
	adminHandler *admin.Handler,
	helpHandler *help.Handler,
	storageStore storage.Store,
	loginLimiter *httpserver.RateLimiter,
	apiLimiter *httpserver.RateLimiter,
) {
	loginLimit := loginLimiter.Middleware(httpserver.ClientIPKey)

	api := router.Group("/api", apiLimiter.Middleware(httpserver.ClientIPKey))

	router.GET(render.StylesPath(), render.StylesHandler())

	router.GET("/account", auth.AuthPage(authService), authHandler.ShowAccount)
	router.GET("/logout", auth.AuthPage(authService), authHandler.ShowLogout)
	router.POST("/logout", auth.AuthPage(authService), authHandler.SubmitLogout)

	router.GET("/login", authHandler.ShowLogin)
	router.POST("/login", loginLimit, authHandler.SubmitLogin)
	router.GET("/register", authHandler.ShowRegister)
	router.POST("/register", loginLimit, authHandler.SubmitRegister)
	router.GET("/verify-email", authHandler.ShowVerifyEmail)
	router.POST("/verify-email", loginLimit, authHandler.SubmitVerifyEmail)
	router.POST("/resend-verification", loginLimit, authHandler.SubmitResendVerification)
	router.GET("/password/reset", authHandler.ShowPasswordReset)
	router.POST("/password/reset", loginLimit, authHandler.SubmitPasswordReset)
	router.POST("/password/reset-request", loginLimit, authHandler.SubmitPasswordResetRequest)

	authRoutes := api.Group("/auth")
	authRoutes.POST("/register", loginLimit, authHandler.Register)
	authRoutes.POST("/login", loginLimit, authHandler.Login)
	authRoutes.POST("/verify-email", loginLimit, authHandler.VerifyEmail)
	authRoutes.POST("/resend-verification", loginLimit, authHandler.ResendVerification)
	authRoutes.POST("/password/reset-request", loginLimit, authHandler.RequestPasswordReset)
	authRoutes.POST("/password/reset", loginLimit, authHandler.ResetPassword)

	authRoutes.POST("/logout", auth.Auth(authService), authHandler.Logout)
	authRoutes.POST("/password/change", auth.Auth(authService), authHandler.ChangePassword)

	me := api.Group("/me", auth.Auth(authService))
	me.GET("", profileHandler.Me)
	me.GET("/profile", profileHandler.GetProfile)
	me.PATCH("/profile", profileHandler.UpdateProfile)
	me.GET("/goals", profileHandler.GetGoals)
	me.PUT("/goals", profileHandler.SaveGoals)
	me.GET("/settings", profileHandler.GetSettings)
	me.PATCH("/settings", profileHandler.UpdateSettings)
	me.POST("/delete-request", profileHandler.DeleteRequest)

	adminGroup := api.Group("/admin", auth.Auth(authService), auth.RequireRole(auth.RoleMethodist, auth.RoleAdmin))
	adminHandler.Register(adminGroup)
	adminOnly := adminGroup.Group("", auth.RequireRole(auth.RoleAdmin))
	adminHandler.RegisterAdminOnly(adminOnly)

	helpGroup := api.Group("", auth.Auth(authService), auth.RequireRole(auth.RoleMethodist, auth.RoleAdmin))
	helpHandler.Register(helpGroup)

	router.GET("/media/*key", apiLimiter.Middleware(httpserver.ClientIPKey), storage.MediaHandler(storageStore))
	router.GET("/admin", webadmin.Handler())
	router.GET("/admin/*path", webadmin.Handler())
}

func setGinMode(env string) {
	switch env {
	case "local", "development":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}
}
