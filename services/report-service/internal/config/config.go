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

	// El servicio sólo lee — consulta 3 bases de datos.
	DBNamePagos     string
	DBNamePrestamos string
	DBNameClientes  string
}

func Load() Config {
	return Config{
		ServiceName:     getenv("SERVICE_NAME", "report-service"),
		ServicePort:     getenv("SERVICE_PORT", "8084"),
		LogLevel:        getenv("LOG_LEVEL", "info"),
		DBHost:          getenv("DB_HOST", "postgres"),
		DBPort:          getenv("DB_PORT", "5432"),
		DBUser:          getenv("DB_USER", "prestamos"),
		DBPassword:      getenv("DB_PASSWORD", "prestamos"),
		DBNamePagos:     getenv("DB_NAME_PAGOS", "pagos"),
		DBNamePrestamos: getenv("DB_NAME_PRESTAMOS", "prestamos"),
		DBNameClientes:  getenv("DB_NAME_CLIENTES", "clientes"),
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
