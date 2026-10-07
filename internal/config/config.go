package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv           string
	AppPort          string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBSSLMode        string
	JWTSecret        string
	JWTExpiresMinute int
}

func Load() *Config {
	_ = godotenv.Load() // .env opsional

	exp, err := strconv.Atoi(get("JWT_EXPIRES_MINUTES", "60"))
	if err != nil {
		exp = 60
	}

	return &Config{
		AppEnv:           get("APP_ENV", "development"),
		AppPort:          get("APP_PORT", "8080"),
		DBHost:           get("DB_HOST", "localhost"),
		DBPort:           get("DB_PORT", "5432"),
		DBUser:           get("DB_USER", "postgres"),
		DBPassword:       get("DB_PASSWORD", "postgres"),
		DBName:           get("DB_NAME", "siakad_mini"),
		DBSSLMode:        get("DB_SSLMODE", "disable"),
		JWTSecret:        get("JWT_SECRET", "dev-secret"),
		JWTExpiresMinute: exp,
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Jakarta",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
