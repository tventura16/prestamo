package config

import (
	"fmt"
	"os"
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
