package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gbh-backend/internal/auth"
	"gbh-backend/internal/cache"
	"gbh-backend/internal/config"
	"gbh-backend/internal/db"
	"gbh-backend/internal/handlers"
	"gbh-backend/internal/middleware"
	"gbh-backend/internal/validation"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, cols, err := db.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		logger.Error("mongo connection failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("mongo connected")
	defer client.Disconnect(context.Background())

	if err := db.EnsureIndexes(ctx, cols); err != nil {
		logger.Error("index creation failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var cacheStore cache.Cache = cache.NewNoop()
	if cfg.RedisURL != "" || cfg.RedisAddr != "" {
		var redisCache *cache.RedisCache
		var err error
		if cfg.RedisURL != "" {
			redisCache, err = cache.NewRedisFromURL(cfg.RedisURL)
		} else {
			redisCache = cache.NewRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
		}
		if err != nil {
			logger.Error("redis connection failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if err := redisCache.Ping(ctx); err != nil {
			logger.Error("redis connection failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if cfg.RedisURL != "" {
			logger.Info("redis connected (url)")
		} else {
			logger.Info("redis connected", slog.String("addr", cfg.RedisAddr))
		}
		cacheStore = redisCache
	}

	var jwtManager *auth.Manager
	if cfg.JWTSecret != "" {
		jwtManager = &auth.Manager{
			Secret:     []byte(cfg.JWTSecret),
			AccessTTL:  time.Duration(cfg.AccessTTLMinutes) * time.Minute,
			RefreshTTL: time.Duration(cfg.RefreshTTLMinutes) * time.Minute,
			Issuer:     "gbh-backend",
		}
	}

	server := &handlers.Server{
		Cfg:  cfg,
		Cols: cols,
		Val:  validation.New(),
		Log:  logger,
		Cache: cacheStore,
	}

	r := chi.NewRouter()
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.FrontendOrigin))
	r.Use(chiMiddleware.Timeout(30 * time.Second))

	appointmentsLimiter := middleware.NewRateLimiter(cfg.RateLimitAppointments, time.Duration(cfg.RateLimitWindowSec)*time.Second)
	contactLimiter := middleware.NewRateLimiter(cfg.RateLimitContact, time.Duration(cfg.RateLimitWindowSec)*time.Second)

	registerAPIRoutes := func(api chi.Router) {
		api.Get("/services", server.GetServices)
		api.Get("/services/{id}/availability", server.GetServiceAvailability)
		api.Get("/availability", server.GetAvailability)
		api.Get("/availability/next", server.GetNextAvailability)
		api.With(appointmentsLimiter.Middleware).Post("/appointments", server.CreateAppointment)
		api.Get("/appointments/{id}", server.GetAppointment)
		api.With(contactLimiter.Middleware).Post("/contact", server.CreateContact)
		api.Post("/payments/intent", server.CreatePaymentIntent)

		api.Route("/admin", func(admin chi.Router) {
			admin.Post("/login", server.AdminLogin)
			admin.Post("/refresh", server.AdminRefresh)
			admin.Post("/logout", server.AdminLogout)
			admin.Use(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager))
			admin.Post("/services", server.AdminCreateService)
			admin.Put("/services/{id}", server.AdminUpdateService)
			admin.Delete("/services/{id}", server.AdminDeleteService)
			admin.Post("/blocks", server.AdminCreateBlock)
			admin.Delete("/blocks/{id}", server.AdminDeleteBlock)
			admin.Get("/appointments", server.AdminListAppointments)
			admin.Patch("/appointments/{id}/status", server.AdminUpdateAppointmentStatus)
			admin.Get("/contacts", server.AdminListContacts)
		})
	}

	// Supporte /api/... (normal) ET /api/api/... (front mal configuré / legacy).
	r.Route("/api", registerAPIRoutes)
	r.Route("/api/api", registerAPIRoutes)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: r,
	}

	go func() {
		logger.Info("server started", slog.String("addr", cfg.ServerAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.String("error", err.Error()))
	}
}
