package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/spf13/viper"
)

type Config struct {
	ServerPort string

	JWTSecret string
	JWTExpire time.Duration

	CORS middleware.CORSConfig
}

func Load() (*Config, error) {
	if err := initConfig(); err != nil {
		return nil, fmt.Errorf("Error initializing config: %v", err)
	}

	expHours := viper.GetInt("JWT_EXPIRE_HOURS")

	cfg := &Config{
		ServerPort: viper.GetString("APP_PORT"),

		JWTSecret: viper.GetString("JWT_SECRET"),
		JWTExpire: time.Duration(expHours) * time.Hour,

		CORS: middleware.CORSConfig{
			AllowedOrigins: splitEnvList(viper.GetString("CORS_ALLOWED_ORIGINS")),
			AllowedMethods: splitEnvList(viper.GetString("CORS_ALLOWED_METHODS")),
			AllowedHeaders: splitEnvList(viper.GetString("CORS_ALLOWED_HEADERS")),
		},
	}
	return cfg, nil
}

func initConfig() error {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AddConfigPath(".")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())

	return nil
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
