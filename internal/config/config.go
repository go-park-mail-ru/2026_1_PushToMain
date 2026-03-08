package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/middleware"
	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string

	JWTSecret string
	JWTExpire time.Duration

	CORS middleware.CORSConfig
}

func Load() *Config {

	_ = godotenv.Load()

	expHours, _ := strconv.Atoi(os.Getenv("JWT_EXPIRE_HOURS"))

	return &Config{
		ServerPort: os.Getenv("APP_PORT"),

		JWTSecret: os.Getenv("JWT_SECRET"),
		JWTExpire: time.Duration(expHours) * time.Hour,

		CORS: middleware.CORSConfig{
			AllowedOrigins: splitEnvList(os.Getenv("CORS_ALLOWED_ORIGINS")),
			AllowedMethods: splitEnvList(os.Getenv("CORS_ALLOWED_METHODS")),
			AllowedHeaders: splitEnvList(os.Getenv("CORS_ALLOWED_HEADERS")),
		},
	}
}

func splitEnvList(env string) []string {
	if env == "" {
		return []string{}
	}
	items := strings.Split(env, ",")
	for i := range items {
		items[i] = strings.TrimSpace(items[i])
	}
	return items
}
