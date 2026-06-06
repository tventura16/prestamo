package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServiceName string
	ServicePort string
	LogLevel    string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string

	// El servicio escribe en DB "pagos" y lee/actualiza en DB "prestamos"
	// (cuotas, prestamos). Database-per-service en su forma pragmática.
	DBNamePagos     string
	DBNamePrestamos string

	KeycloakInternalURL string
	KeycloakPublicURL   string
	KeycloakRealm       string
	KeycloakClientID    string
	AuthEnabled         bool

	// Mensajería / outbox.
	KafkaBrokers       []string
	KafkaTopic         string
	KafkaDLQTopic      string
	ConsumerGroup      string
	RelayInterval      time.Duration
	ConsumerMaxRetries int
	EventsEnabled      bool
}

func Load() Config {
	return Config{
		ServiceName:         getenv("SERVICE_NAME", "payment-service"),
		ServicePort:         getenv("SERVICE_PORT", "8083"),
		LogLevel:            getenv("LOG_LEVEL", "info"),
		DBHost:              getenv("DB_HOST", "postgres"),
		DBPort:              getenv("DB_PORT", "5432"),
		DBUser:              getenv("DB_USER", "prestamos"),
		DBPassword:          getenv("DB_PASSWORD", "prestamos"),
		DBNamePagos:         getenv("DB_NAME", "pagos"),
		DBNamePrestamos:     getenv("DB_NAME_PRESTAMOS", "prestamos"),
		KeycloakInternalURL: getenv("KEYCLOAK_INTERNAL_URL", "http://auth-service:8080"),
		KeycloakPublicURL:   getenv("KEYCLOAK_PUBLIC_URL", "http://localhost:8080"),
		KeycloakRealm:       getenv("KEYCLOAK_REALM", "prestamos"),
		KeycloakClientID:    getenv("KEYCLOAK_CLIENT_ID", "prestamos-frontend"),
		AuthEnabled:         getenv("AUTH_ENABLED", "true") == "true",

		KafkaBrokers:       splitCSV(getenv("KAFKA_BROKERS", "kafka:9092")),
		KafkaTopic:         getenv("KAFKA_TOPIC", "pagos.eventos"),
		KafkaDLQTopic:      getenv("KAFKA_DLQ_TOPIC", "pagos.eventos.dlq"),
		ConsumerGroup:      getenv("KAFKA_CONSUMER_GROUP", "payment-service"),
		RelayInterval:      time.Duration(getenvInt("OUTBOX_RELAY_INTERVAL_MS", 1000)) * time.Millisecond,
		ConsumerMaxRetries: getenvInt("CONSUMER_MAX_RETRIES", 5),
		EventsEnabled:      getenv("EVENTS_ENABLED", "true") == "true",
	}
}

func (c Config) DSN(database string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, database)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
