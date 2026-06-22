package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/prestamos/payment-service/internal/auth"
	"github.com/prestamos/payment-service/internal/config"
	"github.com/prestamos/payment-service/internal/consumer"
	"github.com/prestamos/payment-service/internal/db"
	"github.com/prestamos/payment-service/internal/docs"
	"github.com/prestamos/payment-service/internal/handler"
	"github.com/prestamos/payment-service/internal/messaging"
	"github.com/prestamos/payment-service/internal/outbox"
	"github.com/prestamos/payment-service/internal/repository"
	"github.com/prestamos/payment-service/internal/service"
)

func main() {
	// Modo healthcheck para Docker: la imagen es distroless (sin shell ni curl),
	// así que `<binario> -healthcheck` consulta /health y sale 0/1.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		healthcheck()
	}

	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Register with Consul
	if consulAddr := os.Getenv("CONSUL_ADDR"); consulAddr != "" {
		consulCfg := api.DefaultConfig()
		consulCfg.Address = consulAddr
		consulClient, err := api.NewClient(consulCfg)
		if err != nil {
			logger.Error("cannot create consul client", "err", err)
		} else {
			servicePort := 8083
			reg := &api.AgentServiceRegistration{
				ID:      cfg.ServiceName + "-1",
				Name:    cfg.ServiceName,
				Port:    servicePort,
				Address: os.Getenv("SERVICE_HOST"),
				Check: &api.AgentServiceCheck{
					HTTP:     "http://" + os.Getenv("SERVICE_HOST") + ":" + os.Getenv("SERVICE_PORT") + "/health",
					Interval: "10s",
					Timeout:  "5s",
				},
			}
			if err := consulClient.Agent().ServiceRegister(reg); err != nil {
				logger.Error("cannot register with consul", "err", err)
			} else {
				logger.Info("registered with consul", "service", cfg.ServiceName)
				defer func() {
					if err := consulClient.Agent().ServiceDeregister(cfg.ServiceName + "-1"); err != nil {
						logger.Error("cannot deregister from consul", "err", err)
					}
				}()
			}
		}
	}

	pagosPool, err := db.NewPool(ctx, cfg.DSN(cfg.DBNamePagos))
	if err != nil {
		logger.Error("cannot connect to pagos db", "err", err)
		os.Exit(1)
	}
	defer pagosPool.Close()
	logger.Info("connected to pagos db", "db", cfg.DBNamePagos)

	prestamosPool, err := db.NewPool(ctx, cfg.DSN(cfg.DBNamePrestamos))
	if err != nil {
		logger.Error("cannot connect to prestamos db", "err", err)
		os.Exit(1)
	}
	defer prestamosPool.Close()
	logger.Info("connected to prestamos db", "db", cfg.DBNamePrestamos)

	var verifier *auth.Verifier
	if cfg.AuthEnabled {
		verifier, err = auth.NewVerifier(ctx, cfg.KeycloakInternalURL, cfg.KeycloakPublicURL, cfg.KeycloakRealm, cfg.KeycloakClientID)
		if err != nil {
			logger.Error("cannot initialize auth", "err", err)
			os.Exit(1)
		}
		logger.Info("auth enabled", "realm", cfg.KeycloakRealm)
	} else {
		logger.Warn("AUTH DISABLED")
	}

	pagoRepo := repository.NewPagoRepository(pagosPool)
	loanRepo := repository.NewLoanRepository(prestamosPool)
	outboxRepo := repository.NewOutboxRepository(pagosPool)
	paymentSvc := service.NewPaymentService(pagoRepo, loanRepo, logger)
	pagoHandler := handler.NewPagoHandler(paymentSvc, pagoRepo, verifier)

	// ───── Mensajería: outbox relay + consumer ─────
	var publisher *messaging.Publisher
	if cfg.EventsEnabled {
		topicCtx, cancelTopic := context.WithTimeout(ctx, 30*time.Second)
		if err := messaging.EnsureTopics(topicCtx, cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaDLQTopic); err != nil {
			logger.Error("cannot ensure kafka topics", "err", err)
			cancelTopic()
			os.Exit(1)
		}
		cancelTopic()

		publisher = messaging.NewPublisher(cfg.KafkaBrokers)
		defer publisher.Close()

		relay := outbox.NewRelay(outboxRepo, publisher, cfg.KafkaTopic, cfg.RelayInterval, logger)
		go relay.Run(ctx)

		reader := messaging.NewReader(cfg.KafkaBrokers, cfg.ConsumerGroup, cfg.KafkaTopic)
		defer reader.Close()
		pagoConsumer := consumer.NewPagoConsumer(reader, publisher, loanRepo, cfg.KafkaDLQTopic, cfg.ConsumerMaxRetries, logger)
		go pagoConsumer.Run(ctx)

		logger.Info("eventos habilitados", "topic", cfg.KafkaTopic, "brokers", cfg.KafkaBrokers)
	} else {
		logger.Warn("EVENTS DISABLED: outbox no se publicará (sólo fast-path inline)")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger(logger))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": cfg.ServiceName})
	})
	r.GET("/ready", func(c *gin.Context) {
		pingCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := pagosPool.Ping(pingCtx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "pagos_db": err.Error()})
			return
		}
		if err := prestamosPool.Ping(pingCtx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "prestamos_db": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	docs.Register(r) // /docs (Swagger UI) y /openapi.yaml, públicos

	api := r.Group("")
	if verifier != nil {
		api.Use(verifier.Middleware())
	}
	pagoHandler.Register(api.Group("/payments"))

	srv := &http.Server{
		Addr:              ":" + cfg.ServicePort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("starting server", "service", cfg.ServiceName, "port", cfg.ServicePort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("forced shutdown", "err", err)
	}
	logger.Info("server stopped")
}

// healthcheck consulta el endpoint /health local y termina el proceso con
// código 0 (sano) o 1 (caído). Lo invoca el HEALTHCHECK de Docker.
func healthcheck() {
	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8083"
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", c.GetHeader("X-Request-ID"),
		)
	}
}
