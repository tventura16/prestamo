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
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(getenv("LOG_LEVEL", "info")),
	}))
	slog.SetDefault(logger)

	serviceName := getenv("SERVICE_NAME", "document-service")
	port := getenv("SERVICE_PORT", "8085")
	storePath := getenv("DOCUMENT_STORE_PATH", "/var/documents")

	if err := os.MkdirAll(storePath, 0o755); err != nil {
		logger.Error("cannot create document store", "path", storePath, "err", err)
		os.Exit(1)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger(logger))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": serviceName})
	})
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := r.Group("/documents")
	{
		v1.GET("", listDocs)
		v1.POST("/contract", generateContract)
		v1.POST("/receipt", generateReceipt)
		v1.POST("/statement", generateStatement)
		v1.GET("/:id", getDoc)
		v1.GET("/:id/download", downloadDoc)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("starting server", "service", serviceName, "port", port, "store", storePath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("forced shutdown", "err", err)
	}
	logger.Info("server stopped")
}

// ─── handlers (placeholders) ───

func listDocs(c *gin.Context)            { c.JSON(http.StatusOK, gin.H{"items": []any{}}) }
func generateContract(c *gin.Context)    { c.JSON(http.StatusAccepted, gin.H{"id": "todo", "tipo": "contrato"}) }
func generateReceipt(c *gin.Context)     { c.JSON(http.StatusAccepted, gin.H{"id": "todo", "tipo": "recibo"}) }
func generateStatement(c *gin.Context)   { c.JSON(http.StatusAccepted, gin.H{"id": "todo", "tipo": "estado_cuenta"}) }
func getDoc(c *gin.Context)              { c.JSON(http.StatusOK, gin.H{"id": c.Param("id")}) }
func downloadDoc(c *gin.Context)         { c.JSON(http.StatusNotImplemented, gin.H{"todo": true}) }

// ─── helpers ───

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
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
