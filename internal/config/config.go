package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Supplier    SupplierConfig
	Environment string
	LogLevel    string
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type SupplierConfig struct {
	Timeout          time.Duration
	RetryMaxAttempts int
	RetryBackoff     time.Duration
	SupplierAURL     string
	SupplierBURL     string
}

func Load() *Config {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables: %v", err)
	}

	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "orders"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Supplier: SupplierConfig{
			Timeout:          getDurationEnv("SUPPLIER_TIMEOUT", 3*time.Second),
			RetryMaxAttempts: getIntEnv("SUPPLIER_RETRY_MAX", 3),
			RetryBackoff:     getDurationEnv("SUPPLIER_RETRY_BACKOFF", 1*time.Second),
			SupplierAURL:     getEnv("SUPPLIER_A_URL", "http://localhost:8081"),
			SupplierBURL:     getEnv("SUPPLIER_B_URL", "http://localhost:8082"),
		},
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := viper.GetString(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := viper.GetInt(key); value != 0 {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := viper.GetDuration(key); value != 0 {
		return value
	}
	return defaultValue
}
