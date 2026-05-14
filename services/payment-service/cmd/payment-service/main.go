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
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/prestamos/payment-service/internal/config"
	"github.com/prestamos/payment-service/internal/db"
	"github.com/prestamos/payment-service/internal/handler"
	"github.com/prestamos/payment-service/internal/repository"
	"github.com/prestamos/payment-service/internal/service"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	pagoRepo := repository.NewPagoRepository(pagosPool)
	loanRepo := repository.NewLoanRepository(prestamosPool)
	paymentSvc := service.NewPaymentService(pagoRepo, loanRepo)
	pagoHandler := handler.NewPagoHandler(paymentSvc, pagoRepo)

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

	pagoHandler.Register(r.Group("/payments"))

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
