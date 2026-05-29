package app

import (
	"fmt"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/postgres"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort string `mapstructure:"port"`

	JWTManager utils.JWTManager `mapstructure:"jwt"`

	CORS   middleware.CORSConfig `mapstructure:"cors"`
	Logger logger.Config         `mapstructure:"logger"`

	Db postgres.Config `mapstructure:"postgres"`
}

func Load(path string) (*Config, error) {

	if err := app.Init(path); err != nil {
		return nil, fmt.Errorf(
			"error initializing config: %w",
			err,
		)
	}

	cfg := &Config{}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf(
			"error unmarshaling config: %w",
			err,
		)
	}

	return cfg, nil
}
