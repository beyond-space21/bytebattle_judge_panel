package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	Judge0URL       string
	JWTSecret       string
	AdminUsername   string
	AdminPassword   string
	CORSOrigin      string
	SessionTTL      time.Duration
	PythonLanguageID int
}

func Load() Config {
	return Config{
		HTTPAddr:         getenv("HTTP_ADDR", ":8759"),
		DatabaseURL:      getenv("DATABASE_URL", "postgres://judge:judge@localhost:5432/judge?sslmode=disable"),
		Judge0URL:        getenv("JUDGE0_URL", "http://localhost:2358"),
		JWTSecret:        getenv("JWT_SECRET", "dev-jwt-secret-change-me"),
		AdminUsername:    getenv("ADMIN_USERNAME", "admin"),
		AdminPassword:    getenv("ADMIN_PASSWORD", "admin123"),
		CORSOrigin:       getenv("CORS_ORIGIN", "*"),
		SessionTTL:       durationEnv("SESSION_TTL", 72*time.Hour),
		PythonLanguageID: intEnv("PYTHON_LANGUAGE_ID", 71), // Python 3.8.1 in Judge0 CE
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func intEnv(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func durationEnv(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
