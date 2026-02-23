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
	"gbh-backend/internal/casestudies"
	"gbh-backend/internal/config"
	"gbh-backend/internal/db"
	"gbh-backend/internal/handlers"
	"gbh-backend/internal/middleware"
	"gbh-backend/internal/notifications"
	"gbh-backend/internal/references"
	"gbh-backend/internal/rfp"
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

	mailer := notifications.NewBrevoClient(cfg.BrevoAPIKey, cfg.BrevoSenderEmail, cfg.BrevoSenderName, cfg.BrevoSandbox)
	if mailer == nil {
		logger.Info("brevo mailer disabled")
	} else {
		logger.Info("brevo mailer enabled", slog.String("sender", cfg.BrevoSenderEmail), slog.Bool("sandbox", cfg.BrevoSandbox))
	}

	server := &handlers.Server{
		Cfg:    cfg,
		Cols:   cols,
		Val:    validation.New(),
		Log:    logger,
		Cache:  cacheStore,
		Mailer: mailer,
	}

	rfpRepo := rfp.NewRepository(cols.RFPLeads)
	rfpService := rfp.NewService(rfpRepo, cfg.Timezone, mailer)
	rfpHandler := rfp.NewHandler(rfpService, server.Val, logger)

	referencesRepo := references.NewRepository(cols.References)
	referencesService := references.NewService(referencesRepo, cfg.Timezone)
	referencesHandler := references.NewHandler(referencesService, server.Val, logger)

	caseStudiesRepo := casestudies.NewRepository(cols.CaseStudies)
	caseStudiesService := casestudies.NewService(caseStudiesRepo, cfg.Timezone)
	caseStudiesHandler := casestudies.NewHandler(caseStudiesService, server.Val, logger)

	r := chi.NewRouter()
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.FrontendOrigins))
	r.Use(chiMiddleware.Timeout(30 * time.Second))

	appointmentsLimiter := middleware.NewRateLimiter(cfg.RateLimitAppointments, time.Duration(cfg.RateLimitWindowSec)*time.Second)
	contactLimiter := middleware.NewRateLimiter(cfg.RateLimitContact, time.Duration(cfg.RateLimitWindowSec)*time.Second)

	registerCoreRoutes := func(api chi.Router) {
		api.Get("/services", server.GetServices)
		api.Get("/services/{id}/availability", server.GetServiceAvailability)
		api.Get("/services/{id}/testimonials", server.GetServiceTestimonials)
		api.With(contactLimiter.Middleware).Post("/services/{id}/testimonials", server.CreateServiceTestimonial)
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager))
			protected.Post("/services", server.AdminCreateService)
			protected.Put("/services/{id}", server.AdminUpdateService)
		})
		api.Get("/availability", server.GetAvailability)
		api.Get("/availability/next", server.GetNextAvailability)
		api.With(appointmentsLimiter.Middleware).Post("/appointments", server.CreateAppointment)
		api.Post("/appointments/lookup", server.LookupAppointment)
		api.Get("/appointments/{id}", server.GetAppointment)
		api.With(contactLimiter.Middleware).Post("/contact", server.CreateContact)
		api.Post("/payments/intent", server.CreatePaymentIntent)

		api.Route("/admin", func(admin chi.Router) {
			admin.Post("/register", server.AdminRegister)
			admin.Post("/login", server.AdminLogin)
			admin.Post("/refresh", server.AdminRefresh)
			admin.Post("/logout", server.AdminLogout)

			// Important (chi): middlewares must be attached before defining routes.
			// We keep login/refresh/logout public, and protect the rest via a sub-router.
			admin.Group(func(protected chi.Router) {
				protected.Use(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager))
				protected.Post("/services", server.AdminCreateService)
				protected.Put("/services/{id}", server.AdminUpdateService)
				protected.Delete("/services/{id}", server.AdminDeleteService)
				protected.Post("/blocks", server.AdminCreateBlock)
				protected.Delete("/blocks/{id}", server.AdminDeleteBlock)
				protected.Post("/users", server.AdminCreateUser)
				protected.Patch("/users/{id}/password", server.AdminUpdateUserPassword)
				protected.Get("/appointments", server.AdminListAppointments)
				protected.Patch("/appointments/{id}/status", server.AdminUpdateAppointmentStatus)
				protected.Get("/contacts", server.AdminListContacts)
			})
		})
	}

	registerB2BRoutes := func(api chi.Router) {
		api.Post("/rfp", rfpHandler.Create)
		api.Get("/references", referencesHandler.PublicList)
		api.Get("/case-studies", caseStudiesHandler.PublicList)
		api.Get("/case-studies/{slug}", caseStudiesHandler.PublicGetBySlug)

		api.With(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager)).Get("/admin/rfp", rfpHandler.AdminList)
		api.With(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager)).Get("/admin/rfp/{id}", rfpHandler.AdminGetByID)
		api.With(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager)).Patch("/admin/rfp/{id}", rfpHandler.AdminUpdateStatus)

		api.With(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager)).Get("/admin/references", referencesHandler.AdminList)
		api.With(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager)).Post("/admin/references", referencesHandler.AdminCreate)
		api.With(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager)).Put("/admin/references/{id}", referencesHandler.AdminUpdate)
		api.With(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager)).Delete("/admin/references/{id}", referencesHandler.AdminDelete)

		api.With(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager)).Get("/admin/case-studies", caseStudiesHandler.AdminList)
		api.With(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager)).Post("/admin/case-studies", caseStudiesHandler.AdminCreate)
		api.With(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager)).Put("/admin/case-studies/{id}", caseStudiesHandler.AdminUpdate)
		api.With(middleware.AdminAuth(cfg.AdminAPIKey, jwtManager)).Delete("/admin/case-studies/{id}", caseStudiesHandler.AdminDelete)
	}

	registerV1Routes := func(api chi.Router) {
		registerCoreRoutes(api)
		registerB2BRoutes(api)
	}

	// Supporte /api/... (legacy) et /api/v1/... (nouvelle version), plus alias /api/api/... .
	r.Route("/api", registerCoreRoutes)
	r.Route("/api/api", registerCoreRoutes)
	r.Route("/api/v1", registerV1Routes)
	r.Route("/api/api/v1", registerV1Routes)

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
