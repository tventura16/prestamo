package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	ServiceName string
	ServicePort string
	LogLevel    string

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string

	KeycloakInternalURL string
	KeycloakPublicURL   string
	KeycloakRealm       string
	KeycloakClientID    string
	AuthEnabled         bool

	// Job de mora: devengo automático y transiciones de estado por vencimiento.
	MoraJobEnabled  bool
	MoraJobInterval time.Duration

	// Almacenamiento de imágenes de garantía.
	GarantiasStorePath string
}

func Load() Config {
	return Config{
		ServiceName:         getenv("SERVICE_NAME", "loan-service"),
		ServicePort:         getenv("SERVICE_PORT", "8082"),
		LogLevel:            getenv("LOG_LEVEL", "info"),
		DBHost:              getenv("DB_HOST", "postgres"),
		DBPort:              getenv("DB_PORT", "5432"),
		DBName:              getenv("DB_NAME", "prestamos"),
		DBUser:              getenv("DB_USER", "prestamos"),
		DBPassword:          getenv("DB_PASSWORD", "prestamos"),
		KeycloakInternalURL: getenv("KEYCLOAK_INTERNAL_URL", "http://auth-service:8080"),
		KeycloakPublicURL:   getenv("KEYCLOAK_PUBLIC_URL", "http://localhost:8080"),
		KeycloakRealm:       getenv("KEYCLOAK_REALM", "prestamos"),
		KeycloakClientID:    getenv("KEYCLOAK_CLIENT_ID", "prestamos-frontend"),
		AuthEnabled:         getenv("AUTH_ENABLED", "true") == "true",
		MoraJobEnabled:      getenv("MORA_JOB_ENABLED", "true") == "true",
		MoraJobInterval:     parseDuration(getenv("MORA_JOB_INTERVAL", "24h"), 24*time.Hour),
		GarantiasStorePath:  getenv("GARANTIAS_STORE_PATH", "/var/garantias"),
	}
}

// parseDuration interpreta un valor tipo "24h"/"30m"; ante error o valor no
// positivo cae al default indicado.
func parseDuration(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return def
	}
	return d
}

func (c Config) PostgresDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
