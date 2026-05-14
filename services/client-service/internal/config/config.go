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
}

func Load() Config {
	return Config{
		ServiceName: getenv("SERVICE_NAME", "client-service"),
		ServicePort: getenv("SERVICE_PORT", "8081"),
		LogLevel:    getenv("LOG_LEVEL", "info"),
		DBHost:      getenv("DB_HOST", "postgres"),
		DBPort:      getenv("DB_PORT", "5432"),
		DBName:      getenv("DB_NAME", "clientes"),
		DBUser:      getenv("DB_USER", "prestamos"),
		DBPassword:  getenv("DB_PASSWORD", "prestamos"),
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
