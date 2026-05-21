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
	DBName     string
	DBUser     string
	DBPassword string

	// Keycloak: el servicio se conecta vía URL interna (auth-service:8080),
	// pero los tokens son emitidos con iss = URL pública (localhost:8080).
	KeycloakInternalURL string
	KeycloakPublicURL   string
	KeycloakRealm       string
	KeycloakClientID    string
	AuthEnabled         bool
}

func Load() Config {
	return Config{
		ServiceName:         getenv("SERVICE_NAME", "client-service"),
		ServicePort:         getenv("SERVICE_PORT", "8081"),
		LogLevel:            getenv("LOG_LEVEL", "info"),
		DBHost:              getenv("DB_HOST", "postgres"),
		DBPort:              getenv("DB_PORT", "5432"),
		DBName:              getenv("DB_NAME", "clientes"),
		DBUser:              getenv("DB_USER", "prestamos"),
		DBPassword:          getenv("DB_PASSWORD", "prestamos"),
		KeycloakInternalURL: getenv("KEYCLOAK_INTERNAL_URL", "http://auth-service:8080"),
		KeycloakPublicURL:   getenv("KEYCLOAK_PUBLIC_URL", "http://localhost:8080"),
		KeycloakRealm:       getenv("KEYCLOAK_REALM", "prestamos"),
		KeycloakClientID:    getenv("KEYCLOAK_CLIENT_ID", "prestamos-frontend"),
		AuthEnabled:         getenv("AUTH_ENABLED", "true") == "true",
	}
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
