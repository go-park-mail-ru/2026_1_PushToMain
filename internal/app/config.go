package app

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string

	JWTSecret string
	JWTExpire time.Duration

	CORS   middleware.CORSConfig
	Logger logger.Config
}

func Load() (*Config, error) {

	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	expHours, err := strconv.Atoi(os.Getenv("JWT_EXPIRE_HOURS"))
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ServerPort: os.Getenv("APP_PORT"),

		JWTSecret: os.Getenv("JWT_SECRET"),
		JWTExpire: time.Duration(expHours) * time.Hour,

		CORS: middleware.CORSConfig{
			AllowedOrigins: splitEnvList(os.Getenv("CORS_ALLOWED_ORIGINS")),
			AllowedMethods: splitEnvList(os.Getenv("CORS_ALLOWED_METHODS")),
			AllowedHeaders: splitEnvList(os.Getenv("CORS_ALLOWED_HEADERS")),
		},
		Logger: logger.Config{ //TODO: after viper merge, rewrite using viper config
			Level:           os.Getenv("LOGGER_LEVEL"),
			Environment:     os.Getenv("LOGGER_ENVIRONMENT"),
			OutputPath:      os.Getenv("LOGGER_OUTPUT_PATH"),
			ErrorOutputPath: os.Getenv("LOGGER_ERROR_OUTPUT_PATH"),
		},
	}
	return cfg, nil
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
