package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/spf13/viper"
)

type Config struct {
	ServerPort string `mapstructure:"port"`

	JWTSecret string        `mapstructure:"jwt.secret"`
	JWTExpire time.Duration `mapstructure:"jwt.expireHours"`

	CORS middleware.CORSConfig `mapstructure:"cors"`
}

func Load(path string) (*Config, error) {
	if err := initConfig(path); err != nil {
		return nil, fmt.Errorf("Error initializing config: %v", err)
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("Error unmarshaling config: %v", err)
	}

	fmt.Println(cfg.CORS.AllowedOrigins[0])

	return cfg, nil
}

func initConfig(path string) error {
	viper.SetConfigType("yaml")

	viper.SetConfigFile(path)

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

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
